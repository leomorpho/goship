package controllers

import (
	"errors"
	"fmt"

	modemailsubscriptions "github.com/leomorpho/goship-modules/emailsubscriptions"
	"github.com/leomorpho/goship/app/views"
	"github.com/leomorpho/goship/app/views/emails/gen"
	"github.com/leomorpho/goship/app/views/web/layouts/gen"
	"github.com/leomorpho/goship/app/views/web/pages/gen"
	"github.com/leomorpho/goship/app/web/ui"
	"github.com/leomorpho/goship/app/web/viewmodels"
	"github.com/leomorpho/goship/config"
	"github.com/leomorpho/goship/framework/context"
	"github.com/leomorpho/goship/framework/domain"
	"log/slog"

	"github.com/labstack/echo/v4"
)

type (
	emailSubscribe struct {
		ctr       ui.Controller
		emailSubs modemailsubscriptions.Service
		config    config.Config
	}
)

func NewEmailSubscribeRoute(ctr ui.Controller, emailSubs modemailsubscriptions.Service, config config.Config) emailSubscribe {
	return emailSubscribe{
		ctr:       ctr,
		emailSubs: emailSubs,
		config:    config,
	}
}

func (c *emailSubscribe) Get(ctx echo.Context) error {
	page := ui.NewPage(ctx)
	page.Name = templates.PageEmailSubscribe
	page.Layout = layouts.Main
	page.Component = pages.EmailSubscribe(&page)
	page.Form = viewmodels.NewEmailSubscriptionForm()
	data := viewmodels.NewEmailSubscriptionData()
	data.Description = "Sign up to get our app release announcement."
	data.Placeholder = "Enter email"
	page.Data = data
	if form := ctx.Get(context.FormKey); form != nil {
		page.Form = form.(*viewmodels.EmailSubscriptionForm)
	}
	page.Cache.Enabled = false

	return c.ctr.RenderPage(ctx, page)
}

func (c *emailSubscribe) Post(ctx echo.Context) error {
	var form viewmodels.EmailSubscriptionForm
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

	subscriptionObj, err := c.emailSubs.Subscribe(
		ctx.Request().Context(), form.Email, modemailsubscriptions.List(domain.EmailNewsletter.Value), &form.Latitude, &form.Longitude,
	)
	if err != nil {

		var errMsg string

		if errors.Is(err, modemailsubscriptions.ErrInvalidEmailConfirmationCode) {
			errMsg = "The email confirmation code is invalid."
		} else if errors.Is(err, modemailsubscriptions.ErrEmailSyntaxInvalid) {
			errMsg = "The email address syntax is invalid."
		} else if errors.Is(err, modemailsubscriptions.ErrEmailAddressInvalidCatchAll) {
			errMsg = "The email address is invalid."
		} else if _, ok := err.(*modemailsubscriptions.ErrAlreadySubscribed); ok {
			errMsg = "You're already subscribed."
		} else if e, ok := err.(*modemailsubscriptions.ErrEmailVerificationFailed); ok {
			errMsg = e.Error()
		} else {
			slog.Error("unexpected error in email subscription", "error", err)
			ctx.Echo().Logger.Error(err)
			errMsg = "An unexpected error occurred. We're looking into it. Please try again later."
		}

		form.Submission.SetFieldError("email", errMsg)
		return c.Get(ctx)
	}
	page := ui.NewPage(ctx)
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

	page := ui.NewPage(ctx)
	page.Layout = layouts.Main
	data := viewmodels.NewEmailDefaultData()
	data.AppName = string(c.ctr.Container.Config.App.Name)
	data.ConfirmationLink = fullUrl
	data.SupportEmail = c.ctr.Container.Config.Mail.FromAddress
	data.Domain = c.ctr.Container.Config.HTTP.Domain
	page.Data = data

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
