package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/hibiken/asynq"
	"github.com/labstack/echo/v4"
	"github.com/leomorpho/goship/app/foundation"
	"github.com/leomorpho/goship/app/views/emails/gen"
	"github.com/leomorpho/goship/app/views/web/layouts/gen"
	"github.com/leomorpho/goship/app/web/ui"
	"github.com/leomorpho/goship/app/web/viewmodels"
	"github.com/leomorpho/goship/config"
	"github.com/leomorpho/goship/framework/repos/mailer"
)

// ////////////////////////////////////////////////////////////////////////////
// Send email list confirmation email
// ////////////////////////////////////////////////////////////////////////////

const TypeEmailSubscriptionConfirmation = "email:email_subscription_confirmation"

type (
	EmailSubscriptionConfirmationProcessor struct {
		mailer *mailer.MailClient
		config *config.Config
	}

	EmailSubscriptionConfirmationPayload struct {
		Email string `json:"to"`
		Url   string `json:"url"`
	}
)

func NewEmailSubscriptionConfirmationProcessor(
	mailer *mailer.MailClient, config *config.Config,
) *EmailSubscriptionConfirmationProcessor {
	return &EmailSubscriptionConfirmationProcessor{mailer: mailer, config: config}
}

func (esc *EmailSubscriptionConfirmationProcessor) ProcessTask(ctx context.Context, t *asynq.Task) error {
	var p EmailSubscriptionConfirmationPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		slog.Error("Error unmarshalling payload", "error", err)
		return err
	}

	fullUrl := fmt.Sprintf("%s%s", esc.config.HTTP.Domain, p.Url)

	page := ui.NewPage(echo.New().AcquireContext())
	page.Layout = layouts.Main
	data := viewmodels.NewEmailDefaultData()
	data.AppName = string(esc.config.App.Name)
	data.ConfirmationLink = fullUrl
	data.SupportEmail = esc.config.Mail.FromAddress
	data.Domain = esc.config.HTTP.Domain
	page.Data = data

	err := esc.mailer.
		Compose().
		To(p.Email).
		Subject("Confirm your email subscription for the app release anouncement.").
		TemplateLayout(layouts.Email).
		Component(emails.SubscriptionConfirmation(&page)).
		Send(ctx)

	return err
}

// ----------------------------------------------------------

const TypeEmailUpdates = "email:email_updates"

type (
	EmailUpdateProcessor struct {
		emailSender *UpdateEmailSender
	}
)

// TODO: no need for all the param this takes, some are in Container. Fix later.
func NewEmailUpdateProcessor(container *foundation.Container) *EmailUpdateProcessor {

	updateEmailSender := NewUpdateEmailSender(container)

	return &EmailUpdateProcessor{
		emailSender: updateEmailSender,
	}
}

func (e *EmailUpdateProcessor) ProcessTask(ctx context.Context, t *asynq.Task) error {
	return e.emailSender.PrepareAndSendUpdateEmailForAll(ctx)
}
