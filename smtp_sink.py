"""Minimal local SMTP sink for VYRO screenshot testing.

Listens on 127.0.0.1:1025, accepts any mail, and appends the full
message (including the 6-digit verification code) to smtp_sink.log.
No mail ever leaves this machine.
"""
import socketserver
import sys
from datetime import datetime

LOG = "/home/hatch/workspace/vyro/smtp_sink.log"


class Handler(socketserver.StreamRequestHandler):
    def handle(self):
        self.wfile.write(b"220 vyro-test-sink ESMTP ready\r\n")
        in_data = False
        buf = []
        while True:
            line = self.rfile.readline()
            if not line:
                return
            text = line.decode("utf-8", "replace").strip()
            upper = text.upper()
            if in_data:
                if text == ".":
                    in_data = False
                    with open(LOG, "a", encoding="utf-8") as f:
                        f.write(f"\n=== {datetime.now().isoformat()} ===\n")
                        f.write("\n".join(buf) + "\n")
                    buf = []
                    self.wfile.write(b"250 OK\r\n")
                else:
                    buf.append(text)
                continue
            if upper.startswith("EHLO") or upper.startswith("HELO"):
                self.wfile.write(b"250-Hello\r\n250 AUTH LOGIN PLAIN\r\n")
            elif upper.startswith("MAIL FROM"):
                self.wfile.write(b"250 OK\r\n")
            elif upper.startswith("RCPT TO"):
                self.wfile.write(b"250 OK\r\n")
            elif upper == "DATA":
                in_data = True
                self.wfile.write(b"354 End data with <CR><LF>.<CR><LF>\r\n")
            elif upper == "QUIT":
                self.wfile.write(b"221 Bye\r\n")
                return
            elif upper == "RSET":
                self.wfile.write(b"250 OK\r\n")
            elif upper == "NOOP":
                self.wfile.write(b"250 OK\r\n")
            elif upper.startswith("AUTH"):
                self.wfile.write(b"235 OK\r\n")
            else:
                self.wfile.write(b"250 OK\r\n")


if __name__ == "__main__":
    port = int(sys.argv[1]) if len(sys.argv) > 1 else 1025
    with socketserver.ThreadingTCPServer(("127.0.0.1", port), Handler) as srv:
        srv.daemon_threads = True
        print(f"smtp sink listening on 127.0.0.1:{port}", flush=True)
        srv.serve_forever()
