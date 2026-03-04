package controllers

import (
	"github.com/labstack/echo/v4"
	routeNames "github.com/leomorpho/goship/apps/site/web/routenames"
	"github.com/leomorpho/goship/apps/site/web/ui"
	"github.com/leomorpho/goship/ent"
	"github.com/leomorpho/goship/ent/user"
	"github.com/leomorpho/goship/pkg/context"
	"github.com/leomorpho/goship/pkg/repos/msg"
)

type verifyEmail struct {
	ctr ui.Controller
}

func NewVerifyEmailRoute(ctr ui.Controller) *verifyEmail {
	return &verifyEmail{ctr: ctr}
}
func (c *verifyEmail) Get(ctx echo.Context) error {
	var usr *ent.User

	// Validate the token
	token := ctx.Param("token")
	email, err := c.ctr.Container.Auth.ValidateEmailVerificationToken(token)
	if err != nil {
		msg.Warning(ctx, "The link is either invalid or has expired.")
		return c.ctr.Redirect(ctx, routeNames.RouteNameLandingPage)
	}

	// Check if it matches the authenticated user
	u := ctx.Get(context.AuthenticatedUserKey)
	if u != nil {
		authUser := u.(*ent.User)

		if authUser.Email == email {
			usr = authUser
		}
	}

	// Query to find a matching user, if needed
	if usr == nil {
		usr, err = c.ctr.Container.ORM.User.
			Query().
			Where(user.Email(email)).
			Only(ctx.Request().Context())

		if err != nil {
			return c.ctr.Fail(err, "query failed loading email verification token user")
		}
	}

	// Verify the user, if needed
	if !usr.Verified {
		_, err = usr.
			Update().
			SetVerified(true).
			Save(ctx.Request().Context())

		if err != nil {
			return c.ctr.Fail(err, "failed to set user as verified")
		}
	}

	msg.Success(ctx, "Your email has been successfully verified.")

	// If we have a user, they are already logged in and just redirect them to their home feed
	if u != nil {
		return c.ctr.Redirect(ctx, routeNames.RouteNamePreferences)

	}
	return c.ctr.Redirect(ctx, routeNames.RouteNameLogin)
}
