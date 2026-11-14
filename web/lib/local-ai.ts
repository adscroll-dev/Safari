// v0.6: Local, on-device assistant + translation. HONEST LABELING:
// - There is NO cloud AI and NO paid translation API behind this.
// - The assistant is a small rule-based responder that answers
//   questions about VYRO itself (features, how-to) and a few general
//   prompts. It runs entirely in the browser.
// - Translation is a small built-in English<->Persian dictionary
//   (common words and phrases). It is NOT a full translator: unknown
//   words are left unchanged and `coverage` reports how much of the
//   input was actually in the dictionary, so the UI never pretends a
//   translation is complete when it is not.
// Nothing in this file makes any network request.

export type Lang2 = "en" | "fa";

export function detectLang(text: string): Lang2 {
  return /[\u0600-\u06FF]/.test(text) ? "fa" : "en";
}

function fmtDuration(sec: number): string {
  const m = Math.floor(sec / 60);
  const s = Math.round(sec % 60);
  return `${m}:${s.toString().padStart(2, "0")}`;
}
export { fmtDuration };

// ---- Rule-based VYRO assistant ------------------------------------
// Each rule is a set of keywords (either language) and a bilingual
// answer. The UI must label replies "Local assistant (on-device)".
interface Rule { match: RegExp; en: string; fa: string }

const RULES: Rule[] = [
  {
    match: /^(hi|hello|hey|salam|سلام|درود)\b/i,
    en: "Hello! I'm the local VYRO assistant (on-device, rule-based — not a cloud AI). Ask me how to use VYRO, or type \"help\".",
    fa: "سلام! من دستیار محلی وایرو هستم (روی دستگاه، مبتنی بر قانون — هوش مصنوعی ابری نیست). بپرسید چطور از وایرو استفاده کنید، یا «help» بنویسید.",
  },
  {
    match: /voice|صوتی|میکروفون|میکروفن|ضبط/i,
    en: "Voice messages: tap the 🎤 button next to the message box, allow the microphone, tap again to stop, then send. Voice messages are sent through the normal upload API — note they are NOT end-to-end encrypted, even in an encrypted chat.",
    fa: "پیام صوتی: دکمه 🎤 کنار کادر پیام را بزنید، به میکروفون اجازه دهید، دوباره بزنید تا ضبط متوقف شود، بعد ارسال کنید. پیام صوتی از طریق آپلود معمولی ارسال می‌شود — توجه: حتی در گفت‌وگوی رمزگذاری‌شده هم سرتاسری رمزگذاری نمی‌شود.",
  },
  {
    match: /help|راهنما|کمک|چه کار|چی کار|چطور|چگونه|how (do|can|to)/i,
    en: "I can help with VYRO: send text, photos, files, voice messages (🎤 button), stories, voice calls, disappearing messages, scheduled messages, search, and end-to-end encrypted DMs (🔒 button). What would you like to do?",
    fa: "می‌توانم در استفاده از وایرو کمک کنم: ارسال متن، عکس، فایل، پیام صوتی (دکمه 🎤)، استوری، تماس صوتی، پیام‌های ناپدیدشونده، پیام زمان‌بندی‌شده، جست‌وجو و گفت‌وگوی رمزگذاری‌شده سرتاسری (دکمه 🔒). چه کاری می‌خواهید انجام دهید؟",
  },

  {
    match: /encrypt|رمز|امن|e2ee|lock|قفل/i,
    en: "End-to-end encryption: open a DM and tap the 🔓 button to turn it on. Keys are generated on your device; the server only ever sees ciphertext. It works for 1:1 DMs in this version — group encryption is not available yet.",
    fa: "رمزگذاری سرتاسری: یک گفت‌وگوی دونفره را باز کنید و دکمه 🔓 را بزنید. کلیدها روی دستگاه شما ساخته می‌شوند و سرور فقط متن رمزشده را می‌بیند. در این نسخه فقط برای گفت‌وگوی دونفره کار می‌کند — رمزگذاری گروهی هنوز نیست.",
  },
  {
    match: /call|تماس|زنگ/i,
    en: "Voice calls: open a DM and tap 📞. Calls use WebRTC directly between the two browsers (audio only in v0.6).",
    fa: "تماس صوتی: یک گفت‌وگوی دونفره را باز کنید و 📞 را بزنید. تماس با WebRTC مستقیم بین دو مرورگر برقرار می‌شود (در نسخه ۰٫۶ فقط صوتی).",
  },
  {
    match: /stor(y|ies)|استوری|وضعیت/i,
    en: "Stories: tap the ＋ circle above your chat list, pick a photo or video, and it stays visible for 24 hours.",
    fa: "استوری: دایره ＋ بالای فهرست گفت‌وگوها را بزنید، عکس یا ویدیو انتخاب کنید؛ ۲۴ ساعت نمایش داده می‌شود.",
  },
  {
    match: /disappear|ناپدید|محو|خود.?تخریب/i,
    en: "Disappearing messages: pick a timer (⏱) next to the send box — 1 minute, 1 hour or 1 day — before sending. The message expires after that time.",
    fa: "پیام ناپدیدشونده: پیش از ارسال، زمان‌سنج (⏱) کنار دکمه ارسال را انتخاب کنید — ۱ دقیقه، ۱ ساعت یا ۱ روز. پیام پس از آن زمان منقضی می‌شود.",
  },
  {
    match: /schedul|زمان.?بندی|بعدا|بعداً/i,
    en: "Scheduled messages: type your message, then tap 🕐 and enter in how many minutes it should be sent.",
    fa: "پیام زمان‌بندی‌شده: پیام را بنویسید، بعد 🕐 را بزنید و بگویید بعد از چند دقیقه ارسال شود.",
  },
  {
    match: /translate|ترجمه/i,
    en: "Translation: tap 🌐 under any text message to translate it with the built-in local dictionary (English ↔ Persian). It only knows common words — it is not a full translator, and the result shows how much it actually covered.",
    fa: "ترجمه: زیر هر پیام متنی 🌐 را بزنید تا با فرهنگ لغت محلی (انگلیسی ↔ فارسی) ترجمه شود. فقط واژه‌های رایج را می‌داند — مترجم کامل نیست و نتیجه نشان می‌دهد چقدر پوشش داده است.",
  },
  {
    match: /who are you|what are you|vyro|وایرو|تو کی|چی هستی/i,
    en: "VYRO is a free messenger — 'Connect Beyond Limits'. This assistant is a small on-device helper, not a cloud AI: it only knows how VYRO works.",
    fa: "وایرو یک پیام‌رسان رایگان است — «فراتر از مرزها وصل شو». این دستیار یک کمک‌کننده کوچک روی دستگاه است، نه هوش مصنوعی ابری: فقط نحوه کار وایرو را می‌داند.",
  },
  {
    match: /time|ساعت|date|تاریخ|امروز|today/i,
    en: `The time on your device right now is ${new Date().toLocaleString()}.`,
    fa: `ساعت روی دستگاه شما الان ${new Date().toLocaleString("fa-IR")} است.`,
  },
  {
    match: /thank|مرسی|ممنون|تشکر/i,
    en: "You're welcome! Anything else about VYRO?",
    fa: "خواهش می‌کنم! چیز دیگری درباره وایرو هست؟",
  },
  {
    match: /bye|خداحافظ|بای/i,
    en: "Goodbye! You can reopen me anytime with the 🤖 button.",
    fa: "خداحافظ! هر وقت خواستید با دکمه 🤖 دوباره بازم کنید.",
  },
];

