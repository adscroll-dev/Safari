package ws

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestHubBusyCallRelayAndRelease(t *testing.T) {
	h := NewHub()
	const chatID = "busy-chat"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.Serve(w, r, r.URL.Query().Get("user"), []string{chatID}, nil, nil)
	}))
	defer srv.Close()
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")

	dial := func(userID string) *websocket.Conn {
		conn, _, err := websocket.DefaultDialer.Dial(wsURL+"?user="+userID, nil)
		if err != nil {
			t.Fatalf("dial %s: %v", userID, err)
		}
		return conn
	}
	readEvent := func(conn *websocket.Conn) Event {
		t.Helper()
		for {
			_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
			var ev Event
			if err := conn.ReadJSON(&ev); err != nil {
				t.Fatalf("read event: %v", err)
			}
			if ev.Type != "presence" {
				return ev
			}
		}
	}

	alice := dial("alice")
	bob := dial("bob")
	carol := dial("carol")
	defer alice.Close()
	defer bob.Close()
	defer carol.Close()
	time.Sleep(100 * time.Millisecond)

	if err := alice.WriteJSON(map[string]interface{}{
		"type": "call-offer", "chat_id": chatID, "to_user_id": "bob",
		"payload": map[string]interface{}{"sdp": "offer"},
	}); err != nil {
		t.Fatal(err)
	}
	_ = readEvent(bob) // call-offer

	if err := carol.WriteJSON(map[string]interface{}{
		"type": "call-offer", "chat_id": chatID, "to_user_id": "bob",
		"payload": map[string]interface{}{"sdp": "second"},
	}); err != nil {
		t.Fatal(err)
	}
	ev := readEvent(carol)
	if ev.Type != "call-busy" || ev.ChatID != chatID || ev.UserID != "bob" {
		t.Fatalf("unexpected busy event: %+v", ev)
	}
	_ = bob.SetReadDeadline(time.Now().Add(300 * time.Millisecond))
	var extra Event
	if err := bob.ReadJSON(&extra); err == nil {
		t.Fatalf("bob received blocked offer: %+v", extra)
	}

	if err := alice.WriteJSON(map[string]interface{}{
		"type": "call-end", "chat_id": chatID, "to_user_id": "bob",
		"payload": map[string]interface{}{"reason": "hangup"},
	}); err != nil {
		t.Fatal(err)
	}
	time.Sleep(50 * time.Millisecond)
	h.callMu.Lock()
	_, bobStillBusy := h.activeCalls["bob"]
	h.callMu.Unlock()
	if bobStillBusy {
		t.Fatal("call-end did not release bob")
	}

	if err := bob.WriteJSON(map[string]interface{}{
		"type": "call-offer", "chat_id": chatID, "to_user_id": "alice",
		"payload": map[string]interface{}{"sdp": "after-end"},
	}); err != nil {
		t.Fatal(err)
	}
	_ = readEvent(alice)
}

func TestHubDisconnectAndExpiryFreeSlots(t *testing.T) {
	h := NewHub()
	h.now = func() time.Time { return time.Unix(1000, 0) }

	if _, ok := h.ReserveCall("alice", "bob", "chat"); !ok {
		t.Fatal("initial reserve failed")
	}
	h.ReleaseCall("alice", "", "")
	if h.activeCalls["alice"] != nil || h.activeCalls["bob"] != nil {
		t.Fatal("release did not free both users")
	}

	if _, ok := h.ReserveCall("alice", "bob", "chat"); !ok {
		t.Fatal("reserve failed")
	}
	h.callMu.Lock()
	call := h.activeCalls["alice"]
	call.OfferedAt = h.now().Add(-61 * time.Second)
	h.callMu.Unlock()
	if busy, ok := h.ReserveCall("carol", "bob", "chat"); !ok || busy != "" {
		t.Fatalf("expired ringing slot remained busy: busy=%q ok=%v", busy, ok)
	}

	h.ReleaseCall("carol", "bob", "chat")
	h.now = func() time.Time { return time.Unix(2000, 0) }
	if _, ok := h.ReserveCall("alice", "bob", "chat"); !ok {
		t.Fatal("reserve after expiry failed")
	}
	h.callMu.Lock()
	call = h.activeCalls["alice"]
	call.Answered = true
	call.AnsweredAt = h.now().Add(-5 * time.Hour)
	h.callMu.Unlock()
	if _, ok := h.ReserveCall("carol", "bob", "chat"); !ok {
		t.Fatal("expired answered slot should be free")
	}
}

func TestHubDisconnectFreesSlot(t *testing.T) {
	h := NewHub()
	const chatID = "disconnect-chat"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.Serve(w, r, r.URL.Query().Get("user"), []string{chatID}, nil, nil)
	}))
	defer srv.Close()
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")

	alice, _, err := websocket.DefaultDialer.Dial(wsURL+"?user=alice", nil)
	if err != nil {
		t.Fatal(err)
	}
	bob, _, err := websocket.DefaultDialer.Dial(wsURL+"?user=bob", nil)
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(100 * time.Millisecond)
	if _, ok := h.ReserveCall("alice", "bob", chatID); !ok {
		t.Fatal("reserve failed")
	}
	_ = alice.Close()
	time.Sleep(100 * time.Millisecond)
	h.callMu.Lock()
	_, aliceBusy := h.activeCalls["alice"]
	_, bobBusy := h.activeCalls["bob"]
	h.callMu.Unlock()
	if aliceBusy || bobBusy {
		t.Fatal("disconnect did not release slot")
	}
	_ = bob.Close()
}
