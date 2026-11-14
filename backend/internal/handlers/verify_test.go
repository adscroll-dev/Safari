package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/vyro/backend/internal/mail"
	"github.com/vyro/backend/internal/models"
	"github.com/vyro/backend/internal/store"
	"github.com/vyro/backend/internal/ws"
)

// TestVerifyHappyPath covers the full v0.7 flow: register sends a
// code (no user, no token), verify-register creates the user and
// issues a token, login sends a code (no token), verify-login issues
// a token.
func TestVerifyHappyPath(t *testing.T) {
	h := newTestHandler()
	email := "verify@example.com"

	// Register: 201 code_sent, no token, no user yet.
	rec := doJSON(t, h, http.MethodPost, "/api/auth/register", map[string]string{
		"email": email, "username": "verifyuser", "password": "password-123", "display_name": "Verify",
	}, "")
	if rec.Code != http.StatusCreated {
		t.Fatalf("register: expected 201, got %d %s", rec.Code, rec.Body.String())
	}
	var sent struct {
		Status           string `json:"status"`
		Email            string `json:"email"`
		Purpose          string `json:"purpose"`
		ExpiresInSeconds int    `json:"expires_in_seconds"`
		Token            string `json:"token"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&sent); err != nil {
		t.Fatalf("decode register: %v", err)
	}
	if sent.Status != "code_sent" || sent.Email != email || sent.Purpose != "register" ||
		sent.ExpiresInSeconds != 600 || sent.Token != "" {
		t.Fatalf("unexpected register response: %+v", sent)
	}
	if _, err := h.Store.UserByEmail(email); err == nil {
		t.Fatal("user must not exist before the code is verified")
	}
	// The email carries a 6-digit code, an expiry note, and the
	// never-share warning.
	mail := testMailer(h)
	if mail.count(email) != 1 {
		t.Fatalf("expected exactly 1 email, got %d", mail.count(email))
	}
	body := mail.sent[0].body
	if !strings.Contains(body, "expires in 10 minutes") || !strings.Contains(body, "Never share this code") {
		t.Fatalf("email body missing expiry/warning: %q", body)
	}
	code := mail.lastCode(email)
	if len(code) != 6 {
		t.Fatalf("expected a 6-digit code in the email, got %q", code)
	}
	// The stored code row holds a hash, never the plain code.
	row, err := h.Store.GetEmailCode(email, "register")
	if err != nil {
		t.Fatalf("GetEmailCode: %v", err)
	}
	if row.CodeHash == code || len(row.CodeHash) != 64 {
		t.Fatalf("code must be stored as a SHA-256 hex hash, got %q", row.CodeHash)
	}

	// Verify-register: 201 with user + token. The code is single-use.
	rec = doJSON(t, h, http.MethodPost, "/api/auth/verify-register", map[string]string{
		"email": email, "code": code,
	}, "")
	if rec.Code != http.StatusCreated {
		t.Fatalf("verify-register: expected 201, got %d %s", rec.Code, rec.Body.String())
	}
	var verified struct {
		Token string `json:"token"`
		User  struct {
			ID       string `json:"id"`
			Email    string `json:"email"`
			Username string `json:"username"`
		} `json:"user"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&verified); err != nil {
		t.Fatalf("decode verify-register: %v", err)
	}
	if verified.Token == "" || verified.User.Email != email || verified.User.Username != "verifyuser" {
		t.Fatalf("unexpected verify-register response: %+v", verified)
	}
	rec = doJSON(t, h, http.MethodPost, "/api/auth/verify-register", map[string]string{
		"email": email, "code": code,
	}, "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("reused code: expected 401, got %d %s", rec.Code, rec.Body.String())
	}

	// Login: 200 code_sent, no token.
	rec = doJSON(t, h, http.MethodPost, "/api/auth/login", map[string]string{
		"email": email, "password": "password-123",
	}, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("login: expected 200, got %d %s", rec.Code, rec.Body.String())
	}
	var loginSent struct {
		Status string `json:"status"`
		Token  string `json:"token"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&loginSent); err != nil {
		t.Fatalf("decode login: %v", err)
	}
	if loginSent.Status != "code_sent" || loginSent.Token != "" {
		t.Fatalf("login must return code_sent and no token, got %+v", loginSent)
	}
	loginCode := mail.lastCode(email)
	if loginCode == "" || loginCode == code {
		t.Fatalf("login must send a fresh code, got %q (register code was %q)", loginCode, code)
	}

	// Verify-login: 200 with a token for the same user.
	rec = doJSON(t, h, http.MethodPost, "/api/auth/verify-login", map[string]string{
		"email": email, "code": loginCode,
	}, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("verify-login: expected 200, got %d %s", rec.Code, rec.Body.String())
	}
	var loginOut struct {
		Token string `json:"token"`
		User  struct {
			ID string `json:"id"`
		} `json:"user"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&loginOut); err != nil {
		t.Fatalf("decode verify-login: %v", err)
	}
	if loginOut.Token == "" || loginOut.User.ID != verified.User.ID {
		t.Fatalf("unexpected verify-login response: %+v", loginOut)
	}
}

