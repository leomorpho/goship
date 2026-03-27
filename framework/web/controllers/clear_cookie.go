package controllers

import (
	"time"

	"github.com/labstack/echo/v4"
	"github.com/leomorpho/goship/framework/flash"
	frameworkroutenames "github.com/leomorpho/goship/framework/web/routenames"
	"github.com/leomorpho/goship/framework/web/ui"
)

type ClearCookiesRoute struct {
	Controller ui.Controller
}

func NewClearCookiesRoute(ctr ui.Controller) ClearCookiesRoute {
	return ClearCookiesRoute{Controller: ctr}
}

func (ck *ClearCookiesRoute) Get(ctx echo.Context) error {
	if err := ck.Controller.Container.Auth.Logout(ctx); err == nil {
		uxflashmessages.Success(ctx, "You have successfully cleared this site's cookie.")
	} else {
		uxflashmessages.Danger(ctx, "An error occurred. Please try again.")
	}

	for _, cookie := range ctx.Cookies() {
		cookie.Expires = time.Now().UTC().Add(-100 * time.Hour)
		cookie.MaxAge = -1
		ctx.SetCookie(cookie)
	}

	return ck.Controller.Redirect(ctx, frameworkroutenames.RouteNameLogin)
}
