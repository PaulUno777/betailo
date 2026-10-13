// Package smtp sends the platform's e-mail through an SMTP server.
package smtp

import (
	"context"
	"fmt"
	"log/slog"
	"mime"
	"net"
	netsmtp "net/smtp"
	"strings"
	"time"

	"github.com/pauluno777/betailo/backend/application/ports"
)

// Config is one SMTP account. Username may be empty for a relay that
// takes mail without authentication (a local test inbox).
type Config struct {
	Host     string
	Port     string
	Username string
	Password string
	// From is the sender, as an address or "Name <address>".
	From string
}

// Mailer implements ports.Mailer over SMTP. The server's STARTTLS is
// used whenever it offers it; credentials are only ever sent over TLS
// (or to localhost), which net/smtp enforces.
type Mailer struct {
	Config Config
}

var _ ports.Mailer = (*Mailer)(nil)

// Send implements ports.Mailer.
func (m *Mailer) Send(_ context.Context, mail ports.Mail) error {
	if strings.ContainsAny(mail.To, "\r\n") || strings.ContainsAny(mail.Subject, "\r\n") {
		return fmt.Errorf("smtp: header injection refused")
	}
	var auth netsmtp.Auth
	if m.Config.Username != "" {
		auth = netsmtp.PlainAuth("", m.Config.Username, m.Config.Password, m.Config.Host)
	}
	body := strings.ReplaceAll(strings.ReplaceAll(mail.Text, "\r\n", "\n"), "\n", "\r\n")
	msg := strings.Join([]string{
		"From: " + m.Config.From,
		"To: " + mail.To,
		"Subject: " + mime.QEncoding.Encode("utf-8", mail.Subject),
		"Date: " + time.Now().UTC().Format(time.RFC1123Z),
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=utf-8",
		"Content-Transfer-Encoding: 8bit",
		"",
		body,
	}, "\r\n")
	return netsmtp.SendMail(net.JoinHostPort(m.Config.Host, m.Config.Port), auth, senderAddress(m.Config.From), []string{mail.To}, []byte(msg))
}

// senderAddress is the bare address of a "Name <address>" sender.
func senderAddress(from string) string {
	if i, j := strings.LastIndex(from, "<"), strings.LastIndex(from, ">"); i >= 0 && j > i {
		return from[i+1 : j]
	}
	return from
}

// LogMailer stands in when no SMTP server is configured: it writes each
// message to the log instead of sending it, so a link can still be
// taken from there on a development machine.
type LogMailer struct {
	Logger *slog.Logger
}

var _ ports.Mailer = (*LogMailer)(nil)

// Send implements ports.Mailer.
func (m *LogMailer) Send(_ context.Context, mail ports.Mail) error {
	m.Logger.Info("mail (not sent: no SMTP server configured)", slog.String("to", mail.To), slog.String("subject", mail.Subject), slog.String("text", mail.Text))
	return nil
}
