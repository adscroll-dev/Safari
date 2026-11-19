package handlers

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/vyro/backend/internal/middleware"
	"github.com/vyro/backend/internal/store"
	"github.com/vyro/backend/internal/ws"
)

func newTestHandler() *Handler {
	h := New(store.NewMemoryStore(), ws.NewHub(), "test-secret", time.Hour)
	h.Mailer = newFakeMailer()
	return h
}

// fakeMailer captures verification emails in tests instead of sending
// them. The plain code is only ever read back by the test itself.
type fakeMailer struct {
	mu            sync.Mutex
	sent          []fakeMail
	lastCodeReads map[string]int
}

type fakeMail struct {
	to      string
	subject string
	body    string
}

func newFakeMailer() *fakeMailer { return &fakeMailer{} }

func (f *fakeMailer) Send(to, subject, body string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sent = append(f.sent, fakeMail{to: to, subject: subject, body: body})
	return nil
}

// lastCode returns the 6-digit code from the most recent email sent to
// the given address, or "" when none was sent.
func (f *fakeMailer) lastCode(to string) string {
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		f.mu.Lock()
		count := 0
		var latest string
		for _, sent := range f.sent {
			if sent.to == to {
				count++
				latest = sent.body
			}
		}
		if count > f.lastCodeReads[to] {
			if f.lastCodeReads == nil {
				f.lastCodeReads = make(map[string]int)
			}
			f.lastCodeReads[to] = count
			f.mu.Unlock()
			return codeRe.FindString(latest)
		}
		f.mu.Unlock()
		time.Sleep(time.Millisecond)
	}
	return ""
}

func (f *fakeMailer) count(to string) int {
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if count := f.countNow(to); count > 0 {
			return count
		}
		time.Sleep(time.Millisecond)
	}
	return f.countNow(to)
}

func (f *fakeMailer) countNow(to string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	n := 0
	for _, m := range f.sent {
		if m.to == to {
			n++
		}
	}
	return n
}

func (f *fakeMailer) firstBody(to string) string {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, sent := range f.sent {
		if sent.to == to {
			return sent.body
		}
	}
	return ""
}

type blockingMailer struct {
	started chan struct{}
	release chan struct{}
}

func (m *blockingMailer) Send(string, string, string) error {
	close(m.started)
	<-m.release
	return nil
}

var codeRe = regexp.MustCompile(`\b\d{6}\b`)

// testMailer returns the fake mailer attached to a test handler.
func testMailer(h *Handler) *fakeMailer {
	m, _ := h.Mailer.(*fakeMailer)
	return m
}

