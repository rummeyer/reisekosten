package main

import (
	"github.com/go-gomail/gomail"
)

// ---------------------------------------------------------------------------
// Email
// ---------------------------------------------------------------------------

// sendEmail sends the generated PDFs via SMTP.
func sendEmail(cfg *Config, subject string, filenames ...string) error {
	msg := gomail.NewMessage()
	msg.SetHeader("From", cfg.Email.From)
	msg.SetHeader("To", cfg.Email.To)
	msg.SetHeader("Subject", subject)
	msg.SetBody("text/html", "Dokumente anbei.<br>")

	for _, f := range filenames {
		msg.Attach(f)
	}

	dialer := gomail.NewDialer(cfg.SMTP.Host, cfg.SMTP.Port, cfg.SMTP.Username, cfg.SMTP.Password)
	return dialer.DialAndSend(msg)
}
