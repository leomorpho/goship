package routes

import (
	"fmt"
	"strings"

	"github.com/mikestefanello/pagoda/ent"
	"github.com/mikestefanello/pagoda/ent/user"
	"github.com/mikestefanello/pagoda/pkg/context"
	"github.com/mikestefanello/pagoda/pkg/controller"
	"github.com/mikestefanello/pagoda/pkg/repos/msg"
	routeNames "github.com/mikestefanello/pagoda/pkg/routing/routenames"

	"github.com/mikestefanello/pagoda/pkg/types"
	"github.com/mikestefanello/pagoda/templates"
	"github.com/mikestefanello/pagoda/templates/emails"
	"github.com/mikestefanello/pagoda/templates/layouts"
	"github.com/mikestefanello/pagoda/templates/pages"
	"github.com/mileusna/useragent"

	"github.com/labstack/echo/v4"
)

type (
	forgotPassword struct {
		ctr controller.Controller
	}
)

func NewForgotPasswordRoute(ctr controller.Controller) forgotPassword {
	return forgotPassword{
		ctr: ctr,
	}
}

func (c *forgotPassword) Get(ctx echo.Context) error {
	page := controller.NewPage(ctx)
	page.Layout = layouts.Auth
	page.Name = templates.PageForgotPassword
	page.Title = "Forgot password"
	page.Form = &types.ForgotPasswordForm{}
	page.Component = pages.ForgotPassword(&page)
	page.HTMX.Request.Boosted = true

	if form := ctx.Get(context.FormKey); form != nil {
		page.Form = form.(*types.ForgotPasswordForm)
	}

	return c.ctr.RenderPage(ctx, page)
}

func (c *forgotPassword) Post(ctx echo.Context) error {
	var form types.ForgotPasswordForm
	ctx.Set(context.FormKey, &form)

	succeed := func() error {
		ctx.Set(context.FormKey, nil)
		msg.Success(ctx, "An email was sent to reset your password.")
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
	u, err := c.ctr.Container.ORM.User.
		Query().
		Where(user.Email(strings.ToLower(form.Email))).
		Only(ctx.Request().Context())

	switch err.(type) {
	case *ent.NotFoundError:
		return succeed()
	case nil:
	default:
		return c.ctr.Fail(err, "error querying user during forgot password")
	}

	// Generate the token
	token, pt, err := c.ctr.Container.Auth.GeneratePasswordResetToken(ctx, u.ID)
	if err != nil {
		return c.ctr.Fail(err, "error generating password reset token")
	}

	ctx.Logger().Infof("generated password reset token for user %d", u.ID)

	// Email the user
	url := ctx.Echo().Reverse(routeNames.RouteNameResetPassword, u.ID, pt.ID, token)

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

	page := controller.NewPage(ctx)
	page.Layout = layouts.Main
	page.Data = types.EmailPasswordResetData{
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
