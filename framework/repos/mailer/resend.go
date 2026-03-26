package mailer

import (
	"context"
	"errors"
	"fmt"

	"github.com/resend/resend-go/v2"
)

type ResendMailClient struct {
	sender resendSender
}

type resendSender interface {
	SendWithContext(ctx context.Context, params *resend.SendEmailRequest) (*resend.SendEmailResponse, error)
}

func NewResendMailClient(apiKey string) *ResendMailClient {
	client := resend.NewClient(apiKey)
	return &ResendMailClient{sender: client.Emails}
}

func NewResendMailClientWithSender(sender resendSender) *ResendMailClient {
	return &ResendMailClient{sender: sender}
}

func (r *ResendMailClient) Send(ctx context.Context, email *mail) error {
	if r == nil || r.sender == nil {
		return errors.New("resend mail client not initialized")
	}

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

	_, err := r.sender.SendWithContext(ctx, params)
	if err != nil {
		return fmt.Errorf("resend mail client failed to send email: %w", err)
	}
	return nil
}
