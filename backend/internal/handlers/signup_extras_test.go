package handlers

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/vyro/backend/internal/models"
	"github.com/vyro/backend/internal/store"
)

var signupInviteCodeRe = regexp.MustCompile(`^[ABCDEFGHJKMNPQRSTUVWXYZ23456789]{8}$`)

func startInviteRegistration(t *testing.T, h *Handler, email, inviteCode string) {
	t.Helper()
	body := map[string]string{
		"email": email, "password": "password-123",
		"display_name": "Invite Tester", "invite_code": inviteCode,
	}
	rec := doJSON(t, h, http.MethodPost, "/api/auth/register", body, "")
	if rec.Code != http.StatusCreated {
		t.Fatalf("register %s: expected 201, got %d: %s", email, rec.Code, rec.Body.String())
	}
}

func completeRegistration(t *testing.T, h *Handler, email string) map[string]interface{} {
	t.Helper()
	code := testMailer(h).lastCode(email)
	if code == "" {
		t.Fatalf("registration email for %s did not contain a code", email)
	}
	rec := doJSON(t, h, http.MethodPost, "/api/auth/verify-register",
		map[string]string{"email": email, "code": code}, "")
	var out map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode verify-register: %v", err)
	}
	return out
}

func TestSignupInviteRegistrationFlow(t *testing.T) {
	h := newTestHandler()
	inviterToken, inviterID := register(t, h, "invite-owner@example.com", "inviteowner")

	rec := doJSON(t, h, http.MethodPost, "/api/signup-invites",
		map[string]int{"max_uses": 1, "expires_in_hours": 168}, inviterToken)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create signup invite: %d %s", rec.Code, rec.Body.String())
	}
	var created models.SignupInviteCode
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode signup invite: %v", err)
	}
	if !signupInviteCodeRe.MatchString(created.Code) || created.MaxUses != 1 || created.Uses != 0 {
		t.Fatalf("unexpected signup invite: %+v", created)
	}

	rec = doJSON(t, h, http.MethodGet, "/api/signup-invites", nil, inviterToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("list signup invites: %d %s", rec.Code, rec.Body.String())
	}
	var listed []models.SignupInviteCode
	if err := json.Unmarshal(rec.Body.Bytes(), &listed); err != nil {
		t.Fatalf("decode signup invite list: %v", err)
	}
	if len(listed) != 1 || listed[0].Code != created.Code {
		t.Fatalf("invite missing from list: %+v", listed)
	}

	const invitedEmail = "valid-invite@example.com"
	startInviteRegistration(t, h, invitedEmail, strings.ToLower(created.Code))
	out := completeRegistration(t, h, invitedEmail)
	if out["user"] == nil || out["token"] == nil {
		t.Fatalf("registration did not create a user: %v", out)
	}
	used, err := h.Store.SignupInviteCodeByCode(created.Code)
	if err != nil || used.Uses != 1 {
		t.Fatalf("invite usage should be 1, got %+v, %v", used, err)
	}

	rec = doJSON(t, h, http.MethodPost, "/api/auth/register", map[string]string{
		"email": "unknown-invite@example.com", "password": "password-123",
		"display_name": "Unknown", "invite_code": "ABCDEFGH",
	}, "")
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "invalid invite code") {
		t.Fatalf("unknown code should be rejected: %d %s", rec.Code, rec.Body.String())
	}
	rec = doJSON(t, h, http.MethodPost, "/api/auth/register", map[string]string{
		"email": "used-invite@example.com", "password": "password-123",
		"display_name": "Already used", "invite_code": created.Code,
	}, "")
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "invite code already used") {
		t.Fatalf("used code should be rejected: %d %s", rec.Code, rec.Body.String())
	}

	expired := &models.SignupInviteCode{
		Code: "EXPIRE23", InviterID: inviterID, MaxUses: 1,
		ExpiresAt: time.Now().UTC().Add(-time.Hour),
	}
	if err := h.Store.CreateSignupInviteCode(expired); err != nil {
		t.Fatalf("create expired test code: %v", err)
	}
	rec = doJSON(t, h, http.MethodPost, "/api/auth/register", map[string]string{
		"email": "expired-invite@example.com", "password": "password-123",
		"display_name": "Expired", "invite_code": expired.Code,
	}, "")
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "invite code expired") {
		t.Fatalf("expired code should be rejected: %d %s", rec.Code, rec.Body.String())
	}

	const freePathEmail = "free-path@example.com"
	rec = doJSON(t, h, http.MethodPost, "/api/auth/register", map[string]string{
		"email": freePathEmail, "password": "password-123", "display_name": "No Invite",
	}, "")
	if rec.Code != http.StatusCreated {
		t.Fatalf("registration without invite should work: %d %s", rec.Code, rec.Body.String())
	}
	out = completeRegistration(t, h, freePathEmail)
	if out["user"] == nil {
		t.Fatalf("free signup did not create a user: %v", out)
	}
}

