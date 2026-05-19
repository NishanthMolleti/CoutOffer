package mailer

import (
	"fmt"
	"net/smtp"
	"os"
	"strings"

	"coutoffer/scraper"
)

type Mailer struct {
	host     string
	port     string
	user     string
	password string
}

func New() *Mailer {
	return &Mailer{
		host:     getenv("SMTP_HOST", "smtp.gmail.com"),
		port:     getenv("SMTP_PORT", "587"),
		user:     os.Getenv("SMTP_USER"),
		password: os.Getenv("SMTP_PASS"),
	}
}

func (m *Mailer) SendJobAlert(toEmail, company string, jobs []scraper.Job) error {
	auth := smtp.PlainAuth("", m.user, m.password, m.host)

	var body strings.Builder
	fmt.Fprintf(&body, "cout << offer; // %d new opening(s) at %s\n\n", len(jobs), company)
	for _, j := range jobs {
		fmt.Fprintf(&body, "  [+] %s\n      Location : %s\n      Apply    : %s\n\n", j.Title, j.Location, j.URL)
	}

	msg := fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: [CoutOffer] New jobs at %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s",
		m.user, toEmail, company, body.String(),
	)

	return smtp.SendMail(
		fmt.Sprintf("%s:%s", m.host, m.port),
		auth,
		m.user,
		[]string{toEmail},
		[]byte(msg),
	)
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