func doJSON(t *testing.T, h *Handler, method, path string, body interface{}, token string) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	// Routing must use the path without the query string (as net/http does).
	cleanPath := strings.Split(path, "?")[0]
	path = cleanPath
	// Route through the same auth middleware the server uses, except for
	// the public auth endpoints.
	switch {
	case path == "/api/auth/register":
		h.Register(rec, req)
	case path == "/api/auth/verify-register":
		h.VerifyRegister(rec, req)
	case path == "/api/auth/login":
		h.Login(rec, req)
	case path == "/api/auth/verify-login":
		h.VerifyLogin(rec, req)
	case path == "/api/auth/resend-code":
		h.ResendCode(rec, req)
	default:
		middleware.Auth("test-secret")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch {
			case path == "/api/chats" && method == http.MethodPost:
				h.CreateChat(w, r)
			case path == "/api/chats" && method == http.MethodGet:
				h.ListChats(w, r)
			case strings.HasPrefix(path, "/api/chats/") && strings.HasSuffix(path, "/messages"):
				if method == http.MethodPost {
					h.SendMessage(w, r)
				} else {
					h.ListMessages(w, r)
				}
			case strings.HasPrefix(path, "/api/chats/") && strings.HasSuffix(path, "/sender-keys"):
				if method == http.MethodPost {
					h.UploadGroupSenderKey(w, r)
				} else {
					h.ListGroupSenderKeys(w, r)
				}
			case strings.HasPrefix(path, "/api/chats/") && strings.HasSuffix(path, "/scheduled"):
				if method == http.MethodPost {
					h.ScheduleMessage(w, r)
				} else {
					h.ListScheduled(w, r)
				}
			case strings.HasPrefix(path, "/api/chats/") && strings.Contains(path, "/members/") && strings.HasSuffix(path, "/role"):
				h.SetMemberRole(w, r)
			case strings.HasPrefix(path, "/api/chats/") && strings.Contains(path, "/members/"):
				h.RemoveMember(w, r)
			case strings.HasPrefix(path, "/api/chats/") && strings.HasSuffix(path, "/members"):
				if method == http.MethodPost {
					h.AddMember(w, r)
				} else {
					h.ListMembers(w, r)
				}
			case strings.HasPrefix(path, "/api/chats/") && strings.HasSuffix(path, "/pin"):
				h.SetChatSetting(w, r, "pin")
			case strings.HasPrefix(path, "/api/chats/") && strings.HasSuffix(path, "/unpin"):
				h.SetChatSetting(w, r, "unpin")
			case strings.HasPrefix(path, "/api/chats/") && strings.HasSuffix(path, "/archive"):
				h.SetChatSetting(w, r, "archive")
			case strings.HasPrefix(path, "/api/chats/") && strings.HasSuffix(path, "/unarchive"):
				h.SetChatSetting(w, r, "unarchive")
			case strings.HasPrefix(path, "/api/chats/") && strings.HasSuffix(path, "/mute"):
				h.SetChatSetting(w, r, "mute")
			case strings.HasPrefix(path, "/api/chats/") && strings.HasSuffix(path, "/unmute"):
				h.SetChatSetting(w, r, "unmute")
			case strings.HasPrefix(path, "/api/chats/") && strings.HasSuffix(path, "/pin-message"):
				if method == http.MethodPost {
					h.PinMessage(w, r)
				} else {
					h.UnpinMessage(w, r)
				}
			case strings.HasPrefix(path, "/api/chats/") && strings.HasSuffix(path, "/folder"):
				h.SetChatSetting(w, r, "folder")
			case strings.HasPrefix(path, "/api/chats/") && strings.HasSuffix(path, "/polls"):
				if method == http.MethodPost {
					h.CreatePoll(w, r)
				} else {
					h.ListPolls(w, r)
				}
			case strings.HasPrefix(path, "/api/chats/") && strings.HasSuffix(path, "/follow"):
				h.FollowChannel(w, r)
			case strings.HasPrefix(path, "/api/chats/") && strings.HasSuffix(path, "/unfollow"):
				h.UnfollowChannel(w, r)
			case strings.HasPrefix(path, "/api/chats/") && method == http.MethodPatch:
				h.UpdateChat(w, r)
			case strings.HasPrefix(path, "/api/polls/") && strings.HasSuffix(path, "/vote"):
				h.VotePoll(w, r)
			case strings.HasPrefix(path, "/api/polls/"):
				if method == http.MethodDelete {
					h.DeletePoll(w, r)
				} else {
					h.GetPoll(w, r)
				}
			case strings.HasPrefix(path, "/api/chats/") && strings.HasSuffix(path, "/invites"):
				if method == http.MethodPost {
					h.CreateInvite(w, r)
				} else {
					h.ListInvites(w, r)
				}
			case strings.HasPrefix(path, "/api/chats/") && strings.Contains(path, "/invites/"):
				h.RevokeInvite(w, r)
			case strings.HasPrefix(path, "/api/invites/") && strings.HasSuffix(path, "/join"):
				h.JoinViaInvite(w, r)
			case strings.HasPrefix(path, "/api/invites/"):
				h.PreviewInvite(w, r)
			case path == "/api/stories":
				if method == http.MethodPost {
					h.CreateStory(w, r)
				} else {
					h.ListStories(w, r)
				}
			case strings.HasPrefix(path, "/api/stories/"):
				switch StorySuffix(r) {
				case "":
					h.DeleteStory(w, r)
				case "view":
					h.RecordStoryView(w, r)
				case "views":
					h.ListStoryViews(w, r)
				case "reactions":
					switch method {
					case http.MethodPost:
						h.SetStoryReaction(w, r)
					case http.MethodDelete:
						h.DeleteStoryReaction(w, r)
					default:
						h.ListStoryReactions(w, r)
					}
				case "replies":
					h.ListStoryReplies(w, r)
				case "reply":
					h.ReplyToStory(w, r)
				default:
					middleware.WriteError(w, http.StatusNotFound, "not found")
				}
			case path == "/api/users/me/status-privacy":
				if method == http.MethodPut {
					h.SetStatusPrivacy(w, r)
				} else {
					h.GetStatusPrivacy(w, r)
				}
			case path == "/api/calls":
				if method == http.MethodPost {
					h.CreateCallRecord(w, r)
				} else {
					h.ListCallRecords(w, r)
				}
			case path == "/api/bots":
				if method == http.MethodPost {
					h.CreateBot(w, r)
				} else {
					h.ListBots(w, r)
				}
			case strings.HasPrefix(path, "/api/bots/") && strings.HasSuffix(path, "/webhook"):
				h.SetBotWebhook(w, r)
			case strings.HasPrefix(path, "/api/bots/"):
				h.DeleteBot(w, r)
			case path == "/api/keys/identity":
				h.UploadIdentityKey(w, r)
			case path == "/api/keys/prekeys":
				h.UploadPreKeys(w, r)
			case strings.HasPrefix(path, "/api/keys/") && strings.HasSuffix(path, "/bundle"):
				h.GetKeyBundle(w, r)
			case strings.HasPrefix(path, "/api/keys/") && strings.HasSuffix(path, "/prekey"):
				h.FetchPreKey(w, r)
			case strings.HasPrefix(path, "/api/keys/"):
				h.GetIdentityKey(w, r)
			case path == "/api/devices":
				if method == http.MethodPost {
					h.RegisterDevice(w, r)
				} else {
					h.ListMyDevices(w, r)
				}
			case strings.HasPrefix(path, "/api/devices/"):
				h.RemoveDevice(w, r)
			case path == "/api/push/vapid-public-key":
				h.GetVAPIDPublicKey(w, r)
			case path == "/api/push/subscribe":
				if method == http.MethodPost {
					h.SubscribePush(w, r)
				} else if method == http.MethodDelete {
					h.UnsubscribePush(w, r)
				} else {
					middleware.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
				}
			case path == "/api/push/fcm-token":
				if method == http.MethodPost {
					h.RegisterFCMToken(w, r)
				} else if method == http.MethodDelete {
					h.UnregisterFCMToken(w, r)
				} else {
					middleware.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
				}
			case path == "/api/push/test":
				h.TestPush(w, r)
			case path == "/api/users/me/privacy":
				if method == http.MethodPut {
					h.SetPrivacy(w, r)
				} else {
					h.GetPrivacy(w, r)
				}
			case path == "/api/users/me/privacy-settings":
				if method == http.MethodPut {
					h.SetPrivacySettings(w, r)
				} else {
					h.GetPrivacySettings(w, r)
				}
			case path == "/api/users/me":
				if method == http.MethodPatch {
					h.UpdateMe(w, r)
				} else if method == http.MethodDelete {
					h.DeleteMe(w, r)
				} else {
					h.Me(w, r)
				}
			case path == "/api/users":
				h.ListUsers(w, r)
			case path == "/api/users/me/blocked":
				h.GetBlockedUsers(w, r)
			case path == "/api/users/me/change-email":
				h.RequestEmailChange(w, r)
			case path == "/api/users/me/change-email/verify":
				h.VerifyEmailChange(w, r)
			case path == "/api/users/me/export":
				h.ExportMyData(w, r)
			case path == "/api/users/me/storage":
				h.StorageStats(w, r)
			case strings.HasPrefix(path, "/api/users/") && strings.HasSuffix(path, "/report"):
				h.ReportUser(w, r)
			case strings.HasPrefix(path, "/api/users/") && strings.HasSuffix(path, "/block"):
				if method == http.MethodDelete {
					h.UnblockUser(w, r)
				} else {
					h.BlockUser(w, r)
				}
			case strings.HasPrefix(path, "/api/users/") && strings.HasSuffix(path, "/presence"):
				h.Presence(w, r)
			case strings.HasPrefix(path, "/api/users/") && strings.HasSuffix(path, "/devices"):
				h.ListUserDevices(w, r)
			case strings.HasPrefix(path, "/api/scheduled/"):
				h.CancelScheduled(w, r)
			case strings.HasPrefix(path, "/api/messages/") && strings.HasSuffix(path, "/read"):
				h.MarkRead(w, r)
			case strings.HasPrefix(path, "/api/messages/") && strings.HasSuffix(path, "/delete-for-me"):
				h.DeleteMessageForMe(w, r)
			case strings.HasPrefix(path, "/api/messages/") && strings.HasSuffix(path, "/view"):
				h.RecordMessageView(w, r)
			case strings.HasPrefix(path, "/api/messages/") && strings.HasSuffix(path, "/views"):
				h.GetMessageViews(w, r)
			case path == "/api/channels/search":
				h.SearchChannels(w, r)
			case path == "/api/communities" && method == http.MethodPost:
				h.CreateCommunity(w, r)
			case path == "/api/communities" && method == http.MethodGet:
				h.ListCommunities(w, r)
			case path == "/api/communities/discover":
				h.DiscoverCommunities(w, r)
			case strings.HasPrefix(path, "/api/communities/") && strings.HasSuffix(path, "/join"):
				h.JoinCommunity(w, r)
			case strings.HasPrefix(path, "/api/communities/") && strings.HasSuffix(path, "/leave"):
				h.LeaveCommunity(w, r)
			case strings.HasPrefix(path, "/api/communities/") && strings.Contains(path, "/members/") && strings.HasSuffix(path, "/role"):
				h.SetCommunityMemberRole(w, r)
			case strings.HasPrefix(path, "/api/communities/") && strings.Contains(path, "/members/"):
				h.RemoveCommunityMember(w, r)
			case strings.HasPrefix(path, "/api/communities/") && strings.HasSuffix(path, "/members"):
				if method == http.MethodPost {
					h.AddCommunityMember(w, r)
				} else {
					h.ListCommunityMembers(w, r)
				}
			case strings.HasPrefix(path, "/api/communities/") && strings.Contains(path, "/subgroups/"):
				h.RemoveSubgroup(w, r)
			case strings.HasPrefix(path, "/api/communities/") && strings.HasSuffix(path, "/subgroups"):
				h.AddSubgroup(w, r)
			case strings.HasPrefix(path, "/api/communities/"):
				if method == http.MethodPatch {
					h.UpdateCommunity(w, r)
				} else {
					h.GetCommunity(w, r)
				}
			case path == "/api/messages/search":
				h.SearchMessages(w, r)
			case strings.HasPrefix(path, "/api/messages/"):
				if method == http.MethodPut {
					h.EditMessage(w, r)
				} else if method == http.MethodDelete {
					h.DeleteMessage(w, r)
				}
			default:
				middleware.WriteError(w, http.StatusNotFound, "not found")
			}
		})).ServeHTTP(rec, req)
	}
	return rec
}

