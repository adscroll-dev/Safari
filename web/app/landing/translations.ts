// Translations for the Safari download page. English is the default.
// The visitor picks a language from the switcher; the whole page
// (headings, download section, features, warning, footer) switches
// to it, and the choice is remembered in localStorage.
//
// Copy rules: short, human, benefit-led. No developer jargon
// (no WebSocket, WebRTC, X3DH, webhook, API numbers, or key/signature
// talk) — visitors care what Safari does for them, not how it works.

export type LangCode =
  | "en"
  | "fa"
  | "ar"
  | "ur"
  | "tr"
  | "ru"
  | "es"
  | "fr"
  | "de"
  | "hi"
  | "ps";

export const languages: { code: LangCode; label: string; dir: "ltr" | "rtl" }[] = [
  { code: "en", label: "English", dir: "ltr" },
  { code: "fa", label: "فارسی", dir: "rtl" },
  { code: "ps", label: "پښتو", dir: "rtl" },
  { code: "ar", label: "العربية", dir: "rtl" },
  { code: "ur", label: "اردو", dir: "rtl" },
  { code: "tr", label: "Türkçe", dir: "ltr" },
  { code: "ru", label: "Русский", dir: "ltr" },
  { code: "es", label: "Español", dir: "ltr" },
  { code: "fr", label: "Français", dir: "ltr" },
  { code: "de", label: "Deutsch", dir: "ltr" },
  { code: "hi", label: "हिन्दी", dir: "ltr" },
];

export type Strings = {
  tagline: string;
  intro: string;
  downloadTitle: string;
  versionLine: string;
  warning: string;
  featuresTitle: string;
  features: { title: string; desc: string }[];
  footer: string;
  aboutTitle: string;
  languageLabel: string;
  forYourPhone: string;
  detected: string;
  downloadButton: string;
  verify: string;
  category: string;
  free: string;
  updated: string;
  updatedValue: string;
  requires: string;
  offeredBy: string;
  screenshotsTitle: string;
  slowTitle: string;
  slowDesc: string;
  infoTitle: string;
  versionLabel: string;
  sizeLabel: string;
  heroBadge: string;
  heroSub: string;
  downloadCta: string;
  exploreCta: string;
  showcaseTitle: string;
  showcaseSub: string;
  shotChatList: string;
  shotChat: string;
  shotStories: string;
  shotCalls: string;
  shotCommunities: string;
  securityTitle: string;
  securitySub: string;
  securityItems: { title: string; desc: string }[];
  navDownload: string;
  navGames: string;
  privacyLink: string;
  gamesTitle: string;
  gamesDesc: string;
  gamesVersionLine: string;
  gamesDownloadButton: string;
  gamesNote: string;
  gamesFeatures: string[];
};

