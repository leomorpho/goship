package controllers

import (
	"github.com/labstack/echo/v4"
	routeNames "github.com/leomorpho/goship/app/web/routenames"
	"github.com/leomorpho/goship/app/web/ui"
	"github.com/leomorpho/goship/framework/context"
	"github.com/leomorpho/goship/framework/repos/uxflashmessages"
)

type verifyEmail struct {
	ctr ui.Controller
}

func NewVerifyEmailRoute(ctr ui.Controller) *verifyEmail {
	return &verifyEmail{ctr: ctr}
}
func (c *verifyEmail) Get(ctx echo.Context) error {
	// Validate the token
	token := ctx.Param("token")
	email, err := c.ctr.Container.Auth.ValidateEmailVerificationToken(token)
	if err != nil {
		uxflashmessages.Warning(ctx, "The link is either invalid or has expired.")
		return c.ctr.Redirect(ctx, routeNames.RouteNameLandingPage)
	}

	authEmail, authEmailOK := ctx.Get(context.AuthenticatedUserEmailKey).(string)
	authUserID, authUserIDOK := ctx.Get(context.AuthenticatedUserIDKey).(int)
	if authEmailOK && authUserIDOK && authEmail == email {
		if err = c.ctr.Container.Auth.MarkUserVerifiedByUserID(ctx, authUserID); err != nil {
			return c.ctr.Fail(err, "failed to set authenticated user as verified")
		}
	} else {
		usr, queryErr := c.ctr.Container.Auth.FindUserRecordByEmail(ctx, email)
		if queryErr != nil {
			return c.ctr.Fail(queryErr, "query failed loading email verification token user")
		}

		if !usr.IsVerified {
			if err = c.ctr.Container.Auth.MarkUserVerifiedByUserID(ctx, usr.UserID); err != nil {
				return c.ctr.Fail(err, "failed to set user as verified")
			}
		}
	}

	uxflashmessages.Success(ctx, "Your email has been successfully verified.")

	// If we have a user, they are already logged in and just redirect them to their home feed
	if ctx.Get(context.AuthenticatedUserIDKey) != nil {
		return c.ctr.Redirect(ctx, routeNames.RouteNamePreferences)

	}
	return c.ctr.Redirect(ctx, routeNames.RouteNameLogin)
}
