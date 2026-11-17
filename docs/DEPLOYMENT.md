# Safari — استقرار عمومی / Public Deployment

> وضعیت: **هنوز زنده/عمومی نیست — NOT LIVE YET.**
> هیچ URL عمومی‌ای وجود ندارد و هیچ حسابی در هیچ هاستی ثبت نشده است.
> این سند فقط «قدم ۱» است: آماده‌سازی و نقشه راه. قدم بعدی (ثبت‌نام در هاست)
> به تایید کاربر نیاز دارد.

> Status: **NOT LIVE YET.** No public URL exists and no hosting account
> has been created. This document is Step 1 only: preparation and a plan.
> The next step (hosting signup) needs the user's approval.

---

## ۱. معماری استقرار / Deployment architecture

```
                    ┌────────────────────┐
  کاربر / User ───► │  وب Next.js (web)  │  ← static + SSR, port 3000
                    └─────────┬──────────┘
                              │ NEXT_PUBLIC_API_URL
                              ▼
                    ┌────────────────────┐
                    │  بک‌اند Go         │  ← REST + WebSocket, port 8080
                    │  (backend)         │
                    └────┬──────────┬────┘
                         │          │
                         ▼          ▼
              PostgreSQL 16    Redis 7
              (DATABASE_URL)   (REDIS_URL / REDIS_ADDR)
```

| بخش / Part | چه چیزی / What | کجا اجرا می‌شود / Where it runs |
|---|---|---|
| وب / Web | Next.js 14 (`web/`, `web/Dockerfile`) | هاست وب یا همان سرور Docker |
| بک‌اند / Backend | Go 1.23 (`backend/Dockerfile`) | سرور Docker (VPS / PaaS) |
| دیتابیس / Database | PostgreSQL 16, migration 001–004 | سرویس Postgres هاست، یا کانتینر |
| کش / محدودیت / Cache & limits | Redis 7 (اختیاری؛ بدون آن limiter در حافظه است) | سرویس Redis هاست، یا کانتینر |
| فایل‌ها / Uploads | `UPLOAD_DIR` (پیش‌فرض `./uploads`) | حجم (volume) سرور — باید پشتیبان‌گیری شود |

نکته مهم: فایل‌های آپلودشده روی دیسک محلی سرور ذخیره می‌شوند
(`backend/internal/handlers` — `UPLOAD_DIR`). اگر بک‌اند روی چند
نمونه (replica) اجرا شود، باید یک ذخیره‌سازی مشترک (مثل S3-compatible
رایگان) جایگزین شود؛ در غیر این صورت هر نمونه فقط فایل‌های خودش را
می‌بیند. برای قدم ۱، یک نمونه کافی است.

Important: uploads are stored on the server's local disk (`UPLOAD_DIR`).
With more than one backend replica you need shared object storage
(S3-compatible); for Step 1 a single instance is enough.

---

## ۲. گزینه‌های رایگان — مقایسه / Free options compared

> ⚠️ قیمت و محدودیت‌ها تغییر می‌کنند. قبل از ثبت‌نام، صفحه قیمت
> همان ارائه‌دهنده را بخوانید. هیچ‌کدام از این‌ها در این محیط تست
> زنده نشده‌اند (Docker در این محیط نصب نیست و هیچ حسابی ساخته نشده).
>
> ⚠️ Prices and limits change. Read the provider's own pricing page
> before signing up. None of these was live-tested here (no Docker in
> this environment, no account created).

| گزینه / Option | مناسب برای / Good for | محدودیت‌های شناخته‌شده / Known limits | هزینه / Cost |
|---|---|---|---|
| **Cloudflare Pages** | وب استاتیک / لندینگ (صفحه معرفی) | فقط محتوای استاتیک؛ بک‌اند Go را اجرا نمی‌کند | رایگان / Free |
| **Cloudflare Workers** | APIهای کوچک / لبه (edge) | برای بک‌اند کامل Go مناسب نیست؛ نیاز به بازنویسی دارد | رایگان با سهمیه / Free tier |
| **Fly.io** | بک‌اند Go + Postgres + Redis در یک‌جا (Docker) | سهمیه رایگان محدود است و ممکن است نیاز به کارت داشته باشد؛ پس از اتمام سهمیه هزینه دارد | سهمیه رایگان / Free allowance |
| **Render** | بک‌اند Go + Postgres (Docker / Native) | پلن رایگان سرور را بعد از مدتی بی‌کاری خاموش می‌کند (cold start)؛ دیسک رایگان دائمی نیست | رایگان / Free tier |
| **Oracle Cloud Always Free** | VPS همیشه‌رایگان + دیتابیس همیشه‌رایگان | ثبت‌نام و تنظیم پیچیده‌تر است؛ منابع محدود (۲ ماشین کوچک) | همیشه رایگان / Always Free |
| **VPS ارزان/رایگان دیگر** | Docker Compose کامل | نیاز به مدیریت سرور توسط خودمان | متغیر / Varies |

