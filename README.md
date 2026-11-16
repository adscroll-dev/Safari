# VYRO — Connect Beyond Limits

> **VYRO** یک پیام‌رسان واقعی، سریع، امن و چندسکویی است که با هدف رقابت با
> Telegram و WhatsApp ساخته می‌شود — با معماری مشترک و فقط ابزارهای رایگان /
> متن‌باز.
>
> **VYRO** is a real, fast, secure, multi-platform messenger built to
> compete with Telegram and WhatsApp — one shared backend, only free /
> open-source tools.

![VYRO logo](brand/logo.svg)

- **Slogan / شعار:** Connect Beyond Limits — فراتر از مرزها وصل شو
- **Brand / برند:** آبی و بنفش (`#2563EB` → `#7C3AED`) — see `brand/BRAND.md`
- **Version / نسخه:** 0.7.3 — mobile 0.7.3 with a real Android APK
  (`app.vyro`, see below); backend health reports 0.7.3, and email
  verification (v0.7) is implemented.

---

## ⚠️ قانون صفر هزینه — Zero-cost rule

**تا زمانی که تعداد کاربران ثبت‌نام‌شده VYRO به بیش از ۱۰٬۰۰۰ نفر نرسیده است:**

- هیچ سرویس پولی، هیچ اشتراک پولی، هیچ API پولی، هیچ سرور پولی.
- فقط ابزارهای رایگان / متن‌باز: Go، Flutter، Next.js، PostgreSQL، Redis، Docker.
- قابلیتی که بدون هزینه قابل اجرا نیست، فعلاً **غیرفعال** است و جایگزین
  رایگان آن در `docs/ARCHITECTURE.md` آمده است.

**Until VYRO has more than 10,000 registered users:** no paid service,
no paid subscription, no paid API, no paid server — free / open-source
only. Features that need a paid service stay disabled, with a free
alternative noted in `docs/ARCHITECTURE.md`.

---

## ساختار پروژه — Project structure

```
vyro/
├── README.md                  # this file
├── docker-compose.yml         # dev stack (backend + postgres + redis)
├── backend/                   # Go backend — REST + WebSocket
│   ├── cmd/server/main.go
│   ├── internal/{config,models,store,auth,middleware,handlers,ws,e2ee,push}/
│   ├── migrations/001_init.sql, 002_uploads.sql, 003_v04.sql, 004_v05.sql
│   ├── scripts/live-test.cjs  # live E2EE + calls + push test (Node)
│   ├── Dockerfile
│   ├── docker-compose.yml
│   └── .env.example
├── web/                       # Next.js 14 web client
│   ├── app/ (layout, page, globals.css)
│   ├── public/sw.js           # Web Push service worker (v0.5)
│   └── lib/ (api.ts, i18n.ts — en + fa, e2ee.ts, e2ee-store.ts, push-client.ts, use-call.ts, b64.ts)
├── mobile/                    # Flutter client (Android/iOS/Windows/macOS)
│   ├── lib/main.dart
│   └── lib/api.dart
├── docs/
│   ├── ARCHITECTURE.md
│   └── SECURITY.md            # E2EE design (Signal Protocol, libsignal)
└── brand/
    ├── logo.svg
    └── BRAND.md
```

---

## نحوه اجرا — How to run

### پیش‌نیازها (همه رایگان)

- Go 1.23+ — https://go.dev/dl (در این محیط با `go version` بررسی شد؛
  اگر Go نصب نیست، همان لینک را نصب کنید)
- Node.js 18+ و npm — برای وب
- Flutter SDK — برای موبایل/دسکتاپ (اختیاری در v0.1)
- Docker (اختیاری) — برای PostgreSQL/Redis

### ۱. بک‌اند (Go)

```bash
cd backend
cp .env.example .env   # و JWT_SECRET را عوض کنید
go mod download
go vet ./...
go test ./...
go run ./cmd/server
# → VYRO backend listening on :8080
# → http://localhost:8080/health  →  {"status":"ok","service":"vyro-backend",...}
```

تست دستی (curl):

