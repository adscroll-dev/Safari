package mail

import (
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
