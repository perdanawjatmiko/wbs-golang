package mailer

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"fmt"
	"mime"
	"net"
	"net/mail"
	"net/smtp"
	"net/textproto"
	"strings"
	"time"
	"wbs-golang/backend/internal/config"
	"wbs-golang/backend/internal/modules/notifications"
)

type Client struct {
	config config.SMTPConfig
	roots  *x509.CertPool
}

func New(c config.SMTPConfig) *Client { return &Client{config: c} }
func safe(err error) error {
	var e *textproto.Error
	if errors.As(err, &e) {
		return &notifications.Failure{Code: fmt.Sprintf("SMTP_%d", e.Code), Retryable: e.Code >= 400 && e.Code < 500}
	}
	return &notifications.Failure{Code: "SMTP_UNAVAILABLE", Retryable: true}
}
func (c *Client) Send(ctx context.Context, m notifications.Message) (string, error) {
	if !c.config.Enabled {
		return "", &notifications.Failure{Code: "CHANNEL_DISABLED"}
	}
	recipient, err := mail.ParseAddress(m.Recipient)
	if err != nil || recipient.Address != m.Recipient || strings.ContainsAny(m.Recipient, "\r\n") {
		return "", &notifications.Failure{Code: "RECIPIENT_INVALID"}
	}
	if strings.ContainsAny(m.Subject, "\r\n") {
		return "", &notifications.Failure{Code: "SUBJECT_INVALID"}
	}
	ctx, cancel := context.WithTimeout(ctx, c.config.Timeout)
	defer cancel()
	address := net.JoinHostPort(c.config.Host, c.config.Port)
	tlsConfig := &tls.Config{ServerName: c.config.Host, MinVersion: tls.VersionTLS12, RootCAs: c.roots}
	dialer := &net.Dialer{Timeout: c.config.Timeout}
	var conn net.Conn
	if c.config.TLSMode == "implicit" {
		conn, err = (&tls.Dialer{NetDialer: dialer, Config: tlsConfig}).DialContext(ctx, "tcp", address)
	} else {
		conn, err = dialer.DialContext(ctx, "tcp", address)
	}
	if err != nil {
		return "", safe(err)
	}
	defer conn.Close()
	deadline, _ := ctx.Deadline()
	_ = conn.SetDeadline(deadline)
	stop := context.AfterFunc(ctx, func() { _ = conn.Close() })
	defer stop()
	client, err := smtp.NewClient(conn, c.config.Host)
	if err != nil {
		return "", safe(err)
	}
	defer client.Close()
	if c.config.TLSMode == "starttls" {
		if ok, _ := client.Extension("STARTTLS"); !ok {
			return "", &notifications.Failure{Code: "SMTP_TLS_REQUIRED"}
		}
		if err = client.StartTLS(tlsConfig); err != nil {
			return "", &notifications.Failure{Code: "SMTP_TLS_FAILED"}
		}
	}
	if c.config.Username != "" {
		if err = client.Auth(smtp.PlainAuth("", c.config.Username, c.config.Password, c.config.Host)); err != nil {
			var failure *textproto.Error
			if errors.As(err, &failure) && failure.Code >= 500 {
				return "", &notifications.Failure{Code: "SMTP_AUTH_FAILED"}
			}
			return "", safe(err)
		}
	}
	if err = client.Mail(c.config.FromAddress); err != nil {
		return "", safe(err)
	}
	if err = client.Rcpt(recipient.Address); err != nil {
		return "", safe(err)
	}
	writer, err := client.Data()
	if err != nil {
		return "", safe(err)
	}
	from := (&mail.Address{Name: c.config.FromName, Address: c.config.FromAddress}).String()
	encoded := base64.StdEncoding.EncodeToString([]byte(m.HTML))
	var lines strings.Builder
	for len(encoded) > 76 {
		lines.WriteString(encoded[:76] + "\r\n")
		encoded = encoded[76:]
	}
	lines.WriteString(encoded + "\r\n")
	_, err = fmt.Fprintf(writer, "From: %s\r\nTo: %s\r\nSubject: %s\r\nDate: %s\r\nMessage-ID: <%s@wbs.local>\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=UTF-8\r\nContent-Transfer-Encoding: base64\r\n\r\n%s", from, recipient.String(), mime.QEncoding.Encode("utf-8", m.Subject), time.Now().UTC().Format(time.RFC1123Z), m.DeliveryID, lines.String())
	if err != nil {
		return "", safe(err)
	}
	if err = writer.Close(); err != nil {
		return "", safe(err)
	}
	// DATA accepted; QUIT failure must not turn known acceptance into an unnecessary retry.
	_ = client.Quit()
	return m.DeliveryID, nil
}
