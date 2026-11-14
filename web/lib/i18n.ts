// Tiny i18n for the web client (English + Persian). No paid service.
export type Lang = "en" | "fa";
export const dict = {
  en: {
    tagline: "Connect Beyond Limits", chats: "Chats", newChat: "New chat",
    search: "Search messages…", typeMessage: "Type a message…", send: "Send",
    login: "Log in", register: "Create account", email: "Email", password: "Password",
    username: "Username", logout: "Log out", noChats: "No chats yet — start one from Users.",
    users: "Users", theme: "Theme", attach: "Attach a file", uploading: "Uploading…",
    download: "Download", caption: "Add a caption…",
    assistant: "Assistant", assistantNote: "Local assistant · on-device, rule-based — not a cloud AI.",
    askAssistant: "Ask the local assistant…", translate: "Translate", translating: "Local dictionary translation",
    voiceNote: "Voice messages are not end-to-end encrypted in v0.6",
  },
  fa: {
    tagline: "فراتر از مرزها وصل شو", chats: "گفت‌وگوها", newChat: "گفت‌وگوی جدید",
    search: "جست‌وجوی پیام‌ها…", typeMessage: "پیام بنویس…", send: "ارسال",
    login: "ورود", register: "ساخت حساب", email: "ایمیل", password: "رمز عبور",
    username: "نام کاربری", logout: "خروج", noChats: "هنوز گفت‌وگویی نیست — از بخش کاربران شروع کن.",
    users: "کاربران", theme: "تم", attach: "پیوست فایل", uploading: "در حال آپلود…",
    download: "دانلود", caption: "کپشن بنویس…",
    assistant: "دستیار", assistantNote: "دستیار محلی · روی دستگاه، مبتنی بر قانون — هوش مصنوعی ابری نیست.",
    askAssistant: "از دستیار محلی بپرسید…", translate: "ترجمه", translating: "ترجمه با فرهنگ لغت محلی",
    voiceNote: "پیام‌های صوتی در نسخه ۰٫۶ سرتاسری رمزگذاری نمی‌شوند",
  },
} as const;
export const isRTL = (lang: Lang) => lang === "fa";
