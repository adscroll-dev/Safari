// Email verification (v0.7): registration and login both finish with a
// 6-digit code sent to the user's email.
//
//	POST /api/auth/register      -> 201 {"status":"code_sent", ...} (no token, no user yet)
//	POST /api/auth/verify-register {email, code} -> 201 {"user":{...},"token":"..."}
//	POST /api/auth/login         -> 200 {"status":"code_sent", ...} (no token)
//	POST /api/auth/verify-login    {email, code} -> 200 {"user":{...},"token":"..."}
//	POST /api/auth/resend-code    {email, purpose} -> 200 {"status":"code_sent", ...}
//
// Code rules: random 6 digits (crypto/rand, leading zeros allowed),
// expires after 10 minutes, single-use, max 5 failed verification
// attempts per code (the 5th failure invalidates it and returns 429),
// resend cooldown 60 seconds per (email, purpose). Only a keyed hash
// of a code is stored; the plain code is never logged or
// returned by the API. When no mailer is configured (h.Mailer == nil)
// the endpoints that must send mail return 503 and never fake a code.
package handlers

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/vyro/backend/internal/auth"
	"github.com/vyro/backend/internal/middleware"
	"github.com/vyro/backend/internal/models"
	"github.com/vyro/backend/internal/store"
)

const (
	codeTTL           = 10 * time.Minute
	codeExpirySeconds = 600 // 10 minutes, reported to clients
	maxCodeAttempts   = 5
	resendCooldown    = 60 * time.Second
)

const (
	purposeRegister = "register"
	purposeLogin    = "login"
)

// generateCode returns a random 6-digit code ("000000".."999999"),
// leading zeros allowed.
func generateCode() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

// hashCode stores a keyed, domain-separated digest so a database dump
// alone cannot be used to brute-force the million possible codes.
func hashCode(secret, code string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte("safari-email-code-v1:"))
	_, _ = mac.Write([]byte(code))
	return hex.EncodeToString(mac.Sum(nil))
}

// legacyHashCode permits codes issued before keyed hashing to finish
// during their existing 10-minute lifetime.
func legacyHashCode(code string) string {
	sum := sha256.Sum256([]byte(code))
	return hex.EncodeToString(sum[:])
}

// codeEmailBody is the plain-text body of the verification email. It
// contains the code, its 10-minute expiry, and the never-share warning.
func codeEmailBody(code string) string {
	return "Your Safari verification code is: " + code + "\n\n" +
		"This code expires in 10 minutes. Never share this code with anyone — " +
		"Safari staff will never ask for it.\n"
}

// mailerUnavailable logs the reason and answers 503. It is used by
// every endpoint that must send a code when no SMTP is configured.
func (h *Handler) mailerUnavailable(w http.ResponseWriter) {
	log.Println("Safari email: not configured — verification codes cannot be sent")
	middleware.WriteError(w, http.StatusServiceUnavailable, "email sending is not configured")
}

// issueCode generates a code and stores only its hash before returning.
// Email delivery is asynchronous so a slow mail server does not delay
// the response. A failed send is logged; the normal resend cooldown
// allows the user to request another code.
func (h *Handler) issueCode(email, purpose string) error {
	code, err := generateCode()
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	codeHash := hashCode(h.JWTSecret, code)
	if err := h.Store.SaveEmailCode(&models.EmailCode{
		Email: email, Purpose: purpose, CodeHash: codeHash,
		ExpiresAt: now.Add(codeTTL), CreatedAt: now,
	}); err != nil {
		return err
	}
	go func() {
		if err := h.Mailer.Send(email, "Safari verification code", codeEmailBody(code)); err != nil {
			log.Printf("verification email send failed for %s (%s): %v", email, purpose, err)
		}
	}()
	return nil
}

// verifyCode checks a submitted code against the stored hash. It
// returns (true, false) on success; on failure it returns
// (false, tooMany) where tooMany means the attempt limit was hit and
// the code was invalidated. Expired, consumed, or missing codes fail
// with the same generic outcome as a wrong code, so callers must not
// distinguish them in the response.
func (h *Handler) verifyCode(email, purpose, code string) (valid, tooMany bool) {
	row, err := h.Store.GetEmailCode(email, purpose)
	if err != nil || row.ConsumedAt != nil || time.Now().UTC().After(row.ExpiresAt) {
		return false, false
	}
	currentHashMatch := subtle.ConstantTimeCompare(
		[]byte(hashCode(h.JWTSecret, code)), []byte(row.CodeHash),
	)
	legacyHashMatch := subtle.ConstantTimeCompare(
		[]byte(legacyHashCode(code)), []byte(row.CodeHash),
	)
	if currentHashMatch|legacyHashMatch == 1 {
		_ = h.Store.ConsumeEmailCode(email, purpose)
		return true, false
	}
	attempts, err := h.Store.IncrementCodeAttempts(email, purpose)
	if err != nil {
		return false, false
	}
	if attempts >= maxCodeAttempts {
		_ = h.Store.InvalidateEmailCode(email, purpose)
		return false, true
	}
	return false, false
}

func codeSentResponse(w http.ResponseWriter, status int, email, purpose string) {
	middleware.WriteJSON(w, status, map[string]interface{}{
		"status": "code_sent", "email": email, "purpose": purpose,
		"expires_in_seconds": codeExpirySeconds,
	})
}

