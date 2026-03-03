package routes

import (
	routeNames "github.com/leomorpho/goship/app/goship/web/routenames"
	"github.com/leomorpho/goship/pkg/controller"
	"github.com/leomorpho/goship/pkg/repos/msg"

	"github.com/labstack/echo/v4"
)

type logout struct {
	ctr controller.Controller
}

func NewLogoutRoute(ctr controller.Controller) *logout {
	return &logout{ctr: ctr}
}

func (l *logout) Get(c echo.Context) error {
	if err := l.ctr.Container.Auth.Logout(c); err == nil {

	} else {
		msg.Danger(c, "An error occurred. Please try again.")
	}
	return l.ctr.Redirect(c, routeNames.RouteNameLandingPage)
}
