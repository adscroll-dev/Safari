// Safari local AI for mobile (v0.6) — a faithful Dart port of
// web/lib/local-ai.ts. HONEST LABELING:
// - There is NO cloud AI and NO paid translation API behind this.
// - The assistant is a small rule-based responder that answers
//   questions about Safari itself and a few general prompts. It runs
//   entirely on the device.
// - Translation is a small built-in English<->Persian dictionary
//   (common words and phrases). It is NOT a full translator: unknown
//   words are left unchanged and `coverage` reports how much of the
//   input was actually in the dictionary, so the UI never pretends a
//   translation is complete when it is not.
// - Smart replies are short rule-based suggestions for the last
//   incoming message, generated on-device.
// Nothing in this file makes any network request.

enum AiLang { en, fa }

AiLang detectLang(String text) =>
    RegExp(r'[\u0600-\u06FF]', caseSensitive: false).hasMatch(text)
        ? AiLang.fa
        : AiLang.en;

class _Rule {
  _Rule(this.match, this.en, this.fa);
  final RegExp match;
  final String en;
  final String fa;
}

final List<_Rule> _rules = [
  _Rule(
    // No \b after the greeting: Dart's \b is ASCII-only, so it fails
    // after Persian letters. The lookahead stops at the next letter.
    RegExp(r'^(hi|hello|hey|salam|سلام|درود)(?![a-zA-Z\u0600-\u06FF])',
        caseSensitive: false),
    "Hello! I'm the local Safari assistant (on-device, rule-based — not a cloud AI). Ask me how to use Safari, or type \"help\".",
    "سلام! من دستیار محلی سافاری هستم (روی دستگاه، مبتنی بر قانون — هوش مصنوعی ابری نیست). بپرسید چطور از سافاری استفاده کنید، یا «help» بنویسید.",
  ),
  _Rule(
    RegExp(r'voice|صوتی|میکروفون|میکروفن|ضبط', caseSensitive: false),
    "Voice messages: tap the 🎤 button next to the message box, allow the microphone, tap again to stop, then send. Voice messages are sent through the normal upload API — note they are NOT end-to-end encrypted, even in an encrypted chat.",
    "پیام صوتی: دکمه 🎤 کنار کادر پیام را بزنید، به میکروفون اجازه دهید، دوباره بزنید تا ضبط متوقف شود، بعد ارسال کنید. پیام صوتی از طریق آپلود معمولی ارسال می‌شود — توجه: حتی در گفت‌وگوی رمزگذاری‌شده هم سرتاسری رمزگذاری نمی‌شود.",
  ),
  _Rule(
    RegExp(r'help|راهنما|کمک|چه کار|چی کار|چطور|چگونه|how (do|can|to)',
        caseSensitive: false),
    "I can help with Safari: send text, photos, files, voice messages (🎤 button), stories, voice calls, disappearing messages, scheduled messages, search, and end-to-end encrypted DMs (🔒 button). What would you like to do?",
    "می‌توانم در استفاده از سافاری کمک کنم: ارسال متن، عکس، فایل، پیام صوتی (دکمه 🎤)، استوری، تماس صوتی، پیام‌های ناپدیدشونده، پیام زمان‌بندی‌شده، جست‌وجو و گفت‌وگوی رمزگذاری‌شده سرتاسری (دکمه 🔒). چه کاری می‌خواهید انجام دهید؟",
  ),
  _Rule(
    RegExp(r'encrypt|رمز|امن|e2ee|lock|قفل', caseSensitive: false),
    "End-to-end encryption: open a DM and tap the 🔓 button to turn it on. Keys are generated on your device; the server only ever sees ciphertext. In v0.6 it works for 1:1 DMs and groups (Sender Keys).",
    "رمزگذاری سرتاسری: یک گفت‌وگوی دونفره را باز کنید و دکمه 🔓 را بزنید. کلیدها روی دستگاه شما ساخته می‌شوند و سرور فقط متن رمزشده را می‌بیند. در نسخه ۰٫۶ برای گفت‌وگوی دونفره و گروه‌ها (کلید فرستنده) کار می‌کند.",
  ),
  _Rule(
    RegExp(r'call|تماس|زنگ', caseSensitive: false),
    "Voice calls: open a DM and tap 📞. Calls use WebRTC directly between the two devices (audio first, video starts low quality). Data Saver caps voice at Opus 12 kbps — about 0.09 MB/min.",
    "تماس صوتی: یک گفت‌وگوی دونفره را باز کنید و 📞 را بزنید. تماس با WebRTC مستقیم بین دو دستگاه برقرار می‌شود (اول صوتی، ویدیو با کیفیت پایین شروع می‌شود). ذخیره داده صدا را روی ۱۲ کیلوبیت بر ثانیه محدود می‌کند — حدود ۰٫۰۹ مگابایت در دقیقه.",
  ),
  _Rule(
    RegExp(r'stor(y|ies)|استوری|وضعیت', caseSensitive: false),
    "Stories: open the Stories tab and tap ＋, pick a photo, and it stays visible for 24 hours.",
    "استوری: تب استوری را باز کنید و ＋ را بزنید، عکس انتخاب کنید؛ ۲۴ ساعت نمایش داده می‌شود.",
  ),
  _Rule(
    RegExp(r'disappear|ناپدید|محو|خود.?تخریب', caseSensitive: false),
    "Disappearing messages: tap ＋ next to the send box, choose the timer — 1 minute, 1 hour or 1 day — before sending. The message expires after that time.",
    "پیام ناپدیدشونده: پیش از ارسال، ＋ کنار دکمه ارسال را بزنید و زمان‌سنج را انتخاب کنید — ۱ دقیقه، ۱ ساعت یا ۱ روز. پیام پس از آن زمان منقضی می‌شود.",
  ),
  _Rule(
    RegExp(r'schedul|زمان.?بندی|بعدا|بعداً', caseSensitive: false),
    "Scheduled messages: type your message, then tap ＋ and choose Schedule, and enter in how many minutes it should be sent.",
    "پیام زمان‌بندی‌شده: پیام را بنویسید، بعد ＋ را بزنید و زمان‌بندی را انتخاب کنید و بگویید بعد از چند دقیقه ارسال شود.",
  ),
  _Rule(
    RegExp(r'translate|ترجمه', caseSensitive: false),
    "Translation: tap 🌐 under any text message to translate it with the built-in local dictionary (English ↔ Persian). It only knows common words — it is not a full translator, and the result shows how much it actually covered.",
    "ترجمه: زیر هر پیام متنی 🌐 را بزنید تا با فرهنگ لغت محلی (انگلیسی ↔ فارسی) ترجمه شود. فقط واژه‌های رایج را می‌داند — مترجم کامل نیست و نتیجه نشان می‌دهد چقدر پوشش داده است.",
  ),
  _Rule(
    RegExp(r'who are you|what are you|safari|سافاری|تو کی|چی هستی',
        caseSensitive: false),
    "Safari is a free messenger — 'Connect Beyond Limits'. This assistant is a small on-device helper, not a cloud AI: it only knows how Safari works.",
    "سافاری یک پیام‌رسان رایگان است — «فراتر از مرزها وصل شو». این دستیار یک کمک‌کننده کوچک روی دستگاه است، نه هوش مصنوعی ابری: فقط نحوه کار سافاری را می‌داند.",
  ),
  _Rule(
    RegExp(r'thank|مرسی|ممنون|تشکر', caseSensitive: false),
    "You're welcome! Anything else about Safari?",
    "خواهش می‌کنم! چیز دیگری درباره سافاری هست؟",
  ),
  _Rule(
    RegExp(r'bye|خداحافظ|بای', caseSensitive: false),
    "Goodbye! You can reopen me anytime with the 🤖 button.",
    "خداحافظ! هر وقت خواستید با دکمه 🤖 دوباره بازم کنید.",
  ),
];