// register runs the full v0.7 registration flow: register (which only
// sends a code) then verify-register with the code the fake mailer
// captured. It returns the JWT and user id, like the old helper did.
func register(t *testing.T, h *Handler, email, username string) (token, userID string) {
	t.Helper()
	rec := doJSON(t, h, http.MethodPost, "/api/auth/register", map[string]string{
		"email": email, "username": username, "password": "password-123", "display_name": username,
	}, "")
	if rec.Code != http.StatusCreated {
		t.Fatalf("register %s: status %d body %s", email, rec.Code, rec.Body.String())
	}
	var sent struct {
		Status  string `json:"status"`
		Purpose string `json:"purpose"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&sent); err != nil {
		t.Fatalf("decode register response: %v", err)
	}
	if sent.Status != "code_sent" || sent.Purpose != "register" {
		t.Fatalf("register %s: expected code_sent/register, got %+v", email, sent)
	}
	code := testMailer(h).lastCode(email)
	if code == "" {
		t.Fatalf("register %s: no verification email was sent", email)
	}
	rec = doJSON(t, h, http.MethodPost, "/api/auth/verify-register", map[string]string{
		"email": email, "code": code,
	}, "")
	if rec.Code != http.StatusCreated {
		t.Fatalf("verify-register %s: status %d body %s", email, rec.Code, rec.Body.String())
	}
	var out struct {
		Token string `json:"token"`
		User  struct {
			ID string `json:"id"`
		} `json:"user"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&out); err != nil {
		t.Fatalf("decode verify-register response: %v", err)
	}
	if out.Token == "" || out.User.ID == "" {
		t.Fatalf("verify-register %s: missing token or user id: %s", email, rec.Body.String())
	}
	return out.Token, out.User.ID
}

