package mailer

import (
	"fmt"
	"net/smtp"
)

var smtpSendMail = smtp.SendMail

type SMTPMailClient struct {
	Port     int
	Host     string
	Username string
	Password string
}

func NewSMTPMailClient(host string, port int) *SMTPMailClient {
	return &SMTPMailClient{
		Port: port,
		Host: host,
	}
}

func NewSMTPMailClientWithAuth(host string, port int, user, pass string) *SMTPMailClient {
	return &SMTPMailClient{
		Port:     port,
		Host:     host,
		Username: user,
		Password: pass,
	}
}

// Send sends an email using SMTP
func (c *SMTPMailClient) Send(email *mail) error {
	// Define email headers and body
	msg := fmt.Sprintf("From: %s\nTo: %s\nSubject: %s\nMIME-Version: 1.0\nContent-Type: text/html; charset=\"utf-8\"\n\n%s", email.from, email.to, email.subject, email.body)

	// SMTP server configuration
	smtpHost := c.Host
	smtpPort := c.Port
	smtpAddr := fmt.Sprintf("%s:%d", smtpHost, smtpPort)

	var auth smtp.Auth
	if c.Username != "" || c.Password != "" {
		auth = smtp.PlainAuth("", c.Username, c.Password, smtpHost)
	}

	// Sending email
	err := smtpSendMail(smtpAddr, auth, email.from, []string{email.to}, []byte(msg))
	if err != nil {
		return err
	}

	return nil
}
