#!/usr/bin/env python3
"""Screenshot harness for the Safari download site (local test backend only).

Starts the SMTP sink (:1025) and the real Go backend (:8080, in-memory,
SMTP -> sink), seeds demo users/chats/stories, then serves the helper
API on :1099 that the integration test uses:

  GET /config            -> {"email": ..., "password": ...} (demo login)
  GET /code?email=<addr> -> {"code": "123456"} (latest code in smtp_sink.log)

Run in the background, then from mobile/:
  flutter drive -d web-server --browser-name=chrome \
    --chrome-binary=/opt/meta-chromium/chrome --web-port=3000 \
    --web-renderer=html --driver=test_driver/screenshots_driver.dart \
    --target=integration_test/screenshots_test.dart

Screenshots land in web/public/screenshots/. Stop this script when done;
nothing here ever leaves the machine.
"""
import json
import os
import re
import subprocess
import sys
import time
import urllib.request
import urllib.error
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from urllib.parse import urlparse, parse_qs

ROOT = "/home/hatch/workspace/vyro"
LOG = os.path.join(ROOT, "smtp_sink.log")
API = "http://127.0.0.1:8080"
DEMO1 = {"email": "demo1@vyro.test", "password": "DemoPass123!", "name": "Demo One"}
DEMO2 = {"email": "demo2@vyro.test", "password": "DemoPass123!", "name": "Demo Two"}


def api(method, path, token=None, body=None, timeout=15):
    req = urllib.request.Request(API + path, method=method)
    req.add_header("Content-Type", "application/json")
    if token:
        req.add_header("Authorization", "Bearer " + token)
    data = json.dumps(body).encode() if body is not None else None
    try:
        with urllib.request.urlopen(req, data=data, timeout=timeout) as r:
            return r.status, json.loads(r.read() or b"{}")
    except urllib.error.HTTPError as e:
        return e.code, json.loads(e.read() or b"{}")


def latest_code(email):
    """Latest 6-digit code emailed to `email`, from the sink log."""
    try:
        text = open(LOG, encoding="utf-8", errors="replace").read()
    except FileNotFoundError:
        return None
    # Split into per-message blocks; keep those addressed to this email.
    blocks = re.split(r"\n=== .*? ===\n", text)
    codes = []
    for b in blocks:
        if f"To: {email}" in b:
            m = re.search(r"verification code is: (\d{6})", b)
            if m:
                codes.append(m.group(1))
    return codes[-1] if codes else None


def wait_code(email, timeout=20):
    end = time.time() + timeout
    while time.time() < end:
        c = latest_code(email)
        if c:
            return c
        time.sleep(0.5)
    raise RuntimeError(f"no verification code arrived for {email}")


def register_and_verify(user):
    st, out = api("POST", "/api/auth/register", body={
        "email": user["email"], "password": user["password"],
        "display_name": user["name"]})
    assert st == 201, f"register {user['email']} -> {st} {out}"
    code = wait_code(user["email"])
    st, out = api("POST", "/api/auth/verify-register", body={
        "email": user["email"], "code": code})
    assert st in (200, 201), f"verify {user['email']} -> {st} {out}"
    print(f"registered {user['email']} as @{out['user']['username']}", flush=True)
    return out["user"]["id"], out["token"]


def upload(token, path):
    import uuid
    boundary = uuid.uuid4().hex
    data = open(path, "rb").read()
    body = (f"--{boundary}\r\n".encode()
            + f'Content-Disposition: form-data; name="file"; filename="{os.path.basename(path)}"\r\n'.encode()
            + b"Content-Type: image/png\r\n\r\n" + data
            + f"\r\n--{boundary}--\r\n".encode())
    req = urllib.request.Request(API + "/api/uploads", method="POST", data=body)
    req.add_header("Content-Type", f"multipart/form-data; boundary={boundary}")
    req.add_header("Authorization", "Bearer " + token)
    with urllib.request.urlopen(req, timeout=30) as r:
        return json.loads(r.read())["url"]