// login runs the full v0.7 login flow: login (which only sends a code)
// then verify-login with the captured code. It returns the JWT.
func login(t *testing.T, h *Handler, email, password string) string {
	t.Helper()
	rec := doJSON(t, h, http.MethodPost, "/api/auth/login", map[string]string{
		"email": email, "password": password,
	}, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("login %s: expected 200, got %d %s", email, rec.Code, rec.Body.String())
	}
	var sent struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&sent); err != nil {
		t.Fatalf("decode login response: %v", err)
	}
	if sent.Status != "code_sent" {
		t.Fatalf("login %s: expected code_sent, got %+v", email, sent)
	}
	code := testMailer(h).lastCode(email)
	if code == "" {
		t.Fatalf("login %s: no verification email was sent", email)
	}
	rec = doJSON(t, h, http.MethodPost, "/api/auth/verify-login", map[string]string{
		"email": email, "code": code,
	}, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("verify-login %s: status %d body %s", email, rec.Code, rec.Body.String())
	}
	var out struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&out); err != nil {
		t.Fatalf("decode verify-login response: %v", err)
	}
	if out.Token == "" {
		t.Fatalf("verify-login %s: missing token", email)
	}
	return out.Token
}

func TestRegisterLoginAndMessageFlow(t *testing.T) {
	h := newTestHandler()
	aliceTok, aliceID := register(t, h, "alice@example.com", "alice")
	bobTok, bobID := register(t, h, "bob@example.com", "bob")
	_ = bobTok

	// Duplicate registration is a conflict.
	rec := doJSON(t, h, http.MethodPost, "/api/auth/register", map[string]string{
		"email": "alice@example.com", "username": "alice2", "password": "password-123", "display_name": "Alice Two",
	}, "")
	if rec.Code != http.StatusConflict {
		t.Fatalf("duplicate register: expected 409, got %d", rec.Code)
	}
	// Short password is rejected.
	rec = doJSON(t, h, http.MethodPost, "/api/auth/register", map[string]string{
		"email": "c@example.com", "username": "carol", "password": "short", "display_name": "Carol",
	}, "")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("short password: expected 400, got %d", rec.Code)
	}
	// Wrong password login fails.
	rec = doJSON(t, h, http.MethodPost, "/api/auth/login", map[string]string{
		"email": "alice@example.com", "password": "nope-nope-nope",
	}, "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("wrong password: expected 401, got %d", rec.Code)
	}
	// Correct password only sends a code — no token yet (v0.7).
	rec = doJSON(t, h, http.MethodPost, "/api/auth/login", map[string]string{
		"email": "alice@example.com", "password": "password-123",
	}, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("login: expected 200, got %d %s", rec.Code, rec.Body.String())
	}
	var loginSent struct {
		Status string `json:"status"`
		Token  string `json:"token"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&loginSent); err != nil {
		t.Fatalf("decode login response: %v", err)
	}
	if loginSent.Status != "code_sent" || loginSent.Token != "" {
		t.Fatalf("login must return code_sent and no token, got %+v", loginSent)
	}

	// Create a DM, send, list, edit, search.
	rec = doJSON(t, h, http.MethodPost, "/api/chats", map[string]interface{}{
		"type": "dm", "member_ids": []string{bobID},
	}, aliceTok)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create chat: %d %s", rec.Code, rec.Body.String())
	}
	var chat struct {
		ID string `json:"id"`
	}
	_ = json.NewDecoder(rec.Body).Decode(&chat)

	rec = doJSON(t, h, http.MethodPost, "/api/chats/"+chat.ID+"/messages", map[string]string{
		"content": "سلام بوب!",
	}, aliceTok)
	if rec.Code != http.StatusCreated {
		t.Fatalf("send message: %d %s", rec.Code, rec.Body.String())
	}
	var msg struct {
		ID string `json:"id"`
	}
	_ = json.NewDecoder(rec.Body).Decode(&msg)

	// A stranger cannot read the chat.
	_, strangerID := register(t, h, "eve@example.com", "eve")
	_ = strangerID
	// Eve can log in through the full code flow, but still cannot read
	// a chat she is not a member of.
	eveLoginTok := login(t, h, "eve@example.com", "password-123")
	rec = doJSON(t, h, http.MethodGet, "/api/chats/"+chat.ID+"/messages", nil, eveLoginTok)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("non-member read: expected 403, got %d", rec.Code)
	}

	// Edit by sender works; search finds the message.
	rec = doJSON(t, h, http.MethodPut, "/api/messages/"+msg.ID, map[string]string{
		"content": "سلام بوب! (ویرایش شد)",
	}, aliceTok)
	if rec.Code != http.StatusOK {
		t.Fatalf("edit message: %d %s", rec.Code, rec.Body.String())
	}
	rec = doJSON(t, h, http.MethodGet, "/api/messages/search?q="+url.QueryEscape("ویرایش"), nil, aliceTok)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "ویرایش") {
		t.Fatalf("search: expected edited message, got %d %s", rec.Code, rec.Body.String())
	}
	// Bob cannot edit Alice's message.
	rec = doJSON(t, h, http.MethodPut, "/api/messages/"+msg.ID, map[string]string{
		"content": "hacked",
	}, bobTok)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("edit by non-sender: expected 403, got %d", rec.Code)
	}
	_ = aliceID
}

func TestUnauthenticatedIsRejected(t *testing.T) {
	h := newTestHandler()
	rec := doJSON(t, h, http.MethodGet, "/api/chats", nil, "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without token, got %d", rec.Code)
	}
}

func uploadFile(t *testing.T, h *Handler, token, filename string, content []byte) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, err := mw.CreateFormFile("file", filename)
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := fw.Write(content); err != nil {
		t.Fatalf("write form file: %v", err)
	}
	_ = mw.Close()
	req := httptest.NewRequest(http.MethodPost, "/api/uploads", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	middleware.Auth("test-secret")(http.HandlerFunc(h.Upload)).ServeHTTP(rec, req)
	return rec
}

func TestFileUploadAndImageMessage(t *testing.T) {
	h := newTestHandler()
	h.UploadDir = t.TempDir()
	aliceTok, _ := register(t, h, "alice@example.com", "alice")
	_, bobID := register(t, h, "bob@example.com", "bob")

	rec := doJSON(t, h, http.MethodPost, "/api/chats", map[string]interface{}{
		"type": "dm", "member_ids": []string{bobID},
	}, aliceTok)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create chat: %d %s", rec.Code, rec.Body.String())
	}
	var chat struct {
		ID string `json:"id"`
	}
	_ = json.NewDecoder(rec.Body).Decode(&chat)

	// A tiny valid PNG (header + IHDR chunk bytes are enough for sniffing).
	png := []byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a, 0, 0, 0, 13, 'I', 'H', 'D', 'R'}
	rec = uploadFile(t, h, aliceTok, "photo.png", png)
	if rec.Code != http.StatusCreated {
		t.Fatalf("upload png: expected 201, got %d %s", rec.Code, rec.Body.String())
	}
	var up struct {
		URL      string `json:"url"`
		FileName string `json:"file_name"`
		FileSize int64  `json:"file_size"`
		MimeType string `json:"mime_type"`
		Kind     string `json:"kind"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&up); err != nil {
		t.Fatalf("decode upload: %v", err)
	}
	if !strings.HasPrefix(up.URL, "/uploads/") || up.Kind != "image" || up.FileSize != int64(len(png)) {
		t.Fatalf("unexpected upload response: %+v", up)
	}
	if _, err := os.Stat(filepath.Join(h.UploadDir, filepath.Base(up.URL))); err != nil {
		t.Fatalf("uploaded file not on disk: %v", err)
	}

	// The stored file downloads back byte-for-byte (auth required).
	req := httptest.NewRequest(http.MethodGet, up.URL, nil)
	req.Header.Set("Authorization", "Bearer "+aliceTok)
	down := httptest.NewRecorder()
	middleware.Auth("test-secret")(http.HandlerFunc(h.ServeUpload)).ServeHTTP(down, req)
	if down.Code != http.StatusOK || !bytes.Equal(down.Body.Bytes(), png) {
		t.Fatalf("download: expected 200 with png bytes, got %d len %d", down.Code, down.Body.Len())
	}
	// Path traversal is rejected.
	req = httptest.NewRequest(http.MethodGet, "/uploads/../handlers.go", nil)
	req.Header.Set("Authorization", "Bearer "+aliceTok)
	trav := httptest.NewRecorder()
	middleware.Auth("test-secret")(http.HandlerFunc(h.ServeUpload)).ServeHTTP(trav, req)
	if trav.Code != http.StatusNotFound {
		t.Fatalf("traversal: expected 404, got %d", trav.Code)
	}

	// Send the uploaded photo as an image message with a Persian caption.
	rec = doJSON(t, h, http.MethodPost, "/api/chats/"+chat.ID+"/messages", map[string]interface{}{
		"content": "عکس", "kind": "image", "file_url": up.URL,
		"file_name": up.FileName, "file_size": up.FileSize, "mime_type": up.MimeType,
	}, aliceTok)
	if rec.Code != http.StatusCreated {
		t.Fatalf("send image message: %d %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), up.URL) || !strings.Contains(rec.Body.String(), `"kind":"image"`) {
		t.Fatalf("image message missing file metadata: %s", rec.Body.String())
	}
	// An image message without a file_url is rejected.
	rec = doJSON(t, h, http.MethodPost, "/api/chats/"+chat.ID+"/messages", map[string]string{
		"content": "عکس", "kind": "image",
	}, aliceTok)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("image without file_url: expected 400, got %d", rec.Code)
	}
	// Executables are rejected at upload, and uploads need a login.
	rec = uploadFile(t, h, aliceTok, "evil.exe", []byte("MZ fake"))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("exe upload: expected 400, got %d %s", rec.Code, rec.Body.String())
	}
	rec = uploadFile(t, h, "", "photo.png", png)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated upload: expected 401, got %d", rec.Code)
	}
}