func TestVerifyWrongCodeRejected(t *testing.T) {
	h := newTestHandler()
	email := "wrongcode@example.com"
	rec := doJSON(t, h, http.MethodPost, "/api/auth/register", map[string]string{
		"email": email, "username": "wrongcode", "password": "password-123",
	}, "")
	if rec.Code != http.StatusCreated {
		t.Fatalf("register: %d %s", rec.Code, rec.Body.String())
	}
	real := testMailer(h).lastCode(email)
	wrong := "000000"
	if wrong == real {
		wrong = "000001"
	}
	rec = doJSON(t, h, http.MethodPost, "/api/auth/verify-register", map[string]string{
		"email": email, "code": wrong,
	}, "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("wrong code: expected 401, got %d %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "invalid or expired code") {
		t.Fatalf("wrong code must use the generic message, got %s", rec.Body.String())
	}
	// The real code still works after one wrong guess.
	rec = doJSON(t, h, http.MethodPost, "/api/auth/verify-register", map[string]string{
		"email": email, "code": real,
	}, "")
	if rec.Code != http.StatusCreated {
		t.Fatalf("real code after one wrong guess: expected 201, got %d %s", rec.Code, rec.Body.String())
	}
}

func TestVerifyExpiredCodeRejected(t *testing.T) {
	h := newTestHandler()
	email := "expired@example.com"
	rec := doJSON(t, h, http.MethodPost, "/api/auth/register", map[string]string{
		"email": email, "username": "expired", "password": "password-123",
	}, "")
	if rec.Code != http.StatusCreated {
		t.Fatalf("register: %d %s", rec.Code, rec.Body.String())
	}
	code := testMailer(h).lastCode(email)
	// Age the stored code row past its expiry, as if 10 minutes passed.
	row, err := h.Store.GetEmailCode(email, "register")
	if err != nil {
		t.Fatalf("GetEmailCode: %v", err)
	}
	row.ExpiresAt = time.Now().UTC().Add(-time.Minute)
	if err := h.Store.SaveEmailCode(row); err != nil {
		t.Fatalf("SaveEmailCode: %v", err)
	}
	rec = doJSON(t, h, http.MethodPost, "/api/auth/verify-register", map[string]string{
		"email": email, "code": code,
	}, "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expired code: expected 401, got %d %s", rec.Code, rec.Body.String())
	}
}

