package tasks

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hibiken/asynq"
	"github.com/labstack/echo/v4"
	"github.com/leomorpho/goship/app/goship/views/emails/gen"
	"github.com/leomorpho/goship/app/goship/views/web/layouts/gen"
	"github.com/leomorpho/goship/config"
	"github.com/leomorpho/goship/ent"
	"github.com/leomorpho/goship/pkg/controller"
	"github.com/leomorpho/goship/pkg/repos/emailsmanager"
	"github.com/leomorpho/goship/pkg/repos/mailer"
	"github.com/leomorpho/goship/pkg/services"
	"github.com/leomorpho/goship/pkg/types"
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
		fmt.Printf("Error unmarshalling payload: %v\n", err)
		return err
	}

	fullUrl := fmt.Sprintf("%s%s", esc.config.HTTP.Domain, p.Url)

	page := controller.NewPage(echo.New().AcquireContext())
	page.Layout = layouts.Main
	page.Data = types.EmailDefaultData{
		AppName:          string(esc.config.App.Name),
		ConfirmationLink: fullUrl,
		SupportEmail:     esc.config.Mail.FromAddress,
		Domain:           esc.config.HTTP.Domain,
	}

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
		emailSender *emailsmanager.UpdateEmailSender
	}
)

// TODO: no need for all the param this takes, some are in Container. Fix later.
func NewEmailUpdateProcessor(
	container *services.Container, orm *ent.Client,
) *EmailUpdateProcessor {

	updateEmailSender := emailsmanager.NewUpdateEmailSender(orm, container)

	return &EmailUpdateProcessor{
		emailSender: updateEmailSender,
	}
}

func (e *EmailUpdateProcessor) ProcessTask(ctx context.Context, t *asynq.Task) error {
	return e.emailSender.PrepareAndSendUpdateEmailForAll(ctx)
}
