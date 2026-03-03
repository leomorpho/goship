package routes

import (
	"net/http"
	"strings"

	"github.com/leomorpho/goship/ent"
	"github.com/leomorpho/goship/ent/user"
	"github.com/leomorpho/goship/pkg/context"
	"github.com/leomorpho/goship/pkg/controller"
	"github.com/leomorpho/goship/pkg/repos/msg"
	routeNames "github.com/leomorpho/goship/pkg/routing/routenames"

	"github.com/leomorpho/goship/app/goship/views"
	"github.com/leomorpho/goship/app/goship/views/web/layouts/gen"
	"github.com/leomorpho/goship/app/goship/views/web/pages/gen"
	"github.com/leomorpho/goship/pkg/repos/profilerepo"
	"github.com/leomorpho/goship/pkg/types"

	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
)

type (
	login struct {
		ctr controller.Controller
	}
)

func NewLoginRoute(ctr controller.Controller) login {
	return login{
		ctr: ctr,
	}
}

func (c *login) Get(ctx echo.Context) error {

	page := controller.NewPage(ctx)
	page.Layout = layouts.Auth
	page.Name = templates.PageLogin
	page.Title = "Log in"
	page.Form = &types.LoginForm{}
	page.Component = pages.Login(&page)
	page.HTMX.Request.Boosted = true

	// TODO: below is a bit of a hack. We're sometimes left with a stale CSRF token
	// in the cookies because the user was not actively logged out before their session
	// expired. As a workaround, invalidate any related cookie before attempting to login.
	c.ctr.Container.Auth.Logout(ctx)

	if form := ctx.Get(context.FormKey); form != nil {
		page.Form = form.(*types.LoginForm)
	}

	return c.ctr.RenderPage(ctx, page)
}

func (c *login) Post(ctx echo.Context) error {
	var form types.LoginForm
	ctx.Set(context.FormKey, &form)

	authFailed := func() error {
		// form.Submission.SetFieldError("Email", "")
		// form.Submission.SetFieldError("Password", "")
		msg.Danger(ctx, "Invalid credentials. Please try again.")
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

	// Attempt to load the user
	usr, err := c.ctr.Container.ORM.User.
		Query().
		Where(user.Email((strings.ToLower(form.Email)))).
		Only(ctx.Request().Context())

	switch err.(type) {
	case *ent.NotFoundError:
		ctx.Logger().Debug("ent user not found")
		return authFailed()
	case nil:
	default:
		return c.ctr.Fail(err, "error querying user during login")
	}

	// Check if the password is correct
	err = c.ctr.Container.Auth.CheckPassword(form.Password, usr.Password)
	if err != nil {
		ctx.Logger().Debug("password incorrect")
		return authFailed()
	}

	// Log the user in
	err = c.ctr.Container.Auth.Login(ctx, usr.ID)
	if err != nil {
		return c.ctr.Fail(err, "unable to log in user")
	}

	// msg.Success(ctx, fmt.Sprintf("Welcome back, <strong>%s</strong>. You are now logged in.", usr.Name))

	redirect, err := redirectAfterLogin(ctx)
	if err != nil {
		return err
	}
	if redirect {
		return nil
	}

	profile := usr.QueryProfile().FirstX(ctx.Request().Context())
	if !profilerepo.IsProfileFullyOnboarded(profile) {
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
