package controllers

import (
	"time"

	"github.com/labstack/echo/v4"
	"github.com/leomorpho/goship/app/web/routenames"
	"github.com/leomorpho/goship/app/web/ui"
	"github.com/leomorpho/goship/framework/repos/uxflashmessages"
)

type (
	clearCookie struct {
		ctr ui.Controller
	}
)

func NewClearCookiesRoute(ctr ui.Controller) clearCookie {
	return clearCookie{
		ctr: ctr,
	}
}

func (ck *clearCookie) Get(ctx echo.Context) error {
	if err := ck.ctr.Container.Auth.Logout(ctx); err == nil {
		uxflashmessages.Success(ctx, "You have successfully cleared this site's cookie.")
	} else {
		uxflashmessages.Danger(ctx, "An error occurred. Please try again.")
	}

	// Clear all other cookies
	for _, cookie := range ctx.Cookies() {
		cookie.Expires = time.Now().UTC().Add(-100 * time.Hour) // Set to a time in the past
		cookie.MaxAge = -1
		ctx.SetCookie(cookie)
	}
	return ck.ctr.Redirect(ctx, routenames.RouteNameLogin)
}
