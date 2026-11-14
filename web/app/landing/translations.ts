// Translations for the VYRO download page. English is the default.
// The visitor picks a language from the switcher; the whole page
// (headings, download section, features, warning, footer) switches
// to it, and the choice is remembered in localStorage.

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
  | "hi";

export const languages: { code: LangCode; label: string; dir: "ltr" | "rtl" }[] = [
  { code: "en", label: "English", dir: "ltr" },
  { code: "fa", label: "فارسی", dir: "rtl" },
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
};

export const translations: Record<LangCode, Strings> = {
  en: {
    tagline: "Connect Beyond Limits",
    intro:
      "VYRO is a fast, secure messenger for Android: private and group chat, channels, voice calls, stories, bots, and end-to-end encryption for one-to-one messages. This is a beta — some features are still on the way. The messenger lives only inside the app — download it below to start chatting.",
    downloadTitle: "Download VYRO for Android",
    versionLine:
      "Version 0.6.0 (Beta) · Requires Android 6.0+ (API 23) · The site detects your phone automatically and downloads the build made for it — just tap the button below. Signed with the development key for this first test build; a proper release signature comes before wide distribution.",
    warning:
      "Download VYRO only from this official website. APKs from other sites are not safe and should not be installed.",
    featuresTitle: "Features",
    features: [
      { title: "Private & group chat", desc: "Direct messages, groups and channels with admin roles." },
      { title: "Real-time chat", desc: "Instant messages over WebSocket, read ticks and typing indicator." },
      { title: "Voice & video calls", desc: "Voice calls over WebRTC; video calls start at low quality (Data Saver)." },
      { title: "Photos, files & voice", desc: "Uploads up to 50 MB, image and file messages, voice messages." },
      { title: "24-hour stories", desc: "Stories and status, like the messengers you know." },
      { title: "End-to-end encryption", desc: "One-to-one DMs with real E2EE (X3DH + Double Ratchet); group E2EE with Sender Keys." },
      { title: "Bots", desc: "Bots with a token and webhook for automation." },
      { title: "English & Persian", desc: "English by default; RTL Persian and other languages available in Settings." },
      { title: "Ultra-low data use", desc: "Data Saver by default: voice calls at Opus 12 kbps (~0.09 MB/min), compressed photos, tap-to-download media." },
      { title: "Local AI assistant", desc: "On-device assistant, dictionary translation and smart replies — no cloud AI, no data leaves your device." },
    ],
    footer:
      "Free and open-source only, until VYRO reaches 10,000 registered users. The Android beta APK is available above; iOS is coming later.",
    aboutTitle: "About VYRO",
    languageLabel: "Language",
    forYourPhone: "VYRO 0.6.0 — for your phone",
    detected: "Detected for your phone:",
    downloadButton: "Download VYRO for Android",
    verify: "Verify the download (SHA-256):",
  },
  fa: {
    tagline: "اتصال فراتر از محدودیت‌ها",
    intro:
      "VYRO یک پیام‌رسان سریع و امن برای اندروید است: چت خصوصی و گروهی، کانال، تماس صوتی، استوری، ربات و رمزنگاری سرتاسری برای پیام‌های خصوصی. این نسخه هنوز آزمایشی (بتا) است و بعضی قابلیت‌ها هنوز در راه‌اند. پیام‌رسان فقط داخل اپ موبایل است — برای شروع، اپ را از پایین همین صفحه دانلود کنید.",
    downloadTitle: "دانلود VYRO برای اندروید",
    versionLine:
      "نسخه 0.6.0 (بتا) · نیازمند اندروید ۶٫۰ به بالا (API 23) · سایت گوشی شما را به‌صورت خودکار تشخیص می‌دهد و نسخه مناسب آن را دانلود می‌کند — فقط دکمه زیر را بزنید. با کلید آزمایشی برای این نسخه اول امضا شده است؛ پیش از انتشار گسترده، امضای رسمی خواهد داشت.",
    warning:
      "VYRO را فقط از همین وب‌سایت رسمی دانلود کنید. فایل‌های APK از سایت‌های دیگر امن نیستند و نباید نصب شوند.",
    featuresTitle: "قابلیت‌ها",
    features: [
      { title: "چت خصوصی و گروه", desc: "گفت‌وگوی خصوصی، گروه و کانال با مدیریت مدیران." },
      { title: "چت ریل‌تایم", desc: "پیام آنی با WebSocket، تیک خوانده‌شدن و «در حال تایپ…»." },
      { title: "تماس صوتی و تصویری", desc: "تماس صوتی با WebRTC؛ تماس تصویری با کیفیت پایین شروع می‌شود (ذخیره داده)." },
      { title: "عکس، فایل و پیام صوتی", desc: "آپلود تا ۵۰ مگابایت و پیام تصویری و فایل." },
      { title: "استوری ۲۴ ساعته", desc: "استوری و استاتوس مثل پیام‌رسان‌های آشنا." },
      { title: "رمزنگاری سرتاسری", desc: "پیام‌های خصوصی دونفره با E2EE واقعی (X3DH + Double Ratchet)؛ رمزنگاری گروهی با کلید فرستنده." },
      { title: "ربات‌ها", desc: "ربات با توکن و webhook برای خودکارسازی." },
      { title: "انگلیسی و فارسی", desc: "انگلیسی پیش‌فرض است؛ فارسی راست‌چین و زبان‌های دیگر در تنظیمات در دسترس‌اند." },
      { title: "مصرف بسیار کم اینترنت", desc: "ذخیره داده به‌صورت پیش‌فرض: تماس صوتی با Opus ۱۲ کیلوبیت بر ثانیه (حدود ۰٫۰۹ مگابایت در دقیقه)، عکس‌های فشرده، دانلود با لمس." },
      { title: "دستیار هوش مصنوعی محلی", desc: "دستیار روی دستگاه، ترجمه با فرهنگ لغت و پاسخ‌های هوشمند — بدون هوش مصنوعی ابری، بدون خروج داده از دستگاه شما." },
    ],
    footer:
      "رایگان و متن‌باز، تا زمانی که VYRO به ۱۰٬۰۰۰ کاربر ثبت‌شده برسد. نسخه بتای اندروید در بالا در دسترس است؛ نسخه iOS بعداً می‌آید.",
    aboutTitle: "درباره وایرو",
    languageLabel: "زبان",
    forYourPhone: "VYRO 0.6.0 — برای گوشی شما",
    detected: "برای گوشی شما تشخیص داده شد:",
    downloadButton: "دانلود VYRO برای اندروید",
    verify: "بررسی دانلود (SHA-256):",
  },
  ar: {
    tagline: "تواصل بلا حدود",
    intro:
      "VYRO مراسل سريع وآمن لأندرويد: دردشة خاصة وجماعية، قنوات، مكالمات صوتية، قصص، بوتات، وتشفير كامل للرسائل الخاصة. هذه نسخة تجريبية — بعض الميزات ما زالت قادمة. المراسل يعيش داخل التطبيق فقط — حمّله من الأسفل لبدء الدردشة.",
    downloadTitle: "تنزيل VYRO لأندرويد",
    versionLine:
      "الإصدار 0.6.0 (تجريبي) · يتطلب أندرويد 6.0+ (API 23) · يكتشف الموقع هاتفك تلقائيًا وينزّل النسخة المناسبة له — فقط اضغط الزر أدناه. موقّع بمفتاح تطوير لهذا الإصدار التجريبي الأول؛ وسيحصل على توقيع رسمي قبل الانتشار الواسع.",
    warning:
      "حمّل VYRO من هذا الموقع الرسمي فقط. ملفات APK من مواقع أخرى غير آمنة ويجب عدم تثبيتها.",
    featuresTitle: "الميزات",
    features: [
      { title: "دردشة خاصة وجماعية", desc: "رسائل مباشرة ومجموعات وقنوات مع أدوار للمشرفين." },
      { title: "دردشة فورية", desc: "رسائل فورية عبر WebSocket، وعلامات القراءة ومؤشر الكتابة." },
      { title: "مكالمات صوتية ومرئية", desc: "مكالمات صوتية عبر WebRTC؛ وتبدأ مكالمات الفيديو بجودة منخفضة (توفير البيانات)." },
      { title: "صور وملفات وصوت", desc: "رفع حتى 50 ميغابايت، ورسائل صور وملفات ورسائل صوتية." },
      { title: "قصص 24 ساعة", desc: "قصص وحالات كما في التطبيقات التي تعرفها." },
      { title: "تشفير كامل", desc: "رسائل خاصة فردية بتشفير حقيقي (X3DH + Double Ratchet)؛ وتشفير المجموعات بمفاتيح المرسل." },
      { title: "بوتات", desc: "بوتات برمز وwebhook للأتمتة." },
      { title: "لغات متعددة", desc: "الإنجليزية هي الافتراضية؛ والعربية والفارسية ولغات أخرى متاحة في الإعدادات." },
      { title: "استهلاك بيانات منخفض جدًا", desc: "توفير البيانات افتراضيًا: مكالمات صوتية بترميز Opus 12 كيلوبت/ثانية (~0.09 ميغابايت/دقيقة)، وصور مضغوطة، وتنزيل الوسائط بالنقر." },
      { title: "مساعد ذكاء اصطناعي محلي", desc: "مساعد على الجهاز، وترجمة بالقاموس وردود ذكية — بلا ذكاء اصطناعي سحابي، ولا تغادر بياناتك جهازك." },
    ],
    footer:
      "مجاني ومفتوح المصدر فقط، حتى يصل VYRO إلى 10,000 مستخدم مسجّل. نسخة أندرويد التجريبية متاحة أعلاه؛ ونسخة iOS قادمة لاحقًا.",
    aboutTitle: "عن VYRO",
    languageLabel: "اللغة",
    forYourPhone: "VYRO 0.6.0 — لهاتفك",
    detected: "تم الكشف لهاتفك:",
    downloadButton: "تنزيل VYRO لأندرويد",
    verify: "تحقق من التنزيل (SHA-256):",
  },
  ur: {
    tagline: "حدود سے آگے جڑیں",
    intro:
      "VYRO اینڈرائیڈ کے لیے ایک تیز اور محفوظ میسنجر ہے: نجی اور گروپ چیٹ، چینلز، وائس کالز، اسٹوریز، بوٹس، اور نجی پیغامات کے لیے مکمل خفیہ کاری۔ یہ بیٹا ورژن ہے — کچھ فیچرز ابھی آ رہے ہیں۔ میسنجر صرف ایپ کے اندر ہے — چیٹ شروع کرنے کے لیے نیچے سے ڈاؤن لوڈ کریں۔",
    downloadTitle: "اینڈرائیڈ کے لیے VYRO ڈاؤن لوڈ کریں",
    versionLine:
      "ورژن 0.6.0 (بیٹا) · اینڈرائیڈ 6.0+ (API 23) درکار ہے · سائٹ آپ کے فون کو خود بخود پہچانتی ہے اور اس کے لیے بنائی گئی بلڈ ڈاؤن لوڈ کرتی ہے — بس نیچے والا بٹن دبائیں۔ اس پہلی ٹیسٹ بلڈ پر ڈیولپمنٹ کی دستخط ہے؛ وسیع تقسیم سے پہلے باقاعدہ دستخط آئے گا۔",
    warning:
      "VYRO صرف اسی سرکاری ویب سائٹ سے ڈاؤن لوڈ کریں۔ دوسری سائٹس کی APK فائلیں محفوظ نہیں ہیں اور انہیں انسٹال نہیں کرنا چاہیے۔",
    featuresTitle: "خصوصیات",
    features: [
      { title: "نجی اور گروپ چیٹ", desc: "ڈائریکٹ میسجز، گروپس اور چینلز، ایڈمن کرداروں کے ساتھ۔" },
      { title: "فوری چیٹ", desc: "WebSocket پر فوری پیغامات، پڑھنے کے نشان اور ٹائپنگ انڈیکیٹر۔" },
      { title: "وائس اور ویڈیو کالز", desc: "WebRTC پر وائس کالز؛ ویڈیو کالز کم کوالٹی سے شروع ہوتی ہیں (ڈیٹا سیور)۔" },
      { title: "تصاویر، فائلز اور وائس", desc: "50 MB تک اپ لوڈ، تصویری و فائل پیغامات، وائس پیغامات۔" },
      { title: "24 گھنٹے کی اسٹوریز", desc: "اسٹوریز اور اسٹیٹس، جیسے آپ جانتے ہیں۔" },
      { title: "مکمل خفیہ کاری", desc: "انفرادی DMs میں حقیقی E2EE (X3DH + Double Ratchet)؛ گروپ E2EE سینڈر کیز کے ساتھ۔" },
      { title: "بوٹس", desc: "آٹومیشن کے لیے ٹوکن اور webhook والے بوٹس۔" },
      { title: "زبانیں", desc: "انگریزی بطور پیش فرض؛ اردو اور دیگر زبانیں سیٹنگز میں دستیاب ہیں۔" },
      { title: "بہت کم ڈیٹا استعمال", desc: "ڈیٹا سیور بطور پیش فرض: Opus 12 kbps پر وائس کالز (~0.09 MB/منٹ)، کمپریسڈ تصاویر، ٹیپ کر کے ڈاؤن لوڈ۔" },
      { title: "مقامی AI اسسٹنٹ", desc: "ڈیوائس پر اسسٹنٹ، لغت ترجمہ اور سمارٹ جوابات — کوئی کلاؤڈ AI نہیں، آپ کا ڈیٹا ڈیوائس نہیں چھوڑتا۔" },
    ],
    footer:
      "صرف مفت اور اوپن سورس، جب تک VYRO 10,000 رجسٹرڈ صارفین تک نہ پہنچے۔ اینڈرائیڈ بیٹا APK اوپر دستیاب ہے؛ iOS بعد میں آئے گا۔",
    aboutTitle: "VYRO کے بارے میں",
    languageLabel: "زبان",
    forYourPhone: "VYRO 0.6.0 — آپ کے فون کے لیے",
    detected: "آپ کے فون کے لیے پہچانا گیا:",
    downloadButton: "اینڈرائیڈ کے لیے VYRO ڈاؤن لوڈ کریں",
    verify: "ڈاؤن لوڈ کی تصدیق کریں (SHA-256):",
  },
  tr: {
    tagline: "Sınırların Ötesinde Bağlanın",
    intro:
      "VYRO, Android için hızlı ve güvenli bir mesajlaşma uygulamasıdır: özel ve grup sohbetleri, kanallar, sesli aramalar, hikâyeler, botlar ve bire bir mesajlar için uçtan uca şifreleme. Bu bir beta sürümüdür — bazı özellikler henüz yolda. Mesajlaşma yalnızca uygulamanın içindedir — sohbete başlamak için aşağıdan indirin.",
    downloadTitle: "Android için VYRO'yu indirin",
    versionLine:
      "Sürüm 0.6.0 (Beta) · Android 6.0+ (API 23) gerekir · Site telefonunuzu otomatik olarak algılar ve ona uygun sürümü indirir — aşağıdaki düğmeye dokunmanız yeterli. Bu ilk test sürümü geliştirici anahtarıyla imzalanmıştır; geniş dağıtımdan önce resmi imza gelecektir.",
    warning:
      "VYRO'yu yalnızca bu resmî web sitesinden indirin. Diğer sitelerdeki APK'lar güvenli değildir ve yüklenmemelidir.",
    featuresTitle: "Özellikler",
    features: [
      { title: "Özel ve grup sohbeti", desc: "Yönetici rolleriyle doğrudan mesajlar, gruplar ve kanallar." },
      { title: "Gerçek zamanlı sohbet", desc: "WebSocket ile anında mesajlar, okundu bilgisi ve yazıyor göstergesi." },
      { title: "Sesli ve görüntülü aramalar", desc: "WebRTC ile sesli aramalar; görüntülü aramalar düşük kalitede başlar (Veri Tasarrufu)." },
      { title: "Fotoğraflar, dosyalar ve ses", desc: "50 MB'a kadar yükleme, fotoğraf ve dosya mesajları, sesli mesajlar." },
      { title: "24 saatlik hikâyeler", desc: "Bildiğiniz mesajlaşma uygulamalarındaki gibi hikâyeler ve durumlar." },
      { title: "Uçtan uca şifreleme", desc: "Bire bir DM'lerde gerçek E2EE (X3DH + Double Ratchet); grup E2EE'si Gönderen Anahtarları ile." },
      { title: "Botlar", desc: "Otomasyon için jetonlu ve webhook'lu botlar." },
      { title: "Diller", desc: "Varsayılan olarak İngilizce; Türkçe ve diğer diller Ayarlar'da mevcuttur." },
      { title: "Çok düşük veri kullanımı", desc: "Veri Tasarrufu varsayılan olarak açıktır: Opus 12 kbps ile sesli aramalar (~0,09 MB/dk), sıkıştırılmış fotoğraflar, dokunarak indirme." },
      { title: "Yerel yapay zekâ asistanı", desc: "Cihaz içi asistan, sözlük çevirisi ve akıllı yanıtlar — bulut yapay zekâsı yok, verileriniz cihazınızdan çıkmaz." },
    ],
    footer:
      "Yalnızca ücretsiz ve açık kaynak, VYRO 10.000 kayıtlı kullanıcıya ulaşana kadar. Android beta APK'sı yukarıda mevcuttur; iOS daha sonra gelecektir.",
    aboutTitle: "VYRO Hakkında",
    languageLabel: "Dil",
    forYourPhone: "VYRO 0.6.0 — telefonunuz için",
    detected: "Telefonunuz için algılandı:",
    downloadButton: "Android için VYRO'yu indirin",
    verify: "İndirmeyi doğrulayın (SHA-256):",
  },
  ru: {
    tagline: "Связь без границ",
    intro:
      "VYRO — быстрый и безопасный мессенджер для Android: личные и групповые чаты, каналы, голосовые звонки, истории, боты и сквозное шифрование личных сообщений. Это бета-версия — некоторые функции ещё в пути. Мессенджер живёт только внутри приложения — скачайте его ниже, чтобы начать общение.",
    downloadTitle: "Скачать VYRO для Android",
    versionLine:
      "Версия 0.6.0 (бета) · Требуется Android 6.0+ (API 23) · Сайт автоматически определяет ваш телефон и скачивает подходящую для него сборку — просто нажмите кнопку ниже. Подписано ключом разработчика для этой первой тестовой сборки; перед широким распространением будет официальная подпись.",
    warning:
      "Скачивайте VYRO только с этого официального сайта. APK-файлы с других сайтов небезопасны, их нельзя устанавливать.",
    featuresTitle: "Возможности",
    features: [
      { title: "Личные и групповые чаты", desc: "Личные сообщения, группы и каналы с ролями администраторов." },
      { title: "Чат в реальном времени", desc: "Мгновенные сообщения через WebSocket, отметки о прочтении и индикатор набора." },
      { title: "Голосовые и видеозвонки", desc: "Голосовые звонки через WebRTC; видеозвонки начинаются с низкого качества (экономия трафика)." },
      { title: "Фото, файлы и голос", desc: "Загрузка до 50 МБ, сообщения с фото и файлами, голосовые сообщения." },
      { title: "Истории на 24 часа", desc: "Истории и статусы, как в знакомых мессенджерах." },
      { title: "Сквозное шифрование", desc: "Настоящее E2EE для личных сообщений (X3DH + Double Ratchet); групповое E2EE с ключами отправителя." },
      { title: "Боты", desc: "Боты с токеном и webhook для автоматизации." },
      { title: "Языки", desc: "По умолчанию английский; русский и другие языки доступны в настройках." },
      { title: "Очень низкий расход трафика", desc: "Экономия трафика включена по умолчанию: голосовые звонки на Opus 12 кбит/с (~0,09 МБ/мин), сжатые фото, загрузка по нажатию." },
      { title: "Локальный ИИ-ассистент", desc: "Ассистент на устройстве, словарный перевод и умные ответы — без облачного ИИ, данные не покидают устройство." },
    ],
    footer:
      "Бесплатно и с открытым исходным кодом, пока VYRO не наберёт 10 000 зарегистрированных пользователей. Бета-версия APK для Android доступна выше; iOS появится позже.",
    aboutTitle: "О VYRO",
    languageLabel: "Язык",
    forYourPhone: "VYRO 0.6.0 — для вашего телефона",
    detected: "Определено для вашего телефона:",
    downloadButton: "Скачать VYRO для Android",
    verify: "Проверьте загрузку (SHA-256):",
  },
  es: {
    tagline: "Conecta más allá de los límites",
    intro:
      "VYRO es un mensajero rápido y seguro para Android: chats privados y de grupo, canales, llamadas de voz, historias, bots y cifrado de extremo a extremo para mensajes individuales. Esta es una beta: algunas funciones aún están en camino. El mensajero vive solo dentro de la app: descárgala abajo para empezar a chatear.",
    downloadTitle: "Descargar VYRO para Android",
    versionLine:
      "Versión 0.6.0 (beta) · Requiere Android 6.0+ (API 23) · El sitio detecta tu teléfono automáticamente y descarga la versión hecha para él: solo toca el botón de abajo. Firmado con la clave de desarrollo para esta primera versión de prueba; habrá una firma oficial antes de la distribución amplia.",
    warning:
      "Descarga VYRO solo desde este sitio web oficial. Los APK de otros sitios no son seguros y no deben instalarse.",
    featuresTitle: "Funciones",
    features: [
      { title: "Chat privado y de grupo", desc: "Mensajes directos, grupos y canales con roles de administrador." },
      { title: "Chat en tiempo real", desc: "Mensajes instantáneos por WebSocket, marcas de lectura e indicador de escritura." },
      { title: "Llamadas de voz y video", desc: "Llamadas de voz por WebRTC; las videollamadas empiezan en baja calidad (Ahorro de datos)." },
      { title: "Fotos, archivos y voz", desc: "Subidas de hasta 50 MB, mensajes de foto y archivo, mensajes de voz." },
      { title: "Historias de 24 horas", desc: "Historias y estados, como en los mensajeros que conoces." },
      { title: "Cifrado de extremo a extremo", desc: "E2EE real para mensajes directos (X3DH + Double Ratchet); E2EE de grupo con claves de remitente." },
      { title: "Bots", desc: "Bots con token y webhook para automatización." },
      { title: "Idiomas", desc: "Inglés por defecto; español y otros idiomas disponibles en Ajustes." },
      { title: "Uso de datos ultrabajo", desc: "Ahorro de datos activado por defecto: llamadas de voz con Opus a 12 kbps (~0,09 MB/min), fotos comprimidas, descarga al tocar." },
      { title: "Asistente de IA local", desc: "Asistente en el dispositivo, traducción por diccionario y respuestas inteligentes: sin IA en la nube, tus datos no salen del dispositivo." },
    ],
    footer:
      "Gratis y de código abierto, hasta que VYRO llegue a 10.000 usuarios registrados. La APK beta de Android está arriba; iOS llegará después.",
    aboutTitle: "Acerca de VYRO",
    languageLabel: "Idioma",
    forYourPhone: "VYRO 0.6.0 — para tu teléfono",
    detected: "Detectado para tu teléfono:",
    downloadButton: "Descargar VYRO para Android",
    verify: "Verifica la descarga (SHA-256):",
  },
  fr: {
    tagline: "Connectez-vous au-delà des limites",
    intro:
      "VYRO est une messagerie rapide et sécurisée pour Android : discussions privées et de groupe, chaînes, appels vocaux, stories, bots et chiffrement de bout en bout pour les messages privés. C'est une version bêta — certaines fonctionnalités sont encore en route. La messagerie vit uniquement dans l'application — téléchargez-la ci-dessous pour commencer à discuter.",
    downloadTitle: "Télécharger VYRO pour Android",
    versionLine:
      "Version 0.6.0 (bêta) · Nécessite Android 6.0+ (API 23) · Le site détecte automatiquement votre téléphone et télécharge la version faite pour lui — touchez simplement le bouton ci-dessous. Signé avec la clé de développement pour cette première version de test ; une signature officielle viendra avant la large diffusion.",
    warning:
      "Téléchargez VYRO uniquement depuis ce site officiel. Les APK d'autres sites ne sont pas sûrs et ne doivent pas être installés.",
    featuresTitle: "Fonctionnalités",
    features: [
      { title: "Discussions privées et de groupe", desc: "Messages directs, groupes et chaînes avec des rôles d'administrateur." },
      { title: "Discussion en temps réel", desc: "Messages instantanés via WebSocket, accusés de lecture et indicateur de frappe." },
      { title: "Appels vocaux et vidéo", desc: "Appels vocaux via WebRTC ; les appels vidéo commencent en basse qualité (Économiseur de données)." },
      { title: "Photos, fichiers et voix", desc: "Envois jusqu'à 50 Mo, messages photo et fichier, messages vocaux." },
      { title: "Stories de 24 heures", desc: "Stories et statuts, comme dans les messageries que vous connaissez." },
      { title: "Chiffrement de bout en bout", desc: "Vrai E2EE pour les messages privés (X3DH + Double Ratchet) ; E2EE de groupe avec clés d'expéditeur." },
      { title: "Bots", desc: "Bots avec jeton et webhook pour l'automatisation." },
      { title: "Langues", desc: "Anglais par défaut ; français et autres langues disponibles dans les réglages." },
      { title: "Très faible consommation de données", desc: "Économiseur de données activé par défaut : appels vocaux en Opus 12 kbit/s (~0,09 Mo/min), photos compressées, téléchargement au toucher." },
      { title: "Assistant IA local", desc: "Assistant sur l'appareil, traduction par dictionnaire et réponses intelligentes — pas d'IA cloud, vos données ne quittent pas l'appareil." },
    ],
    footer:
      "Gratuit et open source uniquement, jusqu'à ce que VYRO atteigne 10 000 utilisateurs inscrits. L'APK bêta Android est disponible ci-dessus ; iOS viendra plus tard.",
    aboutTitle: "À propos de VYRO",
    languageLabel: "Langue",
    forYourPhone: "VYRO 0.6.0 — pour votre téléphone",
    detected: "Détecté pour votre téléphone :",
    downloadButton: "Télécharger VYRO pour Android",
    verify: "Vérifiez le téléchargement (SHA-256) :",
  },
  de: {
    tagline: "Verbinde dich über Grenzen hinaus",
    intro:
      "VYRO ist ein schneller, sicherer Messenger für Android: private und Gruppenchats, Kanäle, Sprachanrufe, Stories, Bots und Ende-zu-Ende-Verschlüsselung für private Nachrichten. Dies ist eine Beta — einige Funktionen sind noch unterwegs. Der Messenger lebt nur in der App — lade sie unten herunter, um zu chatten.",
    downloadTitle: "VYRO für Android herunterladen",
    versionLine:
      "Version 0.6.0 (Beta) · Erfordert Android 6.0+ (API 23) · Die Website erkennt dein Telefon automatisch und lädt die dafür gemachte Version herunter — tippe einfach auf die Schaltfläche unten. Mit dem Entwicklungsschlüssel für diesen ersten Test-Build signiert; vor der breiten Verteilung kommt eine offizielle Signatur.",
    warning:
      "Lade VYRO nur von dieser offiziellen Website herunter. APKs von anderen Websites sind nicht sicher und sollten nicht installiert werden.",
    featuresTitle: "Funktionen",
    features: [
      { title: "Private und Gruppenchats", desc: "Direktnachrichten, Gruppen und Kanäle mit Admin-Rollen." },
      { title: "Echtzeit-Chat", desc: "Sofortnachrichten über WebSocket, Lesebestätigungen und Tipp-Anzeige." },
      { title: "Sprach- und Videoanrufe", desc: "Sprachanrufe über WebRTC; Videoanrufe starten in niedriger Qualität (Datensparmodus)." },
      { title: "Fotos, Dateien und Sprache", desc: "Uploads bis 50 MB, Foto- und Dateinachrichten, Sprachnachrichten." },
      { title: "24-Stunden-Stories", desc: "Stories und Status, wie in den Messengern, die du kennst." },
      { title: "Ende-zu-Ende-Verschlüsselung", desc: "Echtes E2EE für private Nachrichten (X3DH + Double Ratchet); Gruppen-E2EE mit Sender-Keys." },
      { title: "Bots", desc: "Bots mit Token und Webhook für Automatisierung." },
      { title: "Sprachen", desc: "Standardmäßig Englisch; Deutsch und andere Sprachen in den Einstellungen verfügbar." },
      { title: "Sehr geringer Datenverbrauch", desc: "Datensparmodus standardmäßig an: Sprachanrufe mit Opus 12 kbit/s (~0,09 MB/Min.), komprimierte Fotos, Download per Tippen." },
      { title: "Lokaler KI-Assistent", desc: "Assistent auf dem Gerät, Wörterbuch-Übersetzung und intelligente Antworten — keine Cloud-KI, deine Daten verlassen das Gerät nicht." },
    ],
    footer:
      "Nur kostenlos und Open Source, bis VYRO 10.000 registrierte Nutzer erreicht. Die Android-Beta-APK ist oben verfügbar; iOS kommt später.",
    aboutTitle: "Über VYRO",
    languageLabel: "Sprache",
    forYourPhone: "VYRO 0.6.0 — für dein Telefon",
    detected: "Für dein Telefon erkannt:",
    downloadButton: "VYRO für Android herunterladen",
    verify: "Download überprüfen (SHA-256):",
  },
  hi: {
    tagline: "सीमाओं से परे जुड़ें",
    intro:
      "VYRO Android के लिए एक तेज़ और सुरक्षित मैसेंजर है: निजी और ग्रुप चैट, चैनल, वॉइस कॉल, स्टोरीज़, बॉट्स और निजी संदेशों के लिए एंड-टू-एंड एन्क्रिप्शन। यह बीटा है — कुछ सुविधाएँ अभी आ रही हैं। मैसेंजर सिर्फ ऐप के अंदर है — चैट शुरू करने के लिए नीचे से डाउनलोड करें।",
    downloadTitle: "Android के लिए VYRO डाउनलोड करें",
    versionLine:
      "संस्करण 0.6.0 (बीटा) · Android 6.0+ (API 23) आवश्यक है · साइट आपके फ़ोन को अपने आप पहचानती है और उसके लिए बना बिल्ड डाउनलोड करती है — बस नीचे वाला बटन दबाएँ। इस पहले टेस्ट बिल्ड पर डेवलपमेंट कुंजी से हस्ताक्षर है; व्यापक वितरण से पहले आधिकारिक हस्ताक्षर आएगा।",
    warning:
      "VYRO सिर्फ इसी आधिकारिक वेबसाइट से डाउनलोड करें। अन्य साइटों की APK सुरक्षित नहीं हैं और उन्हें इंस्टॉल नहीं करना चाहिए।",
    featuresTitle: "सुविधाएँ",
    features: [
      { title: "निजी और ग्रुप चैट", desc: "एडमिन भूमिकाओं के साथ डायरेक्ट संदेश, ग्रुप और चैनल।" },
      { title: "रियल-टाइम चैट", desc: "WebSocket पर तुरंत संदेश, पढ़ने के निशान और टाइपिंग संकेतक।" },
      { title: "वॉइस और वीडियो कॉल", desc: "WebRTC पर वॉइस कॉल; वीडियो कॉल कम गुणवत्ता से शुरू होती हैं (डेटा सेवर)।" },
      { title: "फ़ोटो, फ़ाइलें और आवाज़", desc: "50 MB तक अपलोड, फ़ोटो और फ़ाइल संदेश, वॉइस संदेश।" },
      { title: "24 घंटे की स्टोरीज़", desc: "स्टोरीज़ और स्टेटस, जैसे आप जानते हैं।" },
      { title: "एंड-टू-एंड एन्क्रिप्शन", desc: "निजी संदेशों के लिए असली E2EE (X3DH + Double Ratchet); सेंडर कीज़ के साथ ग्रुप E2EE।" },
      { title: "बॉट्स", desc: "ऑटोमेशन के लिए टोकन और webhook वाले बॉट्स।" },
      { title: "भाषाएँ", desc: "डिफ़ॉल्ट रूप से अंग्रेज़ी; हिंदी और अन्य भाषाएँ सेटिंग्स में उपलब्ध हैं।" },
      { title: "बहुत कम डेटा उपयोग", desc: "डेटा सेवर डिफ़ॉल्ट रूप से चालू: Opus 12 kbps पर वॉइस कॉल (~0.09 MB/मिनट), संपीड़ित फ़ोटो, टैप करके डाउनलोड।" },
      { title: "स्थानीय AI सहायक", desc: "डिवाइस पर सहायक, शब्दकोश अनुवाद और स्मार्ट जवाब — कोई क्लाउड AI नहीं, आपका डेटा डिवाइस नहीं छोड़ता।" },
    ],
    footer:
      "सिर्फ़ मुफ़्त और ओपन-सोर्स, जब तक VYRO 10,000 पंजीकृत उपयोगकर्ताओं तक न पहुँचे। Android बीटा APK ऊपर उपलब्ध है; iOS बाद में आएगा।",
    aboutTitle: "VYRO के बारे में",
    languageLabel: "भाषा",
    forYourPhone: "VYRO 0.6.0 — आपके फ़ोन के लिए",
    detected: "आपके फ़ोन के लिए पहचाना गया:",
    downloadButton: "Android के लिए VYRO डाउनलोड करें",
    verify: "डाउनलोड सत्यापित करें (SHA-256):",
  },
};