const _fallbackEn =
    "I'm a small local assistant and I only know how Safari works — I can't answer general questions or browse the web. Try \"help\", or ask about voice messages, encryption, calls, stories, disappearing or scheduled messages.";
const _fallbackFa =
    "من یک دستیار محلی کوچک هستم و فقط نحوه کار سافاری را می‌دانم — نمی‌توانم به سؤال‌های عمومی پاسخ دهم یا در وب بگردم. «help» را امتحان کنید یا درباره پیام صوتی، رمزگذاری، تماس، استوری، پیام ناپدیدشونده یا زمان‌بندی‌شده بپرسید.";

({String text, AiLang lang}) assistantReply(String input) {
  final lang = detectLang(input);
  final q = input.trim();
  for (final rule in _rules) {
    if (rule.match.hasMatch(q)) {
      return (text: lang == AiLang.fa ? rule.fa : rule.en, lang: lang);
    }
  }
  return (text: lang == AiLang.fa ? _fallbackFa : _fallbackEn, lang: lang);
}

// ---- Local dictionary translation (EN <-> FA) ---------------------

const Map<String, String> _enFa = {
  'hello': 'سلام',
  'hi': 'سلام',
  'hey': 'سلام',
  'goodbye': 'خداحافظ',
  'bye': 'خداحافظ',
  'please': 'لطفاً',
  'thanks': 'ممنون',
  'thank': 'ممنون',
  'thank you': 'ممنونم',
  'yes': 'بله',
  'no': 'نه',
  'ok': 'باشه',
  'okay': 'باشه',
  'good': 'خوب',
  'bad': 'بد',
  'morning': 'صبح',
  'night': 'شب',
  'today': 'امروز',
  'tomorrow': 'فردا',
  'yesterday': 'دیروز',
  'friend': 'دوست',
  'love': 'عشق',
  'happy': 'خوشحال',
  'sad': 'ناراحت',
  'beautiful': 'زیبا',
  'how': 'چطور',
  'are': 'هستی',
  'you': 'تو',
  'i': 'من',
  'we': 'ما',
  'they': 'آنها',
  'what': 'چه',
  'where': 'کجا',
  'when': 'کی',
  'why': 'چرا',
  'who': 'کی',
  'is': 'است',
  'am': 'هستم',
  'the': '',
  'a': '',
  'an': '',
  'message': 'پیام',
  'chat': 'گفت‌وگو',
  'call': 'تماس',
  'voice': 'صوتی',
  'photo': 'عکس',
  'video': 'ویدیو',
  'file': 'فایل',
  'send': 'ارسال',
  'help': 'کمک',
  'time': 'زمان',
  'day': 'روز',
  'week': 'هفته',
  'month': 'ماه',
  'year': 'سال',
  'home': 'خانه',
  'work': 'کار',
  'school': 'مدرسه',
  'book': 'کتاب',
  'water': 'آب',
  'food': 'غذا',
  'mother': 'مادر',
  'father': 'پدر',
  'brother': 'برادر',
  'sister': 'خواهر',
  'boy': 'پسر',
  'girl': 'دختر',
  'man': 'مرد',
  'woman': 'زن',
  'child': 'بچه',
  'house': 'خانه',
  'car': 'ماشین',
  'phone': 'تلفن',
  'computer': 'کامپیوتر',
  'good morning': 'صبح بخیر',
  'good night': 'شب بخیر',
  'good afternoon': 'ظهر بخیر',
  'how are you': 'چطوری',
  'thank you very much': 'خیلی ممنون',
  'welcome': 'خوش آمدید',
  'sorry': 'ببخشید',
  'excuse': 'ببخشید',
  'me': 'من',
  'my': 'من',
  'your': 'تو',
  'his': 'او',
  'her': 'او',
  'its': 'آن',
  'and': 'و',
  'or': 'یا',
  'but': 'اما',
  'with': 'با',
  'without': 'بدون',
  'in': 'در',
  'on': 'روی',
  'at': 'در',
  'to': 'به',
  'from': 'از',
  'for': 'برای',
  'new': 'جدید',
  'old': 'قدیمی',
  'big': 'بزرگ',
  'small': 'کوچک',
  'hot': 'گرم',
  'cold': 'سرد',
  'fast': 'سریع',
  'slow': 'آهسته',
  'easy': 'آسان',
  'hard': 'سخت',
  'right': 'درست',
  'wrong': 'اشتباه',
  'open': 'باز',
  'close': 'بسته',
  'start': 'شروع',
  'end': 'پایان',
  'come': 'بیا',
  'go': 'برو',
  'see': 'ببین',
  'hear': 'بشنو',
  'speak': 'صحبت کن',
  'eat': 'بخور',
  'drink': 'بنوش',
  'sleep': 'بخواب',
  'wake': 'بیدار شو',
  'read': 'بخوان',
  'write': 'بنویس',
  'listen': 'گوش کن',
  'look': 'نگاه کن',
  'play': 'بازی کن',
};

