// Command smtp-sink is a tiny local SMTP server for testing VYRO's
// email verification end to end, with no real mail account and no
// network access.
//
// It accepts mail on 127.0.0.1:2525 (no AUTH, no TLS) and appends each
// message's DATA — headers and body — to a file, so a test can read
// the 6-digit verification code back out of it.
//
// Usage:
//
//	go run ./scripts/smtp-sink -addr 127.0.0.1:2525 -out /tmp/vyro-mail.log
//
// Then start the server with:
//
//	SMTP_HOST=127.0.0.1 SMTP_PORT=2525 SMTP_ALLOW_NOAUTH=true \
//	  SMTP_FROM=vyro-test@example.invalid go run ./cmd/server
//
// and grep the code out of the log file:
//
//	grep -oE '[0-9]{6}' /tmp/vyro-mail.log | tail -1
//
// This is a test tool only: it accepts mail from anyone who can reach
// it, so bind it to loopback and never expose it.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:2525", "listen address")
	out := flag.String("out", "smtp-sink.log", "file to append received mail to")
	flag.Parse()

	ln, err := net.Listen("tcp", *addr)
	if err != nil {
		log.Fatalf("listen %s: %v", *addr, err)
	}
	log.Printf("smtp-sink listening on %s, appending mail to %s", *addr, *out)
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("accept: %v", err)
			continue
		}
		go handle(conn, *out)
	}
}

var fileMu sync.Mutex

func appendMail(path, data string) error {
	fileMu.Lock()
	defer fileMu.Unlock()
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = fmt.Fprintf(f, "=== %s ===\n%s\n", time.Now().UTC().Format(time.RFC3339), data)
	return err
}

// handle speaks just enough SMTP for net/smtp (and any real client):
// EHLO/HELO, MAIL FROM, RCPT TO, DATA, RSET, NOOP, QUIT. Anything
// else gets a 502. DATA is read until a line with a single ".".
func handle(conn net.Conn, outPath string) {
	defer conn.Close()
	defer func() {
		// A malformed client must never take the sink down.
		_ = recover()
	}()
	_ = conn.SetDeadline(time.Now().Add(2 * time.Minute))
	r := bufio.NewReader(conn)
	fmt.Fprintf(conn, "220 smtp-sink VYRO test sink ready\r\n")
	var data strings.Builder
	inData := false
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return
		}
		line = strings.TrimRight(line, "\r\n")
		if inData {
			if line == "." {
				inData = false
				if err := appendMail(outPath, data.String()); err != nil {
					log.Printf("append mail: %v", err)
					fmt.Fprintf(conn, "451 could not store message\r\n")
				} else {
					fmt.Fprintf(conn, "250 OK: message stored\r\n")
				}
				data.Reset()
				continue
			}
			// Undo dot-stuffing.
			data.WriteString(strings.TrimPrefix(line, ".") + "\n")
			continue
		}
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		verb := strings.ToUpper(fields[0])
		switch verb {
		case "EHLO", "HELO":
			fmt.Fprintf(conn, "250-smtp-sink\r\n250 8BITMIME\r\n")
		case "MAIL", "RCPT":
			fmt.Fprintf(conn, "250 OK\r\n")
		case "DATA":
			fmt.Fprintf(conn, "354 End data with <CR><LF>.<CR><LF>\r\n")
			inData = true
		case "RSET":
			data.Reset()
			fmt.Fprintf(conn, "250 OK\r\n")
		case "NOOP":
			fmt.Fprintf(conn, "250 OK\r\n")
		case "QUIT":
			fmt.Fprintf(conn, "221 Bye\r\n")
			return
		default:
			fmt.Fprintf(conn, "502 command not implemented\r\n")
		}
	}
}