func TestVerifyFiveWrongAttemptsInvalidateCode(t *testing.T) {
	h := newTestHandler()
	email := "attempts@example.com"
	rec := doJSON(t, h, http.MethodPost, "/api/auth/register", map[string]string{
		"email": email, "username": "attempts", "password": "password-123",
	}, "")
	if rec.Code != http.StatusCreated {
		t.Fatalf("register: %d %s", rec.Code, rec.Body.String())
	}
	real := testMailer(h).lastCode(email)
	wrong := "000000"
	if wrong == real {
		wrong = "000001"
	}
	// The first four wrong guesses are plain 401s.
	for i := 1; i <= 4; i++ {
		rec = doJSON(t, h, http.MethodPost, "/api/auth/verify-register", map[string]string{
			"email": email, "code": wrong,
		}, "")
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("wrong attempt %d: expected 401, got %d %s", i, rec.Code, rec.Body.String())
		}
	}
	// The 5th wrong guess invalidates the code and returns 429.
	rec = doJSON(t, h, http.MethodPost, "/api/auth/verify-register", map[string]string{
		"email": email, "code": wrong,
	}, "")
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("5th wrong attempt: expected 429, got %d %s", rec.Code, rec.Body.String())
	}
	// Even the correct code no longer works.
	rec = doJSON(t, h, http.MethodPost, "/api/auth/verify-register", map[string]string{
		"email": email, "code": real,
	}, "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("code after invalidation: expected 401, got %d %s", rec.Code, rec.Body.String())
	}
	if _, err := h.Store.GetEmailCode(email, "register"); err == nil {
		t.Fatal("invalidated code row should be gone")
	}
}

func TestResendCooldownAndInvalidation(t *testing.T) {
	h := newTestHandler()
	email := "resend@example.com"
	rec := doJSON(t, h, http.MethodPost, "/api/auth/register", map[string]string{
		"email": email, "username": "resend", "password": "password-123",
	}, "")
	if rec.Code != http.StatusCreated {
		t.Fatalf("register: %d %s", rec.Code, rec.Body.String())
	}
	first := testMailer(h).lastCode(email)

	// Immediate resend is too soon: 429.
	rec = doJSON(t, h, http.MethodPost, "/api/auth/resend-code", map[string]string{
		"email": email, "purpose": "register",
	}, "")
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("immediate resend: expected 429, got %d %s", rec.Code, rec.Body.String())
	}

	// Age the code row past the 60s cooldown, then resend works and
	// the old code stops working.
	row, err := h.Store.GetEmailCode(email, "register")
	if err != nil {
		t.Fatalf("GetEmailCode: %v", err)
	}
	row.CreatedAt = time.Now().UTC().Add(-61 * time.Second)
	if err := h.Store.SaveEmailCode(row); err != nil {
		t.Fatalf("SaveEmailCode: %v", err)
	}
	rec = doJSON(t, h, http.MethodPost, "/api/auth/resend-code", map[string]string{
		"email": email, "purpose": "register",
	}, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("resend after cooldown: expected 200, got %d %s", rec.Code, rec.Body.String())
	}
	second := testMailer(h).lastCode(email)
	if second == "" || second == first {
		t.Fatalf("resend must send a fresh code, got %q (first was %q)", second, first)
	}
	rec = doJSON(t, h, http.MethodPost, "/api/auth/verify-register", map[string]string{
		"email": email, "code": first,
	}, "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("old code after resend: expected 401, got %d %s", rec.Code, rec.Body.String())
	}
	rec = doJSON(t, h, http.MethodPost, "/api/auth/verify-register", map[string]string{
		"email": email, "code": second,
	}, "")
	if rec.Code != http.StatusCreated {
		t.Fatalf("new code after resend: expected 201, got %d %s", rec.Code, rec.Body.String())
	}
}

func TestResendRequiresPendingOrAccount(t *testing.T) {
	h := newTestHandler()
	// No pending registration for this email.
	rec := doJSON(t, h, http.MethodPost, "/api/auth/resend-code", map[string]string{
		"email": "nobody@example.com", "purpose": "register",
	}, "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("resend register without pending: expected 404, got %d %s", rec.Code, rec.Body.String())
	}
	// No account for this email.
	rec = doJSON(t, h, http.MethodPost, "/api/auth/resend-code", map[string]string{
		"email": "nobody@example.com", "purpose": "login",
	}, "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("resend login without account: expected 404, got %d %s", rec.Code, rec.Body.String())
	}
	// Bad purpose.
	rec = doJSON(t, h, http.MethodPost, "/api/auth/resend-code", map[string]string{
		"email": "nobody@example.com", "purpose": "reset",
	}, "")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("resend bad purpose: expected 400, got %d %s", rec.Code, rec.Body.String())
	}

	// Login resend works for a real account (after the cooldown).
	tok, _ := register(t, h, "resendlogin@example.com", "resendlogin")
	_ = tok
	rec = doJSON(t, h, http.MethodPost, "/api/auth/login", map[string]string{
		"email": "resendlogin@example.com", "password": "password-123",
	}, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("login: %d %s", rec.Code, rec.Body.String())
	}
	rec = doJSON(t, h, http.MethodPost, "/api/auth/resend-code", map[string]string{
		"email": "resendlogin@example.com", "purpose": "login",
	}, "")
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("login resend within cooldown: expected 429, got %d %s", rec.Code, rec.Body.String())
	}
}