```bash
# ثبت‌نام — فقط کد تأیید می‌فرستد، هنوز کاربر ساخته نمی‌شود و توکن نمی‌دهد
curl -s -X POST http://localhost:8080/api/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"email":"alice@example.com","username":"alice","password":"password-123","display_name":"Alice"}'
# → {"status":"code_sent","email":"alice@example.com","purpose":"register","expires_in_seconds":600}

# تأیید کد ۶ رقمی ایمیل — حالا کاربر ساخته می‌شود و توکن می‌دهد
curl -s -X POST http://localhost:8080/api/auth/verify-register \
  -H 'Content-Type: application/json' \
  -d '{"email":"alice@example.com","code":"123456"}'
# → {"user":{...},"token":"<jwt>"}

# ورود — رمز درست فقط کد می‌فرستد، توکن نمی‌دهد
curl -s -X POST http://localhost:8080/api/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"alice@example.com","password":"password-123"}'
# → {"status":"code_sent","email":"alice@example.com","purpose":"login","expires_in_seconds":600}

# تأیید کد ورود
curl -s -X POST http://localhost:8080/api/auth/verify-login \
  -H 'Content-Type: application/json' \
  -d '{"email":"alice@example.com","code":"654321"}'
# → {"user":{...},"token":"<jwt>"}

# با توکن: ساخت گفت‌وگو و ارسال پیام
TOKEN="<jwt>"
curl -s http://localhost:8080/api/chats -H "Authorization: Bearer $TOKEN"
```

> **ایمیل (v0.7):** کدها با SMTP فرستاده می‌شوند — پیش‌فرض رله‌ی Gmail
> (`smtp.gmail.com:587`؛ `SMTP_USER` + `SMTP_PASS` = App Password).
> بدون پیکربندی SMTP، ثبت‌نام/ورود `503` می‌دهند و هیچ کدی جعل نمی‌شود.
> کد ۶ رقمی، ۱۰ دقیقه اعتبار، یک‌بارمصرف، حداکثر ۵ تلاش اشتباه،
> ارسال مجدد حداکثر هر ۶۰ ثانیه. فقط هش SHA-256 کد ذخیره می‌شود.
> تست محلی بدون ایمیل واقعی: `backend/scripts/smtp-sink`.

### ۲. وب (Next.js)

```bash
cd web
npm install
npm run dev
# → http://localhost:3000
```

بک‌اند باید روی `http://localhost:8080` در حال اجرا باشد
(یا `NEXT_PUBLIC_API_URL` را تنظیم کنید).

### ۳. موبایل (Flutter)

```bash
cd mobile
flutter pub get
flutter run
```

### ۴. با Docker (بک‌اند + PostgreSQL + Redis)

```bash
docker compose up --build
# backend: http://localhost:8080/health
# postgres: localhost:5432 (vyro/vyro) — schema در اولین بوت اعمال می‌شود
# redis: localhost:6379
```

> توجه: از v0.2 سرور Go با `DATABASE_URL` به PostgreSQL وصل می‌شود
> (`PostgresStore` در `backend/internal/store/postgres.go`)؛ بدون
> `DATABASE_URL` همچنان در حافظه کار می‌کند (فقط برای توسعه).

---

## API (v0.5)

جدید در v0.5: رمزنگاری سرتاسری واقعی برای DMهای دونفره (VYRO E2EE v1 —
X3DH + Double Ratchet، با Safety Number در وب)، سیگنالینگ تماس WebRTC،
اعلان Web Push با VAPID، و تست‌شدن limiter با Redis واقعی و متدهای
Postgres نسخه v0.4 با PostgreSQL واقعی.

| Method & path | توضیح |
|---|---|
| `GET /api/keys/{userId}/bundle` | باندل X3DH همتا: کلید هویت + signed pre-key + یک one-time pre-key (که همان لحظه مصرف می‌شود)؛ امضای signed pre-key را کلاینت بررسی می‌کند |
| `POST /api/chats/{id}/messages` با `kind="e2ee"` | پیام رمزنگاری‌شده: `content` پاکت JSON رمزنگاری است؛ سرور فقط شکل آن را بررسی و ذخیره می‌کند، هرگز رمزگشایی/جست‌وجو نمی‌کند؛ ویرایش و زمان‌بندی آن رد می‌شود |
| `GET /api/push/vapid-public-key` | کلید عمومی VAPID برای اشتراک مرورگر |
| `POST /api/push/subscribe` · `DELETE /api/push/subscribe` | ثبت/لغو اشتراک Web Push (`{endpoint, keys:{p256dh, auth}}`) |
| `POST /api/push/test` | ارسال اعلان آزمایشی به اشتراک‌های خود کاربر |
| `GET /ws?token=` | به‌علاوه فریم‌های `call-offer` / `call-answer` / `call-ice` / `call-end` (با `payload` و `to_user_id` اختیاری) — سرور آن‌ها را بدون بررسی به اعضای همان چت می‌رساند |

## API (v0.4)

جدید در v0.4: تیک تحویل/خوانده‌شدن، در حال تایپ، پیام لوکیشن و
ناپدیدشونده، پیام زمان‌بندی‌شده، پین/آرشیو/پوشه، استوری ۲۴ ساعته،
ربات‌ها، مدیریت مدیران، و زیرساخت کلید E2EE (فقط کلید؛ پیام‌ها هنوز
متن ساده روی سرورند — بخش «چه چیزی آماده نیست» را ببینید).

