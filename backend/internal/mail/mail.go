// Package mail sends VYRO verification emails.
//
// The only mail VYRO sends today is the 6-digit verification code used
// by registration and login (see internal/handlers/verify.go). The
// sender is plain net/smtp against any SMTP server; the default
// configuration is Gmail's SMTP relay (smtp.gmail.com:587, STARTTLS),
// but any host works, including a local test sink. No credentials are
// stored in the repo — see SMTP_* in backend/.env.example.
//
// A nil Sender means "email is not configured": handlers must return
// HTTP 503 and never pretend a code was sent.
package mail

import (
	"fmt"
	"net/smtp"
	"strconv"
	"strings"
)

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
	if err := smtp.SendMail(addr, auth, s.From, []string{to}, []byte(b.String())); err != nil {
		return fmt.Errorf("send mail to %s: %w", to, err)
	}
	return nil
}
