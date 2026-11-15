package handlers

import (
	"net/http"
	"strings"

	"github.com/vyro/backend/internal/middleware"
	"github.com/vyro/backend/internal/store"
)

// Stage 8 (extras): change the account email (account transfer to
// a new address). The new address must prove ownership with the
// same 6-digit code flow as registration/login, so a stolen session
// alone cannot steal the account.

const purposeChangeEmail = "change-email"

type changeEmailReq struct {
	NewEmail string `json:"new_email"`
}

// RequestEmailChange serves POST /api/users/me/change-email:
// validate the new address and send it a 6-digit code.
func (h *Handler) RequestEmailChange(w http.ResponseWriter, r *http.Request) {
	var req changeEmailReq
	if !decode(w, r, &req) {
		return
	}
	email := strings.TrimSpace(strings.ToLower(req.NewEmail))
	if !emailRe.MatchString(email) {
		middleware.WriteError(w, http.StatusBadRequest, "invalid email")
		return
	}
	me := middleware.UserID(r)
	user, err := h.Store.UserByID(me)
	if err != nil {
		middleware.WriteError(w, http.StatusNotFound, "user not found")
		return
	}
	if email == strings.ToLower(strings.TrimSpace(user.Email)) {
		middleware.WriteError(w, http.StatusBadRequest, "this is already your email")
		return
	}
	if _, err := h.Store.UserByEmail(email); err == nil {
		middleware.WriteError(w, http.StatusConflict, "email is already in use")
		return
	}
	if h.Mailer == nil {
		h.mailerUnavailable(w)
		return
	}
	if err := h.issueCode(email, purposeChangeEmail); err != nil {
		middleware.WriteError(w, http.StatusInternalServerError, "could not send code")
		return
	}
	codeSentResponse(w, http.StatusCreated, email, purposeChangeEmail)
}

type verifyChangeEmailReq struct {
	NewEmail string `json:"new_email"`
	Code     string `json:"code"`
}

// VerifyEmailChange serves POST /api/users/me/change-email/verify:
// check the code sent to the new address, then move the account
// to it. Returns the updated public user.
func (h *Handler) VerifyEmailChange(w http.ResponseWriter, r *http.Request) {
	var req verifyChangeEmailReq
	if !decode(w, r, &req) {
		return
	}
	email := strings.TrimSpace(strings.ToLower(req.NewEmail))
	if !emailRe.MatchString(email) {
		middleware.WriteError(w, http.StatusBadRequest, "invalid email")
		return
	}
	valid, tooMany := h.verifyCode(email, purposeChangeEmail, req.Code)
	if tooMany {
		middleware.WriteError(w, http.StatusTooManyRequests, "too many verification attempts, request a new code")
		return
	}
	if !valid {
		middleware.WriteError(w, http.StatusUnauthorized, "invalid or expired code")
		return
	}
	me := middleware.UserID(r)
	if _, err := h.Store.UserByEmail(email); err == nil {
		middleware.WriteError(w, http.StatusConflict, "email is already in use")
		return
	}
	if err := h.Store.ChangeEmail(me, email); err != nil {
		if err == store.ErrNotFound {
			middleware.WriteError(w, http.StatusNotFound, "user not found")
			return
		}
		middleware.WriteError(w, http.StatusConflict, "email is already in use")
		return
	}
	user, err := h.Store.UserByID(me)
	if err != nil {
		middleware.WriteError(w, http.StatusNotFound, "user not found")
		return
	}
	middleware.WriteJSON(w, http.StatusOK, user.Public())
}