| Method & path | توضیح |
|---|---|
| `POST /api/messages/{id}/read` | خوانده‌شدن پیام — تیک آبی؛ رویداد WebSocket `message_read` |
| `GET /api/users/{id}/presence` | آنلاین / آخرین بازدید |
| `POST /api/chats/{id}/messages` | حالا `kind`: `location` (با `lat`/`lng`، و `location_label`)، `sticker` (با `file_url` از `/api/uploads`) و `ttl_seconds` (۶۰ تا ۶۰۴۸۰۰؛ پیام ناپدیدشونده) هم می‌پذیرد |
| `POST /api/chats/{id}/scheduled` · `GET /api/chats/{id}/scheduled` | زمان‌بندی پیام (`send_at` به‌صورت RFC3339)؛ ارسال خودکار هر ۵ ثانیه |
| `DELETE /api/scheduled/{id}` | لغو پیام زمان‌بندی‌شده |
| `POST /api/chats/{id}/pin` · `/unpin` · `/archive` · `/unarchive` · `/folder` | پین / آرشیو / پوشه (به‌ازای هر کاربر) |
| `GET /api/chats/{id}/members` · `POST /api/chats/{id}/members` | فهرست / افزودن عضو (فقط owner/admin) |
| `DELETE /api/chats/{id}/members/{userId}` | حذف عضو (فقط owner/admin؛ owner حذف نمی‌شود) |
| `POST /api/chats/{id}/members/{userId}/role` | ارتقا/تنزل نقش (فقط owner؛ `owner`/`admin`/`member`) |
| `POST /api/stories` · `GET /api/stories[?user_id=]` | استوری ۲۴ ساعته (`media_url` فقط از `/api/uploads`) |
| `POST /api/bots` · `GET /api/bots` · `DELETE /api/bots/{id}` | ساخت ربات — توکن فقط یک‌بار در پاسخ ساخت برمی‌گردد و فقط هش آن ذخیره می‌شود |
| `POST /api/bots/{id}/webhook` | ثبت webhook ربات |
| `GET /api/bot/updates` · `POST /api/bot/send` | پولینگ و ارسال با هدر `X-Bot-Token` (بدون JWT) |
| `POST /api/keys/identity` · `GET /api/keys/{userId}/{deviceId}` | آپلود/گرفتن کلید هویت عمومی X3DH |
| `POST /api/keys/prekeys` · `GET /api/keys/{userId}/prekey` | آپلود/گرفتن یک‌بارمصرف pre-key (هر کلید فقط یک‌بار داده می‌شود) |
| `GET /ws?token=<jwt>` | حالا اتاق به‌ازای هر چت + `typing`، `message_read`، `message_delivered`، `presence`؛ پیام فقط به اعضای همان چت می‌رسد |

Base: `http://localhost:8080` — Auth: `Authorization: Bearer <jwt>` (ادامه جدول v0.3)

| Method & path | توضیح |
|---|---|
| `GET /health` | سلامت سرویس |
| `POST /api/auth/register` | ثبت‌نام (ایمیل، نام کاربری، رمز ≥ ۸ کاراکتر) — فقط کد ۶ رقمی می‌فرستد، کاربر هنوز ساخته نمی‌شود |
| `POST /api/auth/verify-register` | تأیید کد ثبت‌نام → ساخت کاربر + توکن (۲۰۱)؛ کد اشتباه/منقضی 401، ۵ تلاش اشتباه 429 |
| `POST /api/auth/login` | ورود — رمز درست فقط کد می‌فرستد، توکن نمی‌دهد؛ ۵ تلاش / ۱۵ دقیقه، بعد 429 |
| `POST /api/auth/verify-login` | تأیید کد ورود → توکن (۲۰۰) |
| `POST /api/auth/resend-code` | ارسال مجدد کد (`{email, purpose:"register"\|"login"}`) — حداکثر هر ۶۰ ثانیه، بعد 429 |
| `GET /api/users/me` | کاربر جاری |
| `GET /api/users?search=` | جست‌وجوی کاربران |
| `POST /api/users/{id}/block` | مسدودکردن کاربر |
| `GET /api/chats` · `POST /api/chats` | فهرست / ساخت dm، group، channel |
| `GET /api/chats/{id}/messages` | پیام‌های گفت‌وگو |
| `POST /api/chats/{id}/messages` | ارسال پیام متنی یا رسانه‌ای (`kind`: text/image/video/file/voice + `file_url` از `/api/uploads`، پخش زنده با WebSocket) |
| `POST /api/uploads` | آپلود فایل (multipart، فیلد `file`؛ حداکثر ۵۰ مگابایت، فقط پسوندهای مجاز) — برمی‌گرداند `{url, file_name, file_size, mime_type, kind}` |
| `GET /uploads/{name}` | دانلود فایل آپلودشده (با ورود) |
| `PUT /api/messages/{id}` | ویرایش پیام خود |
| `DELETE /api/messages/{id}` | حذف پیام خود (soft delete) |
| `POST/DELETE /api/messages/{id}/reactions` | واکنش ایموجی |
| `GET /api/messages/search?q=` | جست‌وجوی پیام‌ها (فقط گفت‌وگوهای خود) |
| `GET /ws?token=` | WebSocket: `message`, `message_edited`, `message_deleted`, `presence` |

