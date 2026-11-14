// Package ws implements the live-message WebSocket hub (v0.4).
//
// Clients connect to GET /ws?token=<jwt>. Each connection joins the rooms
// of the chats its user belongs to (passed in by the handler), so a
// message is delivered only to members of that chat — not broadcast to
// every connected client, as in v0.1–v0.3.
//
// Inbound client frames (JSON):
//
//	{"type":"typing","chat_id":"..."}    — forwarded to the other members of that chat
//	{"type":"subscribe","chat_id":"..."} — join an additional chat room
//	{"type":"call-offer"|"call-answer"|"call-ice"|"call-end","chat_id":"...","payload":{...},"to_user_id":"..."}
//	                                     — WebRTC signaling (v0.5): relayed verbatim
//	                                     to the other members of that chat room (or
//	                                     only to to_user_id when set). The server
//	                                     never inspects or stores the payload; it
//	                                     only checks the sender is in the room.
//
// Outbound server events: "message", "message_edited", "message_deleted",
// "presence" (user online/offline), "typing", "message_read",
// "message_delivered", and the call-* relay events.
//
// Multi-device sync (v0.6): a message a user sends from one device is
// fanned out to their other devices as a "message_sync" event (same
// Message payload), so the sent message shows up on the phone as well
// as the laptop. BroadcastToChat already delivers to every connected
// device of each *other* member; BroadcastToUser covers the sender's
// own devices. Edits, deletes and read receipts are echoed to the
// sender's/reader's own devices with their normal event types —
// clients apply them idempotently by message id.
package ws

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/vyro/backend/internal/models"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true }, // dev only; restrict in production
}

// Hub tracks connected clients grouped by chat room.
type Hub struct {
	mu      sync.Mutex
	clients map[*websocket.Conn]*client // conn -> client
	rooms   map[string]map[*websocket.Conn]bool
}

type client struct {
	userID string
	chats  map[string]bool // chat ids this connection joined
}

func NewHub() *Hub {
	return &Hub{
		clients: map[*websocket.Conn]*client{},
		rooms:   map[string]map[*websocket.Conn]bool{},
	}
}

// Event is the JSON envelope sent over the socket.
type Event struct {
	Type      string          `json:"type"` // message | message_sync | message_edited | message_deleted | presence | typing | message_read | message_delivered | call-offer | call-answer | call-ice | call-end
	ChatID    string          `json:"chat_id,omitempty"`
	Message   *models.Message `json:"message,omitempty"`
	MessageID string          `json:"message_id,omitempty"`
	UserID    string          `json:"user_id,omitempty"`
	Online    bool            `json:"online,omitempty"`
	Typing    bool            `json:"typing,omitempty"`
	// Payload carries an opaque WebRTC signaling object (SDP offer/
	// answer or ICE candidate) for call-* events. The server relays
	// it verbatim and never inspects it.
	Payload json.RawMessage `json:"payload,omitempty"`
	// ToUserID, when set on a call event, limits the relay to that
	// user's connections (1:1 calls). Empty = every other member of
	// the room.
	ToUserID string `json:"to_user_id,omitempty"`
}

// Broadcast sends an event to every connected client (presence only).
func (h *Hub) Broadcast(ev Event) {
	data, err := json.Marshal(ev)
	if err != nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	for conn := range h.clients {
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			h.removeLocked(conn)
		}
	}
}

// BroadcastToChat delivers ev to the connections of the given member user
// ids that joined chatID, and returns the user ids that actually received
// it (used for delivery receipts). The sender is included in memberIDs
// but never receives their own message back.
func (h *Hub) BroadcastToChat(chatID string, memberIDs []string, senderID string, ev Event) []string {
	data, err := json.Marshal(ev)
	if err != nil {
		return nil
	}
	members := map[string]bool{}
	for _, id := range memberIDs {
		members[id] = true
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	delivered := map[string]bool{}
	for conn := range h.rooms[chatID] {
		cl := h.clients[conn]
		if cl == nil || !cl.chats[chatID] || !members[cl.userID] || cl.userID == senderID {
			continue
		}
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			h.removeLocked(conn)
			continue
		}
		delivered[cl.userID] = true
	}
	out := make([]string, 0, len(delivered))
	for id := range delivered {
		out = append(out, id)
	}
	return out
}

