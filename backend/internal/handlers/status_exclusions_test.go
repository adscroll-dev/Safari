package handlers

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/vyro/backend/internal/middleware"
	"github.com/vyro/backend/internal/models"
)

func TestStatusExclusionsVisibilityAndMedia(t *testing.T) {
	h := newTestHandler()
	h.UploadDir = t.TempDir()
	aliceTok, aliceID := register(t, h, "status-alice@example.com", "status-alice")
	bobTok, bobID := register(t, h, "status-bob@example.com", "status-bob")
	carolTok, _ := register(t, h, "status-carol@example.com", "status-carol")

	mediaURL := "/uploads/status-photo.jpg"
	if err := os.WriteFile(filepath.Join(h.UploadDir, filepath.Base(mediaURL)), []byte("story image"), 0o600); err != nil {
		t.Fatalf("write story media: %v", err)
	}
	storyRec := doJSON(t, h, http.MethodPost, "/api/stories", map[string]string{
		"media_url": mediaURL,
	}, aliceTok)
	if storyRec.Code != http.StatusCreated {
		t.Fatalf("create story: %d %s", storyRec.Code, storyRec.Body.String())
	}
	var story models.Story
	mustDecode(t, storyRec, &story)

	setPrivacy := doJSON(t, h, http.MethodPut, "/api/users/me/status-privacy",
		map[string]string{"status_privacy": "except"}, aliceTok)
	if setPrivacy.Code != http.StatusOK {
		t.Fatalf("set except privacy: %d %s", setPrivacy.Code, setPrivacy.Body.String())
	}
	add := func(userID string) *httptest.ResponseRecorder {
		t.Helper()
		return doJSON(t, h, http.MethodPost, "/api/users/me/status-exclusions",
			map[string]string{"user_id": userID}, aliceTok)
	}
	if rec := add(bobID); rec.Code != http.StatusCreated {
		t.Fatalf("add bob exclusion: %d %s", rec.Code, rec.Body.String())
	}
	if rec := add(bobID); rec.Code != http.StatusOK {
		t.Fatalf("re-add bob exclusion: %d %s", rec.Code, rec.Body.String())
	}
	if rec := add(aliceID); rec.Code != http.StatusBadRequest {
		t.Fatalf("exclude self: %d, want 400", rec.Code)
	}
	if rec := add("missing-user"); rec.Code != http.StatusNotFound {
		t.Fatalf("exclude unknown user: %d, want 404", rec.Code)
	}

	fetchMedia := func(token string) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(http.MethodGet, mediaURL, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		middleware.Auth("test-secret")(http.HandlerFunc(h.ServeUpload)).ServeHTTP(rec, req)
		return rec
	}
	assertAllowed := func(label, token string) {
		t.Helper()
		rec := doJSON(t, h, http.MethodGet, "/api/stories", nil, token)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s list stories: %d %s", label, rec.Code, rec.Body.String())
		}
		var stories []models.Story
		mustDecode(t, rec, &stories)
		found := false
		for _, got := range stories {
			found = found || got.ID == story.ID
		}
		if !found {
			t.Errorf("%s cannot see the owner's story in feed", label)
		}
		if rec := doJSON(t, h, http.MethodPost, "/api/stories/"+story.ID+"/view", nil, token); rec.Code != http.StatusOK {
			t.Errorf("%s view story: %d, want 200", label, rec.Code)
		}
		if rec := doJSON(t, h, http.MethodPost, "/api/stories/"+story.ID+"/reactions",
			map[string]string{"emoji": "❤️"}, token); rec.Code != http.StatusOK {
			t.Errorf("%s react to story: %d, want 200", label, rec.Code)
		}
		if rec := fetchMedia(token); rec.Code != http.StatusOK {
			t.Errorf("%s fetch story media: %d, want 200", label, rec.Code)
		}
	}

	rec := doJSON(t, h, http.MethodGet, "/api/stories", nil, bobTok)
	var bobStories []models.Story
	mustDecode(t, rec, &bobStories)
	for _, got := range bobStories {
		if got.ID == story.ID {
			t.Fatal("excluded viewer sees owner's story in feed")
		}
	}
	if rec := doJSON(t, h, http.MethodPost, "/api/stories/"+story.ID+"/view", nil, bobTok); rec.Code != http.StatusForbidden {
		t.Fatalf("excluded viewer view: %d, want 403", rec.Code)
	}
	if rec := doJSON(t, h, http.MethodPost, "/api/stories/"+story.ID+"/reactions",
		map[string]string{"emoji": "❤️"}, bobTok); rec.Code != http.StatusForbidden {
		t.Fatalf("excluded viewer react: %d, want 403", rec.Code)
	}
	if rec := fetchMedia(bobTok); rec.Code != http.StatusForbidden {
		t.Fatalf("excluded viewer media: %d, want 403", rec.Code)
	}

	assertAllowed("non-excluded viewer", carolTok)
	remove := doJSON(t, h, http.MethodDelete, "/api/users/me/status-exclusions/"+bobID, nil, aliceTok)
	if remove.Code != http.StatusOK {
		t.Fatalf("remove bob exclusion: %d %s", remove.Code, remove.Body.String())
	}
	assertAllowed("removed viewer", bobTok)

	if rec := doJSON(t, h, http.MethodGet, "/api/users/me/status-exclusions", nil, aliceTok); rec.Code != http.StatusOK || rec.Body.String() != "[]\n" {
		t.Fatalf("empty exclusions: %d %s", rec.Code, rec.Body.String())
	}
	assertAllowed("viewer with empty except list", carolTok)

	if rec := doJSON(t, h, http.MethodPut, "/api/users/me/status-privacy",
		map[string]string{"status_privacy": "nobody"}, aliceTok); rec.Code != http.StatusOK {
		t.Fatalf("set nobody privacy: %d %s", rec.Code, rec.Body.String())
	}
	if rec := fetchMedia(carolTok); rec.Code != http.StatusForbidden {
		t.Fatalf("fetch nobody status media: %d, want 403", rec.Code)
	}
	if rec := fetchMedia(aliceTok); rec.Code != http.StatusOK {
		t.Fatalf("owner fetch own nobody status media: %d, want 200", rec.Code)
	}
}