جزئیات کامل: `docs/ARCHITECTURE.md`

---

## ✅ چه چیزی واقعاً کار می‌کند — What actually works (tested)

**تست‌شده در v0.5 (۲۰۲۶-۱۱-۱۲):**

- **E2EE واقعی برای DM دونفره:** X3DH + Double Ratchet با X25519 /
  HKDF-SHA256 / AES-256-GCM / Ed25519 — پیاده‌سازی Go
  (`backend/internal/e2ee/`) و TypeScript/WebCrypto (`web/lib/e2ee.ts`)
  که **در هر دو جهت با هم سازگار اثبات شدند**: پاکت ساخته‌شده توسط Go
  در TypeScript رمزگشایی شد و پاکت ساخته‌شده توسط TypeScript در Go
  (`web/lib/e2ee.interop.cjs` + `web/lib/e2ee-vector.json`).
  تست‌ها: round-trip، تحویل نامرتب، دست‌کاری/کلید اشتباه رد می‌شود،
  forward secrecy، و `TestServerCannotRead` (سرور متن ساده را نمی‌بیند).
  تست سرتاسری HTTP (`TestV05KeyBundleAndE2EEFlow`): باندل با امضای
  معتبر، مصرف یک‌بارمصرف pre-key، ذخیره فقط رمزنگاری روی سرور، و
  رمزگشایی توسط گیرنده.
- **تست زنده سرور واقعی (۲۰۲۶-۱۱-۱۲، PostgreSQL 16 + Redis 7.0.15):**
  `backend/scripts/live-test.cjs` — هر ۱۰ بررسی **PASS** (دو بار
  پشت سر هم): ثبت‌نام دو کاربر، باندل E2EE، ارسال پیام رمزنگاری‌شده
  («live E2EE سلام») و رمزگشایی آن توسط گیرنده، سیگنالینگ تماس
  offer/answer بین دو کلاینت WebSocket زنده، و Web Push به یک endpoint
  محلی با رمزگشایی payload (شامل push هنگام آفلاین‌بودن گیرنده با
  متن «New message from livealice» — بدون محتوای پیام).
- **تماس صوتی وب:** سیگنالینگ WebRTC در بک‌اند + هوک `use-call.ts`
  (فقط صوتی، STUN عمومی گوگل) و دکمه 📞 و بنر تماس ورودی در
  `web/app/page.tsx`. تست سیگنالینگ دو کلاینت **PASS**
  (`TestV05CallSignaling`).
- **Web Push (VAPID):** ثبت/لغو اشتراک، کلید عمومی VAPID، ارسال
  آزمایشی، و push خودکار به اعضای آفلاین هنگام ارسال پیام — همه
  **PASS** در تست زنده. Payload رمزنگاری‌شده به endpoint محلی رسید و
  رمزگشایی شد (`TestSendDecryptsEndToEnd`). اشتراک‌هایی که سرویس push
  آن‌ها را مرده گزارش کند (410/404) حذف می‌شوند. بیلد وب
  (`npm run build`) در ۲۰۲۶-۱۱-۱۲ **پاس شد**.
- **Redis واقعی:** limiter با Redis 7.0.15 **تست شد و کار می‌کند**
  (`TestRedisLimiterLive`)؛ go-redis به v9.8 پین شده چون نسخه‌های
  قدیمی‌تر `CLIENT SETINFO` می‌فرستند که Redis زیر ۷٫۲ رد می‌کند.
  سرور زنده با `REDIS_URL` لاگ «VYRO rate limiting: Redis-backed» را
  نشان داد.
- **PostgreSQL واقعی برای متدهای v0.4:** `TestPostgresStoreV04` با
  PostgreSQL 16 **پاس شد** (رسیدها، نقش‌ها، استوری، زمان‌بندی،
  ربات‌ها، کلیدها، migration 004). یک باگ تست (فراموش‌شدن `OwnerID`
  در تست، نه در store) پیدا و اصلاح شد.