func TestVerifyWithoutMailerIs503(t *testing.T) {
	h := New(store.NewMemoryStore(), ws.NewHub(), "test-secret", time.Hour)
	// No Mailer set: the server must say so, never fake a code.
	rec := doJSON(t, h, http.MethodPost, "/api/auth/register", map[string]string{
		"email": "nomail@example.com", "username": "nomail", "password": "password-123",
	}, "")
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("register without mailer: expected 503, got %d %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "email sending is not configured") {
		t.Fatalf("503 must explain email is not configured, got %s", rec.Body.String())
	}
	// Nothing was stored: no pending user, no code.
	if _, err := h.Store.GetPendingUser("nomail@example.com"); err == nil {
		t.Fatal("no pending user should be stored when email is not configured")
	}
	// A registered user also gets 503 on login when mail is down.
	h.Mailer = newFakeMailer()
	register(t, h, "hasmail@example.com", "hasmail")
	h.Mailer = nil
	rec = doJSON(t, h, http.MethodPost, "/api/auth/login", map[string]string{
		"email": "hasmail@example.com", "password": "password-123",
	}, "")
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("login without mailer: expected 503, got %d %s", rec.Code, rec.Body.String())
	}
}

// TestRegisterWithUnconfiguredSMTPIs503 is a regression test for a
// real crash found by live-testing the server: with no SMTP_FROM /
// SMTP_USER set, main.go assigned mail.NewSMTP's nil *SMTPSender into
// the mail.Sender interface, which made "h.Mailer == nil" false, so
// Register called Send on a nil receiver and the server panicked
// (nil-pointer dereference) instead of answering 503. NewSMTP now
// returns the Sender interface, so unconfigured means a true nil,
// and Send is nil-receiver-safe as a second line of defence.
func TestRegisterWithUnconfiguredSMTPIs503(t *testing.T) {
	h := New(store.NewMemoryStore(), ws.NewHub(), "test-secret", time.Hour)
	// Exactly what main.go does when SMTP is unconfigured.
	h.Mailer = mail.NewSMTP("", 0, "", "", "", false)
	if h.Mailer != nil {
		t.Fatal("NewSMTP with no host/from must return a nil Sender")
	}
	rec := doJSON(t, h, http.MethodPost, "/api/auth/register", map[string]string{
		"email": "nomail2@example.com", "username": "nomail2", "password": "password-123",
	}, "")
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("register with unconfigured SMTP: expected 503, got %d %s", rec.Code, rec.Body.String())
	}
	// A nil *SMTPSender assigned directly must fail closed, never panic.
	var nilSender *mail.SMTPSender
	if err := nilSender.Send("a@b.c", "subject", "body"); err == nil {
		t.Fatal("nil SMTPSender.Send must return an error, not panic or succeed")
	}
}

// TestVerifyLoginWrongCode checks the login code path rejects a wrong
// code the same way the register path does.
func TestVerifyLoginWrongCode(t *testing.T) {
	h := newTestHandler()
	register(t, h, "loginwrong@example.com", "loginwrong")
	rec := doJSON(t, h, http.MethodPost, "/api/auth/login", map[string]string{
		"email": "loginwrong@example.com", "password": "password-123",
	}, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("login: %d %s", rec.Code, rec.Body.String())
	}
	real := testMailer(h).lastCode("loginwrong@example.com")
	wrong := "000000"
	if wrong == real {
		wrong = "000001"
	}
	rec = doJSON(t, h, http.MethodPost, "/api/auth/verify-login", map[string]string{
		"email": "loginwrong@example.com", "code": wrong,
	}, "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("wrong login code: expected 401, got %d %s", rec.Code, rec.Body.String())
	}
	// The real login code still works.
	rec = doJSON(t, h, http.MethodPost, "/api/auth/verify-login", map[string]string{
		"email": "loginwrong@example.com", "code": real,
	}, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("real login code: expected 200, got %d %s", rec.Code, rec.Body.String())
	}
}

