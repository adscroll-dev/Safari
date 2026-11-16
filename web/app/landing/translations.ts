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
  requires: string;
  offeredBy: string;
  screenshotsTitle: string;
  slowTitle: string;
  slowDesc: string;
  infoTitle: string;
  versionLabel: string;
  sizeLabel: string;
};

export const translations: Record<LangCode, Strings> = {
  en: {
    tagline: "Connect Beyond Limits",
    intro:
      "Meet Safari — a fast, private messenger for Android. Chat one-to-one or in groups, follow channels, call your friends, share stories, and send photos, files and voice notes — all protected by end-to-end encryption. Safari sips data instead of gulping it, so it flies even on slow internet. Free to download and use.",
    downloadTitle: "Download Safari for Android",
    versionLine:
      "Version 0.7.1 (Beta) · Android 6.0 and up · Your phone is detected automatically — just tap the button below and the right version downloads for you.",
    warning:
      "Only download Safari from this official website. APKs from other sites are not safe — don't install them.",
    featuresTitle: "Features",
    features: [
      { title: "Private & group chat", desc: "Chat with friends one-to-one or in big groups, with admins, pins and more." },
      { title: "Real-time chat", desc: "Messages arrive instantly, with read ticks and typing indicators." },
      { title: "Voice & video calls", desc: "Clear voice and video calls — even on a slow connection." },
      { title: "Photos, files & voice", desc: "Send photos, videos, files up to 50 MB, and voice messages." },
      { title: "24-hour stories", desc: "Share photos and moments that disappear after a day." },
      { title: "End-to-end encryption", desc: "Your private chats are locked with real end-to-end encryption — only you and your friend can read them." },
      { title: "Bots", desc: "Bots that help you get things done, right inside your chats." },
      { title: "English & Persian", desc: "English by default, with Persian and more languages in Settings." },
      { title: "Ultra-low data use", desc: "Uses far less data than other messengers — calls, photos and media, all squeezed small." },
      { title: "Local AI assistant", desc: "A smart helper on your phone — quick replies and translations, with nothing sent to the cloud." },
    ],
    footer:
      "Free to download and use. The Android beta is available above; iPhone is coming later.",
    aboutTitle: "About Safari",
    languageLabel: "Language",
    forYourPhone: "Safari 0.7.1 — for your phone",
    detected: "Detected for your phone:",
    downloadButton: "Download Safari for Android",
    verify: "Verify the download (SHA-256):",
    category: "Communication",
    free: "Free",
    updated: "Updated Nov 16, 2026",
    requires: "Requires Android 6.0+",
    offeredBy: "Offered by Safari Team",
    screenshotsTitle: "Preview",
    slowTitle: "Built for slow internet",
    slowDesc:
      "Downloads are small and resume if they stop. Data Saver is on by default — voice calls use just 12 kbps, photos arrive compressed, and media downloads only when you tap it.",
    infoTitle: "App info",
    versionLabel: "Version",
    sizeLabel: "Size",
  },
  fa: {
    tagline: "اتصال فراتر از محدودیت‌ها",
    intro:
      "با Safari آشنا شوید — پیام‌رسانی سریع و امن برای اندروید. خصوصی یا گروهی چت کنید، کانال‌ها را دنبال کنید، با دوستانتان تماس بگیرید، استوری بگذارید و عکس، فایل و پیام صوتی بفرستید — همه با رمزنگاری سرتاسری. Safari خیلی کم اینترنت مصرف می‌کند، پس حتی با اینترنت کند هم سریع کار می‌کند. دانلود و استفاده‌اش رایگان است.",
    downloadTitle: "دانلود Safari برای اندروید",
    versionLine:
      "نسخه ۰٫۷٫۱ (بتا) · اندروید ۶٫۰ به بالا · گوشی شما به‌صورت خودکار تشخیص داده می‌شود — فقط دکمه زیر را بزنید و نسخه مناسب گوشی شما دانلود می‌شود.",
    warning:
      "Safari را فقط از همین وب‌سایت رسمی دانلود کنید. فایل‌های APK از سایت‌های دیگر امن نیستند — نصبشان نکنید.",
    featuresTitle: "قابلیت‌ها",
    features: [
      { title: "چت خصوصی و گروه", desc: "با دوستانتان خصوصی یا در گروه‌های بزرگ چت کنید — با مدیر، سنجاق و امکانات بیشتر." },
      { title: "چت آنی", desc: "پیام‌ها فوراً می‌رسند، با تیک خوانده‌شدن و «در حال تایپ…»." },
      { title: "تماس صوتی و تصویری", desc: "تماس صوتی و تصویری شفاف — حتی با اینترنت کند." },
      { title: "عکس، فایل و پیام صوتی", desc: "عکس، ویدیو، فایل تا ۵۰ مگابایت و پیام صوتی بفرستید." },
      { title: "استوری ۲۴ ساعته", desc: "عکس و لحظه‌هایتان را به اشتراک بگذارید؛ بعد از یک روز ناپدید می‌شوند." },
      { title: "رمزنگاری سرتاسری", desc: "چت‌های خصوصی شما با رمزنگاری سرتاسری واقعی قفل شده‌اند — فقط شما و دوستتان می‌توانید آن‌ها را بخوانید." },
      { title: "ربات‌ها", desc: "ربات‌هایی که کارها را درست داخل چت‌هایتان انجام می‌دهند." },
      { title: "انگلیسی و فارسی", desc: "انگلیسی پیش‌فرض است؛ فارسی و زبان‌های دیگر در تنظیمات هستند." },
      { title: "مصرف بسیار کم اینترنت", desc: "خیلی کمتر از پیام‌رسان‌های دیگر اینترنت مصرف می‌کند — تماس، عکس و رسانه، همه فشرده و کوچک." },
      { title: "دستیار هوش مصنوعی محلی", desc: "دستیار هوشمند روی گوشی شما — پاسخ سریع و ترجمه، بدون اینکه چیزی به فضای ابری فرستاده شود." },
    ],
    footer:
      "دانلود و استفاده رایگان است. نسخه بتای اندروید در بالا در دسترس است؛ آیفون بعداً می‌آید.",
    aboutTitle: "درباره Safari",
    languageLabel: "زبان",
    forYourPhone: "Safari ۰٫۷٫۱ — برای گوشی شما",
    detected: "برای گوشی شما تشخیص داده شد:",
    downloadButton: "دانلود Safari برای اندروید",
    verify: "بررسی دانلود (SHA-256):",
    category: "ارتباطات",
    free: "رایگان",
    updated: "به‌روز شده ۱۶ نوامبر ۲۰۲۶",
    requires: "نیاز به اندروید ۶٫۰ به بالا",
    offeredBy: "ارائه‌دهنده: تیم Safari",
    screenshotsTitle: "پیش‌نمایش",
    slowTitle: "ساخته‌شده برای انترنت کم‌سرعت",
    slowDesc:
      "دانلودها کم‌حجم‌اند و اگر قطع شوند ادامه می‌یابند. صرفه‌جویی در مصرف داده به‌صورت پیش‌فرض روشن است — تماس صوتی فقط ۱۲ کیلوبیت بر ثانیه مصرف می‌کند، عکس‌ها فشرده می‌رسند و رسانه فقط وقتی دانلود می‌شود که شما لمس کنید.",
    infoTitle: "اطلاعات برنامه",
    versionLabel: "نسخه",
    sizeLabel: "حجم",
  },
  ar: {
    tagline: "تواصل بلا حدود",
    intro:
      "تعرّف على Safari — مراسل سريع وخاص لأندرويد. تحدّث مع أصدقائك فرديًا أو في مجموعات، وتابع القنوات، واتصل بأصدقائك، وشارك القصص، وأرسل الصور والملفات والرسائل الصوتية — كل ذلك محمي بالتشفير الكامل. يستهلك Safari بيانات أقل بكثير من غيره، فيعمل بسرعة حتى مع الإنترنت البطيء. تنزيله واستخدامه مجاني.",
    downloadTitle: "تنزيل Safari لأندرويد",
    versionLine:
      "الإصدار 0.7.1 (تجريبي) · يتطلب أندرويد 6.0 أو أحدث · يكتشف الموقع هاتفك تلقائيًا — فقط اضغط الزر أدناه وستُنزَّل النسخة المناسبة لهاتفك.",
    warning:
      "حمّل Safari من هذا الموقع الرسمي فقط. ملفات APK من مواقع أخرى غير آمنة — لا تثبّتها.",
    featuresTitle: "الميزات",
    features: [
      { title: "دردشة خاصة وجماعية", desc: "تحدّث مع أصدقائك فرديًا أو في مجموعات كبيرة، مع مشرفين وتثبيت والمزيد." },
      { title: "دردشة فورية", desc: "تصل الرسائل فورًا، مع علامات القراءة ومؤشر الكتابة." },
      { title: "مكالمات صوتية ومرئية", desc: "مكالمات صوتية ومرئية واضحة — حتى مع اتصال بطيء." },
      { title: "صور وملفات وصوت", desc: "أرسل الصور والفيديو والملفات حتى 50 ميغابايت، والرسائل الصوتية." },
      { title: "قصص 24 ساعة", desc: "شارك صورك ولحظاتك التي تختفي بعد يوم." },
      { title: "تشفير كامل", desc: "دردشاتك الخاصة مقفلة بالتشفير الكامل الحقيقي — أنت وصديقك فقط من يستطيع قراءتها." },
      { title: "بوتات", desc: "بوتات تساعدك على إنجاز الأمور داخل دردشاتك." },
      { title: "لغات متعددة", desc: "الإنجليزية هي الافتراضية، مع العربية والفارسية ولغات أخرى في الإعدادات." },
      { title: "استهلاك بيانات منخفض جدًا", desc: "يستهلك بيانات أقل بكثير من المراسلات الأخرى — المكالمات والصور والوسائط، كلها مضغوطة وصغيرة." },
      { title: "مساعد ذكاء اصطناعي محلي", desc: "مساعد ذكي على هاتفك — ردود سريعة وترجمة، دون إرسال أي شيء إلى السحابة." },
    ],
    footer:
      "التنزيل والاستخدام مجاني. نسخة أندرويد التجريبية متاحة أعلاه؛ ونسخة iOS قادمة لاحقًا.",
    aboutTitle: "عن Safari",
    languageLabel: "اللغة",
    forYourPhone: "Safari 0.7.1 — لهاتفك",
    detected: "تم الكشف لهاتفك:",
    downloadButton: "تنزيل Safari لأندرويد",
    verify: "تحقق من التنزيل (SHA-256):",
    category: "الاتصالات",
    free: "مجاني",
    updated: "تم التحديث في 16 نوفمبر 2026",
    requires: "يتطلب Android 6.0+",
    offeredBy: "مقدَّم من فريق Safari",
    screenshotsTitle: "معاينة",
    slowTitle: "مصمَّم لإنترنت بطيء",
    slowDesc:
      "التنزيلات صغيرة وتستأنف إذا توقفت. توفير البيانات مفعَّل افتراضيًا — المكالمات الصوتية تستهلك 12 كيلوبت/ث فقط، والصور تصل مضغوطة، والوسائط تُنزَّل فقط عندما تنقر عليها.",
    infoTitle: "معلومات التطبيق",
    versionLabel: "الإصدار",
    sizeLabel: "الحجم",
  },
  ur: {
    tagline: "حدود سے آگے جڑیں",
    intro:
      "Safari سے ملیں — اینڈرائیڈ کے لیے ایک تیز اور نجی میسنجر۔ دوستوں سے اکیلے یا بڑے گروپس میں چیٹ کریں، چینلز فالو کریں، دوستوں کو کال کریں، اسٹوریز شیئر کریں، اور تصاویر، فائلیں اور وائس نوٹس بھیجیں — سب مکمل خفیہ کاری کے ساتھ۔ Safari دوسرے میسنجرز کے مقابلے میں بہت کم ڈیٹا استعمال کرتا ہے، اس لیے سست انٹرنیٹ پر بھی تیز چلتا ہے۔ ڈاؤن لوڈ اور استعمال مفت ہے۔",
    downloadTitle: "اینڈرائیڈ کے لیے Safari ڈاؤن لوڈ کریں",
    versionLine:
      "ورژن 0.7.1 (بیٹا) · اینڈرائیڈ 6.0 یا اس سے اوپر · سائٹ آپ کے فون کو خود بخود پہچانتی ہے — بس نیچے والا بٹن دبائیں اور آپ کے فون کے لیے صحیح ورژن ڈاؤن لوڈ ہو جائے گا۔",
    warning:
      "Safari صرف اسی سرکاری ویب سائٹ سے ڈاؤن لوڈ کریں۔ دوسری سائٹس کی APK فائلیں محفوظ نہیں ہیں — انہیں انسٹال نہ کریں۔",
    featuresTitle: "خصوصیات",
    features: [
      { title: "نجی اور گروپ چیٹ", desc: "دوستوں سے اکیلے یا بڑے گروپس میں چیٹ کریں — ایڈمن، پن اور مزید سہولتوں کے ساتھ۔" },
      { title: "فوری چیٹ", desc: "پیغامات فوراً پہنچتے ہیں، پڑھنے کے نشان اور ٹائپنگ انڈیکیٹر کے ساتھ۔" },
      { title: "وائس اور ویڈیو کالز", desc: "صاف وائس اور ویڈیو کالز — سست کنکشن پر بھی۔" },
      { title: "تصاویر، فائلز اور وائس", desc: "تصاویر، ویڈیوز، 50 MB تک فائلیں، اور وائس پیغامات بھیجیں۔" },
      { title: "24 گھنٹے کی اسٹوریز", desc: "اپنی تصاویر اور لمحے شیئر کریں جو ایک دن بعد غائب ہو جاتے ہیں۔" },
      { title: "مکمل خفیہ کاری", desc: "آپ کی نجی چیٹس حقیقی مکمل خفیہ کاری سے بند ہیں — صرف آپ اور آپ کا دوست انہیں پڑھ سکتے ہیں۔" },
      { title: "بوٹس", desc: "ایسے بوٹس جو آپ کے کام آپ کی چیٹس کے اندر ہی کر دیں۔" },
      { title: "زبانیں", desc: "انگریزی بطور پیش فرض؛ اردو اور دیگر زبانیں سیٹنگز میں موجود ہیں۔" },
      { title: "بہت کم ڈیٹا استعمال", desc: "دوسرے میسنجرز کے مقابلے میں بہت کم ڈیٹا استعمال کرتا ہے — کالز، تصاویر اور میڈیا، سب چھوٹا اور کمپریسڈ۔" },
      { title: "مقامی AI اسسٹنٹ", desc: "آپ کے فون پر ایک ذہین مددگار — فوری جوابات اور ترجمہ، بغیر کچھ کلاؤڈ کو بھیجے۔" },
    ],
    footer:
      "ڈاؤن لوڈ اور استعمال مفت ہے۔ اینڈرائیڈ بیٹا اوپر دستیاب ہے؛ آئی فون بعد میں آئے گا۔",
    aboutTitle: "Safari کے بارے میں",
    languageLabel: "زبان",
    forYourPhone: "Safari 0.7.1 — آپ کے فون کے لیے",
    detected: "آپ کے فون کے لیے پہچانا گیا:",
    downloadButton: "اینڈرائیڈ کے لیے Safari ڈاؤن لوڈ کریں",
    verify: "ڈاؤن لوڈ کی تصدیق کریں (SHA-256):",
    category: "مواصلات",
    free: "مفت",
    updated: "اپ ڈیٹ: 16 نومبر 2026",
    requires: "Android 6.0+ درکار ہے",
    offeredBy: "Safari ٹیم کی پیشکش",
    screenshotsTitle: "جھلک",
    slowTitle: "سست انٹرنیٹ کے لیے بنایا گیا",
    slowDesc:
      "ڈاؤن لوڈ چھوٹے ہیں اور رک جائیں تو جاری رہتے ہیں۔ ڈیٹا سیور بطور پیش فرض آن ہے — وائس کالز صرف 12 kbps استعمال کرتی ہیں، تصاویر کمپریس ہو کر آتی ہیں، اور میڈیا صرف تب ڈاؤن لوڈ ہوتا ہے جب آپ ٹیپ کریں۔",
    infoTitle: "ایپ کی معلومات",
    versionLabel: "ورژن",
    sizeLabel: "سائز",
  },
  tr: {
    tagline: "Sınırların Ötesinde Bağlanın",
    intro:
      "Safari ile tanışın — Android için hızlı ve özel bir mesajlaşma uygulaması. Arkadaşlarınızla bire bir veya büyük gruplarda sohbet edin, kanalları takip edin, arkadaşlarınızı arayın, hikâyeler paylaşın, fotoğraf, dosya ve sesli notlar gönderin — hepsi uçtan uca şifreleme ile korunuyor. Safari diğer mesajlaşma uygulamalarından çok daha az veri kullanır, bu yüzden yavaş internette bile uçar. İndirmesi ve kullanımı ücretsizdir.",
    downloadTitle: "Android için Safari'yu indirin",
    versionLine:
      "Sürüm 0.7.1 (Beta) · Android 6.0 ve üzeri · Site telefonunuzu otomatik olarak algılar — aşağıdaki düğmeye dokunmanız yeterli, telefonunuza uygun sürüm indirilir.",
    warning:
      "Safari'yu yalnızca bu resmî web sitesinden indirin. Diğer sitelerdeki APK'lar güvenli değildir — yüklemeyin.",
    featuresTitle: "Özellikler",
    features: [
      { title: "Özel ve grup sohbeti", desc: "Arkadaşlarınızla bire bir veya büyük gruplarda sohbet edin; yönetici, sabitleme ve daha fazlası var." },
      { title: "Gerçek zamanlı sohbet", desc: "Mesajlar anında ulaşır; okundu bilgisi ve yazıyor göstergesiyle." },
      { title: "Sesli ve görüntülü aramalar", desc: "Net sesli ve görüntülü aramalar — yavaş bağlantıda bile." },
      { title: "Fotoğraflar, dosyalar ve ses", desc: "Fotoğraf, video, 50 MB'a kadar dosya ve sesli mesaj gönderin." },
      { title: "24 saatlik hikâyeler", desc: "Fotoğraflarınızı ve anılarınızı paylaşın; bir gün sonra kaybolurlar." },
      { title: "Uçtan uca şifreleme", desc: "Özel sohbetleriniz gerçek uçtan uca şifrelemeyle kilitli — onları yalnızca siz ve arkadaşınız okuyabilir." },
      { title: "Botlar", desc: "İşlerinizi sohbetlerinizin içinde halleden botlar." },
      { title: "Diller", desc: "Varsayılan olarak İngilizce; Türkçe ve diğer diller Ayarlar'da." },
      { title: "Çok düşük veri kullanımı", desc: "Diğer mesajlaşma uygulamalarından çok daha az veri kullanır — aramalar, fotoğraflar ve medya, hepsi sıkıştırılmış ve küçük." },
      { title: "Yerel yapay zekâ asistanı", desc: "Telefonunuzda akıllı bir yardımcı — hızlı yanıtlar ve çeviri, buluta hiçbir şey göndermeden." },
    ],
    footer:
      "İndirmesi ve kullanımı ücretsizdir. Android beta sürümü yukarıda mevcuttur; iPhone daha sonra gelecektir.",
    aboutTitle: "Safari Hakkında",
    languageLabel: "Dil",
    forYourPhone: "Safari 0.7.1 — telefonunuz için",
    detected: "Telefonunuz için algılandı:",
    downloadButton: "Android için Safari'yu indirin",
    verify: "İndirmeyi doğrulayın (SHA-256):",
    category: "İletişim",
    free: "Ücretsiz",
    updated: "Güncellendi: 16 Kasım 2026",
    requires: "Android 6.0+ gerekir",
    offeredBy: "Sunucu: Safari Ekibi",
    screenshotsTitle: "Önizleme",
    slowTitle: "Yavaş internet için tasarlandı",
    slowDesc:
      "İndirmeler küçüktür ve durursa devam eder. Veri Tasarrufu varsayılan olarak açıktır — sesli aramalar yalnızca 12 kbps kullanır, fotoğraflar sıkıştırılmış gelir ve medya yalnızca siz dokununca indirilir.",
    infoTitle: "Uygulama bilgisi",
    versionLabel: "Sürüm",
    sizeLabel: "Boyut",
  },
  ru: {
    tagline: "Связь без границ",
    intro:
      "Познакомьтесь с Safari — быстрым и приватным мессенджером для Android. Общайтесь с друзьями один на один или в больших группах, подписывайтесь на каналы, звоните друзьям, делитесь историями, отправляйте фото, файлы и голосовые сообщения — всё защищено сквозным шифрованием. Safari расходует гораздо меньше трафика, чем другие мессенджеры, поэтому летает даже на медленном интернете. Скачивание и использование бесплатны.",
    downloadTitle: "Скачать Safari для Android",
    versionLine:
      "Версия 0.7.1 (бета) · Требуется Android 6.0 или новее · Сайт автоматически определяет ваш телефон — просто нажмите кнопку ниже, и скачается подходящая для него версия.",
    warning:
      "Скачивайте Safari только с этого официального сайта. APK-файлы с других сайтов небезопасны — не устанавливайте их.",
    featuresTitle: "Возможности",
    features: [
      { title: "Личные и групповые чаты", desc: "Общайтесь с друзьями один на один или в больших группах — с администраторами, закреплением и не только." },
      { title: "Чат в реальном времени", desc: "Сообщения приходят мгновенно, с отметками о прочтении и индикатором набора." },
      { title: "Голосовые и видеозвонки", desc: "Чистые голосовые и видеозвонки — даже при медленном соединении." },
      { title: "Фото, файлы и голос", desc: "Отправляйте фото, видео, файлы до 50 МБ и голосовые сообщения." },
      { title: "Истории на 24 часа", desc: "Делитесь фото и моментами, которые исчезают через день." },
      { title: "Сквозное шифрование", desc: "Ваши личные чаты закрыты настоящим сквозным шифрованием — прочитать их можете только вы и ваш друг." },
      { title: "Боты", desc: "Боты, которые помогают делать дела прямо в ваших чатах." },
      { title: "Языки", desc: "По умолчанию английский; русский и другие языки — в настройках." },
      { title: "Очень низкий расход трафика", desc: "Расходует гораздо меньше трафика, чем другие мессенджеры, — звонки, фото и медиа, всё сжато и компактно." },
      { title: "Локальный ИИ-ассистент", desc: "Умный помощник на вашем телефоне — быстрые ответы и перевод, и ничего не уходит в облако." },
    ],
    footer:
      "Скачивание и использование бесплатны. Бета-версия для Android доступна выше; iPhone появится позже.",
    aboutTitle: "О Safari",
    languageLabel: "Язык",
    forYourPhone: "Safari 0.7.1 — для вашего телефона",
    detected: "Определено для вашего телефона:",
    downloadButton: "Скачать Safari для Android",
    verify: "Проверьте загрузку (SHA-256):",
    category: "Связь",
    free: "Бесплатно",
    updated: "Обновлено: 16 ноября 2026 г.",
    requires: "Требуется Android 6.0+",
    offeredBy: "Разработчик: команда Safari",
    screenshotsTitle: "Предпросмотр",
    slowTitle: "Сделано для медленного интернета",
    slowDesc:
      "Загрузки небольшие и возобновляются, если прервутся. Экономия трафика включена по умолчанию — голосовые звонки используют всего 12 кбит/с, фото приходят сжатыми, а медиа скачивается только когда вы нажмёте.",
    infoTitle: "Информация о приложении",
    versionLabel: "Версия",
    sizeLabel: "Размер",
  },
  es: {
    tagline: "Conecta más allá de los límites",
    intro:
      "Conoce Safari: un mensajero rápido y privado para Android. Chatea con tus amigos en privado o en grupos grandes, sigue canales, llama a tus amigos, comparte historias y envía fotos, archivos y notas de voz — todo protegido con cifrado de extremo a extremo. Safari usa mucho menos datos que otros mensajeros, así que vuela incluso con internet lento. Descargarlo y usarlo es gratis.",
    downloadTitle: "Descargar Safari para Android",
    versionLine:
      "Versión 0.7.1 (beta) · Android 6.0 o superior · El sitio detecta tu teléfono automáticamente: solo toca el botón de abajo y se descargará la versión adecuada para ti.",
    warning:
      "Descarga Safari solo desde este sitio web oficial. Los APK de otros sitios no son seguros: no los instales.",
    featuresTitle: "Funciones",
    features: [
      { title: "Chat privado y de grupo", desc: "Chatea con tus amigos en privado o en grupos grandes, con administradores, mensajes fijados y más." },
      { title: "Chat en tiempo real", desc: "Los mensajes llegan al instante, con marcas de lectura e indicador de escritura." },
      { title: "Llamadas de voz y video", desc: "Llamadas de voz y video claras, incluso con una conexión lenta." },
      { title: "Fotos, archivos y voz", desc: "Envía fotos, videos, archivos de hasta 50 MB y mensajes de voz." },
      { title: "Historias de 24 horas", desc: "Comparte fotos y momentos que desaparecen al cabo de un día." },
      { title: "Cifrado de extremo a extremo", desc: "Tus chats privados están protegidos con cifrado real de extremo a extremo: solo tú y tu amigo podéis leerlos." },
      { title: "Bots", desc: "Bots que te ayudan a hacer cosas dentro de tus chats." },
      { title: "Idiomas", desc: "Inglés por defecto, con español y otros idiomas en Ajustes." },
      { title: "Uso de datos ultrabajo", desc: "Usa muchos menos datos que otros mensajeros: llamadas, fotos y medios, todo comprimido y pequeño." },
      { title: "Asistente de IA local", desc: "Un ayudante inteligente en tu teléfono: respuestas rápidas y traducción, sin enviar nada a la nube." },
    ],
    footer:
      "Descargarlo y usarlo es gratis. La beta de Android está disponible arriba; el iPhone llegará después.",
    aboutTitle: "Acerca de Safari",
    languageLabel: "Idioma",
    forYourPhone: "Safari 0.7.1 — para tu teléfono",
    detected: "Detectado para tu teléfono:",
    downloadButton: "Descargar Safari para Android",
    verify: "Verifica la descarga (SHA-256):",
    category: "Comunicación",
    free: "Gratis",
    updated: "Actualizado: 16 de noviembre de 2026",
    requires: "Requiere Android 6.0+",
    offeredBy: "Ofrecido por el equipo de Safari",
    screenshotsTitle: "Vista previa",
    slowTitle: "Hecho para internet lento",
    slowDesc:
      "Las descargas son pequeñas y se reanudan si se detienen. El ahorro de datos está activado por defecto: las llamadas de voz usan solo 12 kbps, las fotos llegan comprimidas y los medios se descargan solo cuando los tocas.",
    infoTitle: "Información de la app",
    versionLabel: "Versión",
    sizeLabel: "Tamaño",
  },
  fr: {
    tagline: "Connectez-vous au-delà des limites",
    intro:
      "Découvrez Safari, une messagerie rapide et privée pour Android. Discutez avec vos amis en privé ou en grands groupes, suivez des chaînes, appelez vos amis, partagez des stories et envoyez photos, fichiers et messages vocaux — le tout protégé par le chiffrement de bout en bout. Safari consomme beaucoup moins de données que les autres messageries, il file donc même sur une connexion lente. Téléchargement et utilisation gratuits.",
    downloadTitle: "Télécharger Safari pour Android",
    versionLine:
      "Version 0.7.1 (bêta) · Android 6.0 ou plus récent · Le site détecte automatiquement votre téléphone — touchez simplement le bouton ci-dessous et la version adaptée se téléchargera.",
    warning:
      "Téléchargez Safari uniquement depuis ce site officiel. Les APK d'autres sites ne sont pas sûrs — ne les installez pas.",
    featuresTitle: "Fonctionnalités",
    features: [
      { title: "Discussions privées et de groupe", desc: "Discutez avec vos amis en privé ou en grands groupes, avec administrateurs, messages épinglés et plus encore." },
      { title: "Discussion en temps réel", desc: "Les messages arrivent instantanément, avec accusés de lecture et indicateur de frappe." },
      { title: "Appels vocaux et vidéo", desc: "Des appels vocaux et vidéo clairs — même avec une connexion lente." },
      { title: "Photos, fichiers et voix", desc: "Envoyez photos, vidéos, fichiers jusqu'à 50 Mo et messages vocaux." },
      { title: "Stories de 24 heures", desc: "Partagez vos photos et vos moments, qui disparaissent au bout d'un jour." },
      { title: "Chiffrement de bout en bout", desc: "Vos discussions privées sont verrouillées par un vrai chiffrement de bout en bout — seuls vous et votre ami pouvez les lire." },
      { title: "Bots", desc: "Des bots qui vous aident à faire les choses, directement dans vos discussions." },
      { title: "Langues", desc: "Anglais par défaut, avec le français et d'autres langues dans les réglages." },
      { title: "Très faible consommation de données", desc: "Consomme beaucoup moins de données que les autres messageries — appels, photos et médias, tout est compressé et léger." },
      { title: "Assistant IA local", desc: "Un assistant intelligent sur votre téléphone — réponses rapides et traduction, sans rien envoyer dans le cloud." },
    ],
    footer:
      "Téléchargement et utilisation gratuits. La bêta Android est disponible ci-dessus ; l'iPhone viendra plus tard.",
    aboutTitle: "À propos de Safari",
    languageLabel: "Langue",
    forYourPhone: "Safari 0.7.1 — pour votre téléphone",
    detected: "Détecté pour votre téléphone :",
    downloadButton: "Télécharger Safari pour Android",
    verify: "Vérifiez le téléchargement (SHA-256) :",
    category: "Communication",
    free: "Gratuit",
    updated: "Mis à jour : 16 novembre 2026",
    requires: "Nécessite Android 6.0+",
    offeredBy: "Proposé par l'équipe Safari",
    screenshotsTitle: "Aperçu",
    slowTitle: "Conçu pour internet lent",
    slowDesc:
      "Les téléchargements sont petits et reprennent s'ils s'arrêtent. L'économiseur de données est activé par défaut : les appels vocaux n'utilisent que 12 kbit/s, les photos arrivent compressées et les médias ne se téléchargent que quand vous les touchez.",
    infoTitle: "Infos sur l'app",
    versionLabel: "Version",
    sizeLabel: "Taille",
  },
  de: {
    tagline: "Verbinde dich über Grenzen hinaus",
    intro:
      "Lerne Safari kennen — ein schneller, privater Messenger für Android. Chatte mit Freunden privat oder in großen Gruppen, folge Kanälen, rufe deine Freunde an, teile Stories und sende Fotos, Dateien und Sprachnachrichten — alles geschützt durch Ende-zu-Ende-Verschlüsselung. Safari verbraucht viel weniger Daten als andere Messenger und fliegt deshalb auch bei langsamem Internet. Download und Nutzung sind kostenlos.",
    downloadTitle: "Safari für Android herunterladen",
    versionLine:
      "Version 0.7.1 (Beta) · Android 6.0 oder neuer · Die Website erkennt dein Telefon automatisch — tippe einfach auf die Schaltfläche unten, und die passende Version wird heruntergeladen.",
    warning:
      "Lade Safari nur von dieser offiziellen Website herunter. APKs von anderen Websites sind nicht sicher — installiere sie nicht.",
    featuresTitle: "Funktionen",
    features: [
      { title: "Private und Gruppenchats", desc: "Chatte mit Freunden privat oder in großen Gruppen — mit Admins, angehefteten Nachrichten und mehr." },
      { title: "Echtzeit-Chat", desc: "Nachrichten kommen sofort an, mit Lesebestätigungen und Tipp-Anzeige." },
      { title: "Sprach- und Videoanrufe", desc: "Klare Sprach- und Videoanrufe — auch bei langsamer Verbindung." },
      { title: "Fotos, Dateien und Sprache", desc: "Sende Fotos, Videos, Dateien bis 50 MB und Sprachnachrichten." },
      { title: "24-Stunden-Stories", desc: "Teile Fotos und Momente, die nach einem Tag verschwinden." },
      { title: "Ende-zu-Ende-Verschlüsselung", desc: "Deine privaten Chats sind mit echter Ende-zu-Ende-Verschlüsselung gesperrt — nur du und dein Freund können sie lesen." },
      { title: "Bots", desc: "Bots, die dir helfen, Dinge direkt in deinen Chats zu erledigen." },
      { title: "Sprachen", desc: "Standardmäßig Englisch; Deutsch und andere Sprachen in den Einstellungen." },
      { title: "Sehr geringer Datenverbrauch", desc: "Verbraucht viel weniger Daten als andere Messenger — Anrufe, Fotos und Medien, alles komprimiert und klein." },
      { title: "Lokaler KI-Assistent", desc: "Ein intelligenter Helfer auf deinem Telefon — schnelle Antworten und Übersetzung, ohne dass etwas in die Cloud geht." },
    ],
    footer:
      "Download und Nutzung sind kostenlos. Die Android-Beta ist oben verfügbar; das iPhone kommt später.",
    aboutTitle: "Über Safari",
    languageLabel: "Sprache",
    forYourPhone: "Safari 0.7.1 — für dein Telefon",
    detected: "Für dein Telefon erkannt:",
    downloadButton: "Safari für Android herunterladen",
    verify: "Download überprüfen (SHA-256):",
    category: "Kommunikation",
    free: "Kostenlos",
    updated: "Aktualisiert: 16. November 2026",
    requires: "Erfordert Android 6.0+",
    offeredBy: "Angeboten vom Safari-Team",
    screenshotsTitle: "Vorschau",
    slowTitle: "Für langsames Internet gemacht",
    slowDesc:
      "Downloads sind klein und werden fortgesetzt, wenn sie abbrechen. Der Datensparmodus ist standardmäßig an: Sprachanrufe nutzen nur 12 kbit/s, Fotos kommen komprimiert an, und Medien werden nur heruntergeladen, wenn du darauf tippst.",
    infoTitle: "App-Info",
    versionLabel: "Version",
    sizeLabel: "Größe",
  },
  hi: {
    tagline: "सीमाओं से परे जुड़ें",
    intro:
      "Safari से मिलिए — Android के लिए एक तेज़ और निजी मैसेंजर। दोस्तों से अकेले या बड़े ग्रुप में चैट करें, चैनल फॉलो करें, दोस्तों को कॉल करें, स्टोरीज़ शेयर करें और फ़ोटो, फ़ाइलें व वॉइस नोट्स भेजें — सब कुछ एंड-टू-एंड एन्क्रिप्शन से सुरक्षित। Safari दूसरे मैसेंजरों से बहुत कम डेटा इस्तेमाल करता है, इसलिए धीमे इंटरनेट पर भी तेज़ चलता है। डाउनलोड और इस्तेमाल मुफ़्त है।",
    downloadTitle: "Android के लिए Safari डाउनलोड करें",
    versionLine:
      "संस्करण 0.7.1 (बीटा) · Android 6.0 या उससे नया · साइट आपके फ़ोन को अपने आप पहचानती है — बस नीचे वाला बटन दबाएँ और आपके फ़ोन के लिए सही संस्करण डाउनलोड हो जाएगा।",
    warning:
      "Safari सिर्फ़ इसी आधिकारिक वेबसाइट से डाउनलोड करें। अन्य साइटों की APK सुरक्षित नहीं हैं — उन्हें इंस्टॉल न करें।",
    featuresTitle: "सुविधाएँ",
    features: [
      { title: "निजी और ग्रुप चैट", desc: "दोस्तों से अकेले या बड़े ग्रुप में चैट करें — एडमिन, पिन और बहुत कुछ के साथ।" },
      { title: "रियल-टाइम चैट", desc: "संदेश तुरंत पहुँचते हैं, पढ़ने के निशान और टाइपिंग संकेतक के साथ।" },
      { title: "वॉइस और वीडियो कॉल", desc: "साफ़ वॉइस और वीडियो कॉल — धीमे कनेक्शन पर भी।" },
      { title: "फ़ोटो, फ़ाइलें और आवाज़", desc: "फ़ोटो, वीडियो, 50 MB तक की फ़ाइलें और वॉइस संदेश भेजें।" },
      { title: "24 घंटे की स्टोरीज़", desc: "अपनी फ़ोटो और पल शेयर करें जो एक दिन बाद गायब हो जाते हैं।" },
      { title: "एंड-टू-एंड एन्क्रिप्शन", desc: "आपकी निजी चैट असली एंड-टू-एंड एन्क्रिप्शन से बंद हैं — सिर्फ़ आप और आपका दोस्त ही उन्हें पढ़ सकते हैं।" },
      { title: "बॉट्स", desc: "ऐसे बॉट्स जो आपके काम आपकी चैट के अंदर ही कर दें।" },
      { title: "भाषाएँ", desc: "डिफ़ॉल्ट रूप से अंग्रेज़ी; हिंदी और अन्य भाषाएँ सेटिंग्स में हैं।" },
      { title: "बहुत कम डेटा उपयोग", desc: "दूसरे मैसेंजरों से बहुत कम डेटा इस्तेमाल करता है — कॉल, फ़ोटो और मीडिया, सब छोटा और कंप्रेस्ड।" },
      { title: "स्थानीय AI सहायक", desc: "आपके फ़ोन पर एक स्मार्ट मददगार — तेज़ जवाब और अनुवाद, बिना कुछ क्लाउड को भेजे।" },
    ],
    footer:
      "डाउनलोड और इस्तेमाल मुफ़्त है। Android बीटा ऊपर उपलब्ध है; iPhone बाद में आएगा।",
    aboutTitle: "Safari के बारे में",
    languageLabel: "भाषा",
    forYourPhone: "Safari 0.7.1 — आपके फ़ोन के लिए",
    detected: "आपके फ़ोन के लिए पहचाना गया:",
    downloadButton: "Android के लिए Safari डाउनलोड करें",
    verify: "डाउनलोड सत्यापित करें (SHA-256):",
    category: "संचार",
    free: "निःशुल्क",
    updated: "अपडेट किया गया: 16 नवंबर 2026",
    requires: "Android 6.0+ आवश्यक",
    offeredBy: "Safari टीम द्वारा प्रस्तुत",
    screenshotsTitle: "पूर्वावलोकन",
    slowTitle: "धीमे इंटरनेट के लिए बनाया गया",
    slowDesc:
      "डाउनलोड छोटे हैं और रुक जाएँ तो फिर से शुरू होते हैं। डेटा सेवर डिफ़ॉल्ट रूप से चालू है — वॉइस कॉल सिर्फ़ 12 kbps इस्तेमाल करती हैं, फ़ोटो कंप्रेस होकर आती हैं, और मीडिया सिर्फ़ तब डाउनलोड होता है जब आप उसे टैप करें।",
    infoTitle: "ऐप की जानकारी",
    versionLabel: "संस्करण",
    sizeLabel: "आकार",
  },
  ps: {
    tagline: "له حدودو هاخوا وصل شئ",
    intro:
      "له Safari سره وپېژنئ — د Android لپاره یو چټک او شخصي مسینجر. له ملګرو سره یو په یو یا په لویو ډلو کې چیټ وکړئ، چینلونه تعقیب کړئ، ملګرو ته زنګ ووهئ، کیسې شریکې کړئ او انځورونه، فایلونه او غږیز یادښتونه ولېږئ — هر څه د له پایه تر پایه کوډ کولو سره خوندي دي. Safari له نورو مسینجرونو څخه ډېر کم ډاټا کاروي، نو په کم سرعت انټرنېټ کې هم چټک کار کوي. ډاونلوډ او کارول یې وړیا دي.",
    downloadTitle: "د Android لپاره Safari ډاونلوډ کړئ",
    versionLine:
      "نسخه 0.7.1 (بیټا) · Android 6.0 یا نوی · سایټ ستاسو تلیفون په اوتومات ډول پېژني — یوازې لاندې تڼۍ کېکاږئ او ستاسو تلیفون لپاره سمه نسخه به ډاونلوډ شي.",
    warning:
      "Safari یوازې له دې رسمي ویب‌سایټ څخه ډاونلوډ کړئ. له نورو سایټونو څخه APK خوندي نه دي — مه یې نصب کوئ.",
    featuresTitle: "ځانګړتیاوې",
    features: [
      { title: "شخصي او ډله‌ییز چیټ", desc: "له ملګرو سره یو په یو یا په لویو ډلو کې چیټ وکړئ — د مدیر، پین او نورو سره." },
      { title: "په ژوندۍ بڼه چیټ", desc: "پیغامونه سمدستي رسېږي، د لوستلو نښو او د ټایپ کولو نښې سره." },
      { title: "غږیز او ویډیويي زنګونه", desc: "روښانه غږیز او ویډیويي زنګونه — حتی په کم سرعت اړیکه کې." },
      { title: "انځورونه، فایلونه او غږ", desc: "انځورونه، ویډیوګانې، تر 50 MB پورې فایلونه او غږیز پیغامونه ولېږئ." },
      { title: "24 ساعته کیسې", desc: "خپل انځورونه او شېبې شریکې کړئ چې له یوې ورځې وروسته ورکېږي." },
      { title: "له پایه تر پایه کوډ کول", desc: "ستاسو شخصي چیټونه د رښتیني له پایه تر پایه کوډ کولو سره تړل شوي — یوازې تاسو او ستاسو ملګری یې لوستلی شئ." },
      { title: "بوټان", desc: "هغه بوټان چې ستاسو کارونه ستاسو د چیټونو په دننه کې کوي." },
      { title: "ژبې", desc: "په ډیفالټ انګلیسي؛ پښتو او نورې ژبې په ترتیباتو کې دي." },
      { title: "ډېر ټیټ ډاټا کارول", desc: "له نورو مسینجرونو څخه ډېر کم ډاټا کاروي — زنګونه، انځورونه او رسنۍ، ټول کمپریس شوي او کوچني." },
      { title: "ځایي AI مرستیال", desc: "ستاسو په تلیفون کې یو هوښیار مرستیال — چټک ځوابونه او ژباړه، پرته له دې چې څه کلاوډ ته ولېږي." },
    ],
    footer:
      "ډاونلوډ او کارول وړیا دي. د Android بیټا پورته شتون لري؛ iPhone وروسته راځي.",
    aboutTitle: "د Safari په اړه",
    languageLabel: "ژبه",
    forYourPhone: "Safari 0.7.1 — ستاسو تلیفون لپاره",
    detected: "ستاسو تلیفون لپاره وپېژندل شو:",
    downloadButton: "د Android لپاره Safari ډاونلوډ کړئ",
    verify: "ډاونلوډ تایید کړئ (SHA-256):",
    category: "اړیکې",
    free: "وړیا",
    updated: "تازه شوی: نومبر ۱۶، ۲۰۲۶",
    requires: "Android 6.0+ ته اړتیا لري",
    offeredBy: "د Safari ټیم لخوا وړاندې شوی",
    screenshotsTitle: "مخکتنه",
    slowTitle: "د کم سرعت انټرنېټ لپاره جوړ شوی",
    slowDesc:
      "ډاونلوډونه کوچني دي او که ودرېږي، دوام مومي. ډاټا سیور په ډیفالټ فعال دی — غږیز زنګونه یوازې 12 kbps کاروي، انځورونه کمپریس شوي راځي، او رسنۍ یوازې هغه وخت ډاونلوډ کېږي کله چې تاسو یې ټچ کړئ.",
    infoTitle: "د اپ معلومات",
    versionLabel: "نسخه",
    sizeLabel: "سایز",
  },
};