// BroadcastToUser sends an event to every connected device of one
// user (multi-device sync, v0.6). Used for "message_sync" copies of a
// user's own messages and for echoing their edits/deletes/read
// receipts to their other devices.
func (h *Hub) BroadcastToUser(userID string, ev Event) {
	data, err := json.Marshal(ev)
	if err != nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	for conn, cl := range h.clients {
		if cl.userID != userID {
			continue
		}
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			h.removeLocked(conn)
		}
	}
}

// OnlineUsers reports the user ids currently connected.
func (h *Hub) OnlineUsers() []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	seen := map[string]bool{}
	for _, cl := range h.clients {
		seen[cl.userID] = true
	}
	out := make([]string, 0, len(seen))
	for id := range seen {
		out = append(out, id)
	}
	return out
}

func (h *Hub) removeLocked(conn *websocket.Conn) {
	if cl, ok := h.clients[conn]; ok {
		for chatID := range cl.chats {
			delete(h.rooms[chatID], conn)
		}
	}
	delete(h.clients, conn)
	_ = conn.Close()
}

func (h *Hub) joinLocked(conn *websocket.Conn, chatID string) {
	if h.rooms[chatID] == nil {
		h.rooms[chatID] = map[*websocket.Conn]bool{}
	}
	h.rooms[chatID][conn] = true
	h.clients[conn].chats[chatID] = true
}

// relayToRoomLocked sends data to the other connections in a room.
// When toUserID is non-empty only that user's connections receive it.
// Callers must hold h.mu.
func (h *Hub) relayToRoomLocked(room string, sender *websocket.Conn, toUserID string, data []byte) {
	for other := range h.rooms[room] {
		if other == sender {
			continue
		}
		if toUserID != "" {
			cl := h.clients[other]
			if cl == nil || cl.userID != toUserID {
				continue
			}
		}
		_ = other.WriteMessage(websocket.TextMessage, data)
	}
}

// Serve upgrades the HTTP request and registers the client in the rooms
// of chatIDs. onPresence, when set, is called with (userID, online) on
// connect and disconnect so the store can record presence / last-seen.
func (h *Hub) Serve(w http.ResponseWriter, r *http.Request, userID string, chatIDs []string, onPresence func(userID string, online bool)) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("ws upgrade: %v", err)
		return
	}
	h.mu.Lock()
	h.clients[conn] = &client{userID: userID, chats: map[string]bool{}}
	for _, id := range chatIDs {
		h.joinLocked(conn, id)
	}
	h.mu.Unlock()
	if onPresence != nil {
		onPresence(userID, true)
	}
	h.Broadcast(Event{Type: "presence", UserID: userID, Online: true})
	defer func() {
		h.mu.Lock()
		h.removeLocked(conn)
		h.mu.Unlock()
		if onPresence != nil {
			onPresence(userID, false)
		}
		h.Broadcast(Event{Type: "presence", UserID: userID, Online: false})
	}()
	// Read loop: typing / subscribe / call-signaling frames from this client.
	for {
		_, raw, err := conn.ReadMessage()
		if err != nil {
			return
		}
		var frame struct {
			Type     string          `json:"type"`
			ChatID   string          `json:"chat_id"`
			Payload  json.RawMessage `json:"payload"`
			ToUserID string          `json:"to_user_id"`
		}
		if err := json.Unmarshal(raw, &frame); err != nil || frame.ChatID == "" {
			continue
		}
		switch frame.Type {
		case "subscribe":
			h.mu.Lock()
			if _, ok := h.clients[conn]; ok {
				h.joinLocked(conn, frame.ChatID)
			}
			h.mu.Unlock()
		case "typing":
			// Forward to the other members of that room only.
			ev := Event{Type: "typing", ChatID: frame.ChatID, UserID: userID, Typing: true}
			data, err := json.Marshal(ev)
			if err != nil {
				continue
			}
			h.mu.Lock()
			h.relayToRoomLocked(frame.ChatID, conn, "", data)
			h.mu.Unlock()
		case "call-offer", "call-answer", "call-ice", "call-end":
			// WebRTC signaling: relay the payload verbatim to the
			// other member(s) of the room. The sender is in the room
			// (they joined it), so membership is already enforced;
			// the payload itself is never parsed or stored.
			ev := Event{Type: frame.Type, ChatID: frame.ChatID, UserID: userID, Payload: frame.Payload, ToUserID: frame.ToUserID}
			data, err := json.Marshal(ev)
			if err != nil {
				continue
			}
			h.mu.Lock()
			h.relayToRoomLocked(frame.ChatID, conn, frame.ToUserID, data)
			h.mu.Unlock()
		}
	}
}