export const translations: Record<LangCode, Strings> = {
  en: {
    tagline: "Connect Beyond Limits",
    intro:
      "Meet Safari — a fast, private messenger for Android. Chat one-to-one or in groups, follow channels, call your friends, share stories, and send photos, files and voice notes — all protected by end-to-end encryption. Safari sips data instead of gulping it, so it flies even on slow internet. Free to download and use.",
    downloadTitle: "Download Safari for Android",
    versionLine:
      "Version 0.7.12 (Beta) · Android 6.0 and up · Your phone is detected automatically — just tap the button below and the right version downloads for you.",
    warning:
      "Only download Safari from this official website. APKs from other sites are not safe — don't install them.",
    featuresTitle: "Features",
    features: [
      { title: "Private Messaging", desc: "One-to-one chats, protected by real end-to-end encryption — only you and your friend can read them." },
      { title: "Group Conversations", desc: "Big groups with admins, pins, polls and invite links. Everyone stays in the loop." },
      { title: "Voice Calls", desc: "Clear voice calls that keep working even on a slow connection." },
      { title: "Video Calls", desc: "Face-to-face video calls, one-to-one or with your whole group." },
      { title: "Stories", desc: "Share photos and moments that disappear after 24 hours." },
      { title: "Privacy & Security", desc: "App lock, privacy settings, block and report — your space stays yours." },
      { title: "Bots", desc: "Bots that help you get things done, right inside your chats." },
      { title: "Ultra-low data", desc: "Uses far less data than other messengers — calls, photos and media, all squeezed small." },
    ],
    footer:
      "Free to download and use. The Android beta is available above; iPhone is coming later.",
    aboutTitle: "About Safari",
    languageLabel: "Language",
    forYourPhone: "Safari 0.7.12 — for your phone",
    detected: "Detected for your phone:",
    downloadButton: "Download Safari for Android",
    verify: "Verify the download (SHA-256):",
    category: "Communication",
    free: "Free",
    updated: "Updated",
    updatedValue: "Nov 19, 2026",
    requires: "Requires",
    offeredBy: "Offered by",
    screenshotsTitle: "Preview",
    slowTitle: "Built for slow internet",
    slowDesc:
      "Downloads are small and resume if they stop. Data Saver is on by default — voice calls use just 12 kbps, photos arrive compressed, and media downloads only when you tap it.",
    infoTitle: "App info",
    versionLabel: "Version",
    sizeLabel: "Size",
    heroBadge: "Safari Messenger · Beta",
    heroSub: "Experience a new generation of messaging. Connect, communicate, and share without boundaries.",
    downloadCta: "Download Safari",
    exploreCta: "Explore features",
    showcaseTitle: "See Safari in action",
    showcaseSub: "Real screens from the app — chats, conversations and stories.",
    shotChatList: "Your chats, at a glance",
    shotChat: "A conversation",
    shotStories: "Stories",
    shotCalls: "Calls",
    shotCommunities: "Communities",
    securityTitle: "Private by design",
    securitySub: "Security that works quietly in the background — so you can just talk.",
    navDownload: "Download",
    navGames: "Games",
    privacyLink: "Privacy Policy",
    gamesTitle: "Games",
    gamesDesc:
      "Safari City — an open-world Android game. Explore a 3D city on foot or by car, take on missions, outrun the police, sail boats, fly a seaplane, and dive for treasure. Free to download and play, and it works fully offline.",
    gamesVersionLine: "Version 0.15.0 · Android · 22.3 MB (23,431,772 bytes)",
    gamesDownloadButton: "Download Safari City for Android",
    gamesNote:
      "Safari City is in active development — new versions with new features arrive regularly.",
    gamesFeatures: [
      "Open 3D city — walk, drive, ride a motorbike",
      "Missions, races, taxi and bus jobs",
      "Police chases, boats, jet-ski and seaplane",
      "Diving, fishing and sunken treasure",
    ],
    securityItems: [
      { title: "End-to-end encryption", desc: "Your private chats are locked so only you and the person you’re talking to can read them." },
      { title: "App lock", desc: "Lock Safari with a PIN or your fingerprint. Your chats stay hidden until it’s you." },
      { title: "Privacy controls", desc: "Choose who sees your photo and status, block anyone, and report spam in one tap." },
    ],
  },
  fa: {
    tagline: "اتصال فراتر از محدودیت‌ها",
    intro:
      "با Safari آشنا شوید — پیام‌رسانی سریع و امن برای اندروید. خصوصی یا گروهی چت کنید، کانال‌ها را دنبال کنید، با دوستانتان تماس بگیرید، استوری بگذارید و عکس، فایل و پیام صوتی بفرستید — همه با رمزنگاری سرتاسری. Safari خیلی کم اینترنت مصرف می‌کند، پس حتی با اینترنت کند هم سریع کار می‌کند. دانلود و استفاده‌اش رایگان است.",
    downloadTitle: "دانلود Safari برای اندروید",
    versionLine:
      "نسخه ۰٫۷٫۲ (بتا) · اندروید ۶٫۰ به بالا · گوشی شما به‌صورت خودکار تشخیص داده می‌شود — فقط دکمه زیر را بزنید و نسخه مناسب گوشی شما دانلود می‌شود.",
    warning:
      "Safari را فقط از همین وب‌سایت رسمی دانلود کنید. فایل‌های APK از سایت‌های دیگر امن نیستند — نصبشان نکنید.",
    featuresTitle: "قابلیت‌ها",
    features: [
      { title: "پیام‌رسانی خصوصی", desc: "چت یک‌به‌یک، با رمزنگاری سرتاسری واقعی — فقط شما و دوستتان می‌توانید آن را بخوانید." },
      { title: "گفتگوهای گروهی", desc: "گروه‌های بزرگ با مدیر، سنجاق، نظرسنجی و لینک دعوت. همه در جریان می‌مانند." },
      { title: "تماس صوتی", desc: "تماس صوتی شفاف که حتی با اینترنت کند هم کار می‌کند." },
      { title: "تماس تصویری", desc: "تماس تصویری روبه‌رو، یک‌به‌یک یا با کل گروه." },
      { title: "استوری‌ها", desc: "عکس و لحظه‌هایتان را به اشتراک بگذارید؛ بعد از ۲۴ ساعت ناپدید می‌شوند." },
      { title: "حریم خصوصی و امنیت", desc: "قفل برنامه، تنظیمات حریم خصوصی، مسدود و گزارش — فضای شما مال شماست." },
      { title: "ربات‌ها", desc: "ربات‌هایی که کارها را درست داخل چت‌هایتان انجام می‌دهند." },
      { title: "مصرف بسیار کم اینترنت", desc: "خیلی کمتر از پیام‌رسان‌های دیگر اینترنت مصرف می‌کند — تماس، عکس و رسانه، همه فشرده و کوچک." },
    ],
    footer:
      "دانلود و استفاده رایگان است. نسخه بتای اندروید در بالا در دسترس است؛ آیفون بعداً می‌آید.",
    aboutTitle: "درباره Safari",
    languageLabel: "زبان",
    forYourPhone: "Safari ۰٫۷٫۲ — برای گوشی شما",
    detected: "برای گوشی شما تشخیص داده شد:",
    downloadButton: "دانلود Safari برای اندروید",
    verify: "بررسی دانلود (SHA-256):",
    category: "ارتباطات",
    free: "رایگان",
    updated: "به‌روز شده",
    updatedValue: "۱۹ نوامبر ۲۰۲۶",
    requires: "نیاز به",
    offeredBy: "ارائه‌دهنده",
    screenshotsTitle: "پیش‌نمایش",
    slowTitle: "ساخته‌شده برای انترنت کم‌سرعت",
    slowDesc:
      "دانلودها کم‌حجم‌اند و اگر قطع شوند ادامه می‌یابند. صرفه‌جویی در مصرف داده به‌صورت پیش‌فرض روشن است — تماس صوتی فقط ۱۲ کیلوبیت بر ثانیه مصرف می‌کند، عکس‌ها فشرده می‌رسند و رسانه فقط وقتی دانلود می‌شود که شما لمس کنید.",
    infoTitle: "اطلاعات برنامه",
    versionLabel: "نسخه",
    sizeLabel: "حجم",
    heroBadge: "پیام‌رسان Safari · بتا",
    heroSub: "نسل جدید پیام‌رسانی را تجربه کنید. بدون مرز، متصل شوید، ارتباط بگیرید و به اشتراک بگذارید.",
    downloadCta: "دانلود Safari",
    exploreCta: "دیدن قابلیت‌ها",
    showcaseTitle: "Safari را در عمل ببینید",
    showcaseSub: "صفحه‌های واقعی برنامه — چت‌ها، گفتگوها و استوری‌ها.",
    shotChatList: "چت‌های شما، در یک نگاه",
    shotChat: "یک گفتگو",
    shotStories: "استوری‌ها",
    shotCalls: "تماس‌ها",
    shotCommunities: "انجمن‌ها",
    securityTitle: "حریم خصوصی، از پایه",
    securitySub: "امنیتی که بی‌صدا در پس‌زمینه کار می‌کند — تا شما فقط حرف بزنید.",
    navDownload: "دانلود",
    navGames: "بازی‌ها",
    privacyLink: "سیاست حریم خصوصی",
    gamesTitle: "بازی‌ها",
    gamesDesc:
      "Safari City — یک بازی جهان‌باز برای اندروید. در یک شهر سه‌بعدی پیاده یا با ماشین بگردید، مأموریت انجام دهید، از پلیس فرار کنید، با قایق و هواپیمای دریایی سفر کنید و زیر آب گنج پیدا کنید. دانلود و بازی رایگان است و کاملاً آفلاین کار می‌کند.",
    gamesVersionLine: "نسخه ۰٫۱۵٫۰ · اندروید · ۲۲٫۳ مگابایت (۲۳٬۴۳۱٬۷۷۲ بایت)",
    gamesDownloadButton: "دانلود Safari City برای اندروید",
    gamesNote:
      "Safari City در حال توسعه فعال است — نسخه‌های جدید با قابلیت‌های تازه مرتب می‌رسند.",
    gamesFeatures: [
      "شهر سه‌بعدی باز — پیاده، با ماشین و موتور",
      "مأموریت‌ها، مسابقه‌ها، تاکسی و اتوبوس",
      "تعقیب پلیس، قایق، جت‌اسکی و هواپیمای دریایی",
      "غواصی، ماهیگیری و گنج غرق‌شده",
    ],
    securityItems: [
      { title: "رمزنگاری سرتاسری", desc: "چت‌های خصوصی شما طوری قفل شده‌اند که فقط شما و طرف مقابلتان می‌توانید آن‌ها را بخوانید." },
      { title: "قفل برنامه", desc: "Safari را با پین یا اثر انگشت قفل کنید. چت‌هایتان تا وقتی شما نباشید پنهان می‌مانند." },
      { title: "کنترل‌های حریم خصوصی", desc: "انتخاب کنید چه کسی عکس و وضعیت شما را ببیند، هر کسی را مسدود کنید و اسپم را با یک لمس گزارش دهید." },
    ],
  },
  ar: {
    tagline: "تواصل بلا حدود",
    intro:
      "تعرّف على Safari — مراسل سريع وخاص لأندرويد. تحدّث مع أصدقائك فرديًا أو في مجموعات، وتابع القنوات، واتصل بأصدقائك، وشارك القصص، وأرسل الصور والملفات والرسائل الصوتية — كل ذلك محمي بالتشفير الكامل. يستهلك Safari بيانات أقل بكثير من غيره، فيعمل بسرعة حتى مع الإنترنت البطيء. تنزيله واستخدامه مجاني.",
    downloadTitle: "تنزيل Safari لأندرويد",
    versionLine:
      "الإصدار 0.7.12 (تجريبي) · يتطلب أندرويد 6.0 أو أحدث · يكتشف الموقع هاتفك تلقائيًا — فقط اضغط الزر أدناه وستُنزَّل النسخة المناسبة لهاتفك.",
    warning:
      "حمّل Safari من هذا الموقع الرسمي فقط. ملفات APK من مواقع أخرى غير آمنة — لا تثبّتها.",
    featuresTitle: "الميزات",
    features: [
      { title: "مراسلة خاصة", desc: "دردشات فردية محمية بتشفير كامل حقيقي — أنت وصديقك فقط من يستطيع قراءتها." },
      { title: "محادثات جماعية", desc: "مجموعات كبيرة مع مشرفين وتثبيت واستطلاعات وروابط دعوة. الجميع يبقى على اطلاع." },
      { title: "مكالمات صوتية", desc: "مكالمات صوتية واضحة تعمل حتى مع اتصال بطيء." },
      { title: "مكالمات مرئية", desc: "مكالمات فيديو وجهًا لوجه، فرديًا أو مع مجموعتك كلها." },
      { title: "القصص", desc: "شارك الصور واللحظات التي تختفي بعد 24 ساعة." },
      { title: "الخصوصية والأمان", desc: "قفل التطبيق، وإعدادات الخصوصية، والحظر والإبلاغ — مساحتك تبقى لك." },
      { title: "بوتات", desc: "بوتات تساعدك على إنجاز الأمور داخل دردشاتك." },
      { title: "استهلاك بيانات منخفض جدًا", desc: "يستهلك بيانات أقل بكثير من المراسلات الأخرى — المكالمات والصور والوسائط، كلها مضغوطة وصغيرة." },
    ],
    footer:
      "التنزيل والاستخدام مجاني. نسخة أندرويد التجريبية متاحة أعلاه؛ ونسخة iOS قادمة لاحقًا.",
    aboutTitle: "عن Safari",
    languageLabel: "اللغة",
    forYourPhone: "Safari 0.7.12 — لهاتفك",
    detected: "تم الكشف لهاتفك:",
    downloadButton: "تنزيل Safari لأندرويد",
    verify: "تحقق من التنزيل (SHA-256):",
    category: "الاتصالات",
    free: "مجاني",
    updated: "تم التحديث",
    updatedValue: "19 نوفمبر 2026",
    requires: "يتطلب",
    offeredBy: "مقدَّم من",
    screenshotsTitle: "معاينة",
    slowTitle: "مصمَّم لإنترنت بطيء",
    slowDesc:
      "التنزيلات صغيرة وتستأنف إذا توقفت. توفير البيانات مفعَّل افتراضيًا — المكالمات الصوتية تستهلك 12 كيلوبت/ث فقط، والصور تصل مضغوطة، والوسائط تُنزَّل فقط عندما تنقر عليها.",
    infoTitle: "معلومات التطبيق",
    versionLabel: "الإصدار",
    sizeLabel: "الحجم",
    heroBadge: "Safari Messenger · تجريبي",
    heroSub: "جرّب جيلًا جديدًا من المراسلة. تواصل، وتحدّث، وشارك بلا حدود.",
    downloadCta: "تنزيل Safari",
    exploreCta: "استكشف الميزات",
    showcaseTitle: "شاهد Safari أثناء العمل",
    showcaseSub: "شاشات حقيقية من التطبيق — الدردشات والمحادثات والقصص.",
    shotChatList: "دردشاتك، في لمحة",
    shotChat: "محادثة",
    shotStories: "القصص",
    shotCalls: "المكالمات",
    shotCommunities: "المجتمعات",
    securityTitle: "الخصوصية بالتصميم",
    securitySub: "أمان يعمل بهدوء في الخلفية — لتتحدث أنت فقط.",
    navDownload: "تنزيل",
    navGames: "الألعاب",
    privacyLink: "سياسة الخصوصية",
    gamesTitle: "الألعاب",
    gamesDesc:
      "Safari City — لعبة عالم مفتوح لأندرويد. استكشف مدينة ثلاثية الأبعاد سيرًا على الأقدام أو بالسيارة، ونفّذ المهام، واهرب من الشرطة، وأبحر بالقوارب، وحلّق بالطائرة المائية، واغص للبحث عن الكنز. مجانية للتنزيل واللعب، وتعمل دون إنترنت تمامًا.",
    gamesVersionLine: "الإصدار 0.15.0 · أندرويد · 22.3 م.ب. (23,431,772 بايت)",
    gamesDownloadButton: "تنزيل Safari City لأندرويد",
    gamesNote:
      "Safari City قيد التطوير النشط — إصدارات جديدة بميزات جديدة تصل بانتظام.",
    gamesFeatures: [
      "مدينة مفتوحة ثلاثية الأبعاد — سيرًا أو بالسيارة أو بالدراجة",
      "مهام وسباقات ووظائف سيارة أجرة وحافلة",
      "مطاردات الشرطة والقوارب والدراجة المائية والطائرة المائية",
      "الغوص والصيد والكنز الغارق",
    ],
    securityItems: [
      { title: "تشفير كامل", desc: "دردشاتك الخاصة مقفلة بحيث لا يستطيع قراءتها سواك والشخص الذي تتحدث معه." },
      { title: "قفل التطبيق", desc: "اقفل Safari برمز PIN أو ببصمتك. تبقى دردشاتك مخفية حتى تكون أنت." },
      { title: "عناصر تحكم الخصوصية", desc: "اختر من يرى صورتك وحالتك، واحظر أي شخص، وأبلغ عن البريد المزعج بنقرة واحدة." },
    ],
  },
  ur: {
    tagline: "حدود سے آگے جڑیں",
    intro:
      "Safari سے ملیں — اینڈرائیڈ کے لیے ایک تیز اور نجی میسنجر۔ دوستوں سے اکیلے یا بڑے گروپس میں چیٹ کریں، چینلز فالو کریں، دوستوں کو کال کریں، اسٹوریز شیئر کریں، اور تصاویر، فائلیں اور وائس نوٹس بھیجیں — سب مکمل خفیہ کاری کے ساتھ۔ Safari دوسرے میسنجرز کے مقابلے میں بہت کم ڈیٹا استعمال کرتا ہے، اس لیے سست انٹرنیٹ پر بھی تیز چلتا ہے۔ ڈاؤن لوڈ اور استعمال مفت ہے۔",
    downloadTitle: "اینڈرائیڈ کے لیے Safari ڈاؤن لوڈ کریں",
    versionLine:
      "ورژن 0.7.12 (بیٹا) · اینڈرائیڈ 6.0 یا اس سے اوپر · سائٹ آپ کے فون کو خود بخود پہچانتی ہے — بس نیچے والا بٹن دبائیں اور آپ کے فون کے لیے صحیح ورژن ڈاؤن لوڈ ہو جائے گا۔",
    warning:
      "Safari صرف اسی سرکاری ویب سائٹ سے ڈاؤن لوڈ کریں۔ دوسری سائٹس کی APK فائلیں محفوظ نہیں ہیں — انہیں انسٹال نہ کریں۔",
    featuresTitle: "خصوصیات",
    features: [
      { title: "نجی میسجنگ", desc: "اکیلے کی چیٹس، حقیقی مکمل خفیہ کاری کے ساتھ — صرف آپ اور آپ کا دوست انہیں پڑھ سکتے ہیں۔" },
      { title: "گروپ گفتگوئیں", desc: "ایڈمن، پن، پولز اور انوائٹ لنکس کے ساتھ بڑے گروپس۔ سب باخبر رہتے ہیں۔" },
      { title: "وائس کالز", desc: "صاف وائس کالز جو سست کنکشن پر بھی کام کرتی ہیں۔" },
      { title: "ویڈیو کالز", desc: "آمنے سامنے ویڈیو کالز، اکیلے یا پورے گروپ کے ساتھ۔" },
      { title: "اسٹوریز", desc: "اپنی تصاویر اور لمحے شیئر کریں جو 24 گھنٹے بعد غائب ہو جاتے ہیں۔" },
      { title: "پرائیویسی اور سیکیورٹی", desc: "ایپ لاک، پرائیویسی سیٹنگز، بلاک اور رپورٹ — آپ کی جگہ آپ کی ہی رہتی ہے۔" },
      { title: "بوٹس", desc: "ایسے بوٹس جو آپ کے کام آپ کی چیٹس کے اندر ہی کر دیں۔" },
      { title: "بہت کم ڈیٹا استعمال", desc: "دوسرے میسنجرز کے مقابلے میں بہت کم ڈیٹا استعمال کرتا ہے — کالز، تصاویر اور میڈیا، سب چھوٹا اور کمپریسڈ۔" },
    ],
    footer:
      "ڈاؤن لوڈ اور استعمال مفت ہے۔ اینڈرائیڈ بیٹا اوپر دستیاب ہے؛ آئی فون بعد میں آئے گا۔",
    aboutTitle: "Safari کے بارے میں",
    languageLabel: "زبان",
    forYourPhone: "Safari 0.7.12 — آپ کے فون کے لیے",
    detected: "آپ کے فون کے لیے پہچانا گیا:",
    downloadButton: "اینڈرائیڈ کے لیے Safari ڈاؤن لوڈ کریں",
    verify: "ڈاؤن لوڈ کی تصدیق کریں (SHA-256):",
    category: "مواصلات",
    free: "مفت",
    updated: "اپ ڈیٹ",
    updatedValue: "19 نومبر 2026",
    requires: "ضرورت",
    offeredBy: "پیشکش",
    screenshotsTitle: "جھلک",
    slowTitle: "سست انٹرنیٹ کے لیے بنایا گیا",
    slowDesc:
      "ڈاؤن لوڈ چھوٹے ہیں اور رک جائیں تو جاری رہتے ہیں۔ ڈیٹا سیور بطور پیش فرض آن ہے — وائس کالز صرف 12 kbps استعمال کرتی ہیں، تصاویر کمپریس ہو کر آتی ہیں، اور میڈیا صرف تب ڈاؤن لوڈ ہوتا ہے جب آپ ٹیپ کریں۔",
    infoTitle: "ایپ کی معلومات",
    versionLabel: "ورژن",
    sizeLabel: "سائز",
    heroBadge: "Safari Messenger · بیٹا",
    heroSub: "میسجنگ کی نئی نسل کا تجربہ کریں۔ بغیر کسی حد کے جڑیں، بات کریں اور شیئر کریں۔",
    downloadCta: "Safari ڈاؤن لوڈ کریں",
    exploreCta: "خصوصیات دیکھیں",
    showcaseTitle: "Safari کو عمل میں دیکھیں",
    showcaseSub: "ایپ کی اصل اسکرینیں — چیٹس، گفتگوئیں اور اسٹوریز۔",
    shotChatList: "آپ کی چیٹس، ایک نظر میں",
    shotChat: "ایک گفتگو",
    shotStories: "اسٹوریز",
    shotCalls: "کالز",
    shotCommunities: "کمیونٹیز",
    securityTitle: "پرائیویسی، ڈیزائن سے",
    securitySub: "سیکیورٹی جو پس منظر میں خاموشی سے کام کرتی ہے — تاکہ آپ بس بات کریں۔",
    navDownload: "ڈاؤن لوڈ",
    navGames: "گیمز",
    privacyLink: "پرائیویسی پالیسی",
    gamesTitle: "گیمز",
    gamesDesc:
      "Safari City — اینڈرائیڈ کے لیے ایک اوپن ورلڈ گیم۔ 3D شہر میں پیدل یا گاڑی سے گھومیں، مشن مکمل کریں، پولیس سے بچیں، کشتی چلائیں، سی پلین اڑائیں اور خزانہ ڈھونڈیں۔ ڈاؤن لوڈ اور کھیلنا مفت ہے اور یہ مکمل طور پر آف لائن کام کرتی ہے۔",
    gamesVersionLine: "ورژن 0.15.0 · اینڈرائیڈ · 22.3 MB (23,431,772 بائٹس)",
    gamesDownloadButton: "اینڈرائیڈ کے لیے Safari City ڈاؤن لوڈ کریں",
    gamesNote:
      "Safari City مسلسل تیار ہو رہی ہے — نئے فیچرز کے ساتھ نئے ورژن باقاعدگی سے آتے ہیں۔",
    gamesFeatures: [
      "کھلا 3D شہر — پیدل، گاڑی یا موٹر بائیک سے",
      "مشن، ریس، ٹیکسی اور بس کی نوکریاں",
      "پولیس کا پیچھا، کشتی، جیٹ اسکی اور سی پلین",
      "ڈائیونگ، ماہی گیری اور ڈوبا ہوا خزانہ",
    ],
    securityItems: [
      { title: "مکمل خفیہ کاری", desc: "آپ کی نجی چیٹس اس طرح بند ہیں کہ صرف آپ اور جس سے آپ بات کر رہے ہیں وہی انہیں پڑھ سکتے ہیں۔" },
      { title: "ایپ لاک", desc: "Safari کو پن یا فنگر پرنٹ سے لاک کریں۔ آپ کی چیٹس تب تک چھپی رہتی ہیں جب تک آپ نہ ہوں۔" },
      { title: "پرائیویسی کنٹرولز", desc: "چنیں کہ آپ کی تصویر اور اسٹیٹس کون دیکھے، کسی کو بھی بلاک کریں، اور اسپام کی ایک ٹیپ سے رپورٹ کریں۔" },
    ],
  },
  tr: {
    tagline: "Sınırların Ötesinde Bağlanın",
    intro:
      "Safari ile tanışın — Android için hızlı ve özel bir mesajlaşma uygulaması. Arkadaşlarınızla bire bir veya büyük gruplarda sohbet edin, kanalları takip edin, arkadaşlarınızı arayın, hikâyeler paylaşın, fotoğraf, dosya ve sesli notlar gönderin — hepsi uçtan uca şifreleme ile korunuyor. Safari diğer mesajlaşma uygulamalarından çok daha az veri kullanır, bu yüzden yavaş internette bile uçar. İndirmesi ve kullanımı ücretsizdir.",
    downloadTitle: "Android için Safari'yu indirin",
    versionLine:
      "Sürüm 0.7.12 (Beta) · Android 6.0 ve üzeri · Site telefonunuzu otomatik olarak algılar — aşağıdaki düğmeye dokunmanız yeterli, telefonunuza uygun sürüm indirilir.",
    warning:
      "Safari'yu yalnızca bu resmî web sitesinden indirin. Diğer sitelerdeki APK'lar güvenli değildir — yüklemeyin.",
    featuresTitle: "Özellikler",
    features: [
      { title: "Özel mesajlaşma", desc: "Gerçek uçtan uca şifrelemeyle korunan bire bir sohbetler — onları yalnızca siz ve arkadaşınız okuyabilir." },
      { title: "Grup sohbetleri", desc: "Yöneticiler, sabitleme, anketler ve davet bağlantılarıyla büyük gruplar. Herkes haberdar olur." },
      { title: "Sesli aramalar", desc: "Yavaş bağlantıda bile çalışan net sesli aramalar." },
      { title: "Görüntülü aramalar", desc: "Bire bir veya tüm grubunuzla yüz yüze görüntülü aramalar." },
      { title: "Hikâyeler", desc: "24 saat sonra kaybolan fotoğraf ve anılarınızı paylaşın." },
      { title: "Gizlilik ve güvenlik", desc: "Uygulama kilidi, gizlilik ayarları, engelleme ve bildirme — alanınız size ait kalır." },
      { title: "Botlar", desc: "İşlerinizi sohbetlerinizin içinde halleden botlar." },
      { title: "Çok düşük veri kullanımı", desc: "Diğer mesajlaşma uygulamalarından çok daha az veri kullanır — aramalar, fotoğraflar ve medya, hepsi sıkıştırılmış ve küçük." },
    ],
    footer:
      "İndirmesi ve kullanımı ücretsizdir. Android beta sürümü yukarıda mevcuttur; iPhone daha sonra gelecektir.",
    aboutTitle: "Safari Hakkında",
    languageLabel: "Dil",
    forYourPhone: "Safari 0.7.12 — telefonunuz için",
    detected: "Telefonunuz için algılandı:",
    downloadButton: "Android için Safari'yu indirin",
    verify: "İndirmeyi doğrulayın (SHA-256):",
    category: "İletişim",
    free: "Ücretsiz",
    updated: "Güncellendi",
    updatedValue: "19 Kasım 2026",
    requires: "Gerekir",
    offeredBy: "Sunucu",
    screenshotsTitle: "Önizleme",
    slowTitle: "Yavaş internet için tasarlandı",
    slowDesc:
      "İndirmeler küçüktür ve durursa devam eder. Veri Tasarrufu varsayılan olarak açıktır — sesli aramalar yalnızca 12 kbps kullanır, fotoğraflar sıkıştırılmış gelir ve medya yalnızca siz dokununca indirilir.",
    infoTitle: "Uygulama bilgisi",
    versionLabel: "Sürüm",
    sizeLabel: "Boyut",
    heroBadge: "Safari Messenger · Beta",
    heroSub: "Yeni nesil mesajlaşmayı deneyimleyin. Sınırsız bağlanın, iletişim kurun ve paylaşın.",
    downloadCta: "Safari’yi indir",
    exploreCta: "Özellikleri keşfet",
    showcaseTitle: "Safari’yi iş başında görün",
    showcaseSub: "Uygulamadan gerçek ekranlar — sohbetler, görüşmeler ve hikâyeler.",
    shotChatList: "Sohbetleriniz, bir bakışta",
    shotChat: "Bir sohbet",
    shotStories: "Hikâyeler",
    shotCalls: "Aramalar",
    shotCommunities: "Topluluklar",
    securityTitle: "Tasarım gereği gizlilik",
    securitySub: "Arka planda sessizce çalışan güvenlik — siz sadece konuşun.",
    navDownload: "İndir",
    navGames: "Oyunlar",
    privacyLink: "Gizlilik Politikası",
    gamesTitle: "Oyunlar",
    gamesDesc:
      "Safari City — Android için açık dünya oyunu. 3B bir şehirde yürüyerek veya arabayla gezin, görevleri tamamla, polisten kaç, tekneyle yol al, deniz uçağıyla uç ve hazine için dal. İndirmesi ve oynaması ücretsizdir ve tamamen çevrimdışı çalışır.",
    gamesVersionLine: "Sürüm 0.15.0 · Android · 22,3 MB (23.431.772 bayt)",
    gamesDownloadButton: "Android için Safari City'yi indirin",
    gamesNote:
      "Safari City aktif olarak geliştiriliyor — yeni özelliklerle yeni sürümler düzenli olarak geliyor.",
    gamesFeatures: [
      "Açık 3B şehir — yürüyerek, arabayla veya motosikletle",
      "Görevler, yarışlar, taksi ve otobüs işleri",
      "Polis kovalamacaları, tekne, jet-ski ve deniz uçağı",
      "Dalış, balık tutma ve batık hazine",
    ],
    securityItems: [
      { title: "Uçtan uca şifreleme", desc: "Özel sohbetleriniz öyle kilitli ki onları yalnızca siz ve konuştuğunuz kişi okuyabilir." },
      { title: "Uygulama kilidi", desc: "Safari’yi PIN veya parmak iziyle kilitleyin. Sohbetleriniz siz olana kadar gizli kalır." },
      { title: "Gizlilik kontrolleri", desc: "Fotoğrafınızı ve durumunuzu kimin göreceğini seçin, istediğinizi engelleyin ve istenmeyen mesajları tek dokunuşla bildirin." },
    ],
  },
  ru: {
    tagline: "Связь без границ",
    intro:
      "Познакомьтесь с Safari — быстрым и приватным мессенджером для Android. Общайтесь с друзьями один на один или в больших группах, подписывайтесь на каналы, звоните друзьям, делитесь историями, отправляйте фото, файлы и голосовые сообщения — всё защищено сквозным шифрованием. Safari расходует гораздо меньше трафика, чем другие мессенджеры, поэтому летает даже на медленном интернете. Скачивание и использование бесплатны.",
    downloadTitle: "Скачать Safari для Android",
    versionLine:
      "Версия 0.7.12 (бета) · Требуется Android 6.0 или новее · Сайт автоматически определяет ваш телефон — просто нажмите кнопку ниже, и скачается подходящая для него версия.",
    warning:
      "Скачивайте Safari только с этого официального сайта. APK-файлы с других сайтов небезопасны — не устанавливайте их.",
    featuresTitle: "Возможности",
    features: [
      { title: "Личные сообщения", desc: "Чаты один на один с настоящим сквозным шифрованием — прочитать их можете только вы и ваш друг." },
      { title: "Групповые беседы", desc: "Большие группы с администраторами, закреплением, опросами и ссылками-приглашениями. Все в курсе." },
      { title: "Голосовые звонки", desc: "Чистые голосовые звонки, которые работают даже при медленном соединении." },
      { title: "Видеозвонки", desc: "Видеозвонки лицом к лицу — один на один или со всей группой." },
      { title: "Истории", desc: "Делитесь фото и моментами, которые исчезают через 24 часа." },
      { title: "Приватность и безопасность", desc: "Блокировка приложения, настройки приватности, блокировка и жалобы — ваше пространство остаётся вашим." },
      { title: "Боты", desc: "Боты, которые помогают делать дела прямо в ваших чатах." },
      { title: "Очень низкий расход трафика", desc: "Расходует гораздо меньше трафика, чем другие мессенджеры, — звонки, фото и медиа, всё сжато и компактно." },
    ],
    footer:
      "Скачивание и использование бесплатны. Бета-версия для Android доступна выше; iPhone появится позже.",
    aboutTitle: "О Safari",
    languageLabel: "Язык",
    forYourPhone: "Safari 0.7.12 — для вашего телефона",
    detected: "Определено для вашего телефона:",
    downloadButton: "Скачать Safari для Android",
    verify: "Проверьте загрузку (SHA-256):",
    category: "Связь",
    free: "Бесплатно",
    updated: "Обновлено",
    updatedValue: "19 ноября 2026 г.",
    requires: "Требуется",
    offeredBy: "Разработчик",
    screenshotsTitle: "Предпросмотр",
    slowTitle: "Сделано для медленного интернета",
    slowDesc:
      "Загрузки небольшие и возобновляются, если прервутся. Экономия трафика включена по умолчанию — голосовые звонки используют всего 12 кбит/с, фото приходят сжатыми, а медиа скачивается только когда вы нажмёте.",
    infoTitle: "Информация о приложении",
    versionLabel: "Версия",
    sizeLabel: "Размер",
    heroBadge: "Safari Messenger · Бета",
    heroSub: "Попробуйте мессенджер нового поколения. Общайтесь, созванивайтесь и делитесь без границ.",
    downloadCta: "Скачать Safari",
    exploreCta: "Смотреть возможности",
    showcaseTitle: "Посмотрите Safari в деле",
    showcaseSub: "Настоящие экраны приложения — чаты, беседы и истории.",
    shotChatList: "Ваши чаты — с первого взгляда",
    shotChat: "Беседа",
    shotStories: "Истории",
    shotCalls: "Звонки",
    shotCommunities: "Сообщества",
    securityTitle: "Приватность по замыслу",
    securitySub: "Безопасность тихо работает в фоне — а вы просто общаетесь.",
    navDownload: "Скачать",
    navGames: "Игры",
    privacyLink: "Политика конфиденциальности",
    gamesTitle: "Игры",
    gamesDesc:
      "Safari City — игра с открытым миром для Android. Исследуйте 3D-город пешком или на машине, выполняйте задания, уходите от полиции, плавайте на лодках, летайте на гидросамолёте и ныряйте за сокровищами. Скачивание и игра бесплатны, а игра полностью работает офлайн.",
    gamesVersionLine: "Версия 0.15.0 · Android · 22,3 МБ (23 431 772 байт)",
    gamesDownloadButton: "Скачать Safari City для Android",
    gamesNote:
      "Safari City активно развивается — новые версии с новыми возможностями выходят регулярно.",
    gamesFeatures: [
      "Открытый 3D-город — пешком, на машине или мотоцикле",
      "Задания, гонки, работа таксистом и водителем автобуса",
      "Погони полиции, лодки, гидроцикл и гидросамолёт",
      "Дайвинг, рыбалка и затонувшие сокровища",
    ],
    securityItems: [
      { title: "Сквозное шифрование", desc: "Ваши личные чаты закрыты так, что прочитать их можете только вы и ваш собеседник." },
      { title: "Блокировка приложения", desc: "Запирайте Safari ПИН-кодом или отпечатком пальца. Чаты скрыты, пока это не вы." },
      { title: "Настройки приватности", desc: "Выбирайте, кто видит ваше фото и статус, блокируйте кого угодно и жалуйтесь на спам в одно касание." },
    ],
  },
  es: {
    tagline: "Conecta más allá de los límites",
    intro:
      "Conoce Safari: un mensajero rápido y privado para Android. Chatea con tus amigos en privado o en grupos grandes, sigue canales, llama a tus amigos, comparte historias y envía fotos, archivos y notas de voz — todo protegido con cifrado de extremo a extremo. Safari usa mucho menos datos que otros mensajeros, así que vuela incluso con internet lento. Descargarlo y usarlo es gratis.",
    downloadTitle: "Descargar Safari para Android",
    versionLine:
      "Versión 0.7.12 (beta) · Android 6.0 o superior · El sitio detecta tu teléfono automáticamente: solo toca el botón de abajo y se descargará la versión adecuada para ti.",
    warning:
      "Descarga Safari solo desde este sitio web oficial. Los APK de otros sitios no son seguros: no los instales.",
    featuresTitle: "Funciones",
    features: [
      { title: "Mensajería privada", desc: "Chats individuales protegidos con cifrado real de extremo a extremo: solo tú y tu amigo podéis leerlos." },
      { title: "Conversaciones de grupo", desc: "Grupos grandes con administradores, mensajes fijados, encuestas y enlaces de invitación. Todos al tanto." },
      { title: "Llamadas de voz", desc: "Llamadas de voz claras que funcionan incluso con una conexión lenta." },
      { title: "Videollamadas", desc: "Videollamadas cara a cara, en privado o con todo tu grupo." },
      { title: "Historias", desc: "Comparte fotos y momentos que desaparecen a las 24 horas." },
      { title: "Privacidad y seguridad", desc: "Bloqueo de la app, ajustes de privacidad, bloqueo y denuncias: tu espacio sigue siendo tuyo." },
      { title: "Bots", desc: "Bots que te ayudan a hacer cosas dentro de tus chats." },
      { title: "Uso de datos ultrabajo", desc: "Usa muchos menos datos que otros mensajeros: llamadas, fotos y medios, todo comprimido y pequeño." },
    ],
    footer:
      "Descargarlo y usarlo es gratis. La beta de Android está disponible arriba; el iPhone llegará después.",
    aboutTitle: "Acerca de Safari",
    languageLabel: "Idioma",
    forYourPhone: "Safari 0.7.12 — para tu teléfono",
    detected: "Detectado para tu teléfono:",
    downloadButton: "Descargar Safari para Android",
    verify: "Verifica la descarga (SHA-256):",
    category: "Comunicación",
    free: "Gratis",
    updated: "Actualizado",
    updatedValue: "19 de noviembre de 2026",
    requires: "Requiere",
    offeredBy: "Ofrecido por",
    screenshotsTitle: "Vista previa",
    slowTitle: "Hecho para internet lento",
    slowDesc:
      "Las descargas son pequeñas y se reanudan si se detienen. El ahorro de datos está activado por defecto: las llamadas de voz usan solo 12 kbps, las fotos llegan comprimidas y los medios se descargan solo cuando los tocas.",
    infoTitle: "Información de la app",
    versionLabel: "Versión",
    sizeLabel: "Tamaño",
    heroBadge: "Safari Messenger · Beta",
    heroSub: "Vive una nueva generación de mensajería. Conecta, comunica y comparte sin límites.",
    downloadCta: "Descargar Safari",
    exploreCta: "Explorar funciones",
    showcaseTitle: "Mira Safari en acción",
    showcaseSub: "Pantallas reales de la app: chats, conversaciones e historias.",
    shotChatList: "Tus chats, de un vistazo",
    shotChat: "Una conversación",
    shotStories: "Historias",
    shotCalls: "Llamadas",
    shotCommunities: "Comunidades",
    securityTitle: "Privacidad por diseño",
    securitySub: "Seguridad que trabaja en silencio en segundo plano, para que tú solo hables.",
    navDownload: "Descargar",
    navGames: "Juegos",
    privacyLink: "Política de privacidad",
    gamesTitle: "Juegos",
    gamesDesc:
      "Safari City: un juego de mundo abierto para Android. Explora una ciudad 3D a pie o en coche, completa misiones, escapa de la policía, navega en barco, vuela en hidroavión y bucea en busca de tesoros. Descargarlo y jugar es gratis, y funciona totalmente sin conexión.",
    gamesVersionLine: "Versión 0.15.0 · Android · 22,3 MB (23 431 772 bytes)",
    gamesDownloadButton: "Descargar Safari City para Android",
    gamesNote:
      "Safari City está en desarrollo activo: llegan nuevas versiones con novedades con regularidad.",
    gamesFeatures: [
      "Ciudad 3D abierta: a pie, en coche o en moto",
      "Misiones, carreras, taxi y rutas de autobús",
      "Persecuciones policiales, barco, moto acuática e hidroavión",
      "Buceo, pesca y tesoros hundidos",
    ],
    securityItems: [
      { title: "Cifrado de extremo a extremo", desc: "Tus chats privados están cerrados de forma que solo tú y la persona con la que hablas podéis leerlos." },
      { title: "Bloqueo de la app", desc: "Bloquea Safari con un PIN o tu huella. Tus chats permanecen ocultos hasta que seas tú." },
      { title: "Controles de privacidad", desc: "Elige quién ve tu foto y tu estado, bloquea a quien quieras y denuncia el spam con un toque." },
    ],
  },
  fr: {
    tagline: "Connectez-vous au-delà des limites",
    intro:
      "Découvrez Safari, une messagerie rapide et privée pour Android. Discutez avec vos amis en privé ou en grands groupes, suivez des chaînes, appelez vos amis, partagez des stories et envoyez photos, fichiers et messages vocaux — le tout protégé par le chiffrement de bout en bout. Safari consomme beaucoup moins de données que les autres messageries, il file donc même sur une connexion lente. Téléchargement et utilisation gratuits.",
    downloadTitle: "Télécharger Safari pour Android",
    versionLine:
      "Version 0.7.12 (bêta) · Android 6.0 ou plus récent · Le site détecte automatiquement votre téléphone — touchez simplement le bouton ci-dessous et la version adaptée se téléchargera.",
    warning:
      "Téléchargez Safari uniquement depuis ce site officiel. Les APK d'autres sites ne sont pas sûrs — ne les installez pas.",
    featuresTitle: "Fonctionnalités",
    features: [
      { title: "Messagerie privée", desc: "Des discussions en tête-à-tête protégées par un vrai chiffrement de bout en bout — seuls vous et votre ami pouvez les lire." },
      { title: "Conversations de groupe", desc: "De grands groupes avec administrateurs, messages épinglés, sondages et liens d’invitation. Tout le monde reste informé." },
      { title: "Appels vocaux", desc: "Des appels vocaux clairs qui fonctionnent même avec une connexion lente." },
      { title: "Appels vidéo", desc: "Des appels vidéo en face à face, en privé ou avec tout votre groupe." },
      { title: "Stories", desc: "Partagez vos photos et vos moments, qui disparaissent au bout de 24 heures." },
      { title: "Confidentialité et sécurité", desc: "Verrouillage de l’app, réglages de confidentialité, blocage et signalement — votre espace reste à vous." },
      { title: "Bots", desc: "Des bots qui vous aident à faire les choses, directement dans vos discussions." },
      { title: "Très faible consommation de données", desc: "Consomme beaucoup moins de données que les autres messageries — appels, photos et médias, tout est compressé et léger." },
    ],
    footer:
      "Téléchargement et utilisation gratuits. La bêta Android est disponible ci-dessus ; l'iPhone viendra plus tard.",
    aboutTitle: "À propos de Safari",
    languageLabel: "Langue",
    forYourPhone: "Safari 0.7.12 — pour votre téléphone",
    detected: "Détecté pour votre téléphone :",
    downloadButton: "Télécharger Safari pour Android",
    verify: "Vérifiez le téléchargement (SHA-256) :",
    category: "Communication",
    free: "Gratuit",
    updated: "Mis à jour",
    updatedValue: "19 novembre 2026",
    requires: "Nécessite",
    offeredBy: "Proposé par",
    screenshotsTitle: "Aperçu",
    slowTitle: "Conçu pour internet lent",
    slowDesc:
      "Les téléchargements sont petits et reprennent s'ils s'arrêtent. L'économiseur de données est activé par défaut : les appels vocaux n'utilisent que 12 kbit/s, les photos arrivent compressées et les médias ne se téléchargent que quand vous les touchez.",
    infoTitle: "Infos sur l'app",
    versionLabel: "Version",
    sizeLabel: "Taille",
    heroBadge: "Safari Messenger · Bêta",
    heroSub: "Découvrez une nouvelle génération de messagerie. Connectez-vous, communiquez et partagez sans limites.",
    downloadCta: "Télécharger Safari",
    exploreCta: "Explorer les fonctionnalités",
    showcaseTitle: "Voyez Safari en action",
    showcaseSub: "De vrais écrans de l’app — discussions, conversations et stories.",
    shotChatList: "Vos discussions, en un coup d’œil",
    shotChat: "Une conversation",
    shotStories: "Stories",
    shotCalls: "Calls",
    shotCommunities: "Communities",
    securityTitle: "La confidentialité par design",
    securitySub: "Une sécurité qui travaille silencieusement en arrière-plan — pour que vous n’ayez qu’à parler.",
    navDownload: "Télécharger",
    navGames: "Jeux",
    privacyLink: "Politique de confidentialité",
    gamesTitle: "Jeux",
    gamesDesc:
      "Safari City — un jeu en monde ouvert pour Android. Explorez une ville en 3D à pied ou en voiture, accomplissez des missions, échappez à la police, naviguez en bateau, volez en hydravion et plongez à la recherche de trésors. Téléchargement et jeu gratuits, et le jeu fonctionne entièrement hors ligne.",
    gamesVersionLine: "Version 0.15.0 · Android · 22,3 Mo (23 431 772 octets)",
    gamesDownloadButton: "Télécharger Safari City pour Android",
    gamesNote:
      "Safari City est en développement actif — de nouvelles versions avec de nouvelles fonctionnalités arrivent régulièrement.",
    gamesFeatures: [
      "Ville 3D ouverte — à pied, en voiture ou à moto",
      "Missions, courses, taxi et lignes de bus",
      "Poursuites policières, bateau, jet-ski et hydravion",
      "Plongée, pêche et trésors engloutis",
    ],
    securityItems: [
      { title: "Chiffrement de bout en bout", desc: "Vos discussions privées sont verrouillées de sorte que seuls vous et votre interlocuteur pouvez les lire." },
      { title: "Verrouillage de l’app", desc: "Verrouillez Safari avec un code PIN ou votre empreinte. Vos discussions restent cachées jusqu’à ce que ce soit vous." },
      { title: "Contrôles de confidentialité", desc: "Choisissez qui voit votre photo et votre statut, bloquez qui vous voulez et signalez le spam en un toucher." },
    ],
  },
  de: {
    tagline: "Verbinde dich über Grenzen hinaus",
    intro:
      "Lerne Safari kennen — ein schneller, privater Messenger für Android. Chatte mit Freunden privat oder in großen Gruppen, folge Kanälen, rufe deine Freunde an, teile Stories und sende Fotos, Dateien und Sprachnachrichten — alles geschützt durch Ende-zu-Ende-Verschlüsselung. Safari verbraucht viel weniger Daten als andere Messenger und fliegt deshalb auch bei langsamem Internet. Download und Nutzung sind kostenlos.",
    downloadTitle: "Safari für Android herunterladen",
    versionLine:
      "Version 0.7.12 (Beta) · Android 6.0 oder neuer · Die Website erkennt dein Telefon automatisch — tippe einfach auf die Schaltfläche unten, und die passende Version wird heruntergeladen.",
    warning:
      "Lade Safari nur von dieser offiziellen Website herunter. APKs von anderen Websites sind nicht sicher — installiere sie nicht.",
    featuresTitle: "Funktionen",
    features: [
      { title: "Private Nachrichten", desc: "Einzelchats mit echter Ende-zu-Ende-Verschlüsselung — nur du und dein Freund können sie lesen." },
      { title: "Gruppengespräche", desc: "Große Gruppen mit Admins, angehefteten Nachrichten, Umfragen und Einladungslinks. Alle bleiben auf dem Laufenden." },
      { title: "Sprachanrufe", desc: "Klare Sprachanrufe, die auch bei langsamer Verbindung funktionieren." },
      { title: "Videoanrufe", desc: "Videoanrufe von Angesicht zu Angesicht — privat oder mit deiner ganzen Gruppe." },
      { title: "Stories", desc: "Teile Fotos und Momente, die nach 24 Stunden verschwinden." },
      { title: "Privatsphäre und Sicherheit", desc: "App-Sperre, Datenschutzeinstellungen, Blockieren und Melden — dein Bereich bleibt deiner." },
      { title: "Bots", desc: "Bots, die dir helfen, Dinge direkt in deinen Chats zu erledigen." },
      { title: "Sehr geringer Datenverbrauch", desc: "Verbraucht viel weniger Daten als andere Messenger — Anrufe, Fotos und Medien, alles komprimiert und klein." },
    ],
    footer:
      "Download und Nutzung sind kostenlos. Die Android-Beta ist oben verfügbar; das iPhone kommt später.",
    aboutTitle: "Über Safari",
    languageLabel: "Sprache",
    forYourPhone: "Safari 0.7.12 — für dein Telefon",
    detected: "Für dein Telefon erkannt:",
    downloadButton: "Safari für Android herunterladen",
    verify: "Download überprüfen (SHA-256):",
    category: "Kommunikation",
    free: "Kostenlos",
    updated: "Aktualisiert",
    updatedValue: "19. November 2026",
    requires: "Erfordert",
    offeredBy: "Angeboten vom",
    screenshotsTitle: "Vorschau",
    slowTitle: "Für langsames Internet gemacht",
    slowDesc:
      "Downloads sind klein und werden fortgesetzt, wenn sie abbrechen. Der Datensparmodus ist standardmäßig an: Sprachanrufe nutzen nur 12 kbit/s, Fotos kommen komprimiert an, und Medien werden nur heruntergeladen, wenn du darauf tippst.",
    infoTitle: "App-Info",
    versionLabel: "Version",
    sizeLabel: "Größe",
    heroBadge: "Safari Messenger · Beta",
    heroSub: "Erlebe eine neue Generation des Messaging. Verbinde dich, kommuniziere und teile ohne Grenzen.",
    downloadCta: "Safari herunterladen",
    exploreCta: "Funktionen entdecken",
    showcaseTitle: "Safari in Aktion sehen",
    showcaseSub: "Echte Bildschirme aus der App — Chats, Gespräche und Stories.",
    shotChatList: "Deine Chats auf einen Blick",
    shotChat: "Ein Gespräch",
    shotStories: "Stories",
    shotCalls: "Calls",
    shotCommunities: "Communities",
    securityTitle: "Privatsphäre von Anfang an",
    securitySub: "Sicherheit, die leise im Hintergrund arbeitet — damit du einfach nur reden kannst.",
    navDownload: "Herunterladen",
    navGames: "Spiele",
    privacyLink: "Datenschutzerklärung",
    gamesTitle: "Spiele",
    gamesDesc:
      "Safari City — ein Open-World-Spiel für Android. Erkunde eine 3D-Stadt zu Fuß oder mit dem Auto, erfülle Missionen, entkomme der Polizei, fahre Boot, fliege mit dem Wasserflugzeug und tauche nach Schätzen. Download und Spielen sind kostenlos, und das Spiel funktioniert komplett offline.",
    gamesVersionLine: "Version 0.15.0 · Android · 22,3 MB (23.431.772 Bytes)",
    gamesDownloadButton: "Safari City für Android herunterladen",
    gamesNote:
      "Safari City wird aktiv weiterentwickelt — neue Versionen mit neuen Funktionen erscheinen regelmäßig.",
    gamesFeatures: [
      "Offene 3D-Stadt — zu Fuß, mit dem Auto oder Motorrad",
      "Missionen, Rennen, Taxi- und Busjobs",
      "Polizeiverfolgungen, Boot, Jet-Ski und Wasserflugzeug",
      "Tauchen, Angeln und versunkene Schätze",
    ],
    securityItems: [
      { title: "Ende-zu-Ende-Verschlüsselung", desc: "Deine privaten Chats sind so gesperrt, dass nur du und dein Gesprächspartner sie lesen können." },
      { title: "App-Sperre", desc: "Sperre Safari mit einer PIN oder deinem Fingerabdruck. Deine Chats bleiben verborgen, bis du es bist." },
      { title: "Datenschutzkontrollen", desc: "Wähle, wer dein Foto und deinen Status sieht, blockiere jeden und melde Spam mit einem Tipp." },
    ],
  },
  hi: {
    tagline: "सीमाओं से परे जुड़ें",
    intro:
      "Safari से मिलिए — Android के लिए एक तेज़ और निजी मैसेंजर। दोस्तों से अकेले या बड़े ग्रुप में चैट करें, चैनल फॉलो करें, दोस्तों को कॉल करें, स्टोरीज़ शेयर करें और फ़ोटो, फ़ाइलें व वॉइस नोट्स भेजें — सब कुछ एंड-टू-एंड एन्क्रिप्शन से सुरक्षित। Safari दूसरे मैसेंजरों से बहुत कम डेटा इस्तेमाल करता है, इसलिए धीमे इंटरनेट पर भी तेज़ चलता है। डाउनलोड और इस्तेमाल मुफ़्त है।",
    downloadTitle: "Android के लिए Safari डाउनलोड करें",
    versionLine:
      "संस्करण 0.7.12 (बीटा) · Android 6.0 या उससे नया · साइट आपके फ़ोन को अपने आप पहचानती है — बस नीचे वाला बटन दबाएँ और आपके फ़ोन के लिए सही संस्करण डाउनलोड हो जाएगा।",
    warning:
      "Safari सिर्फ़ इसी आधिकारिक वेबसाइट से डाउनलोड करें। अन्य साइटों की APK सुरक्षित नहीं हैं — उन्हें इंस्टॉल न करें।",
    featuresTitle: "सुविधाएँ",
    features: [
      { title: "निजी मैसेजिंग", desc: "असली एंड-टू-एंड एन्क्रिप्शन से सुरक्षित एक-से-एक चैट — सिर्फ़ आप और आपका दोस्त ही उन्हें पढ़ सकते हैं।" },
      { title: "ग्रुप बातचीत", desc: "एडमिन, पिन, पोल और इन्वाइट लिंक के साथ बड़े ग्रुप। सभी अपडेट रहते हैं।" },
      { title: "वॉइस कॉल", desc: "साफ़ वॉइस कॉल जो धीमे कनेक्शन पर भी काम करती हैं।" },
      { title: "वीडियो कॉल", desc: "आमने-सामने वीडियो कॉल — अकेले या पूरे ग्रुप के साथ।" },
      { title: "स्टोरीज़", desc: "अपनी फ़ोटो और पल शेयर करें जो 24 घंटे बाद गायब हो जाते हैं।" },
      { title: "प्राइवेसी और सुरक्षा", desc: "ऐप लॉक, प्राइवेसी सेटिंग्स, ब्लॉक और रिपोर्ट — आपकी जगह आपकी ही रहती है।" },
      { title: "बॉट्स", desc: "ऐसे बॉट्स जो आपके काम आपकी चैट के अंदर ही कर दें।" },
      { title: "बहुत कम डेटा उपयोग", desc: "दूसरे मैसेंजरों से बहुत कम डेटा इस्तेमाल करता है — कॉल, फ़ोटो और मीडिया, सब छोटा और कंप्रेस्ड।" },
    ],
    footer:
      "डाउनलोड और इस्तेमाल मुफ़्त है। Android बीटा ऊपर उपलब्ध है; iPhone बाद में आएगा।",
    aboutTitle: "Safari के बारे में",
    languageLabel: "भाषा",
    forYourPhone: "Safari 0.7.12 — आपके फ़ोन के लिए",
    detected: "आपके फ़ोन के लिए पहचाना गया:",
    downloadButton: "Android के लिए Safari डाउनलोड करें",
    verify: "डाउनलोड सत्यापित करें (SHA-256):",
    category: "संचार",
    free: "निःशुल्क",
    updated: "अपडेट किया गया",
    updatedValue: "19 नवंबर 2026",
    requires: "आवश्यक",
    offeredBy: "प्रस्तुतकर्ता",
    screenshotsTitle: "पूर्वावलोकन",
    slowTitle: "धीमे इंटरनेट के लिए बनाया गया",
    slowDesc:
      "डाउनलोड छोटे हैं और रुक जाएँ तो फिर से शुरू होते हैं। डेटा सेवर डिफ़ॉल्ट रूप से चालू है — वॉइस कॉल सिर्फ़ 12 kbps इस्तेमाल करती हैं, फ़ोटो कंप्रेस होकर आती हैं, और मीडिया सिर्फ़ तब डाउनलोड होता है जब आप उसे टैप करें।",
    infoTitle: "ऐप की जानकारी",
    versionLabel: "संस्करण",
    sizeLabel: "आकार",
    heroBadge: "Safari Messenger · बीटा",
    heroSub: "मैसेजिंग की नई पीढ़ी का अनुभव करें। बिना किसी सीमा के जुड़ें, बात करें और शेयर करें।",
    downloadCta: "Safari डाउनलोड करें",
    exploreCta: "सुविधाएँ देखें",
    showcaseTitle: "Safari को काम करते देखें",
    showcaseSub: "ऐप की असली स्क्रीनें — चैट, बातचीत और स्टोरीज़।",
    shotChatList: "आपकी चैट, एक नज़र में",
    shotChat: "एक बातचीत",
    shotStories: "स्टोरीज़",
    shotCalls: "कॉल",
    shotCommunities: "समुदाय",
    securityTitle: "डिज़ाइन से ही निजी",
    securitySub: "सुरक्षा जो पृष्ठभूमि में चुपचाप काम करती है — ताकि आप बस बात करें।",
    navDownload: "डाउनलोड",
    navGames: "गेम्स",
    privacyLink: "गोपनीयता नीति",
    gamesTitle: "गेम्स",
    gamesDesc:
      "Safari City — Android के लिए एक ओपन-वर्ल्ड गेम। 3D शहर में पैदल या गाड़ी से घूमें, मिशन पूरे करें, पुलिस से बचें, नाव चलाएँ, सीप्लेन उड़ाएँ और खज़ाने के लिए गोता लगाएँ। डाउनलोड और खेलना मुफ़्त है, और यह पूरी तरह ऑफ़लाइन काम करता है।",
    gamesVersionLine: "संस्करण 0.15.0 · Android · 22.3 MB (23,431,772 बाइट्स)",
    gamesDownloadButton: "Android के लिए Safari City डाउनलोड करें",
    gamesNote:
      "Safari City लगातार विकसित हो रहा है — नई सुविधाओं के साथ नए संस्करण नियमित रूप से आते रहते हैं।",
    gamesFeatures: [
      "खुला 3D शहर — पैदल, गाड़ी या मोटरबाइक से",
      "मिशन, रेस, टैक्सी और बस की नौकरियाँ",
      "पुलिस का पीछा, नाव, जेट-स्की और सीप्लेन",
      "गोताखोरी, मछली पकड़ना और डूबा हुआ खज़ाना",
    ],
    securityItems: [
      { title: "एंड-टू-एंड एन्क्रिप्शन", desc: "आपकी निजी चैट इस तरह बंद हैं कि सिर्फ़ आप और जिससे आप बात कर रहे हैं वही उन्हें पढ़ सकते हैं।" },
      { title: "ऐप लॉक", desc: "Safari को पिन या अपने फ़िंगरप्रिंट से लॉक करें। आपकी चैट तब तक छिपी रहती हैं जब तक आप न हों।" },
      { title: "प्राइवेसी कंट्रोल", desc: "चुनें कि आपकी फ़ोटो और स्टेटस कौन देखे, किसी को भी ब्लॉक करें, और स्पैम को एक टैप से रिपोर्ट करें।" },
    ],
  },
  ps: {
    tagline: "له حدودو هاخوا وصل شئ",
    intro:
      "له Safari سره وپېژنئ — د Android لپاره یو چټک او شخصي مسینجر. له ملګرو سره یو په یو یا په لویو ډلو کې چیټ وکړئ، چینلونه تعقیب کړئ، ملګرو ته زنګ ووهئ، کیسې شریکې کړئ او انځورونه، فایلونه او غږیز یادښتونه ولېږئ — هر څه د له پایه تر پایه کوډ کولو سره خوندي دي. Safari له نورو مسینجرونو څخه ډېر کم ډاټا کاروي، نو په کم سرعت انټرنېټ کې هم چټک کار کوي. ډاونلوډ او کارول یې وړیا دي.",
    downloadTitle: "د Android لپاره Safari ډاونلوډ کړئ",
    versionLine:
      "نسخه 0.7.12 (بیټا) · Android 6.0 یا نوی · سایټ ستاسو تلیفون په اوتومات ډول پېژني — یوازې لاندې تڼۍ کېکاږئ او ستاسو تلیفون لپاره سمه نسخه به ډاونلوډ شي.",
    warning:
      "Safari یوازې له دې رسمي ویب‌سایټ څخه ډاونلوډ کړئ. له نورو سایټونو څخه APK خوندي نه دي — مه یې نصب کوئ.",
    featuresTitle: "ځانګړتیاوې",
    features: [
      { title: "شخصي مسینج کول", desc: "یو په یو چیټونه، د رښتیني له پایه تر پایه کوډ کولو سره خوندي — یوازې تاسو او ستاسو ملګری یې لوستلی شئ." },
      { title: "ډله‌ییزې خبرې", desc: "د مدیر، پین، رایې او بلنې لینکونو سره لویې ډلې. هر څوک خبر پاتې کېږي." },
      { title: "غږیز زنګونه", desc: "روښانه غږیز زنګونه چې حتی په کم سرعت اړیکه کې کار کوي." },
      { title: "ویډیويي زنګونه", desc: "مخامخ ویډیويي زنګونه، یو په یو یا له خپلې ټولې ډلې سره." },
      { title: "کیسې", desc: "خپل انځورونه او شېبې شریکې کړئ چې له 24 ساعتونو وروسته ورکېږي." },
      { title: "محرمیت او امنیت", desc: "د اپ قلف، د محرمیت ترتیبات، بلاک او راپور — ستاسو ځای ستاسو پاتې کېږي." },
      { title: "بوټان", desc: "هغه بوټان چې ستاسو کارونه ستاسو د چیټونو په دننه کې کوي." },
      { title: "ډېر ټیټ ډاټا کارول", desc: "له نورو مسینجرونو څخه ډېر کم ډاټا کاروي — زنګونه، انځورونه او رسنۍ، ټول کمپریس شوي او کوچني." },
    ],
    footer:
      "ډاونلوډ او کارول وړیا دي. د Android بیټا پورته شتون لري؛ iPhone وروسته راځي.",
    aboutTitle: "د Safari په اړه",
    languageLabel: "ژبه",
    forYourPhone: "Safari 0.7.12 — ستاسو تلیفون لپاره",
    detected: "ستاسو تلیفون لپاره وپېژندل شو:",
    downloadButton: "د Android لپاره Safari ډاونلوډ کړئ",
    verify: "ډاونلوډ تایید کړئ (SHA-256):",
    category: "اړیکې",
    free: "وړیا",
    updated: "تازه شوی",
    updatedValue: "نومبر ۱۹، ۲۰۲۶",
    requires: "ته اړتیا لري",
    offeredBy: "وړاندې شوی",
    screenshotsTitle: "مخکتنه",
    slowTitle: "د کم سرعت انټرنېټ لپاره جوړ شوی",
    slowDesc:
      "ډاونلوډونه کوچني دي او که ودرېږي، دوام مومي. ډاټا سیور په ډیفالټ فعال دی — غږیز زنګونه یوازې 12 kbps کاروي، انځورونه کمپریس شوي راځي، او رسنۍ یوازې هغه وخت ډاونلوډ کېږي کله چې تاسو یې ټچ کړئ.",
    infoTitle: "د اپ معلومات",
    versionLabel: "نسخه",
    sizeLabel: "سایز",
    heroBadge: "Safari Messenger · بیټا",
    heroSub: "د مسینج کولو نو نسل تجربه کړئ. له حدودو پرته وصل شئ، اړیکه ونیسئ او شریک کړئ.",
    downloadCta: "Safari ډاونلوډ کړئ",
    exploreCta: "ځانګړتیاوې وګورئ",
    showcaseTitle: "Safari په عمل کې وګورئ",
    showcaseSub: "له اپ څخه رښتیني سکرینونه — چیټونه، خبرې او کیسې.",
    shotChatList: "ستاسو چیټونه، په یوه کتنه کې",
    shotChat: "یوه خبره",
    shotStories: "کیسې",
    shotCalls: "اړیکې",
    shotCommunities: "ټولنې",
    securityTitle: "له ډیزاین څخه شخصي",
    securitySub: "امنیت چې په شالید کې په خاموشۍ کار کوي — ترڅو تاسو یوازې خبرې وکړئ.",
    navDownload: "ډاونلوډ",
    navGames: "لوبې",
    privacyLink: "د محرمیت تګلاره",
    gamesTitle: "لوبې",
    gamesDesc:
      "Safari City — د Android لپاره د خلاصې نړۍ لوبه. په 3D ښار کې په پښو یا په موټر کې وګرځئ، ماموریتونه بشپړ کړئ، له پولیسو څخه وتښتئ، په کښتۍ کې سفر وکړئ، په سمندري الوتکه کې والوزئ او د خزانې لپاره غوټه ووهئ. ډاونلوډ او لوبېدل یې وړیا دي، او دا په بشپړ ډول آف‌لاین کار کوي.",
    gamesVersionLine: "نسخه 0.15.0 · Android · 22.3 MB (23,431,772 بایټ)",
    gamesDownloadButton: "د Android لپاره Safari City ډاونلوډ کړئ",
    gamesNote:
      "Safari City په فعاله توګه پراختیا مومي — د نویو ځانګړتیاوو سره نوې نسخې په منظمه توګه راځي.",
    gamesFeatures: [
      "خلاص 3D ښار — په پښو، په موټر یا موټرسایکل",
      "ماموریتونه، سیالۍ، ټکسي او بس دندې",
      "د پولیسو تعقیب، کښتۍ، جیټ سکي او سمندري الوتکه",
      "غوټه وهل، کب نیول او ډوب شوی خزانه",
    ],
    securityItems: [
      { title: "له پایه تر پایه کوډ کول", desc: "ستاسو شخصي چیټونه داسې تړل شوي چې یوازې تاسو او هغه څوک چې ورسره خبرې کوئ، لوستلی شئ." },
      { title: "د اپ قلف", desc: "Safari په PIN یا خپل ګوتې نښې سره قلف کړئ. ستاسو چیټونه تر هغه پټ پاتې کېږي تر څو چې تاسو نه یاست." },
      { title: "د محرمیت کنټرولونه", desc: "وټاکئ چې ستاسو انځور او حالت څوک ویني، هر څوک بلاک کړئ، او سپیم په یوه ټچ سره راپور کړئ." },
    ],
  },
};
