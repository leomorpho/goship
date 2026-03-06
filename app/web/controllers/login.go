package controllers

import (
	"net/http"

	"github.com/leomorpho/goship/app/foundation"
	routeNames "github.com/leomorpho/goship/app/web/routenames"
	"github.com/leomorpho/goship/app/web/ui"
	"github.com/leomorpho/goship/framework/context"
	"github.com/leomorpho/goship/framework/repos/uxflashmessages"

	"github.com/leomorpho/goship/app/views"
	"github.com/leomorpho/goship/app/views/web/layouts/gen"
	"github.com/leomorpho/goship/app/views/web/pages/gen"
	"github.com/leomorpho/goship/app/web/viewmodels"

	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
)

type (
	login struct {
		ctr ui.Controller
	}
)

func NewLoginRoute(ctr ui.Controller) login {
	return login{
		ctr: ctr,
	}
}

func (c *login) Get(ctx echo.Context) error {

	page := ui.NewPage(ctx)
	page.Layout = layouts.Auth
	page.Name = templates.PageLogin
	page.Title = "Log in"
	page.Form = &viewmodels.LoginForm{}
	page.Component = pages.Login(&page)
	page.HTMX.Request.Boosted = true

	// TODO: below is a bit of a hack. We're sometimes left with a stale CSRF token
	// in the cookies because the user was not actively logged out before their session
	// expired. As a workaround, invalidate any related cookie before attempting to login.
	c.ctr.Container.Auth.Logout(ctx)

	if form := ctx.Get(context.FormKey); form != nil {
		page.Form = form.(*viewmodels.LoginForm)
	}

	return c.ctr.RenderPage(ctx, page)
}

func (c *login) Post(ctx echo.Context) error {
	var form viewmodels.LoginForm
	ctx.Set(context.FormKey, &form)

	authFailed := func() error {
		// form.Submission.SetFieldError("Email", "")
		// form.Submission.SetFieldError("Password", "")
		uxflashmessages.Danger(ctx, "Invalid credentials. Please try again.")
		return c.Get(ctx)
	}

	// Parse the form values
	if err := ctx.Bind(&form); err != nil {
		return c.ctr.Fail(err, "unable to parse login form")
	}

	if err := form.Submission.Process(ctx, form); err != nil {
		return c.ctr.Fail(err, "unable to process form submission")
	}

	if form.Submission.HasErrors() {
		return c.Get(ctx)
	}

	usr, err := c.ctr.Container.Auth.AuthenticateUserByEmailPassword(ctx, form.Email, form.Password)
	switch err.(type) {
	case nil:
	case foundation.InvalidCredentialsError:
		ctx.Logger().Debug("credentials incorrect")
		return authFailed()
	default:
		return c.ctr.Fail(err, "error authenticating user during login")
	}

	// Log the user in
	err = c.ctr.Container.Auth.Login(ctx, usr.UserID)
	if err != nil {
		return c.ctr.Fail(err, "unable to log in user")
	}

	// uxflashmessages.Success(ctx, fmt.Sprintf("Welcome back, <strong>%s</strong>. You are now logged in.", usr.Name))

	redirect, err := redirectAfterLogin(ctx)
	if err != nil {
		return err
	}
	if redirect {
		return nil
	}

	identity, err := c.ctr.Container.Auth.GetIdentityByUserID(ctx.Request().Context(), usr.UserID)
	if err != nil {
		return c.ctr.Fail(err, "unable to determine profile onboarding status")
	}
	if identity == nil || !identity.ProfileFullyOnboarded {
		return c.ctr.Redirect(ctx, routeNames.RouteNamePreferences)

	}
	return c.ctr.Redirect(ctx, routeNames.RouteNameHomeFeed)
}

// redirectAfterLogin redirects a now logged-in user to a previously requested page.
func redirectAfterLogin(ctx echo.Context) (bool, error) {
	sess, _ := session.Get("session", ctx)

	// Retrieve the redirect URL if it exists
	redirectURL, ok := sess.Values["redirectAfterLogin"].(string)
	if ok && redirectURL != "" {
		// Clear the redirect URL from session
		delete(sess.Values, "redirectAfterLogin")
		sess.Save(ctx.Request(), ctx.Response())

		// Redirect to the originally requested URL
		return true, ctx.Redirect(http.StatusFound, redirectURL)
	}
	return false, nil // Or redirect to a default route if nothing is in the session
}
