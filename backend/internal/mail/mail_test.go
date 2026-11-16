package mail

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestResendSenderSuccess(t *testing.T) {
	var gotAuth, gotCT string
	var gotBody resendEmailRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotCT = r.Header.Get("Content-Type")
		if r.URL.Path != "/emails" || r.Method != http.MethodPost {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		defer r.Body.Close()
		b, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(b, &gotBody); err != nil {
			t.Fatalf("bad json: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"abc123"}`))
	}))
	defer srv.Close()

	s := &ResendSender{APIKey: "re_test123", From: "onboarding@resend.dev", BaseURL: srv.URL, Client: srv.Client()}
	if err := s.Send("user@example.com", "VYRO verification code", "Your VYRO verification code is: 123456"); err != nil {
		t.Fatalf("Send failed: %v", err)
	}
	if gotAuth != "Bearer re_test123" {
		t.Errorf("auth header = %q", gotAuth)
	}
	if gotCT != "application/json" {
		t.Errorf("content-type = %q", gotCT)
	}
	if gotBody.From != "onboarding@resend.dev" || len(gotBody.To) != 1 || gotBody.To[0] != "user@example.com" ||
		gotBody.Subject != "VYRO verification code" || !strings.Contains(gotBody.Text, "123456") {
		t.Errorf("unexpected body: %+v", gotBody)
	}
}

func TestResendSenderError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"message":"invalid API key"}`))
	}))
	defer srv.Close()

	s := &ResendSender{APIKey: "re_bad", From: "onboarding@resend.dev", BaseURL: srv.URL, Client: srv.Client()}
	err := s.Send("user@example.com", "subject", "body")
	if err == nil {
		t.Fatal("expected error for 401, got nil")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Errorf("error should mention 401: %v", err)
	}
}

func TestNewResendNil(t *testing.T) {
	if NewResend("", "onboarding@resend.dev") != nil {
		t.Error("empty key should give nil sender")
	}
	if NewResend("re_x", "") != nil {
		t.Error("empty from should give nil sender")
	}
	var s *ResendSender
	if err := s.Send("a@b.c", "s", "b"); err == nil {
		t.Error("nil receiver should error, not panic")
	}
}

func TestGmailAPISenderSuccess(t *testing.T) {
	tokenCalls := 0
	tokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenCalls++
		if r.URL.Path != "/token" || r.Method != http.MethodPost {
			t.Errorf("unexpected token request %s %s", r.Method, r.URL.Path)
		}
		if got := r.FormValue("grant_type"); got != "refresh_token" {
			t.Errorf("grant_type = %q", got)
		}
		if got := r.FormValue("refresh_token"); got != "refresh123" {
			t.Errorf("refresh_token = %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"ya29.test","expires_in":3600,"token_type":"Bearer"}`))
	}))
	defer tokenSrv.Close()

	var gotAuth, gotRaw string
	gmailSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		if r.URL.Path != "/gmail/v1/users/me/messages/send" || r.Method != http.MethodPost {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		defer r.Body.Close()
		var req gmailSendRequest
		b, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(b, &req); err != nil {
			t.Fatalf("bad json: %v", err)
		}
		gotRaw = req.Raw
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"msg1"}`))
	}))
	defer gmailSrv.Close()

	s := &GmailAPISender{
		ClientID: "cid", ClientSecret: "csecret", RefreshToken: "refresh123",
		From: "najmuddinsafari00@gmail.com", BaseURL: gmailSrv.URL, TokenURL: tokenSrv.URL + "/token",
		Client: gmailSrv.Client(),
	}
	if err := s.Send("user@example.com", "VYRO verification code", "Your VYRO verification code is: 123456"); err != nil {
		t.Fatalf("Send failed: %v", err)
	}
	if tokenCalls != 1 {
		t.Errorf("token endpoint called %d times, want 1", tokenCalls)
	}
	if gotAuth != "Bearer ya29.test" {
		t.Errorf("auth header = %q", gotAuth)
	}
	decoded, err := base64.RawURLEncoding.DecodeString(gotRaw)
	if err != nil {
		t.Fatalf("raw is not base64url: %v", err)
	}
	text := string(decoded)
	for _, want := range []string{
		"From: najmuddinsafari00@gmail.com",
		"To: user@example.com",
		"Subject: VYRO verification code",
		"123456",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("decoded message missing %q:\n%s", want, text)
		}
	}
	// Second send reuses the cached access token: no second refresh.
	if err := s.Send("user2@example.com", "subject", "body"); err != nil {
		t.Fatalf("second Send failed: %v", err)
	}
	if tokenCalls != 1 {
		t.Errorf("token endpoint called %d times after second send, want 1 (cached)", tokenCalls)
	}
}

func TestGmailAPIRefreshesOn401(t *testing.T) {
	tokenCalls := 0
	tokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenCalls++
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"ya29.fresh","expires_in":3600,"token_type":"Bearer"}`))
	}))
	defer tokenSrv.Close()

	sendCalls := 0
	gmailSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sendCalls++
		if sendCalls == 1 {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":"invalid token"}`))
			return
		}
		if got := r.Header.Get("Authorization"); got != "Bearer ya29.fresh" {
			t.Errorf("retry auth header = %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"msg2"}`))
	}))
	defer gmailSrv.Close()

	s := &GmailAPISender{
		ClientID: "cid", ClientSecret: "csecret", RefreshToken: "refresh123",
		From: "a@b.c", BaseURL: gmailSrv.URL, TokenURL: tokenSrv.URL,
		Client: gmailSrv.Client(),
	}
	// Seed a stale cached token so the first send uses it and gets 401.
	s.accessToken = "ya29.stale"
	s.tokenExpiry = time.Now().Add(time.Hour)
	if err := s.Send("user@example.com", "subject", "body"); err != nil {
		t.Fatalf("Send failed after 401 retry: %v", err)
	}
	if sendCalls != 2 {
		t.Errorf("gmail endpoint called %d times, want 2", sendCalls)
	}
	if tokenCalls != 1 {
		t.Errorf("token endpoint called %d times, want 1 (forced refresh)", tokenCalls)
	}
}

