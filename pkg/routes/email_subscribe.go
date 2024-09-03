package routes

import (
	"errors"
	"fmt"

	"github.com/mikestefanello/pagoda/config"
	"github.com/mikestefanello/pagoda/pkg/context"
	"github.com/mikestefanello/pagoda/pkg/controller"
	"github.com/mikestefanello/pagoda/pkg/domain"
	"github.com/mikestefanello/pagoda/pkg/repos/emailsmanager"
	"github.com/mikestefanello/pagoda/pkg/types"
	"github.com/mikestefanello/pagoda/templates"
	"github.com/mikestefanello/pagoda/templates/emails"
	"github.com/mikestefanello/pagoda/templates/layouts"
	"github.com/mikestefanello/pagoda/templates/pages"
	"github.com/rs/zerolog/log"

	"github.com/labstack/echo/v4"
)

type (
	emailSubscribe struct {
		ctr       controller.Controller
		emailRepo emailsmanager.EmailSubscriptionRepo
		config    config.Config
	}
)

func NewEmailSubscribeRoute(ctr controller.Controller, emailRepo emailsmanager.EmailSubscriptionRepo, config config.Config) emailSubscribe {
	return emailSubscribe{
		ctr:       ctr,
		emailRepo: emailRepo,
		config:    config,
	}
}

func (c *emailSubscribe) Get(ctx echo.Context) error {
	page := controller.NewPage(ctx)
	page.Name = templates.PageEmailSubscribe
	page.Layout = layouts.Main
	page.Component = pages.EmailSubscribe(&page)
	page.Form = types.EmailSubscriptionForm{}
	page.Data = types.EmailSubscriptionData{
		Description: "Sign up to get our app release announcement.",
		Placeholder: "Enter email",
		Latitude:    0,
		Longitude:   0,
	}
	if form := ctx.Get(context.FormKey); form != nil {
		page.Form = form.(*types.EmailSubscriptionForm)
	}
	page.Cache.Enabled = false

	return c.ctr.RenderPage(ctx, page)
}

func (c *emailSubscribe) Post(ctx echo.Context) error {
	var form types.EmailSubscriptionForm
	ctx.Set(context.FormKey, &form)

	// Parse the form values
	if err := ctx.Bind(&form); err != nil {
		return c.ctr.Fail(err, "unable to bind form")
	}

	if err := form.Submission.Process(ctx, form); err != nil {
		return c.ctr.Fail(err, "unable to process form submission")
	}

	if form.Submission.HasErrors() {
		return c.Get(ctx)
	}

	subscriptionObj, err := c.emailRepo.SSESubscribe(
		ctx.Request().Context(), form.Email, domain.EmailNewsletter, &form.Latitude, &form.Longitude,
	)
	if err != nil {

		var errMsg string

		if errors.Is(err, emailsmanager.ErrInvalidEmailConfirmationCode) {
			errMsg = "The email confirmation code is invalid."
		} else if errors.Is(err, emailsmanager.ErrEmailSyntaxInvalid) {
			errMsg = "The email address syntax is invalid."
		} else if errors.Is(err, emailsmanager.ErrEmailAddressInvalidCatchAll) {
			errMsg = "The email address is invalid."
		} else if _, ok := err.(*emailsmanager.ErrAlreadySubscribed); ok {
			errMsg = "You're already subscribed."
		} else if e, ok := err.(*emailsmanager.ErrEmailVerificationFailed); ok {
			errMsg = e.Error()
		} else {
			log.Error().Err(err)
			ctx.Echo().Logger.Error(err)
			errMsg = "An unexpected error occurred. We're looking into it. Please try again later."
		}

		form.Submission.SetFieldError("email", errMsg)
		return c.Get(ctx)
	}
	page := controller.NewPage(ctx)
	page.Name = "email-subscribe-success"

	// Send the verification email
	c.sendSubscriptionVerificationEmail(ctx, form.Email, subscriptionObj.ConfirmationCode)

	return c.ctr.RenderPage(ctx, page)
}

func (c *emailSubscribe) sendSubscriptionVerificationEmail(ctx echo.Context, email, code string) {
	url := ctx.Echo().Reverse("verify_email_subscription", code)

	fullUrl := fmt.Sprintf("%s%s", c.ctr.Container.Config.HTTP.Domain, url)

	type EmailData struct {
		AppName          string
		ConfirmationLink string
		SupportEmail     string
		Domain           string
	}

	page := controller.NewPage(ctx)
	page.Layout = layouts.Main
	page.Data = types.EmailDefaultData{
		AppName:          string(c.ctr.Container.Config.App.Name),
		ConfirmationLink: fullUrl,
		SupportEmail:     c.ctr.Container.Config.Mail.FromAddress,
		Domain:           c.ctr.Container.Config.HTTP.Domain,
	}

	err := c.ctr.Container.Mail.
		Compose().
		To(email).
		Subject("Confirm your email subscription for the app release anouncement.").
		TemplateLayout(layouts.Email).
		Component(emails.SubscriptionConfirmation(&page)).
		Send(ctx.Request().Context())

	if err != nil {
		ctx.Logger().Errorf("unable to send email subscription verification link: %v", err)
		return
	}
}
