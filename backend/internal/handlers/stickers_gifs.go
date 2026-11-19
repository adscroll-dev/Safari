package handlers

import (
	"embed"
	"net/http"
	"strconv"
	"strings"

	"github.com/vyro/backend/internal/middleware"
)

//go:embed static/stickers/*.webp static/gifs/*.gif
var catalogFiles embed.FS

type catalogFile struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	FileURL string `json:"file_url"`
	Width   int    `json:"width"`
	Height  int    `json:"height"`
	Size    int    `json:"size"`
}

type stickerPack struct {
	ID           string        `json:"id"`
	Title        string        `json:"title"`
	Cover        string        `json:"cover"`
	StickerCount int           `json:"sticker_count"`
	Stickers     []catalogFile `json:"stickers,omitempty"`
}

var safariStickerDefinitions = []catalogFile{
	{ID: "safari-planet", Name: "Safari Planet", FileURL: "/stickers/safari-planet.webp", Width: 384, Height: 384},
	{ID: "happy", Name: "Happy", FileURL: "/stickers/happy.webp", Width: 384, Height: 384},
	{ID: "laughing", Name: "Laughing", FileURL: "/stickers/laughing.webp", Width: 384, Height: 384},
	{ID: "love", Name: "Love", FileURL: "/stickers/love.webp", Width: 384, Height: 384},
	{ID: "sad", Name: "Sad", FileURL: "/stickers/sad.webp", Width: 384, Height: 384},
	{ID: "angry", Name: "Angry", FileURL: "/stickers/angry.webp", Width: 384, Height: 384},
	{ID: "wow", Name: "Wow", FileURL: "/stickers/wow.webp", Width: 384, Height: 384},
	{ID: "thinking", Name: "Thinking", FileURL: "/stickers/thinking.webp", Width: 384, Height: 384},
	{ID: "thumbs-up", Name: "Thumbs Up", FileURL: "/stickers/thumbs-up.webp", Width: 384, Height: 384},
	{ID: "good-night", Name: "Good Night", FileURL: "/stickers/good-night.webp", Width: 384, Height: 384},
	{ID: "celebration", Name: "Celebration", FileURL: "/stickers/celebration.webp", Width: 384, Height: 384},
}

var safariGIFDefinitions = []catalogFile{
	{ID: "planet-spin", Name: "Spinning Safari Planet", FileURL: "/gifs/planet-spin.gif", Width: 256, Height: 256},
	{ID: "heart-bounce", Name: "Bouncing Heart", FileURL: "/gifs/heart-bounce.gif", Width: 256, Height: 256},
	{ID: "waving-planet", Name: "Waving Planet", FileURL: "/gifs/waving-planet.gif", Width: 256, Height: 256},
	{ID: "pulsing-hi", Name: "Pulsing Hi", FileURL: "/gifs/pulsing-hi.gif", Width: 256, Height: 256},
	{ID: "laughing-face", Name: "Laughing Face", FileURL: "/gifs/laughing-face.gif", Width: 256, Height: 256},
	{ID: "rocket-launch", Name: "Rocket Launch", FileURL: "/gifs/rocket-launch.gif", Width: 256, Height: 256},
}

func (h *Handler) ListStickerPacks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		middleware.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	middleware.WriteJSON(w, http.StatusOK, []stickerPack{{
		ID: "safari-expressions", Title: "Safari Expressions",
		Cover: safariStickerDefinitions[0].FileURL, StickerCount: len(safariStickerDefinitions),
	}})
}

func (h *Handler) GetStickerPack(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		middleware.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/api/stickers/packs/")
	if id != "safari-expressions" {
		middleware.WriteError(w, http.StatusNotFound, "sticker pack not found")
		return
	}
	stickers, err := catalogFilesWithSizes(safariStickerDefinitions)
	if err != nil {
		middleware.WriteError(w, http.StatusInternalServerError, "could not read sticker catalog")
		return
	}
	middleware.WriteJSON(w, http.StatusOK, stickerPack{
		ID: id, Title: "Safari Expressions", Cover: safariStickerDefinitions[0].FileURL,
		StickerCount: len(stickers), Stickers: stickers,
	})
}

func (h *Handler) ListGIFs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		middleware.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	query := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q")))
	filtered := make([]catalogFile, 0, len(safariGIFDefinitions))
	for _, item := range safariGIFDefinitions {
		if query == "" || strings.Contains(strings.ToLower(item.Name), query) {
			filtered = append(filtered, item)
		}
	}
	items, err := catalogFilesWithSizes(filtered)
	if err != nil {
		middleware.WriteError(w, http.StatusInternalServerError, "could not read GIF catalog")
		return
	}
	middleware.WriteJSON(w, http.StatusOK, items)
}

func catalogFilesWithSizes(definitions []catalogFile) ([]catalogFile, error) {
	items := make([]catalogFile, 0, len(definitions))
	for _, item := range definitions {
		data, err := catalogFileBytes(item.FileURL)
		if err != nil {
			return nil, err
		}
		item.Size = len(data)
		items = append(items, item)
	}
	return items, nil
}

func catalogFileBytes(fileURL string) ([]byte, error) {
	switch {
	case strings.HasPrefix(fileURL, "/stickers/"):
		return catalogFiles.ReadFile("static/stickers/" + strings.TrimPrefix(fileURL, "/stickers/"))
	case strings.HasPrefix(fileURL, "/gifs/"):
		return catalogFiles.ReadFile("static/gifs/" + strings.TrimPrefix(fileURL, "/gifs/"))
	default:
		return nil, http.ErrMissingFile
	}
}

func isCatalogFileURL(fileURL string) bool {
	if !strings.HasPrefix(fileURL, "/stickers/") && !strings.HasPrefix(fileURL, "/gifs/") {
		return false
	}
	_, err := catalogFileBytes(fileURL)
	return err == nil
}

func validMediaFileURL(fileURL string) bool {
	if strings.HasPrefix(fileURL, "/uploads/") {
		return !strings.Contains(fileURL, "..")
	}
	return isCatalogFileURL(fileURL)
}

func (h *Handler) ServeCatalogFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		middleware.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	data, err := catalogFileBytes(r.URL.Path)
	if err != nil {
		middleware.WriteError(w, http.StatusNotFound, "not found")
		return
	}
	contentType := "image/webp"
	if strings.HasPrefix(r.URL.Path, "/gifs/") {
		contentType = "image/gif"
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	w.Header().Set("Cache-Control", "private, max-age=86400, immutable")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}