**توصیه قدم‌به‌قدم (پیشنهاد، نه تصمیم نهایی):**
1. **اول:** لندینگ استاتیک روی Cloudflare Pages (رایگان، سریع، بدون سرور).
2. **بعد:** بک‌اند + Postgres + Redis روی Fly.io یا Render با
   `docker compose up --build` — همان فایل‌هایی که در ریپو هستند.
3. **بعدتر:** دامنه اختصاصی و HTTPS کامل (Cloudflare خودش HTTPS رایگان
   می‌دهد).

**Recommended step-by-step (proposal, not a final decision):**
1. Static landing page on Cloudflare Pages (free, fast, serverless).
2. Backend + Postgres + Redis on Fly.io or Render with the repo's
   `docker compose up --build`.
3. Later: a custom domain with full HTTPS (Cloudflare provides free HTTPS).

---

## ۳. دامنه / Domain

| مرحله / Stage | دامنه / Domain | وضعیت / Status |
|---|---|---|
| قدم ۱ (الان) | زیردامنه رایگان هاست (مثلاً `vyro.pages.dev` یا `vyro.onrender.com`) | آماده‌سازی انجام شد؛ ثبت‌نام نشده |
| قدم ۲ | دامنه اختصاصی (مثلاً `vyro.app`) — **خریداری نشده** | نیاز به خرید/ثبت دامنه و تایید کاربر دارد |
| قدم ۳ | ساب‌دامنه API (مثلاً `api.vyro.app`) | بعد از قدم ۲ |

- دامنه جایگزین (placeholder) در فایل‌های SEO فعلاً `https://vyro.example`
  است (دامنه رزروشده `.example` — عمداً واقعی نیست). **قبل از انتشار
  عمومی، همه `vyro.example`ها را با دامنه واقعی جایگزین کنید.**
- The placeholder domain in SEO files is `https://vyro.example`
  (reserved `.example` — intentionally not real). **Replace every
  `vyro.example` with the real domain before going live.**

---

## ۴. SEO — چه چیزی آماده شد / What is ready

در کلاینت وب (`web/`) آماده شد:
- متادیتای SEO در `web/app/layout.tsx`: عنوان، توضیح، کلمات کلیدی،
  Open Graph و Twitter Card.
- صفحه لندینگ عمومی: `web/app/landing/page.tsx` — فارسی + انگلیسی،
  معرفی Safari، ویژگی‌ها، و اعلام «بتا / beta». مسیر: `/landing`.
- `web/public/robots.txt` — به خزنده‌ها می‌گوید چه چیزی را بخوانند.
- `web/public/sitemap.xml` — صفحات `/` و `/landing` (با دامنه جایگزین).
- `web/Dockerfile` — بیلد وب برای Docker (قبلاً نبود).
- `web/.env.example` — متغیر `NEXT_PUBLIC_API_URL`.

In the web client (`web/`):
- SEO metadata in `web/app/layout.tsx` (title, description, keywords,
  Open Graph, Twitter Card).
- Public landing page: `web/app/landing/page.tsx` (Persian + English,
  features, "beta" notice). Route: `/landing`.
- `web/public/robots.txt` and `web/public/sitemap.xml` (`/` and `/landing`).
- `web/Dockerfile` (new) and `web/.env.example` (`NEXT_PUBLIC_API_URL`).

بعد از انتشار عمومی، این‌ها را هم انجام دهید:
- دامنه واقعی را در Google Search Console ثبت کنید و sitemap را بدهید.
- در `layout.tsx` مقدار `metadataBase` را به دامنه واقعی تغییر دهید
  (فعلاً روی `https://vyro.example` است).
- After going live: register the real domain in Google Search Console,
  submit the sitemap, and set `metadataBase` in `layout.tsx` to the
  real domain (currently `https://vyro.example`).

---

## ۵. اجرای محلی — بررسی سازگاری / Local run & consistency check

```bash
# همه با هم (Docker) — در این محیط تست نشد، Docker نصب نیست
docker compose up --build
# backend:  http://localhost:8080/health   → {"version":"0.5.0", ...}
# web:      http://localhost:3000          → صفحه اصلی Safari
# landing:  http://localhost:3000/landing  → صفحه معرفی

# یا جداگانه:
cd backend && go run ./cmd/server          # نیاز به DATABASE_URL برای Postgres
cd web && npm install && npm run build && npm start
```

بررسی سازگاری انجام‌شده در قدم ۱:
- `docker-compose.yml` ریشه: بک‌اند + Postgres + Redis + **وب (جدید)**،
  با `DATABASE_URL`، `REDIS_ADDR` و `CORS_ALLOW_ORIGINS` یکسان با
  `backend/docker-compose.yml`.
- `backend/Dockerfile` موجود و سالم است؛ `web/Dockerfile` اضافه شد.
- `backend/.env.example`: همه متغیرها (`PORT`, `JWT_SECRET`,
  `DATABASE_URL`, `REDIS_URL`/`REDIS_ADDR`, `CORS_ALLOW_ORIGINS`,
  `UPLOAD_DIR`, `MAX_UPLOAD_MB`, `VAPID_*`) با `backend/internal/config`
  مطابقت دارند.
