package mailer

import (
	"fmt"
	"net/smtp"
)

type SMTPMailClient struct {
	Port int
	Host string
}

func NewSMTPMailClient(host string, port int) *SMTPMailClient {
	return &SMTPMailClient{
		Port: port,
		Host: host,
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

	// Authentication - Mailpit does not require authentication, but you could add it here if needed
	auth := smtp.PlainAuth("", "", "", smtpHost)

	// Sending email
	err := smtp.SendMail(smtpAddr, auth, email.from, []string{email.to}, []byte(msg))
	if err != nil {
		return err
	}

	return nil
}
