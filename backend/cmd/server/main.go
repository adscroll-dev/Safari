// Command server is the VYRO backend entry point.
//
// Run: go run ./cmd/server        (or: go build -o vyro-server ./cmd/server)
// Env: PORT (default 8080), JWT_SECRET, JWT_EXPIRY_HOURS, DATABASE_URL,
//
//	REDIS_URL / REDIS_ADDR, CORS_ALLOW_ORIGINS, UPLOAD_DIR,
//	SMTP_HOST / SMTP_PORT / SMTP_USER / SMTP_PASS / SMTP_FROM /
//	SMTP_ALLOW_NOAUTH (email verification, v0.7),
//	FIREBASE_SERVICE_ACCOUNT_JSON / FIREBASE_SERVICE_ACCOUNT_PATH
//	(FCM push for the mobile app; empty = FCM disabled).
//
// In v0.2+ the store is PostgreSQL when DATABASE_URL is set: data survives
// a restart. Without DATABASE_URL it stays in-memory (dev only).
// File uploads (POST /api/uploads, GET /uploads/{name}) store files under
// UPLOAD_DIR (default ./uploads), capped at MAX_UPLOAD_MB (default 50).
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/vyro/backend/internal/auth"
	"github.com/vyro/backend/internal/config"
	"github.com/vyro/backend/internal/handlers"
	"github.com/vyro/backend/internal/mail"
	"github.com/vyro/backend/internal/middleware"
	"github.com/vyro/backend/internal/push"
	"github.com/vyro/backend/internal/store"
	"github.com/vyro/backend/internal/ws"
)