const FALLBACK = {
  en: "I'm a small local assistant and I only know how VYRO works — I can't answer general questions or browse the web. Try \"help\", or ask about voice messages, encryption, calls, stories, disappearing or scheduled messages.",
  fa: "من یک دستیار محلی کوچک هستم و فقط نحوه کار وایرو را می‌دانم — نمی‌توانم به سؤال‌های عمومی پاسخ دهم یا در وب بگردم. «help» را امتحان کنید یا درباره پیام صوتی، رمزگذاری، تماس، استوری، پیام ناپدیدشونده یا زمان‌بندی‌شده بپرسید.",
} as const;

export function assistantReply(input: string): { text: string; lang: Lang2 } {
  const lang = detectLang(input);
  const q = input.trim();
  for (const rule of RULES) {
    if (rule.match.test(q)) return { text: lang === "fa" ? rule.fa : rule.en, lang };
  }
  return { text: lang === "fa" ? FALLBACK.fa : FALLBACK.en, lang };
}

// ---- Local dictionary translation (EN <-> FA) ---------------------
// Common words/phrases only. Word-by-word with case/punctuation
// preserved; unknown words pass through unchanged. `coverage` =
// fraction of words found in the dictionary (0..1).
type Dict = Record<string, string>;

const EN_FA: Dict = {
  hello: "سلام", hi: "سلام", hey: "سلام", goodbye: "خداحافظ", bye: "خداحافظ",
  please: "لطفاً", thanks: "ممنون", thank: "ممنون", "thank you": "ممنونم",
  yes: "بله", no: "نه", ok: "باشه", okay: "باشه", good: "خوب", bad: "بد",
  morning: "صبح", night: "شب", today: "امروز", tomorrow: "فردا", yesterday: "دیروز",
  friend: "دوست", love: "عشق", happy: "خوشحال", sad: "ناراحت", beautiful: "زیبا",
  how: "چطور", are: "هستی", you: "تو", i: "من", we: "ما", they: "آنها",
  what: "چه", where: "کجا", when: "کی", why: "چرا", who: "کی",
  is: "است", am: "هستم", the: "", a: "", an: "",
  message: "پیام", chat: "گفت‌وگو", call: "تماس", voice: "صوتی", photo: "عکس",
  video: "ویدیو", file: "فایل", send: "ارسال", help: "کمک", time: "زمان",
  day: "روز", week: "هفته", month: "ماه", year: "سال", home: "خانه",
  work: "کار", school: "مدرسه", book: "کتاب", water: "آب", food: "غذا",
  mother: "مادر", father: "پدر", brother: "برادر", sister: "خواهر",
  boy: "پسر", girl: "دختر", man: "مرد", woman: "زن", child: "بچه",
  house: "خانه", car: "ماشین", phone: "تلفن", computer: "کامپیوتر",
  "good morning": "صبح بخیر", "good night": "شب بخیر", "good afternoon": "ظهر بخیر",
  "how are you": "چطوری", "thank you very much": "خیلی ممنون",
  welcome: "خوش آمدید", sorry: "ببخشید", excuse: "ببخشید", me: "من",
  my: "من", your: "تو", his: "او", her: "او", its: "آن",
  and: "و", or: "یا", but: "اما", with: "با", without: "بدون",
  in: "در", on: "روی", at: "در", to: "به", from: "از", for: "برای",
  new: "جدید", old: "قدیمی", big: "بزرگ", small: "کوچک", hot: "گرم", cold: "سرد",
  fast: "سریع", slow: "آهسته", easy: "آسان", hard: "سخت", right: "درست",
  wrong: "اشتباه", open: "باز", close: "بسته", start: "شروع", end: "پایان",
  come: "بیا", go: "برو", see: "ببین", hear: "بشنو", speak: "صحبت کن",
  eat: "بخور", drink: "بنوش", sleep: "بخواب", wake: "بیدار شو",
  read: "بخوان", write: "بنویس", listen: "گوش کن", look: "نگاه کن",
  play: "بازی کن", work2: "کار کن",
};