func TestNewGmailAPINil(t *testing.T) {
	if NewGmailAPI("", "sec", "ref", "a@b.c") != nil {
		t.Error("empty client id should give nil sender")
	}
	if NewGmailAPI("cid", "", "ref", "a@b.c") != nil {
		t.Error("empty client secret should give nil sender")
	}
	if NewGmailAPI("cid", "sec", "", "a@b.c") != nil {
		t.Error("empty refresh token should give nil sender")
	}
	if NewGmailAPI("cid", "sec", "ref", "") != nil {
		t.Error("empty from should give nil sender")
	}
	var s *GmailAPISender
	if err := s.Send("a@b.c", "s", "b"); err == nil {
		t.Error("nil receiver should error, not panic")
	}
}

func TestSMTPSenderTimeout(t *testing.T) {
	old := sendTimeout
	sendTimeout = 200 * time.Millisecond
	defer func() { sendTimeout = old }()

	// A server that accepts the connection but never speaks: without
	// a deadline, smtp.NewClient would block on the greeting forever.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			// Hold the connection open without sending anything.
			go func(conn net.Conn) {
				time.Sleep(5 * time.Second)
				conn.Close()
			}(c)
		}
	}()

	s := &SMTPSender{Host: "127.0.0.1", Port: ln.Addr().(*net.TCPAddr).Port, From: "a@b.c", AllowNoAuth: true}
	start := time.Now()
	err = s.Send("user@example.com", "subject", "body")
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	if took := time.Since(start); took > 5*time.Second {
		t.Errorf("Send took %v, expected it to fail fast", took)
	}
}