- **اصلاح re-publish برای pre-keyها:** انتشار دوباره یک key_id حالا
  جایگزین کلید عمومی قبلی می‌شود (قبلاً نادیده گرفته می‌شد و کلید
  قدیمیِ غیرقابل رمزگشایی به همتاها داده می‌شد) —
  `TestPreKeyRepublish` برای هر دو store.
- کلیدهای VAPID از متغیر محیطی خوانده می‌شوند؛ بدون آن‌ها سرور یک
  جفت موقت برای همان اجرا می‌سازد و هشدار می‌دهد.

****تست‌شده در v0.7 (۲۰۲۶-۱۱-۱۴):**

- **فوروارد پیام:** `forwarded_from` / `forwarded_from_id` در مدل
  پیام و migration ‏`007_forward.sql`؛ تست‌های handler
  (`TestForwardMessage`، `TestForwardMediaMessage`،
  `TestForwardLocationMessage`، `TestForwardValidation`): فوروارد متن
  با حفظ فرستنده اصلی، فوروارد عکس بدون آپلود دوباره، فوروارد لوکیشن
  با مختصات، رد فوروارد E2EE و فرستنده ناموجود و نام بیش‌ازحد طولانی.
  تست یکپارچه‌سازی PostgreSQL واقعی
  (`TestPostgresStoreForwardedMessage`): ستون‌های جدید در خواندن و
  نوشتن درست کار می‌کنند.
- **موبایل:** پوشه‌های ساخته‌شده توسط کاربر، قفل چت با PIN (فقط هش
  SHA-256 با salt ذخیره می‌شود)، پیش‌نویس پیام، سرعت پخش پیام صوتی
  1x/1.5x/2x — تست‌های `mobile/test/chat_extras_test.dart` (۱۲ تست
  جدید؛ مجموع `flutter test` ‏۳۳/۳۳‏، `flutter analyze` بدون مشکل).
- **باگ کرش SMTP اصلاح شد:** ثبت‌نام بدون SMTP قبلاً سرور را با panic
  از کار می‌انداخت؛ حالا HTTP 503 می‌دهد و سرور بالا می‌ماند (تست
  زنده با سرور واقعی + PostgreSQL). تست رگرسیون
  `TestRegisterWithUnconfiguredSMTPIs503`.
- **کل سوئیت Go با PostgreSQL واقعی پاس شد** (`go test ./...`،
  PostgreSQL 16 محلی): همه پکیج‌ها OK.

تست‌شده در v0.4 (۲۰۲۶-۱۱-۱۲):**

- `go vet ./...` — **پاس شد**؛ `go test ./...` — **پاس شد**
  (پکیج‌های `auth`, `handlers`, `store`؛ تست‌های جدید v0.4:
  تیک‌ها، لوکیشن، پیام ناپدیدشونده، زمان‌بندی، پین/آرشیو/پوشه،
  مدیریت مدیران، استوری، ربات‌ها، کلیدهای E2EE، و Redis limiter
  در حالت بدون Redis).
- **تست زنده دو کلاینت WebSocket (۲۰۲۶-۱۱-۱۲):** سرور واقعی با
  `store` در حافظه اجرا شد؛ دو کاربر، DM، دو اتصال WebSocket.
  ارسال پیام از یکی و دریافت توسط دیگری: **۵ پیام، حداقل ۰٫۱۷،
  میانه ۰٫۳۵، حداکثر ۰٫۵۷ میلی‌ثانیه** (loopback روی همان ماشین،
  `store` در حافظه — سرعت در شبکه واقعی و با PostgreSQL متفاوت
  خواهد بود و هنوز اندازه‌گیری نشده است). رویداد **typing** و
  **read receipt** سرتاسری (REST + WebSocket) هر دو **PASS** شدند؛
  وضعیت پیام در API پس از خواندن `read` بود.
- اتاق WebSocket به‌ازای هر چت: پیام فقط به اعضای همان چت می‌رسد
  (دیگر پخش به همه نیست)؛ تیک‌ها `sent` → `delivered` → `read`
  در API و WebSocket؛ حضور آنلاین/آخرین بازدید.
- پیام لوکیشن (`lat`/`lng` با اعتبارسنجی)، پیام ناپدیدشونده
  (`ttl_seconds` ۶۰ تا ۶۰۴۸۰۰)، پیام زمان‌بندی‌شده (ارسال‌کننده
  هر ۵ ثانیه)، پین/آرشیو/پوشه به‌ازای هر کاربر، استوری ۲۴ ساعته،
  استیکر به‌صورت فایل آپلودشده.
- مدیریت مدیران گروه/کانال: owner/admin/member با اعمال واقعی
  (عضو نمی‌تواند اضافه کند، admin نمی‌تواند owner را حذف/تنزل دهد).