def seed():
    id1, tok1 = register_and_verify(DEMO1)
    id2, tok2 = register_and_verify(DEMO2)
    # DM Demo One <-> Demo Two.
    st, dm = api("POST", "/api/chats", token=tok1,
                 body={"type": "dm", "member_ids": [id2]})
    assert st in (200, 201), f"create DM -> {st} {dm}"
    dm_id = dm["id"]
    for tok, msg in [(tok1, "Hi Demo Two!"),
                     (tok1, "Perfect for slow internet. See you tomorrow!"),
                     (tok2, "See you tomorrow!")]:
        st, _ = api("POST", f"/api/chats/{dm_id}/messages", token=tok,
                    body={"content": msg})
        assert st in (200, 201), f"send message -> {st}"
    # Group "Weekend Plans".
    st, grp = api("POST", "/api/chats", token=tok1,
                  body={"type": "group", "member_ids": [id2],
                        "title": "Weekend Plans"})
    assert st in (200, 201), f"create group -> {st} {grp}"
    st, _ = api("POST", f"/api/chats/{grp['id']}/messages", token=tok1,
                body={"content": "Who is coming to the park on Saturday?"})
    assert st in (200, 201), f"group message -> {st}"
    # Story with a real uploaded image.
    url = upload(tok1, os.path.join(ROOT, "mobile/assets/safari-logo.png"))
    st, _ = api("POST", "/api/stories", token=tok1,
                body={"media_url": url, "caption": "Morning in Kabul ☀️"})
    assert st in (200, 201), f"create story -> {st} {_}"
    print("seeded demo data (DM, Weekend Plans group, story)", flush=True)


class Helper(BaseHTTPRequestHandler):
    def log_message(self, *a):
        pass

    def _json(self, obj, status=200):
        body = json.dumps(obj).encode()
        self.send_response(status)
        self.send_header("Content-Type", "application/json")
        # The app runs in Chrome (origin localhost:3000); without this
        # the browser blocks the cross-origin response (CORS).
        self.send_header("Access-Control-Allow-Origin", "*")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def do_OPTIONS(self):
        self.send_response(204)
        self.send_header("Access-Control-Allow-Origin", "*")
        self.send_header("Access-Control-Allow-Methods", "GET, OPTIONS")
        self.send_header("Access-Control-Allow-Headers", "Content-Type")
        self.send_header("Content-Length", "0")
        self.end_headers()

    def do_GET(self):
        u = urlparse(self.path)
        if u.path == "/config":
            self._json({"email": DEMO1["email"], "password": DEMO1["password"]})
        elif u.path == "/code":
            email = parse_qs(u.query).get("email", [""])[0]
            code = latest_code(email)
            self._json({"code": code} if code else {"code": None})
        else:
            self._json({"error": "not found"}, 404)


def main():
    env = dict(os.environ)
    env["PATH"] = os.path.expanduser("~/go/bin") + ":" + env.get("PATH", "")
    env.update({
        "PORT": "8080",
        "JWT_SECRET": os.urandom(16).hex(),
        "SMTP_HOST": "127.0.0.1", "SMTP_PORT": "1025",
        "SMTP_FROM": "safari@test.local", "SMTP_ALLOW_NOAUTH": "true",
    })
    # Fresh sink log per run: stale codes from earlier runs would
    # otherwise be picked up as the "latest" code by the test.
    open(LOG, "w").close()
    sink = subprocess.Popen(
        [sys.executable, os.path.join(ROOT, "smtp_sink.py"), "1025"],
        stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
    backend = subprocess.Popen(
        ["go", "run", "./cmd/server"], cwd=os.path.join(ROOT, "backend"),
        env=env, stdout=open("/tmp/screenshot-backend.log", "w"),
        stderr=subprocess.STDOUT)
    try:
        for _ in range(60):
            try:
                st, _ = api("GET", "/health", timeout=2)
                if st == 200:
                    break
            except Exception:
                pass
            time.sleep(1)
        else:
            raise RuntimeError("backend did not come up on :8080")
        print("backend up on :8080, seeding...", flush=True)
        seed()
        srv = ThreadingHTTPServer(("127.0.0.1", 1099), Helper)
        print("helper serving on :1099 — ready for flutter drive", flush=True)
        srv.serve_forever()
    finally:
        backend.terminate()
        sink.terminate()


if __name__ == "__main__":
    main()
