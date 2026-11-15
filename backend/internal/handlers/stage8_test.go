package handlers

import (
	"net/http"
	"testing"
)

// Stage 8 (extras): change the account email with a 6-digit code
// sent to the NEW address.
func TestStage8ChangeEmail(t *testing.T) {
	h := newTestHandler()
	aliceTok, aliceID := register(t, h, "alice8@vyro.test", "alice8")
	_, _ = register(t, h, "bob8@vyro.test", "bob8")
	mailer := testMailer(h)

	newEmail := "alice-new8@vyro.test"
	rec := doJSON(t, h, http.MethodPost, "/api/users/me/change-email",
		map[string]interface{}{"new_email": newEmail}, aliceTok)
	if rec.Code != http.StatusCreated {
		t.Fatalf("request change: status %d body %s", rec.Code, rec.Body.String())
	}
	code := mailer.lastCode(newEmail)
	if code == "" {
		t.Fatal("no code was sent to the new address")
	}
	// Wrong code fails; the right code moves the account.
	rec = doJSON(t, h, http.MethodPost, "/api/users/me/change-email/verify",
		map[string]interface{}{"new_email": newEmail, "code": "000000"}, aliceTok)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("wrong code: status %d, want 401", rec.Code)
	}
	rec = doJSON(t, h, http.MethodPost, "/api/users/me/change-email/verify",
		map[string]interface{}{"new_email": newEmail, "code": code}, aliceTok)
	if rec.Code != http.StatusOK {
		t.Fatalf("verify change: status %d body %s", rec.Code, rec.Body.String())
	}
	var me map[string]interface{}
	mustDecode(t, rec, &me)
	if me["email"] != newEmail {
		t.Fatalf("email after change = %v, want %s", me["email"], newEmail)
	}
	// Login now works with the new address (register flow proves
	// the email index moved); the old address is free again.
	if _, err := h.Store.UserByEmail("alice8@vyro.test"); err == nil {
		t.Fatal("old email still resolves to a user")
	}
	if u, err := h.Store.UserByEmail(newEmail); err != nil || u.ID != aliceID {
		t.Fatalf("new email lookup failed: %v", err)
	}
	// Validation: bad address, own current address, taken address.
	for _, body := range []map[string]interface{}{
		{"new_email": "not-an-email"},
		{"new_email": newEmail},
		{"new_email": "bob8@vyro.test"},
	} {
		rec = doJSON(t, h, http.MethodPost, "/api/users/me/change-email", body, aliceTok)
		if rec.Code != http.StatusBadRequest && rec.Code != http.StatusConflict {
			t.Fatalf("change-email validation %v: status %d, want 400/409", body, rec.Code)
		}
	}
}
