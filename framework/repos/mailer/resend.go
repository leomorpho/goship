package mailer

import (
	"context"
	"fmt"

	"github.com/resend/resend-go/v2"
)

type ResendMailClient struct {
	client *resend.Client
}

func NewResendMailClient(apiKey string) *ResendMailClient {
	client := resend.NewClient(apiKey)
	return &ResendMailClient{client: client}
}

func (r *ResendMailClient) Send(email *mail) error {
	params := &resend.SendEmailRequest{
		To:      []string{email.to},
		From:    email.from,
		Subject: email.subject,
	}

	if email.component != nil {
		params.Html = email.body
	} else {
		params.Text = email.body
	}

	ctx := context.TODO()
	_, err := r.client.Emails.SendWithContext(ctx, params)
	if err != nil {
		return fmt.Errorf("resend mail client failed to send email: %w", err)
	}
	return nil
}