// Build the reverse dictionary automatically (first spelling wins).
const FA_EN: Dict = {};
for (const [en, fa] of Object.entries(EN_FA)) {
  if (fa && !(fa in FA_EN)) FA_EN[fa] = en;
}

const PHRASES_EN = Object.keys(EN_FA).filter((k) => k.includes(" "));
const PHRASES_FA = Object.keys(FA_EN).filter((k) => k.includes(" "));

export interface Translation {
  text: string;
  coverage: number; // 0..1 of words found in the dictionary
  from: Lang2;
  to: Lang2;
  knownWords: number;
  totalWords: number;
}

export function translateLocal(text: string, to: Lang2): Translation {
  const from: Lang2 = to === "fa" ? "en" : "fa";
  const dict = to === "fa" ? EN_FA : FA_EN;
  const phrases = to === "fa" ? PHRASES_EN : PHRASES_FA;
  let out = text;
  // Longest phrases first, case-insensitive for English.
  const sorted = [...phrases].sort((a, b) => b.length - a.length);
  for (const p of sorted) {
    const re = new RegExp(p.replace(/[.*+?^${}()|[\]\\]/g, "\\$&"), "gi");
    out = out.replace(re, dict[p]);
  }
  // Word by word.
  const words = out.split(/(\s+)/);
  let known = 0;
  let total = 0;
  const translated = words.map((w) => {
    if (!/\S/.test(w)) return w;
    const m = w.match(/^([^A-Za-z\u0600-\u06FF]*)([A-Za-z\u0600-\u06FF]+)([^A-Za-z\u0600-\u06FF]*)$/);
    if (!m) return w;
    const [, pre, core, post] = m;
    total++;
    const key = to === "fa" ? core.toLowerCase() : core;
    const hit = dict[key];
    if (hit) { known++; return pre + (hit === "i" ? "I" : hit) + post; }
    return w;
  }).join("");
  return {
    text: translated,
    coverage: total === 0 ? 0 : known / total,
    from, to, knownWords: known, totalWords: total,
  };
}