- Migrationها 001 تا 004 در `backend/migrations/` هستند و
  `PostgresStore` آن‌ها را هنگام شروع سرور اعمال می‌کند؛
  `docker-compose` فقط `001_init.sql` را در اولین بوت Postgres می‌گذارد.
- `npx next build` در وب **پاس شد** (بیلد production بدون خطا).
- `go vet` در قدم ۱ اجرا نشد — Go در این محیط نصب نیست (نصب‌های قبلی
  در `/tmp` موقتی بودند و از بین رفته‌اند)؛ این بررسی در قدم بعدی،
  پس از نصب دوباره Go، انجام می‌شود.

Consistency checked in Step 1:
- Root `docker-compose.yml`: backend + Postgres + Redis + **web (new)**,
  with the same `DATABASE_URL`, `REDIS_ADDR`, `CORS_ALLOW_ORIGINS` as
  `backend/docker-compose.yml`.
- `backend/Dockerfile` exists; `web/Dockerfile` added.
- `backend/.env.example` matches `backend/internal/config`.
- Migrations 001–004 exist and `PostgresStore` applies them on startup.
- `npx next build` **passed**. `go vet` was not run in this step
  (Go is at `/tmp/go/bin/go`); it belongs to the next verification step.

---

## ۶. چه چیزی هنوز قبل از انتشار عمومی لازم است / Still missing before going live

| مورد / Item | وضعیت / Status |
|---|---|
| E2EE گروهی (Sender Keys) | ❌ فقط DM دونفره رمزنگاری می‌شود؛ گروه/کانال متن ساده‌اند |
| تماس تصویری + TURN | ❌ فقط تماس صوتی وب (سیگنالینگ تست شده)؛ رسانه زنده تست نشده؛ TURN نیست |
| Push واقعی مرورگر | ⚠️ فقط تحویل به endpoint محلی اثبات شده؛ اشتراک‌ها در حافظه‌اند |
| چنددستگاه | ❌ پیاده نشده |
| ضبط پیام صوتی در کلاینت | ❌ در API پذیرفته می‌شود، ولی ضبط/پخش در وب/موبایل نیست |
| دستیار AI، ترجمه، ابزار کسب‌وکار، ذخیره ابری | ❌ پیاده نشده‌اند |
| اسکن بدافزار فایل‌ها | ❌ فقط پسوند و نوع محتوا بررسی می‌شود |
| بیلد APK/IPA بومی | ❌ Flutter کامپایل می‌شود، ولی پوشه پلتفرم و Android SDK نیست |
| تست `docker compose` | ❌ در این محیط Docker نصب نیست |
| سرعت در شبکه واقعی | ❌ عدد ۰٫۳۵ms فقط loopback است |
| امنیت production | ⚠️ `JWT_SECRET` و `VAPID_*` واقعی، `CORS_ALLOW_ORIGINS` به دامنه واقعی، رمزهای Postgres قوی، پشتیبان‌گیری Postgres + uploads |

---

## ۷. قدم بعدی — دقیقاً چه چیزی نیاز به ثبت‌نام دارد / Next step

**قدم ۲ (انجام نشده — نیاز به تایید کاربر):**
1. انتخاب هاست از جدول بخش ۲ (پیشنهاد: Cloudflare Pages برای لندینگ
   + Fly.io یا Render برای بک‌اند).
2. ثبت‌نام در آن هاست (ممکن است ایمیل/کارت بخواهد — طبق قانون صفر
   هزینه، فقط پلن رایگان انتخاب می‌شود؛ هیچ پرداختی انجام نمی‌شود).
3. ساخت پروژه/سرویس، اتصال ریپو، تنظیم متغیرهای محیطی
   (`DATABASE_URL`, `JWT_SECRET`, `REDIS_URL`, `VAPID_*`,
   `NEXT_PUBLIC_API_URL`, `CORS_ALLOW_ORIGINS`).
4. دیپلوی و تست سلامت (`/health` باید `0.5.0` برگرداند)، سپس ثبت
   دامنه و SEO در Search Console.

**Step 2 (NOT done — needs user approval):**
1. Pick a host from §2 (proposal: Cloudflare Pages for the landing
   page + Fly.io or Render for the backend).
2. Sign up there (may ask for email/card — free tier only, no payment,
   per the zero-cost rule).
3. Create the project/service, connect the repo, set the env vars
   (`DATABASE_URL`, `JWT_SECRET`, `REDIS_URL`, `VAPID_*`,
   `NEXT_PUBLIC_API_URL`, `CORS_ALLOW_ORIGINS`).
4. Deploy, check `/health` returns `0.5.0`, then domain + Search Console.

**هیچ URL عمومی‌ای در قدم ۱ ساخته نشده است — No public URL was created in Step 1.**
