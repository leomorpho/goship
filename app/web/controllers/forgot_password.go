package controllers

import (
	"fmt"

	routeNames "github.com/leomorpho/goship/app/web/routenames"
	"github.com/leomorpho/goship/app/web/ui"
	"github.com/leomorpho/goship/framework/context"
	"github.com/leomorpho/goship/framework/dberrors"
	"github.com/leomorpho/goship/framework/repos/uxflashmessages"

	"github.com/leomorpho/goship/app/views"
	"github.com/leomorpho/goship/app/views/emails/gen"
	"github.com/leomorpho/goship/app/views/web/layouts/gen"
	"github.com/leomorpho/goship/app/views/web/pages/gen"
	"github.com/leomorpho/goship/app/web/viewmodels"
	"github.com/mileusna/useragent"

	"github.com/labstack/echo/v4"
)

type (
	forgotPassword struct {
		ctr ui.Controller
	}
)

func NewForgotPasswordRoute(ctr ui.Controller) forgotPassword {
	return forgotPassword{
		ctr: ctr,
	}
}

func (c *forgotPassword) Get(ctx echo.Context) error {
	page := ui.NewPage(ctx)
	page.Layout = layouts.Auth
	page.Name = templates.PageForgotPassword
	page.Title = "Forgot password"
	page.Form = &viewmodels.ForgotPasswordForm{}
	page.Component = pages.ForgotPassword(&page)
	page.HTMX.Request.Boosted = true

	if form := ctx.Get(context.FormKey); form != nil {
		page.Form = form.(*viewmodels.ForgotPasswordForm)
	}

	return c.ctr.RenderPage(ctx, page)
}

func (c *forgotPassword) Post(ctx echo.Context) error {
	var form viewmodels.ForgotPasswordForm
	ctx.Set(context.FormKey, &form)

	succeed := func() error {
		ctx.Set(context.FormKey, nil)
		uxflashmessages.Success(ctx, "An email was sent to reset your password.")
		return c.Get(ctx)
	}

	// Parse the form values
	if err := ctx.Bind(&form); err != nil {
		return c.ctr.Fail(err, "unable to parse forgot password form")
	}

	if err := form.Submission.Process(ctx, form); err != nil {
		return c.ctr.Fail(err, "unable to process form submission")
	}

	if form.Submission.HasErrors() {
		return c.Get(ctx)
	}

	// Attempt to load the user
	u, err := c.ctr.Container.Auth.FindUserRecordByEmail(ctx, form.Email)

	switch {
	case dberrors.IsNotFound(err):
		return succeed()
	case err != nil:
		return c.ctr.Fail(err, "error querying user during forgot password")
	}

	// Generate the token
	token, tokenID, err := c.ctr.Container.Auth.GeneratePasswordResetToken(ctx, u.UserID)
	if err != nil {
		return c.ctr.Fail(err, "error generating password reset token")
	}

	ctx.Logger().Infof("generated password reset token for user %d", u.UserID)

	// Email the user
	url := ctx.Echo().Reverse(routeNames.RouteNameResetPassword, u.UserID, tokenID, token)

	err = c.sendPasswordResetEmail(ctx, u.Name, u.Email, url)
	if err != nil {
		return err
	}

	return succeed()
}

func (c *forgotPassword) sendPasswordResetEmail(ctx echo.Context, profileName, email, url string) error {

	fullUrl := fmt.Sprintf("%s%s", c.ctr.Container.Config.HTTP.Domain, url)
	// Parse User-Agent string
	ua := useragent.Parse(ctx.Request().UserAgent())

	page := ui.NewPage(ctx)
	page.Layout = layouts.Main
	page.Data = viewmodels.EmailPasswordResetData{
		AppName:           string(c.ctr.Container.Config.App.Name),
		ProfileName:       profileName,
		PasswordResetLink: fullUrl,
		SupportEmail:      c.ctr.Container.Config.Mail.FromAddress,
		OperatingSystem:   ua.OS,
		BrowserName:       ua.Name,
		Domain:            c.ctr.Container.Config.HTTP.Domain,
	}

	err := c.ctr.Container.Mail.
		Compose().
		To(email).
		Subject("Reset your password").
		TemplateLayout(layouts.Email).
		Component(emails.PasswordReset(&page)).
		Send(ctx.Request().Context())

	if err != nil {
		ctx.Logger().Errorf("unable to send email reset link: %v", err)
		return err
	}
	return nil
}
