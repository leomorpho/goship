package routes

import (
	"time"

	"github.com/labstack/echo/v4"
	"github.com/mikestefanello/pagoda/pkg/controller"
	"github.com/mikestefanello/pagoda/pkg/repos/msg"
	"github.com/mikestefanello/pagoda/pkg/routing/routenames"
)

type (
	clearCookie struct {
		ctr controller.Controller
	}
)

func NewClearCookiesRoute(ctr controller.Controller) clearCookie {
	return clearCookie{
		ctr: ctr,
	}
}

func (ck *clearCookie) Get(ctx echo.Context) error {
	if err := ck.ctr.Container.Auth.Logout(ctx); err == nil {
		msg.Success(ctx, "You have successfully cleared this site's cookie.")
	} else {
		msg.Danger(ctx, "An error occurred. Please try again.")
	}

	// Clear all other cookies
	for _, cookie := range ctx.Cookies() {
		cookie.Expires = time.Now().UTC().Add(-100 * time.Hour) // Set to a time in the past
		cookie.MaxAge = -1
		ctx.SetCookie(cookie)
	}
	return ck.ctr.Redirect(ctx, routenames.RouteNameLogin)
}
