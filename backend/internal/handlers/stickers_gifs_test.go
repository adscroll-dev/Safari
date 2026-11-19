package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestStickerAndGIFCatalog(t *testing.T) {
	h := newTestHandler()
	token, _ := register(t, h, "catalog@example.com", "cataloguser")

	packsRec := doJSON(t, h, http.MethodGet, "/api/stickers/packs", nil, token)
	if packsRec.Code != http.StatusOK {
		t.Fatalf("list sticker packs: status %d body %s", packsRec.Code, packsRec.Body.String())
	}
	var packs []stickerPack
	if err := json.Unmarshal(packsRec.Body.Bytes(), &packs); err != nil {
		t.Fatalf("decode sticker packs: %v", err)
	}
	if len(packs) != 1 || packs[0].ID != "safari-expressions" || packs[0].StickerCount != 11 {
		t.Fatalf("unexpected sticker packs: %+v", packs)
	}
	packRec := doJSON(t, h, http.MethodGet, "/api/stickers/packs/safari-expressions", nil, token)
	if packRec.Code != http.StatusOK {
		t.Fatalf("get sticker pack: status %d body %s", packRec.Code, packRec.Body.String())
	}
	var pack stickerPack
	if err := json.Unmarshal(packRec.Body.Bytes(), &pack); err != nil {
		t.Fatalf("decode sticker pack: %v", err)
	}
	if len(pack.Stickers) != 11 {
		t.Fatalf("expected 11 stickers, got %d", len(pack.Stickers))
	}
	for _, sticker := range pack.Stickers {
		if !isCatalogFileURL(sticker.FileURL) || !strings.HasPrefix(sticker.FileURL, "/stickers/") || sticker.Size <= 0 {
			t.Fatalf("invalid sticker catalog file: %+v", sticker)
		}
	}
	unknownRec := doJSON(t, h, http.MethodGet, "/api/stickers/packs/missing", nil, token)
	if unknownRec.Code != http.StatusNotFound {
		t.Fatalf("unknown sticker pack: expected 404, got %d", unknownRec.Code)
	}

	gifsRec := doJSON(t, h, http.MethodGet, "/api/gifs", nil, token)
	if gifsRec.Code != http.StatusOK {
		t.Fatalf("list GIFs: status %d body %s", gifsRec.Code, gifsRec.Body.String())
	}
	var gifs []catalogFile
	if err := json.Unmarshal(gifsRec.Body.Bytes(), &gifs); err != nil {
		t.Fatalf("decode GIF list: %v", err)
	}
	if len(gifs) != 6 {
		t.Fatalf("expected 6 GIFs, got %d", len(gifs))
	}
	for _, gif := range gifs {
		if !isCatalogFileURL(gif.FileURL) || !strings.HasPrefix(gif.FileURL, "/gifs/") || gif.Size <= 0 {
			t.Fatalf("invalid GIF catalog file: %+v", gif)
		}
	}
	filteredRec := doJSON(t, h, http.MethodGet, "/api/gifs?q=planet", nil, token)
	if filteredRec.Code != http.StatusOK {
		t.Fatalf("filter GIFs: status %d body %s", filteredRec.Code, filteredRec.Body.String())
	}
	var filtered []catalogFile
	if err := json.Unmarshal(filteredRec.Body.Bytes(), &filtered); err != nil {
		t.Fatalf("decode filtered GIF list: %v", err)
	}
	if len(filtered) != 2 {
		t.Fatalf("expected 2 planet GIFs, got %d: %+v", len(filtered), filtered)
	}
	for _, gif := range filtered {
		if !strings.Contains(strings.ToLower(gif.Name), "planet") {
			t.Fatalf("non-planet result in filtered GIF list: %+v", gif)
		}
	}

	for _, tc := range []struct {
		path        string
		contentType string
	}{
		{"/stickers/happy.webp", "image/webp"},
		{"/gifs/planet-spin.gif", "image/gif"},
	} {
		fileRec := doJSON(t, h, http.MethodGet, tc.path, nil, token)
		if fileRec.Code != http.StatusOK || fileRec.Header().Get("Content-Type") != tc.contentType || fileRec.Body.Len() == 0 {
			t.Errorf("serve %s: status %d content-type %q bytes %d", tc.path, fileRec.Code, fileRec.Header().Get("Content-Type"), fileRec.Body.Len())
		}
	}
}

func TestSendGIFMessageAndRejectUnknownKind(t *testing.T) {
	h := newTestHandler()
	aliceToken, _ := register(t, h, "gif-alice@example.com", "gifalice")
	_, bobID := register(t, h, "gif-bob@example.com", "gifbob")
	chatRec := doJSON(t, h, http.MethodPost, "/api/chats", map[string]interface{}{
		"type": "dm", "member_ids": []string{bobID},
	}, aliceToken)
	if chatRec.Code != http.StatusCreated {
		t.Fatalf("create chat: status %d body %s", chatRec.Code, chatRec.Body.String())
	}
	var chat struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(chatRec.Body.Bytes(), &chat); err != nil {
		t.Fatalf("decode chat: %v", err)
	}
	if chat.ID == "" {
		t.Fatalf("chat response missing id: %s", chatRec.Body.String())
	}
	gifRec := doJSON(t, h, http.MethodPost, "/api/chats/"+chat.ID+"/messages", map[string]interface{}{
		"content": "", "kind": "gif", "file_url": "/gifs/planet-spin.gif",
		"file_name": "planet-spin.gif", "file_size": 100, "mime_type": "image/gif",
	}, aliceToken)
	if gifRec.Code != http.StatusCreated && gifRec.Code != http.StatusOK {
		t.Fatalf("send GIF message: status %d body %s", gifRec.Code, gifRec.Body.String())
	}
	unknownRec := doJSON(t, h, http.MethodPost, "/api/chats/"+chat.ID+"/messages", map[string]interface{}{
		"content": "not valid", "kind": "unknown-kind",
	}, aliceToken)
	if unknownRec.Code != http.StatusBadRequest {
		t.Fatalf("unknown message kind: expected 400, got %d body %s", unknownRec.Code, unknownRec.Body.String())
	}
}