- ربات‌ها: ساخت با توکن (فقط یک‌بار نمایش، فقط هش ذخیره می‌شود)،
  webhook (بهترین تلاش، تایم‌اوت ۲ ثانیه)، پولینگ `X-Bot-Token`،
  ارسال از طرف ربات.
- زیرساخت کلید E2EE: آپلود کلید هویت + signed pre-key، آپلود
  one-time pre-key، گرفتن هر pre-key فقط یک‌بار (دومین درخواست
  404 می‌گیرد). **پیام‌ها هنوز روی سرور متن ساده‌اند.**
- محدودیت ورود با Redis: وقتی `REDIS_URL` تنظیم شده و Redis جواب
  بدهد، limiter با Redis کار می‌کند؛ در غیر این صورت به‌صورت
  خودکار به limiter در حافظه برمی‌گردد (fail-open با لاگ).
- وب Next.js (`npx next build` در ۲۰۲۶-۱۱-۱۲ **پاس شد**): چیدمان
  آشنای فهرست چت (آواتار، نشان 📌/📁، بخش آرشیو)، نوار استوری،
  حباب پیام با تیک ✓/✓✓/✓✓ آبی و ساعت، نشان «در حال تایپ…»،
  دکمه‌های بزرگ ارسال/پیوست/لوکیشن، انتخاب TTL پیام ناپدیدشونده،
  زمان‌بندی پیام، پین/آرشیو از سربرگ چت، دکمه لوکیشن با
  geolocation مرورگر. برند آبی-بنفش VYRO حفظ شده؛ طرح تلگرام و
  واتساپ کپی نشده است.
- موبایل Flutter: فهرست چت با `ListTile`/آواتار، حباب با تیک و
  ساعت، نشان تایپ، دکمه ارسال بزرگ، ناوبری پایین (Chats/Stories)،
  پین/آرشیو، TTL و زمان‌بندی پیام، اشتراک لوکیشن (در موبایل فعلاً
  با ورود دستی مختصات، چون وابستگی geolocator اضافه نشده است).
  **موبایل در v0.6 (۲۰۲۶-۱۱-۱۲):** برابری کامل با وب — E2EE
  (X3DH + Double Ratchet و Sender Keys گروهی)، تماس صوتی/تصویری
  WebRTC با Opus ۱۲ کیلوبیت/ثانیه، پیام صوتی، استوری و استوری
  تبلیغاتی، ربات‌ها، دستیار محلی و ترجمه، Data Saver پیش‌فرض روشن،
  و **نخستین APK واقعی اندروید**: `mobile/build/app/outputs/
  flutter-apk/app-release.apk` — بسته `app.vyro`، نسخه ۰٫۶٫۰،
  ۹۶٫۳ مگابایت، SHA-256
  `8a717aa83b26cf916b4f109aa16a2a3b27a84bd93a0fdf41e2b8ca6bc97fc592`
  (در `web/public/downloads/vyro-0.6.0.apk` کپی شده و در بخش دانلود
  صفحه `/landing` با هش و هشدار دانلود فقط از سایت رسمی آمده است).
  `flutter analyze` بدون مشکل و `flutter test` ‏۲۱/۲۱‏ پاس شد.
  توجه: APK با کلید debug امضا شده است (نه keystore انتشار)؛ پیش
  از توزیع گسترده باید keystore انتشار ساخته شود. IPA برای iOS
  هنوز ساخته نشده است (به Mac با Xcode نیاز دارد).

تست‌شده در این محیط (۲۰۲۶-۱۱-۱۱):

- `go vet ./...` — **پاس شد (بدون خطا)**
- `go test ./...` — **پاس شد** (پکیج‌های `auth`, `handlers`, `store`؛
  بقیه پکیج‌ها فایل تست ندارند)
- `go build ./cmd/server` — **ساخته شد و اجرا شد**؛ تست زنده با curl:
  `/health`، ثبت‌نام، ورود، ساخت DM، ارسال پیام و جست‌وجوی پیام
  با موفقیت انجام شد.
- ثبت‌نام/ورود با ایمیل، هش bcrypt (هزینه ۱۲، هرگز متن ساده)،
  JWT HS256، محدودیت تلاش ورود (۵ در ۱۵ دقیقه، 429).
- گفت‌وگوی خصوصی / گروه / کانال، ارسال پیام متنی، پاسخ (`reply_to_id`)،
  ویرایش و حذف پیام (فقط فرستنده)، واکنش ایموجی، جست‌وجوی پیام،
  مسدودکردن کاربر، بررسی عضویت در گفت‌وگو (غیرعضو 403 می‌گیرد).