final Map<String, String> _faEn = () {
  final out = <String, String>{};
  for (final e in _enFa.entries) {
    if (e.value.isNotEmpty && !out.containsKey(e.value)) {
      out[e.value] = e.key;
    }
  }
  return out;
}();

class Translation {
  const Translation({
    required this.text,
    required this.coverage,
    required this.from,
    required this.to,
    required this.knownWords,
    required this.totalWords,
  });
  final String text;
  final double coverage; // 0..1 of words found in the dictionary
  final AiLang from;
  final AiLang to;
  final int knownWords;
  final int totalWords;
}

Translation translateLocal(String text, AiLang to) {
  final from = to == AiLang.fa ? AiLang.en : AiLang.fa;
  final dict = to == AiLang.fa ? _enFa : _faEn;
  final phrases = dict.keys.where((k) => k.contains(' ')).toList()
    ..sort((a, b) => b.length.compareTo(a.length));
  var out = text;
  for (final p in phrases) {
    final re = RegExp(RegExp.escape(p), caseSensitive: false);
    out = out.replaceAll(re, dict[p]!);
  }
  final words = out.split(RegExp(r'(\s+)', caseSensitive: false));
  final translated = words.map((w) {
    if (!RegExp(r'\S', caseSensitive: false).hasMatch(w)) return w;
    final m = RegExp(
            r'^([^A-Za-z\u0600-\u06FF]*)([A-Za-z\u0600-\u06FF]+)([^A-Za-z\u0600-\u06FF]*)$')
        .firstMatch(w);
    if (m == null) return w;
    final pre = m.group(1)!;
    final core = m.group(2)!;
    final post = m.group(3)!;
    final key = to == AiLang.fa ? core.toLowerCase() : core;
    final hit = dict[key];
    if (hit != null && hit.isNotEmpty) {
      return pre + hit + post;
    }
    return w;
  }).join('');
  // Coverage is measured on the ORIGINAL text: words inside a matched
  // phrase count as known, otherwise a word counts if it is in the
  // dictionary. (Phrase replacement turns words into the other
  // language, which would otherwise count as "unknown".)
  final origWords =
      text.split(RegExp(r'\s+')).where((w) => w.isNotEmpty).toList();
  final lowerText = text.toLowerCase();
  final covered = <int>{};
  for (final p in phrases) {
    var from = 0;
    while (true) {
      final at = lowerText.indexOf(p, from);
      if (at < 0) break;
      final wordIdx = text
          .substring(0, at)
          .split(RegExp(r'\s+'))
          .where((w) => w.isNotEmpty)
          .length;
      for (var k = 0; k < p.split(' ').length; k++) {
        covered.add(wordIdx + k);
      }
      from = at + p.length;
    }
  }
  var total = 0;
  var known = 0;
  for (var i = 0; i < origWords.length; i++) {
    total++;
    if (covered.contains(i)) {
      known++;
      continue;
    }
    final m = RegExp(
            r'^([^A-Za-z\u0600-\u06FF]*)([A-Za-z\u0600-\u06FF]+)([^A-Za-z\u0600-\u06FF]*)$')
        .firstMatch(origWords[i]);
    if (m == null) continue;
    final key = to == AiLang.fa ? m.group(2)!.toLowerCase() : m.group(2)!;
    if (dict.containsKey(key) && dict[key]!.isNotEmpty) known++;
  }
  return Translation(
    text: translated,
    coverage: total == 0 ? 0 : known / total,
    from: from,
    to: to,
    knownWords: known,
    totalWords: total,
  );
}

