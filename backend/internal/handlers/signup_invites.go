package handlers

import (
	"crypto/rand"
	"math/big"
	"net/http"
	"time"

	"github.com/vyro/backend/internal/middleware"
	"github.com/vyro/backend/internal/models"
	"github.com/vyro/backend/internal/store"
)

const signupInviteAlphabet = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"

func generateSignupInviteCode() (string, error) {
	code := make([]byte, 8)
	for i := range code {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(signupInviteAlphabet))))
		if err != nil {
			return "", err
		}
		code[i] = signupInviteAlphabet[n.Int64()]
	}
	return string(code), nil
}

type createSignupInviteReq struct {
	MaxUses        *int `json:"max_uses"`
	ExpiresInHours *int `json:"expires_in_hours"`
}

// SignupInvites creates and lists the authenticated user's signup codes.
func (h *Handler) SignupInvites(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		var req createSignupInviteReq
		if !decode(w, r, &req) {
			return
		}
		maxUses, expiresInHours := 1, 168
		if req.MaxUses != nil {
			maxUses = *req.MaxUses
		}
		if req.ExpiresInHours != nil {
			expiresInHours = *req.ExpiresInHours
		}
		if maxUses < 1 {
			maxUses = 1
		} else if maxUses > 20 {
			maxUses = 20
		}
		if expiresInHours < 1 {
			expiresInHours = 1
		} else if expiresInHours > 720 {
			expiresInHours = 720
		}
		now := time.Now().UTC()
		for attempt := 0; attempt < 5; attempt++ {
			code, err := generateSignupInviteCode()
			if err != nil {
				middleware.WriteError(w, http.StatusInternalServerError, "could not generate invite code")
				return
			}
			invite := &models.SignupInviteCode{
				Code: code, InviterID: middleware.UserID(r),
				MaxUses: maxUses, ExpiresAt: now.Add(time.Duration(expiresInHours) * time.Hour),
				CreatedAt: now,
			}
			if err := h.Store.CreateSignupInviteCode(invite); err != nil {
				if err == store.ErrAlreadyExists {
					continue
				}
				middleware.WriteError(w, http.StatusInternalServerError, "could not create invite code")
				return
			}
			middleware.WriteJSON(w, http.StatusCreated, invite)
			return
		}
		middleware.WriteError(w, http.StatusInternalServerError, "could not create a unique invite code")
	case http.MethodGet:
		codes := h.Store.ListSignupInviteCodes(middleware.UserID(r))
		if codes == nil {
			codes = []*models.SignupInviteCode{}
		}
		middleware.WriteJSON(w, http.StatusOK, codes)
	default:
		middleware.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