func TestInviteConsumedBeforeVerificationDoesNotCreateUser(t *testing.T) {
	h := newTestHandler()
	_, inviterID := register(t, h, "racing-owner@example.com", "raceowner")
	invite := &models.SignupInviteCode{
		Code: "RACE2345", InviterID: inviterID, MaxUses: 1,
		ExpiresAt: time.Now().UTC().Add(time.Hour),
	}
	if err := h.Store.CreateSignupInviteCode(invite); err != nil {
		t.Fatalf("create invite: %v", err)
	}

	const email = "racing-signup@example.com"
	startInviteRegistration(t, h, email, invite.Code)
	if err := h.Store.UseSignupInviteCode(invite.Code); err != nil {
		t.Fatalf("consume invite concurrently: %v", err)
	}
	code := testMailer(h).lastCode(email)
	rec := doJSON(t, h, http.MethodPost, "/api/auth/verify-register",
		map[string]string{"email": email, "code": code}, "")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("verification with exhausted invite should fail: %d %s", rec.Code, rec.Body.String())
	}
	if _, err := h.Store.UserByEmail(email); err != store.ErrNotFound {
		t.Fatalf("user must not be created when invite consumption fails, got %v", err)
	}
}

func TestRegisterAttachesFirstDeviceWithoutNewDeviceNotice(t *testing.T) {
	h := newTestHandler()
	const email = "first-device@example.com"
	rec := doJSON(t, h, http.MethodPost, "/api/auth/register", map[string]string{
		"email": email, "password": "password-123", "display_name": "First Device",
		"device_id": "first-device", "device_name": "My phone", "platform": "android",
	}, "")
	if rec.Code != http.StatusCreated {
		t.Fatalf("register: %d %s", rec.Code, rec.Body.String())
	}
	code := testMailer(h).lastCode(email)
	rec = doJSON(t, h, http.MethodPost, "/api/auth/verify-register", map[string]string{
		"email": email, "code": code,
	}, "")
	if rec.Code != http.StatusCreated {
		t.Fatalf("verify-register: %d %s", rec.Code, rec.Body.String())
	}
	var response map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode verify-register: %v", err)
	}
	if response["new_device"] != false {
		t.Fatalf("first registered device should not be reported as new: %v", response)
	}
	user, err := h.Store.UserByEmail(email)
	if err != nil {
		t.Fatalf("find registered user: %v", err)
	}
	devices := h.Store.ListDevices(user.ID)
	if len(devices) != 1 || devices[0].DeviceID != "first-device" {
		t.Fatalf("first device was not stored: %+v", devices)
	}
}

func TestDeviceVerificationAndRevocation(t *testing.T) {
	h := newTestHandler()
	legacyToken, userID := register(t, h, "device-user@example.com", "deviceuser")
	const deviceID = "phone-verified"
	device := map[string]string{
		"device_id": deviceID, "device_name": "Test phone", "platform": "ios",
	}

	loginAndVerify := func() map[string]interface{} {
		t.Helper()
		loginBody := map[string]string{
			"email": "device-user@example.com", "password": "password-123",
			"device_id": deviceID, "device_name": "Test phone", "platform": "ios",
		}
		rec := doJSON(t, h, http.MethodPost, "/api/auth/login", loginBody, "")
		if rec.Code != http.StatusOK {
			t.Fatalf("login: %d %s", rec.Code, rec.Body.String())
		}
		code := testMailer(h).lastCode("device-user@example.com")
		if code == "" {
			t.Fatal("login email did not contain a code")
		}
		verifyBody := map[string]string{
			"email": "device-user@example.com", "code": code,
			"device_id": device["device_id"], "device_name": device["device_name"],
			"platform": device["platform"],
		}
		rec = doJSON(t, h, http.MethodPost, "/api/auth/verify-login", verifyBody, "")
		if rec.Code != http.StatusOK {
			t.Fatalf("verify-login: %d %s", rec.Code, rec.Body.String())
		}
		var out map[string]interface{}
		if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
			t.Fatalf("decode verify-login: %v", err)
		}
		return out
	}

	first := loginAndVerify()
	if first["new_device"] != true || first["device_id"] != deviceID {
		t.Fatalf("first login should be a new device: %v", first)
	}
	devices := h.Store.ListDevices(userID)
	if len(devices) != 1 || devices[0].DeviceID != deviceID {
		t.Fatalf("new device was not registered: %+v", devices)
	}
	second := loginAndVerify()
	if second["new_device"] != false {
		t.Fatalf("second login should be trusted: %v", second)
	}

	rec := doJSON(t, h, http.MethodDelete, "/api/devices/"+deviceID, nil,
		second["token"].(string))
	if rec.Code != http.StatusOK {
		t.Fatalf("remove device: %d %s", rec.Code, rec.Body.String())
	}
	rec = doJSON(t, h, http.MethodGet, "/api/devices", nil, second["token"].(string))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("revoked device token should fail, got %d: %s", rec.Code, rec.Body.String())
	}
	rec = doJSON(t, h, http.MethodGet, "/api/devices", nil, legacyToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("legacy token without device claim should remain valid: %d %s", rec.Code, rec.Body.String())
	}

	rec = doJSON(t, h, http.MethodPost, "/api/auth/login", map[string]string{
		"email": "device-user@example.com", "password": "password-123",
		"device_id": deviceID, "platform": "phone",
	}, "")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("invalid platform should be rejected: %d %s", rec.Code, rec.Body.String())
	}
}