// ---- Smart replies (local, rule-based) ----------------------------

/// Up to 3 quick replies for the last incoming message. Returns an
/// empty list when nothing sensible matches — the UI then shows no
/// suggestion chips rather than a fake one.
List<String> smartReplies(String lastMessage, AiLang lang) {
  final q = lastMessage.trim().toLowerCase();
  if (RegExp(r'^(hi|hello|hey|salam|سلام|درود)(?![a-zA-Z\u0600-\u06FF])',
          caseSensitive: false)
      .hasMatch(q)) {
    return lang == AiLang.fa
        ? ['سلام! 👋', 'سلام، چطوری؟', 'خوشحالم که پیام دادی!']
        : ['Hello! 👋', 'Hi, how are you?', 'Hey there!'];
  }
  if (q.contains('how are you') || q.contains('چطوری')) {
    return lang == AiLang.fa
        ? ['خوبم، ممنون! تو چطوری؟', 'عالی‌ام! 😊', 'خوبم، ممنون']
        : ["I'm good, thanks!", 'Doing great, you?', 'All good here!'];
  }
  if (RegExp(r'thank|thanks|ممنون|مرسی|تشکر', caseSensitive: false)
      .hasMatch(q)) {
    return lang == AiLang.fa
        ? ['خواهش می‌کنم! 😊', 'قابلی نداشت!']
        : ["You're welcome! 😊", 'Anytime!'];
  }
  if (RegExp(r'bye|goodbye|خداحافظ|بای', caseSensitive: false).hasMatch(q)) {
    return lang == AiLang.fa
        ? ['خداحافظ! 👋', 'به‌زودی صحبت می‌کنیم!']
        : ['Goodbye! 👋', 'Talk soon!'];
  }
  if (q.endsWith('?') || q.endsWith('؟')) {
    return lang == AiLang.fa
        ? ['بذار بررسی کنم.', 'سؤال خوبی است!', 'بله، حتماً.']
        : ['Let me check.', 'Good question!', 'Sure thing.'];
  }
  if (q.isEmpty) return [];
  return lang == AiLang.fa
      ? ['👍', 'باشه', 'متشکرم!']
      : ['👍', 'Got it.', 'Thanks!'];
}
