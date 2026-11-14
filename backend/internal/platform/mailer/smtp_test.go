package mailer

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"net"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"strings"
	"testing"
	"time"
	"wbs-golang/backend/internal/config"
	"wbs-golang/backend/internal/modules/notifications"
)

func TestSMTPRejectsUnsafeAndUnavailable(t *testing.T) {
	c := New(config.SMTPConfig{Enabled: true, Host: "127.0.0.1", Port: "1", Timeout: 100 * time.Millisecond})
	if _, err := c.Send(context.Background(), notifications.Message{Recipient: "not an email"}); notifications.Classify(err).Code != "RECIPIENT_INVALID" {
		t.Fatal(err)
	}
	if _, err := c.Send(context.Background(), notifications.Message{Recipient: "qa@example.test", Subject: "x\r\nBcc: evil"}); notifications.Classify(err).Code != "SUBJECT_INVALID" {
		t.Fatal(err)
	}
	if _, err := c.Send(context.Background(), notifications.Message{Recipient: "qa@example.test"}); !notifications.Classify(err).Retryable {
		t.Fatal(err)
	}
	for _, code := range []int{450, 550} {
		if notifications.Classify(safe(&textproto.Error{Code: code, Msg: "private@example.test"})).Retryable != (code == 450) {
			t.Fatal(code)
		}
	}
	// A local cleartext SMTP server without STARTTLS is refused, never downgraded.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	go func() {
		conn, e := ln.Accept()
		if e != nil {
			return
		}
		defer conn.Close()
		_ = conn.SetDeadline(time.Now().Add(time.Second))
		text := textproto.NewConn(conn)
		_ = text.PrintfLine("220 localhost SMTP")
		_, _ = text.ReadLine()
		_ = text.PrintfLine("250 localhost")
		_, _ = text.ReadLine()
	}()
	host, port, _ := net.SplitHostPort(ln.Addr().String())
	c = New(config.SMTPConfig{Enabled: true, Host: host, Port: port, Timeout: time.Second, TLSMode: "starttls"})
	if _, err = c.Send(context.Background(), notifications.Message{Recipient: "qa@example.test"}); notifications.Classify(err).Code != "SMTP_TLS_REQUIRED" {
		t.Fatal(err)
	}
}

func TestSMTPLocalVerifiedTLSDelivery(t *testing.T) {
	fixture := httptest.NewTLSServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	certificate := fixture.TLS.Certificates[0]
	fixture.Close()
	parsed, err := x509.ParseCertificate(certificate.Certificate[0])
	if err != nil {
		t.Fatal(err)
	}
	roots := x509.NewCertPool()
	roots.AddCert(parsed)
	for _, mode := range []string{"implicit", "starttls"} {
		t.Run(mode, func(t *testing.T) {
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}
			defer listener.Close()
			received := make(chan string, 1)
			go func() {
				conn, e := listener.Accept()
				if e != nil {
					return
				}
				defer conn.Close()
				_ = conn.SetDeadline(time.Now().Add(3 * time.Second))
				serverTLS := &tls.Config{Certificates: []tls.Certificate{certificate}, MinVersion: tls.VersionTLS12}
				if mode == "implicit" {
					conn = tls.Server(conn, serverTLS)
				}
				text := textproto.NewConn(conn)
				_ = text.PrintfLine("220 localhost SMTP")
				var message strings.Builder
				for {
					line, e := text.ReadLine()
					if e != nil {
						return
					}
					switch {
					case strings.HasPrefix(line, "EHLO"):
						if mode == "starttls" {
							_ = text.PrintfLine("250-localhost\r\n250 STARTTLS")
						} else {
							_ = text.PrintfLine("250 localhost")
						}
					case line == "STARTTLS":
						_ = text.PrintfLine("220 ready")
						conn = tls.Server(conn, serverTLS)
						text = textproto.NewConn(conn)
					case strings.HasPrefix(line, "MAIL FROM:"), strings.HasPrefix(line, "RCPT TO:"):
						_ = text.PrintfLine("250 OK")
					case line == "DATA":
						_ = text.PrintfLine("354 send data")
						for {
							v, e := text.ReadLine()
							if e != nil {
								return
							}
							if v == "." {
								break
							}
							message.WriteString(v + "\n")
						}
						_ = text.PrintfLine("250 accepted")
						received <- message.String()
					case line == "QUIT":
						_ = text.PrintfLine("221 bye")
						return
					default:
						_ = text.PrintfLine("500 unsupported")
					}
				}
			}()
			host, port, _ := net.SplitHostPort(listener.Addr().String())
			c := New(config.SMTPConfig{Enabled: true, Host: host, Port: port, FromAddress: "wbs@example.test", FromName: "WBS", TLSMode: mode, Timeout: 2 * time.Second})
			c.roots = roots
			id, err := c.Send(context.Background(), notifications.Message{Recipient: "qa@example.test", Subject: "Ticket WBS", HTML: "<p>Safe</p>", DeliveryID: "00000000-0000-0000-0000-000000000001"})
			if err != nil || id == "" {
				t.Fatal(err)
			}
			select {
			case raw := <-received:
				if !strings.Contains(raw, "Content-Transfer-Encoding: base64") || !strings.Contains(raw, "Message-ID:") {
					t.Fatal("invalid MIME")
				}
			case <-time.After(time.Second):
				t.Fatal("no local delivery")
			}
		})
	}
}
