// Package mail sends VYRO verification emails.
//
// The only mail VYRO sends today is the 6-digit verification code used
// by registration and login (see internal/handlers/verify.go).
//
// Two senders exist:
//   - SMTPSender: plain net/smtp against any SMTP server (default
//     configuration is Gmail's SMTP relay, smtp.gmail.com:587,
//     STARTTLS). NOTE: on Render's free tier outbound SMTP TCP is
//     blocked entirely — an SMTP dial to smtp.gmail.com:587 times
//     out — so SMTP cannot be the live sender there.
//   - GmailAPISender: the Gmail API over HTTPS (POST
//     gmail.googleapis.com/gmail/v1/users/me/messages/send), OAuth2
//     with a refresh token, scope gmail.send only. Use it when
//     outbound SMTP is blocked on the host: HTTPS works fine.
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
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/smtp"
	"net/url"
	"strconv"
	"strings"
	"sync"
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

// --- Gmail API (HTTPS) sender ---

// GmailAPISender delivers mail through the Gmail API over HTTPS
// (POST https://gmail.googleapis.com/gmail/v1/users/me/messages/send).
// It exists because Render's free tier blocks outbound SMTP TCP
// entirely: an SMTP dial to smtp.gmail.com:587 times out, while
// HTTPS works fine. Auth is OAuth2 with a refresh token: the sender
// exchanges it for a short-lived access token (cached until near
// expiry) and sends with scope gmail.send only (send-only; the token
// cannot read the mailbox). Standard library only.
type GmailAPISender struct {
	ClientID     string
	ClientSecret string
	RefreshToken string
	From         string
	BaseURL      string // https://gmail.googleapis.com; overridden in tests
	TokenURL     string // https://oauth2.googleapis.com/token; overridden in tests
	Client       *http.Client

	mu          sync.Mutex
	accessToken string
	tokenExpiry time.Time
}

// NewGmailAPI returns a sender, or nil when any credential (or the
// From address) is missing.
func NewGmailAPI(clientID, clientSecret, refreshToken, from string) Sender {
	if strings.TrimSpace(clientID) == "" || strings.TrimSpace(clientSecret) == "" ||
		strings.TrimSpace(refreshToken) == "" || strings.TrimSpace(from) == "" {
		return nil
	}
	return &GmailAPISender{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RefreshToken: refreshToken,
		From:         from,
		BaseURL:      "https://gmail.googleapis.com",
		TokenURL:     "https://oauth2.googleapis.com/token",
		Client:       &http.Client{Timeout: sendTimeout},
	}
}

type gmailSendRequest struct {
	Raw string `json:"raw"`
}

type gmailTokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	TokenType   string `json:"token_type"`
}

// token returns a valid access token, refreshing it when the cached
// one is missing or about to expire. force forces a refresh (used
// after a 401).
func (s *GmailAPISender) token(force bool) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !force && s.accessToken != "" && time.Now().Add(60*time.Second).Before(s.tokenExpiry) {
		return s.accessToken, nil
	}
	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("client_id", s.ClientID)
	form.Set("client_secret", s.ClientSecret)
	form.Set("refresh_token", s.RefreshToken)
	tokenURL := s.TokenURL
	if tokenURL == "" {
		tokenURL = "https://oauth2.googleapis.com/token"
	}
	client := s.Client
	if client == nil {
		client = &http.Client{Timeout: sendTimeout}
	}
	ctx, cancel := context.WithTimeout(context.Background(), sendTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("gmail token refresh: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		detail, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return "", fmt.Errorf("gmail token refresh: status %d: %s", resp.StatusCode, strings.TrimSpace(string(detail)))
	}
	var tr gmailTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tr); err != nil {
		return "", fmt.Errorf("gmail token refresh: bad response: %w", err)
	}
	if tr.AccessToken == "" {
		return "", fmt.Errorf("gmail token refresh: empty access token")
	}
	expiresIn := tr.ExpiresIn
	if expiresIn <= 0 {
		expiresIn = 3600
	}
	s.accessToken = tr.AccessToken
	s.tokenExpiry = time.Now().Add(time.Duration(expiresIn) * time.Second)
	return s.accessToken, nil
}

// Send delivers one plain-text email via the Gmail API. A nil receiver
// reports "not configured" instead of panicking, like the other senders.
func (s *GmailAPISender) Send(to, subject, body string) error {
	if s == nil {
		return fmt.Errorf("mail sender not configured")
	}
	var msg strings.Builder
	msg.WriteString("From: " + s.From + "\r\n")
	msg.WriteString("To: " + to + "\r\n")
	msg.WriteString("Subject: " + subject + "\r\n")
	msg.WriteString("MIME-Version: 1.0\r\n")
	msg.WriteString("Content-Type: text/plain; charset=utf-8\r\n")
	msg.WriteString("\r\n")
	msg.WriteString(body)
	raw := base64.RawURLEncoding.EncodeToString([]byte(msg.String()))
	payload, err := json.Marshal(gmailSendRequest{Raw: raw})
	if err != nil {
		return fmt.Errorf("send mail to %s: %w", to, err)
	}
	base := s.BaseURL
	if base == "" {
		base = "https://gmail.googleapis.com"
	}
	client := s.Client
	if client == nil {
		client = &http.Client{Timeout: sendTimeout}
	}
	for attempt := 0; attempt < 2; attempt++ {
		token, err := s.token(attempt == 1)
		if err != nil {
			return fmt.Errorf("send mail to %s: %w", to, err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), sendTimeout)
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimSuffix(base, "/")+"/gmail/v1/users/me/messages/send", bytes.NewReader(payload))
		if err != nil {
			cancel()
			return fmt.Errorf("send mail to %s: %w", to, err)
		}
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		resp, err := client.Do(req)
		cancel()
		if err != nil {
			return fmt.Errorf("send mail to %s: %w", to, err)
		}
		if resp.StatusCode == http.StatusUnauthorized && attempt == 0 {
			// The access token may have expired; force a refresh
			// and try once more.
			_, _ = io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			continue
		}
		defer resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			detail, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
			return fmt.Errorf("send mail to %s: gmail API error %d: %s", to, resp.StatusCode, strings.TrimSpace(string(detail)))
		}
		return nil
	}
	return fmt.Errorf("send mail to %s: gmail API error 401", to)
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