- WebSocket زنده: پیام جدید به همه کلاینت‌های متصل پخش می‌شود.
- وب Next.js: فرم واقعی ثبت‌نام/ورود، فهرست گفت‌وگوها، ارسال پیام،
  جست‌وجو، تم روشن/تاریک، فارسی + راست‌چین و انگلیسی.
  (`npm install` و `npx next build` در این محیط با موفقیت اجرا شد —
  Next 14.2.35، بیلد production بدون خطا.)
- موبایل Flutter: صفحه واقعی ورود/ثبت‌نام و چت با همان API.
  (وضعیت کامپایل در v0.5 در بخش بالا به‌روز شد: analyze بدون
  مشکل و build web موفق؛ بیلد بومی APK/IPA هنوز نشده است.)
- اسکیمای PostgreSQL کامل: `users`, `identity_keys`,
  `one_time_prekeys`, `chats`, `chat_members`, `messages`, `reactions`,
  `stories`, `blocks`, `reports`, `devices` (+ ایندکس‌ها و تریگرها).
- **آپلود فایل و عکس (v0.3، ۲۰۲۶-۱۱-۱۲):** `POST /api/uploads`
  (multipart، نام تصادفی، حد ۵۰ مگابایت، پسوند و نوع محتوا بررسی
  می‌شود؛ فایل اجرایی/اسکریپت/HTML رد می‌شود) و
  `GET /uploads/{name}`؛ پیام `image/video/file/voice` با `file_url`
  همان آپلود در PostgreSQL ذخیره می‌شود و پس از ری‌استارت سرور
  باقی می‌ماند (تست زنده: آپلود PNG، ارسال پیام تصویری با کپشن
  فارسی، ری‌استارت، پیام و فایل هنوز در دسترس بودند).
  کلاینت وب ریپو هم حالا به همین API وصل است: دکمه 📎 در
  `web/app/page.tsx` فایل را آپلود می‌کند و پیام رسانه‌ای می‌فرستد،
  و عکس/ویدئو/صدا درون چت نمایش داده می‌شوند (فایل‌ها با توکن ورود
  گرفته می‌شوند، چون `/uploads/*` احراز هویت می‌خواهد) — بیلد
  production وب در ۲۰۲۶-۱۱-۱۲ پاس شد؛ ارسال زنده در این محیط تست
  نشد (Go در این محیط نصب نیست).
- **PostgreSQL واقعی (v0.2، ۲۰۲۶-۱۱-۱۲):** `PostgresStore`
  (`backend/internal/store/postgres.go`, pgx v5) اینترفیس `StoreIface`
  را پیاده می‌کند؛ با `DATABASE_URL` سرور از PostgreSQL استفاده می‌کند
  و داده‌ها با ری‌استارت باقی می‌مانند (تست یکپارچه‌سازی با PostgreSQL
  واقعی: ثبت‌نام، چت، پیام، واکنش، ویرایش/حذف، جستجو، مسدودکردن، و
  بقای داده پس از بازکردن دوباره اتصال — `go test` با
  `VYRO_TEST_DATABASE_URL`). بدون `DATABASE_URL` همچنان `MemoryStore`.
- لوگوی SVG آبی-بنفش و راهنمای برند.

## ❌ چه چیزی هنوز ناتمام / ساخته نشده — What is NOT done

صادقانه — این‌ها **در کد نیستند** و هیچ دکمه ساختگی برایشان ساخته نشده:

- **E2EE گروهی:** فقط DM دونفره رمزنگاری می‌شود؛ پیام‌های گروه/کانال
  روی سرور متن ساده‌اند (Sender Keys پیاده نشده). همچنین E2EE
  ساخته‌شده VYRO **libsignal نیست، با Signal سازگار نیست، و به‌عنوان
  یک پروتکل به‌صورت مستقل بررسی (audit) نشده است** — فقط از
  primitiveهای بررسی‌شده استفاده می‌کند (جزئیات در
  `docs/SECURITY.md`). چنددستگاه هم نیست (یک دستگاه، `"default"`).
  کلیدهای خصوصی کلاینت وب در `localStorage` ذخیره می‌شوند (درجه
  توسعه — نه keystore سیستم‌عامل).
- **تماس تصویری و تماس با رسانه واقعی:** فقط تماس **صوتی** در وب
  پیاده شده و فقط **سیگنالینگ** آن تست شده است؛ تبادل رسانه زنده بین
  دو همتای واقعی با میکروفون/دوربین تست نشده است. تماس تصویری پیاده
  نشده و TURN server در نسخه ۰.۶ پیکربندی و به وب وصل شده است
  (coturn خودمیزبان؛ با `docker compose` اجرا می‌شود) — ولی تبادل
  رسانه زنده بین دو همتای واقعی هنوز تست نشده و تماس تصویری پیاده
  نشده است.
