package controllers

import (
	routeNames "github.com/leomorpho/goship/app/web/routenames"
	"github.com/leomorpho/goship/app/web/ui"
	"github.com/leomorpho/goship/db/ent"
	"github.com/leomorpho/goship/framework/context"
	"github.com/leomorpho/goship/framework/repos/msg"

	"github.com/leomorpho/goship/app/views"
	"github.com/leomorpho/goship/app/views/web/layouts/gen"
	"github.com/leomorpho/goship/app/views/web/pages/gen"
	"github.com/leomorpho/goship/app/web/viewmodels"

	"github.com/labstack/echo/v4"
)

type (
	resetPassword struct {
		ctr ui.Controller
	}
)

func NewResetPasswordRoute(ctr ui.Controller) resetPassword {
	return resetPassword{
		ctr: ctr,
	}
}

func (c *resetPassword) Get(ctx echo.Context) error {
	page := ui.NewPage(ctx)
	page.Layout = layouts.Auth
	page.Name = templates.PageResetPassword
	page.Title = "Reset password"
	page.Form = &viewmodels.ResetPasswordForm{}
	page.Component = pages.ResetPassword(&page)

	if form := ctx.Get(context.FormKey); form != nil {
		page.Form = form.(*viewmodels.ResetPasswordForm)
	}

	return c.ctr.RenderPage(ctx, page)
}

func (c *resetPassword) Post(ctx echo.Context) error {
	var form viewmodels.ResetPasswordForm
	ctx.Set(context.FormKey, &form)

	// Parse the form values
	if err := ctx.Bind(&form); err != nil {
		return c.ctr.Fail(err, "unable to parse password reset form")
	}

	if err := form.Submission.Process(ctx, form); err != nil {
		return c.ctr.Fail(err, "unable to process form submission")
	}

	if form.Submission.HasErrors() {
		return c.Get(ctx)
	}

	// Hash the new password
	hash, err := c.ctr.Container.Auth.HashPassword(form.Password)
	if err != nil {
		return c.ctr.Fail(err, "unable to hash password")
	}

	// Get the requesting user
	usr := ctx.Get(context.UserKey).(*ent.User)

	// Update the user
	_, err = usr.
		Update().
		SetPassword(hash).
		Save(ctx.Request().Context())

	if err != nil {
		return c.ctr.Fail(err, "unable to update password")
	}

	// Delete all password tokens for this user
	err = c.ctr.Container.Auth.DeletePasswordTokens(ctx, usr.ID)
	if err != nil {
		return c.ctr.Fail(err, "unable to delete password tokens")
	}

	msg.Success(ctx, "Your password has been updated.")
	return c.ctr.Redirect(ctx, routeNames.RouteNameLogin)
}
