import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "Privacy Policy — Safari Messenger",
  description:
    "How Safari Messenger collects, uses and protects your information. The messenger lives only inside the mobile app; this website is only for downloading it.",
};

export default function PrivacyPolicyPage() {
  return (
    <main
      style={{
        maxWidth: 760,
        margin: "0 auto",
        padding: "48px 20px 64px",
        fontFamily: "system-ui, -apple-system, Segoe UI, Roboto, sans-serif",
        color: "#1e293b",
        lineHeight: 1.7,
      }}
    >
      <h1 style={{ fontSize: 32, marginBottom: 8 }}>Privacy Policy</h1>
      <p style={{ color: "#64748b", marginTop: 0 }}>
        Safari Messenger — last updated 14 November 2026.
      </p>

      <h2>What Safari is</h2>
      <p>
        Safari is a messenger app for Android. The messenger itself lives
        only inside the mobile app. This website is only for downloading the
        app; there is no chatting or messaging in the browser here.
      </p>

      <h2>Information we collect</h2>
      <ul>
        <li>
          <strong>Your email address</strong> — when you register or log in,
          we send a 6-digit verification code to the email address you
          provide. We store your email address and a one-way hash of your
          password (never the password itself).
        </li>
        <li>
          <strong>Your display name</strong> — the name you enter at
          registration. Your username is chosen automatically from your name.
        </li>
        <li>
          <strong>Messages, photos and files you send</strong> — stored on
          our servers so they can be delivered and synced across your
          devices. One-to-one text chats are end-to-end encrypted.
        </li>
        <li>
          <strong>Contacts you add and groups you join</strong> — stored so
          the app can show your chats.
        </li>
        <li>
          <strong>Technical data</strong> — such as your device type and
          push-notification token, only to deliver messages and
          notifications to you.
        </li>
      </ul>

      <h2>What we do not do</h2>
      <ul>
        <li>We do not sell your personal information to anyone.</li>
        <li>We do not show you ads based on the content of your messages.</li>
        <li>
          We do not share your data with advertisers. Your data is used only
          to run the messenger for you.
        </li>
      </ul>

      <h2>How we use your information</h2>
      <p>
        We use the information above only to operate Safari: to create your
        account, verify your email, deliver your messages and calls, keep
        your chats in sync, and notify you about new messages. We send only
        transactional emails — 6-digit verification codes for registration
        and login. We never send marketing emails, and we never buy, rent or
        use purchased email lists.
      </p>

      <h2>Data retention and deletion</h2>
      <p>
        Your messages and files stay on our servers while your account
        exists. You can delete your messages in the app, and you can ask us
        to delete your account and its data by writing to us at the address
        below.
      </p>

      <h2>Security</h2>
      <p>
        Passwords are stored only as one-way hashes. One-to-one text chats
        are end-to-end encrypted. We use industry-standard encryption for
        data in transit.
      </p>

      <h2>Children</h2>
      <p>
        Safari is not directed at children under 13. If you are under 13,
        please do not create an account.
      </p>

      <h2>Changes to this policy</h2>
      <p>
        If this policy changes, we will update this page and the date above.
      </p>

      <h2>Contact</h2>
      <p>
        Safari Messenger — contact us at{" "}
        <a href="mailto:najmuddinsafari00@gmail.com">
          najmuddinsafari00@gmail.com
        </a>
        .
      </p>

      <p style={{ marginTop: 40 }}>
        <a href="/landing" style={{ color: "#2563eb" }}>
          ← Back to download Safari
        </a>
      </p>
    </main>
  );
}
