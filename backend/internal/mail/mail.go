// Package mail sends VYRO verification emails.
//
// The only mail VYRO sends today is the 6-digit verification code used
// by registration and login (see internal/handlers/verify.go).
//
// Two senders exist:
//   - SMTPSender: plain net/smtp against any SMTP server (default
//     configuration is Gmail's SMTP relay, smtp.gmail.com:587,
//     STARTTLS).
//   - ResendSender: the Resend HTTPS API (api.resend.com). Use it when
//     outbound SMTP is unreliable on the host: on Render's free tier an
//     SMTP dial to smtp.resend.com:587 never completed, hanging every
//     registration with no error at all, while HTTPS works fine. Set
//     RESEND_API_KEY and the server prefers this sender over SMTP.
//
// Every send is bounded by sendTimeout: a hung mail server must fail
// fast (HTTP 500, retryable) instead of blocking a registration
// forever. Verification codes are useless if they arrive minutes late.
//
// No credentials are stored in the repo — see SMTP_* / RESEND_API_KEY
// in backend/.env.example.
//
// A nil Sender means "email is not configured": handlers must return
// HTTP 503 and never pretend a code was sent.
package mail

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/smtp"
	"strconv"
	"strings"
	"time"
)

// sendTimeout bounds any single email send (SMTP dial + greeting +
// conversation, or one Resend HTTPS request). It is a variable so
// tests can shorten it.
var sendTimeout = 20 * time.Second

// Sender delivers one plain-text email. Implementations must never log
// the body (it contains the verification code).
type Sender interface {
	Send(to, subject, body string) error
}

// SMTPSender sends mail through an SMTP server with opportunistic
// STARTTLS (net/smtp upgrades the connection when the server
// advertises STARTTLS, as Gmail does on port 587).
type SMTPSender struct {
	Host        string
	Port        int
	User        string
	Pass        string
	From        string
	AllowNoAuth bool // when true, send without SMTP AUTH (local test sinks)
}

// NewSMTP returns a sender, or nil when email is not configured.
// "Configured" means a host and a From address are both set; the
// From address defaults to the SMTP user when SMTP_FROM is empty
// (config.Load handles that default).
//
// It returns the Sender interface (not *SMTPSender): a nil concrete
// pointer assigned to an interface is a non-nil interface, so
// returning *SMTPSender would make "Mailer == nil" checks silently
// pass and the server would panic on the first register/login
// (nil-receiver Send) instead of answering 503.
func NewSMTP(host string, port int, user, pass, from string, allowNoAuth bool) Sender {
	if strings.TrimSpace(host) == "" || strings.TrimSpace(from) == "" {
		return nil
	}
	if port <= 0 {
		port = 587
	}
	return &SMTPSender{Host: host, Port: port, User: user, Pass: pass, From: from, AllowNoAuth: allowNoAuth}
}

// Send delivers one plain-text email. The caller supplies the full
// body; this method only adds the headers. A nil receiver can never
// panic: it reports "not configured" instead, so a mis-wired sender
// fails closed like the 503 path.
func (s *SMTPSender) Send(to, subject, body string) error {
	if s == nil {
		return fmt.Errorf("mail sender not configured")
	}
	addr := s.Host + ":" + strconv.Itoa(s.Port)
	var auth smtp.Auth
	if !s.AllowNoAuth && s.User != "" {
		auth = smtp.PlainAuth("", s.User, s.Pass, s.Host)
	}
	var b strings.Builder
	b.WriteString("From: " + s.From + "\r\n")
	b.WriteString("To: " + to + "\r\n")
	b.WriteString("Subject: " + subject + "\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/plain; charset=utf-8\r\n")
	b.WriteString("\r\n")
	b.WriteString(body)
	// Dial with a timeout: net/smtp's SendMail dials without one, so a
	// blackholed SMTP server used to hang registration forever.
	d := net.Dialer{Timeout: sendTimeout}
	conn, err := d.Dial("tcp", addr)
	if err != nil {
		return fmt.Errorf("send mail to %s: %w", to, err)
	}
	defer conn.Close()
	// Bound the greeting and the whole conversation too: a server that
	// accepts the connection but never answers must not block the
	// request either.
	_ = conn.SetDeadline(time.Now().Add(sendTimeout))
	c, err := smtp.NewClient(conn, s.Host)
	if err != nil {
		return fmt.Errorf("send mail to %s: %w", to, err)
	}
	defer c.Close()
	if err := sendViaClient(c, auth, s.Host, s.From, to, b.String()); err != nil {
		return fmt.Errorf("send mail to %s: %w", to, err)
	}
	return nil
}

// sendViaClient runs the SMTP conversation: STARTTLS when advertised,
// AUTH when offered and configured, then MAIL/RCPT/DATA.
func sendViaClient(c *smtp.Client, auth smtp.Auth, host, from, to, msg string) error {
	if ok, _ := c.Extension("STARTTLS"); ok {
		if err := c.StartTLS(&tls.Config{ServerName: host}); err != nil {
			return err
		}
	}
	if auth != nil {
		if ok, _ := c.Extension("AUTH"); ok {
			if err := c.Auth(auth); err != nil {
				return err
			}
		}
	}
	if err := c.Mail(from); err != nil {
		return err
	}
	if err := c.Rcpt(to); err != nil {
		return err
	}
	w, err := c.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write([]byte(msg)); err != nil {
		return err
	}
	return w.Close()
}

// --- Resend HTTPS API sender ---

// ResendSender delivers mail through the Resend HTTP API
// (POST https://api.resend.com/emails). It exists because plain SMTP
// is not reliable everywhere the server runs: on Render's free tier an
// SMTP dial to smtp.resend.com:587 never completed, hanging every
// registration with no error at all, while HTTPS works fine. The API
// key is a Resend API key (re_...); the From address must be a sender
// Resend accepts — onboarding@resend.dev works with no domain setup.
type ResendSender struct {
	APIKey  string
	From    string
	BaseURL string // https://api.resend.com; overridden in tests
	Client  *http.Client
}

// NewResend returns a sender, or nil when no API key (or no From) is given.
func NewResend(apiKey, from string) Sender {
	if strings.TrimSpace(apiKey) == "" || strings.TrimSpace(from) == "" {
		return nil
	}
	return &ResendSender{
		APIKey:  apiKey,
		From:    from,
		BaseURL: "https://api.resend.com",
		Client:  &http.Client{Timeout: sendTimeout},
	}
}

type resendEmailRequest struct {
	From    string   `json:"from"`
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	Text    string   `json:"text"`
}

// Send delivers one plain-text email via the Resend API. A nil receiver
// reports "not configured" instead of panicking, like SMTPSender.
func (s *ResendSender) Send(to, subject, body string) error {
	if s == nil {
		return fmt.Errorf("mail sender not configured")
	}
	payload, err := json.Marshal(resendEmailRequest{
		From:    s.From,
		To:      []string{to},
		Subject: subject,
		Text:    body,
	})
	if err != nil {
		return fmt.Errorf("send mail to %s: %w", to, err)
	}
	base := s.BaseURL
	if base == "" {
		base = "https://api.resend.com"
	}
	client := s.Client
	if client == nil {
		client = &http.Client{Timeout: sendTimeout}
	}
	ctx, cancel := context.WithTimeout(context.Background(), sendTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimSuffix(base, "/")+"/emails", bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("send mail to %s: %w", to, err)
	}
	req.Header.Set("Authorization", "Bearer "+s.APIKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("send mail to %s: %w", to, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		detail, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("send mail to %s: resend API error %d: %s", to, resp.StatusCode, strings.TrimSpace(string(detail)))
	}
	return nil
}