// codeSentResponseWithUsername is codeSentResponse plus the username
// the system generated for this registration, so the client can show
// it to the user on the code screen.
func codeSentResponseWithUsername(w http.ResponseWriter, status int, email, purpose, username string) {
	middleware.WriteJSON(w, status, map[string]interface{}{
		"status": "code_sent", "email": email, "purpose": purpose,
		"expires_in_seconds": codeExpirySeconds, "username": username,
	})
}

type verifyReq struct {
	Email string `json:"email"`
	Code  string `json:"code"`
}

// VerifyRegister completes a pending registration: the emailed code
// is checked, the pending user becomes a real user, and a JWT is
// issued. Wrong/expired/consumed codes get the generic 401; the 5th
// failed attempt invalidates the code and gets 429.
func (h *Handler) VerifyRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		middleware.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req verifyReq
	if !decode(w, r, &req) {
		return
	}
	email := strings.TrimSpace(strings.ToLower(req.Email))
	if !emailRe.MatchString(email) {
		middleware.WriteError(w, http.StatusBadRequest, "invalid email")
		return
	}
	valid, tooMany := h.verifyCode(email, purposeRegister, req.Code)
	if tooMany {
		middleware.WriteError(w, http.StatusTooManyRequests, "too many verification attempts, request a new code")
		return
	}
	if !valid {
		middleware.WriteError(w, http.StatusUnauthorized, "invalid or expired code")
		return
	}
	pending, err := h.Store.GetPendingUser(email)
	if err != nil || time.Now().UTC().After(pending.ExpiresAt) {
		middleware.WriteError(w, http.StatusUnauthorized, "invalid or expired code")
		return
	}
	_ = h.Store.DeletePendingUser(email)
	user := &models.User{
		Email: pending.Email, Username: pending.Username,
		DisplayName: pending.DisplayName, PasswordHash: pending.PasswordHash,
	}
	if err := h.Store.CreateUser(user); err != nil {
		if err == store.ErrAlreadyExists {
			middleware.WriteError(w, http.StatusConflict, "email or username already registered")
			return
		}
		middleware.WriteError(w, http.StatusInternalServerError, "could not create user")
		return
	}
	h.issueToken(w, user, http.StatusCreated)
}

// VerifyLogin completes a login: the emailed code is checked and a
// JWT is issued for the existing account.
func (h *Handler) VerifyLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		middleware.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req verifyReq
	if !decode(w, r, &req) {
		return
	}
	email := strings.TrimSpace(strings.ToLower(req.Email))
	if !emailRe.MatchString(email) {
		middleware.WriteError(w, http.StatusBadRequest, "invalid email")
		return
	}
	valid, tooMany := h.verifyCode(email, purposeLogin, req.Code)
	if tooMany {
		middleware.WriteError(w, http.StatusTooManyRequests, "too many verification attempts, request a new code")
		return
	}
	if !valid {
		middleware.WriteError(w, http.StatusUnauthorized, "invalid or expired code")
		return
	}
	user, err := h.Store.UserByEmail(email)
	if err != nil {
		middleware.WriteError(w, http.StatusUnauthorized, "invalid or expired code")
		return
	}
	h.issueToken(w, user, http.StatusOK)
}

// issueToken mints a JWT for user and writes the {"user","token"}
// response used by both verify endpoints.
func (h *Handler) issueToken(w http.ResponseWriter, user *models.User, status int) {
	token, err := auth.GenerateJWT(h.JWTSecret, user.ID, user.Email, user.Username, h.JWTExpiry)
	if err != nil {
		middleware.WriteError(w, http.StatusInternalServerError, "could not issue token")
		return
	}
	middleware.WriteJSON(w, status, map[string]interface{}{
		"user": user.Public(), "token": token,
	})
}

type resendReq struct {
	Email   string `json:"email"`
	Purpose string `json:"purpose"`
}

// ResendCode issues a fresh code for a pending registration or an
// existing login, invalidating the previous one. At most one code per
// 60 seconds per (email, purpose).
func (h *Handler) ResendCode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		middleware.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req resendReq
	if !decode(w, r, &req) {
		return
	}
	email := strings.TrimSpace(strings.ToLower(req.Email))
	purpose := strings.TrimSpace(req.Purpose)
	if !emailRe.MatchString(email) {
		middleware.WriteError(w, http.StatusBadRequest, "invalid email")
		return
	}
	switch purpose {
	case purposeRegister:
		pending, err := h.Store.GetPendingUser(email)
		if err != nil || time.Now().UTC().After(pending.ExpiresAt) {
			middleware.WriteError(w, http.StatusNotFound, "no pending registration for this email")
			return
		}
	case purposeLogin:
		if _, err := h.Store.UserByEmail(email); err != nil {
			middleware.WriteError(w, http.StatusNotFound, "no account with this email")
			return
		}
	default:
		middleware.WriteError(w, http.StatusBadRequest, `purpose must be "register" or "login"`)
		return
	}
	if h.Mailer == nil {
		h.mailerUnavailable(w)
		return
	}
	if row, err := h.Store.GetEmailCode(email, purpose); err == nil {
		if time.Since(row.CreatedAt) < resendCooldown {
			middleware.WriteError(w, http.StatusTooManyRequests, "verification code was sent recently, please wait before requesting another")
			return
		}
	}
	if err := h.issueCode(email, purpose); err != nil {
		middleware.WriteError(w, http.StatusInternalServerError, "could not send verification email")
		return
	}
	codeSentResponse(w, http.StatusOK, email, purpose)
}
