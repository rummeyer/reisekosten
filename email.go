package main

import (
	"io"

	"github.com/go-gomail/gomail"
)

// ---------------------------------------------------------------------------
// Email
// ---------------------------------------------------------------------------

// Attachment represents an in-memory email attachment.
type Attachment struct {
	Filename string
	Data     []byte
}

// sendEmail sends the generated PDFs via SMTP using in-memory attachments.
func sendEmail(cfg *Config, subject string, attachments ...Attachment) error {
	msg := gomail.NewMessage()
	msg.SetHeader("From", cfg.Email.From)
	msg.SetHeader("To", cfg.Email.To)
	msg.SetHeader("Subject", subject)
	msg.SetBody("text/html", "Dokumente anbei.<br>")

	for _, a := range attachments {
		data := a.Data // capture for closure
		msg.Attach(a.Filename, gomail.SetCopyFunc(func(w io.Writer) error {
			_, err := w.Write(data)
			return err
		}))
	}

	dialer := gomail.NewDialer(cfg.SMTP.Host, cfg.SMTP.Port, cfg.SMTP.Username, cfg.SMTP.Password)
	return dialer.DialAndSend(msg)
}