// TestPendingUserExpiry: a pending registration whose window passed
// cannot be verified even with the right code.
func TestPendingUserExpiry(t *testing.T) {
	h := newTestHandler()
	email := "pendingexp@example.com"
	rec := doJSON(t, h, http.MethodPost, "/api/auth/register", map[string]string{
		"email": email, "username": "pendingexp", "password": "password-123",
	}, "")
	if rec.Code != http.StatusCreated {
		t.Fatalf("register: %d %s", rec.Code, rec.Body.String())
	}
	code := testMailer(h).lastCode(email)
	pending, err := h.Store.GetPendingUser(email)
	if err != nil {
		t.Fatalf("GetPendingUser: %v", err)
	}
	pending.ExpiresAt = time.Now().UTC().Add(-time.Minute)
	if err := h.Store.SavePendingUser(pending); err != nil {
		t.Fatalf("SavePendingUser: %v", err)
	}
	// Keep the code itself valid so only the pending expiry is tested.
	row, _ := h.Store.GetEmailCode(email, "register")
	row.ExpiresAt = time.Now().UTC().Add(10 * time.Minute)
	_ = h.Store.SaveEmailCode(row)
	rec = doJSON(t, h, http.MethodPost, "/api/auth/verify-register", map[string]string{
		"email": email, "code": code,
	}, "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expired pending: expected 401, got %d %s", rec.Code, rec.Body.String())
	}
}

// TestStoreVerificationRoundTrip exercises the store methods the
// handlers depend on (MemoryStore), including that only the hash is
// persisted.
func TestStoreVerificationRoundTrip(t *testing.T) {
	h := newTestHandler()
	s := h.Store
	email := "store@example.com"
	now := time.Now().UTC()
	if err := s.SavePendingUser(&models.PendingUser{
		Email: email, Username: "storeuser", DisplayName: "Store",
		PasswordHash: "bcrypt-hash", ExpiresAt: now.Add(10 * time.Minute),
	}); err != nil {
		t.Fatalf("SavePendingUser: %v", err)
	}
	got, err := s.GetPendingUser(email)
	if err != nil || got.Username != "storeuser" {
		t.Fatalf("GetPendingUser: %+v, %v", got, err)
	}
	if err := s.SaveEmailCode(&models.EmailCode{
		Email: email, Purpose: "login", CodeHash: hashCode("123456"),
		ExpiresAt: now.Add(10 * time.Minute),
	}); err != nil {
		t.Fatalf("SaveEmailCode: %v", err)
	}
	row, err := s.GetEmailCode(email, "login")
	if err != nil || row.CodeHash != hashCode("123456") || row.Attempts != 0 {
		t.Fatalf("GetEmailCode: %+v, %v", row, err)
	}
	n, err := s.IncrementCodeAttempts(email, "login")
	if err != nil || n != 1 {
		t.Fatalf("IncrementCodeAttempts: %d, %v", n, err)
	}
	if err := s.ConsumeEmailCode(email, "login"); err != nil {
		t.Fatalf("ConsumeEmailCode: %v", err)
	}
	row, _ = s.GetEmailCode(email, "login")
	if row.ConsumedAt == nil {
		t.Fatal("consumed code must carry ConsumedAt")
	}
	if err := s.InvalidateEmailCode(email, "login"); err != nil {
		t.Fatalf("InvalidateEmailCode: %v", err)
	}
	if _, err := s.GetEmailCode(email, "login"); err == nil {
		t.Fatal("invalidated code must be gone")
	}
	if err := s.DeletePendingUser(email); err != nil {
		t.Fatalf("DeletePendingUser: %v", err)
	}
	if _, err := s.GetPendingUser(email); err == nil {
		t.Fatal("deleted pending user must be gone")
	}
}
