// Join-page strings. The whole page translates, like the download
// page: the visitor's language choice is shared with the landing
// page through the same localStorage key (vyro-site-lang).
import type { LangCode } from "./translations";

export type JoinStrings = {
  joinGroup: string;
  joinChannel: string;
  membersLabel: string;
  invitedBy: string;
  downloadToJoin: string;
  haveApp: string;
  invalidTitle: string;
  invalidBody: string;
};

export const joinTranslations: Record<LangCode, JoinStrings> = {
  en: {
    joinGroup: "Join this group",
    joinChannel: "Join this channel",
    membersLabel: "Members",
    invitedBy: "Invited by",
    downloadToJoin: "Download Safari to join",
    haveApp:
      "Already have Safari? In the app, open Chats, tap Join via invite link, and paste this link.",
    invalidTitle: "Invite not found",
    invalidBody: "This invite link is invalid or has been revoked.",
  },
  fa: {
    joinGroup: "به این گروه بپیوندید",
    joinChannel: "به این کانال بپیوندید",
    membersLabel: "اعضا",
    invitedBy: "دعوت‌کننده",
    downloadToJoin: "برای پیوستن، Safari را دانلود کنید",
    haveApp:
      "Safari را دارید؟ در برنامه، گفتگوها را باز کنید، «پیوستن با لینک دعوت» را بزنید و این لینک را بچسبانید.",
    invalidTitle: "دعوت پیدا نشد",
    invalidBody: "این لینک دعوت نامعتبر است یا لغو شده است.",
  },
  ar: {
    joinGroup: "انضم إلى هذه المجموعة",
    joinChannel: "انضم إلى هذه القناة",
    membersLabel: "الأعضاء",
    invitedBy: "دعاك",
    downloadToJoin: "نزّل Safari للانضمام",
    haveApp:
      "هل لديك Safari؟ افتح التطبيق، ثم الدردشات، واضغط على «الانضمام عبر رابط دعوة» والصق هذا الرابط.",
    invalidTitle: "الدعوة غير موجودة",
    invalidBody: "رابط الدعوة هذا غير صالح أو تم إلغاؤه.",
  },
  ur: {
    joinGroup: "اس گروپ میں شامل ہوں",
    joinChannel: "اس چینل میں شامل ہوں",
    membersLabel: "ارکان",
    invitedBy: "دعوت دینے والا",
    downloadToJoin: "شامل ہونے کے لیے Safari ڈاؤن لوڈ کریں",
    haveApp:
      "کیا آپ کے پاس Safari ہے؟ ایپ میں چیٹس کھولیں، «دعوت لنک سے شامل ہوں» پر ٹیپ کریں اور یہ لنک پیسٹ کریں۔",
    invalidTitle: "دعوت نہیں ملی",
    invalidBody: "یہ دعوت لنک غلط ہے یا منسوخ کر دیا گیا ہے۔",
  },
  tr: {
    joinGroup: "Bu gruba katıl",
    joinChannel: "Bu kanala katıl",
    membersLabel: "Üyeler",
    invitedBy: "Davet eden",
    downloadToJoin: "Katılmak için Safari'yi indir",
    haveApp:
      "Safari zaten var mı? Uygulamada Sohbetler'i aç, «Davet bağlantısıyla katıl»a dokun ve bu bağlantıyı yapıştır.",
    invalidTitle: "Davet bulunamadı",
    invalidBody: "Bu davet bağlantısı geçersiz veya iptal edilmiş.",
  },
  ru: {
    joinGroup: "Вступить в эту группу",
    joinChannel: "Вступить в этот канал",
    membersLabel: "Участники",
    invitedBy: "Пригласил",
    downloadToJoin: "Скачайте Safari, чтобы вступить",
    haveApp:
      "Safari уже установлен? В приложении откройте Чаты, нажмите «Вступить по ссылке-приглашению» и вставьте эту ссылку.",
    invalidTitle: "Приглашение не найдено",
    invalidBody: "Эта ссылка-приглашение недействительна или отозвана.",
  },
  es: {
    joinGroup: "Unirse a este grupo",
    joinChannel: "Unirse a este canal",
    membersLabel: "Miembros",
    invitedBy: "Invitado por",
    downloadToJoin: "Descarga Safari para unirte",
    haveApp:
      "¿Ya tienes Safari? En la app, abre Chats, toca «Unirse con enlace de invitación» y pega este enlace.",
    invalidTitle: "Invitación no encontrada",
    invalidBody: "Este enlace de invitación no es válido o ha sido revocado.",
  },
  fr: {
    joinGroup: "Rejoindre ce groupe",
    joinChannel: "Rejoindre cette chaîne",
    membersLabel: "Membres",
    invitedBy: "Invité par",
    downloadToJoin: "Téléchargez Safari pour rejoindre",
    haveApp:
      "Vous avez déjà Safari ? Dans l'app, ouvrez Discussions, touchez « Rejoindre via un lien d'invitation » et collez ce lien.",
    invalidTitle: "Invitation introuvable",
    invalidBody: "Ce lien d'invitation est invalide ou a été révoqué.",
  },
  de: {
    joinGroup: "Dieser Gruppe beitreten",
    joinChannel: "Diesem Kanal beitreten",
    membersLabel: "Mitglieder",
    invitedBy: "Eingeladen von",
    downloadToJoin: "Safari herunterladen, um beizutreten",
    haveApp:
      "Hast du Safari schon? Öffne in der App Chats, tippe auf «Über Einladungslink beitreten» und füge diesen Link ein.",
    invalidTitle: "Einladung nicht gefunden",
    invalidBody: "Dieser Einladungslink ist ungültig oder wurde widerrufen.",
  },
  hi: {
    joinGroup: "इस ग्रुप से जुड़ें",
    joinChannel: "इस चैनल से जुड़ें",
    membersLabel: "सदस्य",
    invitedBy: "द्वारा आमंत्रित",
    downloadToJoin: "जुड़ने के लिए Safari डाउनलोड करें",
    haveApp:
      "क्या आपके पास Safari है? ऐप में चैट खोलें, «निमंत्रण लिंक से जुड़ें» पर टैप करें और यह लिंक पेस्ट करें।",
    invalidTitle: "निमंत्रण नहीं मिला",
    invalidBody: "यह निमंत्रण लिंक अमान्य है या रद्द कर दिया गया है।",
  },
  ps: {
    joinGroup: "له دې ډلې سره یوځای شئ",
    joinChannel: "له دې چینل سره یوځای شئ",
    membersLabel: "غړي",
    invitedBy: "بلونکی",
    downloadToJoin: "د یوځای کېدو لپاره Safari ډاونلوډ کړئ",
    haveApp:
      "ایا Safari لرئ؟ په اپ کې چیټونه پرانیزئ، «د بلنې لینک له لارې یوځای شئ» ټیپ کړئ او دا لینک پیست کړئ.",
    invalidTitle: "بلنه ونه موندل شوه",
    invalidBody: "دا د بلنې لینک ناسم دی یا لغوه شوی دی.",
  },
};