func main() {
	cfg := config.Load()
	if os.Getenv("JWT_SECRET") == "" {
		log.Println("WARNING: JWT_SECRET is not set; using dev default. Set a strong secret in production.")
	}
	var s store.StoreIface
	storeName := "memory"
	if cfg.DatabaseURL != "" {
		pg, err := store.NewPostgresStore(context.Background(), cfg.DatabaseURL)
		if err != nil {
			// Fail loudly: silently falling back to memory would lose data.
			log.Fatalf("PostgreSQL store failed: %v", err)
		}
		defer pg.Close()
		s = pg
		storeName = "postgres"
		log.Println("VYRO store: PostgreSQL (data survives restart)")
	} else {
		s = store.NewMemoryStore()
		log.Println("VYRO store: in-memory (data lost on restart; set DATABASE_URL for PostgreSQL)")
	}
	hub := ws.NewHub()
	h := handlers.New(s, hub, cfg.JWTSecret, cfg.JWTExpiry)
	// Email verification (v0.7): Resend HTTPS API when RESEND_API_KEY
	// is set (preferred: plain SMTP hangs from some hosts — on Render's
	// free tier an SMTP dial to Resend never completed, hanging every
	// registration with no error at all), else SMTP via env (defaults
	// to Gmail's relay, smtp.gmail.com:587). No sender means email is
	// not configured: register/login answer 503 and never fake a code.
	// SMTP_ALLOW_NOAUTH=true skips SMTP AUTH (local test sink).
	if cfg.ResendAPIKey != "" {
		h.Mailer = mail.NewResend(cfg.ResendAPIKey, cfg.SMTPFrom)
		if h.Mailer != nil {
			log.Printf("VYRO email: configured (Resend API, from %s)", cfg.SMTPFrom)
		}
	}
	if h.Mailer == nil {
		h.Mailer = mail.NewSMTP(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUser, cfg.SMTPPass, cfg.SMTPFrom, cfg.SMTPAllowNoAuth)
		if h.Mailer != nil {
			log.Printf("VYRO email: configured (SMTP %s:%d, from %s)", cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPFrom)
		}
	}
	if h.Mailer == nil {
		log.Println("VYRO email: not configured — verification codes cannot be sent")
	}
	// Web Push (VAPID, RFC 8292): free, no FCM/APNs account. Keys from
	// env when set; otherwise generate an ephemeral pair for this run
	// and warn — subscriptions from a previous pair stop validating
	// after a restart, so production must set VAPID_* env vars.
	// Subscriptions are persisted in the store (PostgreSQL when
	// DATABASE_URL is set), so they survive a restart (v0.6).
	h.Push = push.NewRegistryWithStore(s)
	h.VAPIDSubject = cfg.VAPIDSubject
	if cfg.VAPIDPublicKey != "" && cfg.VAPIDPrivateKey != "" {
		h.VAPIDPublicKey, h.VAPIDPrivateKey = cfg.VAPIDPublicKey, cfg.VAPIDPrivateKey
		log.Println("VYRO push: Web Push (VAPID) enabled")
	} else {
		pub, priv, err := push.GenerateVAPIDKeys()
		if err != nil {
			log.Printf("WARNING: could not generate VAPID keys (%v); push disabled", err)
		} else {
			h.VAPIDPublicKey, h.VAPIDPrivateKey = pub, priv
			log.Println("WARNING: VAPID keys generated for this run only; set VAPID_PUBLIC_KEY/VAPID_PRIVATE_KEY to keep push subscriptions across restarts")
			// The public key is safe to log; the operator can persist
			// this pair (public key here, private key is NOT printed)
			// by generating a pair out-of-band instead.
			log.Printf("VYRO push: ephemeral VAPID public key: %s", pub)
		}
	}
	// FCM push for the mobile app (Firebase Cloud Messaging): the
	// service-account JSON comes only from the environment —
	// FIREBASE_SERVICE_ACCOUNT_JSON (raw JSON, e.g. a Render env var)
	// or FIREBASE_SERVICE_ACCOUNT_PATH (a secret file). When neither
	// is set FCM stays disabled and the server behaves exactly as
	// before (Web Push only). A set but invalid credential fails the
	// startup loudly: a half-configured push path must not fail
	// silently and then surprise users with missing notifications.
	// Tokens are persisted in the store (PostgreSQL when DATABASE_URL
	// is set), so they survive a restart.
	fcmClient, err := push.NewFCMClientFromEnv(context.Background())
	if err != nil {
		log.Fatalf("FCM configuration failed: %v", err)
	}
	if fcmClient != nil {
		h.FCM = push.NewFCMRegistry(s, fcmClient)
		log.Println("VYRO push: FCM enabled")
	} else {
		log.Println("VYRO push: FCM not configured — set FIREBASE_SERVICE_ACCOUNT_JSON (or FIREBASE_SERVICE_ACCOUNT_PATH) to enable mobile push")
	}
	// Redis-backed login rate limiting when REDIS_URL is set and Redis
	// answers; otherwise the in-memory limiter stays in place. Redis
	// 7.0.15 was tested live in v0.5 (see README): the limiter works
	// against real Redis; go-redis v9.8+ is pinned because older
	// versions send CLIENT SETINFO, which Redis < 7.2 rejects.
	if cfg.RedisURL != "" {
		if rl, err := auth.NewRedisLimiter(cfg.RedisURL, 5, 15*time.Minute); err == nil {
			h.LoginLimits = rl
			log.Println("VYRO rate limiting: Redis-backed")
		} else {
			log.Printf("WARNING: REDIS_URL set but Redis unreachable (%v); using in-memory rate limiting", err)
		}
	}
	// Scheduled-message dispatcher: every 5s, send what is due.
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			h.ProcessDueScheduled()
		}
	}()
	h.StoreName = storeName
	h.UploadDir = cfg.UploadDir
	h.MaxUploadMB = cfg.MaxUploadMB

	mux := http.NewServeMux()
	authMW := middleware.Auth(cfg.JWTSecret)

	// Public.
	mux.HandleFunc("/health", h.Health)
	mux.HandleFunc("/api/auth/register", h.Register)
	mux.HandleFunc("/api/auth/verify-register", h.VerifyRegister)
	mux.HandleFunc("/api/auth/login", h.Login)
	mux.HandleFunc("/api/auth/verify-login", h.VerifyLogin)
	mux.HandleFunc("/api/auth/resend-code", h.ResendCode)
	// Bot token endpoints (X-Bot-Token header, no JWT).
	mux.HandleFunc("/api/bot/updates", h.BotUpdates)
	mux.HandleFunc("/api/bot/send", h.BotSend)

	// Authenticated.
	mux.Handle("/api/uploads", authMW(http.HandlerFunc(h.Upload)))
	mux.Handle("/uploads/", authMW(http.HandlerFunc(h.ServeUpload)))
	mux.Handle("/api/users/me", authMW(http.HandlerFunc(h.Me)))
	mux.Handle("/api/users", authMW(http.HandlerFunc(h.ListUsers)))
	mux.Handle("/api/users/", authMW(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/presence") {
			h.Presence(w, r)
			return
		}
		if strings.HasSuffix(r.URL.Path, "/devices") {
			h.ListUserDevices(w, r)
			return
		}
		h.BlockUser(w, r)
	})))
	// Multi-device registry (v0.6).
	mux.Handle("/api/devices", authMW(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			h.ListMyDevices(w, r)
		case http.MethodPost:
			h.RegisterDevice(w, r)
		default:
			middleware.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	})))
	mux.Handle("/api/devices/", authMW(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			h.RemoveDevice(w, r)
		} else {
			middleware.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	})))
	mux.Handle("/api/chats", authMW(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			h.ListChats(w, r)
		case http.MethodPost:
			h.CreateChat(w, r)
		default:
			middleware.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	})))
	mux.Handle("/api/chats/", authMW(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		switch {
		case strings.HasSuffix(path, "/messages"):
			switch r.Method {
			case http.MethodGet:
				h.ListMessages(w, r)
			case http.MethodPost:
				h.SendMessage(w, r)
			default:
				middleware.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
			}
		case strings.HasSuffix(path, "/scheduled"):
			switch r.Method {
			case http.MethodGet:
				h.ListScheduled(w, r)
			case http.MethodPost:
				h.ScheduleMessage(w, r)
			default:
				middleware.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
			}
		case strings.HasSuffix(path, "/sender-keys"):
			// Group E2EE sender-key distribution (v0.6).
			switch r.Method {
			case http.MethodGet:
				h.ListGroupSenderKeys(w, r)
			case http.MethodPost:
				h.UploadGroupSenderKey(w, r)
			default:
				middleware.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
			}
		case strings.Contains(path, "/members/") && strings.HasSuffix(path, "/role"):
			if r.Method == http.MethodPost {
				h.SetMemberRole(w, r)
			} else {
				middleware.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
			}
		case strings.Contains(path, "/members/"):
			if r.Method == http.MethodDelete {
				h.RemoveMember(w, r)
			} else {
				middleware.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
			}
		case strings.HasSuffix(path, "/members"):
			switch r.Method {
			case http.MethodGet:
				h.ListMembers(w, r)
			case http.MethodPost:
				h.AddMember(w, r)
			default:
				middleware.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
			}
		case strings.HasSuffix(path, "/invites"):
			// Group/channel invite links (Telegram-style).
			switch r.Method {
			case http.MethodGet:
				h.ListInvites(w, r)
			case http.MethodPost:
				h.CreateInvite(w, r)
			default:
				middleware.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
			}
		case strings.Contains(path, "/invites/"):
			if r.Method == http.MethodDelete {
				h.RevokeInvite(w, r)
			} else {
				middleware.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
			}
		case strings.HasSuffix(path, "/pin"):
			h.SetChatSetting(w, r, "pin")
		case strings.HasSuffix(path, "/unpin"):
			h.SetChatSetting(w, r, "unpin")
		case strings.HasSuffix(path, "/archive"):
			h.SetChatSetting(w, r, "archive")
		case strings.HasSuffix(path, "/unarchive"):
			h.SetChatSetting(w, r, "unarchive")
		case strings.HasSuffix(path, "/folder"):
			h.SetChatSetting(w, r, "folder")
		default:
			middleware.WriteError(w, http.StatusNotFound, "not found")
		}
	})))
	mux.Handle("/api/messages/search", authMW(http.HandlerFunc(h.SearchMessages)))
	// Invite-link preview/join (Telegram-style): /api/invites/{token}
	// and /api/invites/{token}/join.
	mux.Handle("/api/invites/", authMW(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/join") {
			if r.Method == http.MethodPost {
				h.JoinViaInvite(w, r)
			} else {
				middleware.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
			}
			return
		}
		if r.Method == http.MethodGet {
			h.PreviewInvite(w, r)
		} else {
			middleware.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	})))
	mux.Handle("/api/messages/", authMW(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/reactions") {
			switch r.Method {
			case http.MethodPost:
				h.AddReaction(w, r)
			case http.MethodDelete:
				h.RemoveReaction(w, r)
			default:
				middleware.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
			}
			return
		}
		if strings.HasSuffix(r.URL.Path, "/read") {
			if r.Method == http.MethodPost {
				h.MarkRead(w, r)
			} else {
				middleware.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
			}
			return
		}
		switch r.Method {
		case http.MethodPut:
			h.EditMessage(w, r)
		case http.MethodDelete:
			h.DeleteMessage(w, r)
		default:
			middleware.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	})))
	mux.Handle("/api/scheduled/", authMW(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			h.CancelScheduled(w, r)
		} else {
			middleware.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	})))
	mux.Handle("/api/stories", authMW(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			h.ListStories(w, r)
		case http.MethodPost:
			h.CreateStory(w, r)
		default:
			middleware.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	})))
	mux.Handle("/api/bots", authMW(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			h.ListBots(w, r)
		case http.MethodPost:
			h.CreateBot(w, r)
		default:
			middleware.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	})))
	mux.Handle("/api/bots/", authMW(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/webhook") {
			h.SetBotWebhook(w, r)
			return
		}
		if r.Method == http.MethodDelete {
			h.DeleteBot(w, r)
			return
		}
		middleware.WriteError(w, http.StatusNotFound, "not found")
	})))
	mux.Handle("/api/keys/identity", authMW(http.HandlerFunc(h.UploadIdentityKey)))
	mux.Handle("/api/keys/prekeys", authMW(http.HandlerFunc(h.UploadPreKeys)))
	mux.Handle("/api/keys/", authMW(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/bundle") {
			h.GetKeyBundle(w, r)
			return
		}
		if strings.HasSuffix(r.URL.Path, "/prekey") {
			h.FetchPreKey(w, r)
			return
		}
		h.GetIdentityKey(w, r)
	})))
	mux.Handle("/api/push/vapid-public-key", authMW(http.HandlerFunc(h.GetVAPIDPublicKey)))
	mux.Handle("/api/push/subscribe", authMW(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			h.SubscribePush(w, r)
		case http.MethodDelete:
			h.UnsubscribePush(w, r)
		default:
			middleware.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	})))
	// FCM token registration for the mobile app.
	mux.Handle("/api/push/fcm-token", authMW(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			h.RegisterFCMToken(w, r)
		case http.MethodDelete:
			h.UnregisterFCMToken(w, r)
		default:
			middleware.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	})))
	mux.Handle("/api/push/test", authMW(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			h.TestPush(w, r)
		} else {
			middleware.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	})))
	mux.Handle("/ws", authMW(http.HandlerFunc(h.WebSocketHandler)))

	handler := middleware.CORS(cfg.AllowOrigins)(mux)
	addr := ":" + cfg.Port
	log.Printf("VYRO backend listening on %s (store=%s)", addr, storeName)
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