- **Push واقعی مرورگر:** تحویل از طریق سرویس push واقعی مرورگر
  (مثل FCM) تست نشده است — فقط مسیر رمزنگاری و تحویل به یک endpoint
  محلی اثبات شده. اشتراک‌ها در حافظه‌اند؛ با ری‌استارت سرور از بین
  می‌روند و کلاینت باید دوباره اشتراک بگیرد (کلید VAPID موقت هم
  همین مشکل را دارد — برای production متغیرهای `VAPID_*` را تنظیم
  کنید).
- **همگام‌سازی چنددستگاه:** پیاده نشده است.
- **اپ‌های بومی اندروید/iOS/دسکتاپ:** APK واقعی اندروید در v0.6
  ساخته شد (بخش بالا — `app.vyro` ۰٫۶٫۰، با کلید debug امضا شده).
  IPA برای iOS هنوز ساخته نشده است (به Mac با Xcode نیاز دارد) و
  بیلد دسکتاپ هنوز انجام نشده است.
- **ضبط پیام صوتی در کلاینت‌ها:** در API پیام `voice` با فایل
  آپلودشده پذیرفته می‌شود، ولی ضبط/پخش در وب و موبایل پیاده
  نشده است.
- **ضبط پیام صوتی در کلاینت‌ها:** در API پیام `voice` با فایل
  آپلودشده پذیرفته می‌شود، ولی ضبط/پخش در وب و موبایل پیاده
  نشده است.
- **دستیار هوش مصنوعی، ترجمه خودکار، ابزارهای کسب‌وکار، ذخیره‌سازی
  ابری:** پیاده نشده‌اند.
- **اسکن بدافزار فایل‌های آپلودشده:** انجام نمی‌شود (فقط پسوند و
  نوع محتوا بررسی می‌شود).
- **Docker:** `docker compose` در این محیط تست نشد (Docker نصب
  نیست)؛ فایل‌ها نوشته شده‌اند و باید روی دستگاه دارای Docker
  بررسی شوند.
- **سرعت در شبکه واقعی:** عدد ۰٫۳۵ میلی‌ثانیه فقط loopback روی
  همان ماشین با `store` در حافظه است؛ با PostgreSQL و شبکه واقعی
  هنوز اندازه‌گیری نشده است.

## نقشه راه — Roadmap

1. **v0.1 (این نسخه)** — احراز هویت، چت متنی، گروه/کانال، WebSocket.
2. **v0.2** — `PostgresStore` ✅ (انجام شد ۲۰۲۶-۱۱-۱۲).
3. **v0.3** — آپلود فایل/عکس ✅ (انجام شد ۲۰۲۶-۱۱-۱۲).
4. **v0.4** — ✅ (انجام شد ۲۰۲۶-۱۱-۱۲): اتاق‌های WebSocket به‌ازای
   هر چت، تیک‌ها و تایپ، لوکیشن، پیام ناپدیدشونده و زمان‌بندی‌شده،
   پین/آرشیو/پوشه، استوری، ربات‌ها، مدیریت مدیران، limiter با Redis،
   زیرساخت کلید E2EE، چیدمان آشنای وب/موبایل.
5. **v0.5** — ✅ (انجام شد ۲۰۲۶-۱۱-۱۲): E2EE واقعی برای DM دونفره
   (VYRO E2EE v1 — X3DH + Double Ratchet، Go و TypeScript، اثبات
   سازگاری دوسویه؛ Safety Number در وب)، سیگنالینگ تماس WebRTC +
   تماس صوتی در وب، Web Push با VAPID، تست limiter با Redis واقعی،
   تست متدهای Postgres نسخه v0.4 با PostgreSQL واقعی، اصلاح
   re-publish برای pre-keyها.
6. **v0.6** — E2EE گروهی (Sender Keys)، چنددستگاه، تماس تصویری +
   TURN خودمیزبان، تست رسانه زنده با همتای واقعی، ضبط پیام صوتی،
   دستیار AI، ترجمه، ابزار کسب‌وکار، بررسی مستقل ساختار E2EE.
7. **v0.7 (در حال توسعه، ۲۰۲۶-۱۱-۱۴)** — فوروارد پیام با «Forwarded
   from» (بک‌اند + موبایل، تست‌شده با PostgreSQL واقعی)، پوشه‌های
   ساخته‌شده توسط کاربر، قفل چت با PIN، پیش‌نویس پیام، سرعت پخش
   پیام صوتی (1x/1.5x/2x)؛ و اصلاح باگ کرش سرور هنگام ثبت‌نام بدون
   SMTP (به‌جای panic حالا 503 می‌دهد — تست رگرسیون
   `TestRegisterWithUnconfiguredSMTPIs503`).

مستندات کامل: `docs/ARCHITECTURE.md` و `docs/SECURITY.md`
