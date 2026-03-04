package controllers

import (
	routeNames "github.com/leomorpho/goship/apps/site/web/routenames"
	"github.com/leomorpho/goship/apps/site/web/ui"
	"github.com/leomorpho/goship/framework/repos/msg"

	"github.com/labstack/echo/v4"
)

type logout struct {
	ctr ui.Controller
}

func NewLogoutRoute(ctr ui.Controller) *logout {
	return &logout{ctr: ctr}
}

func (l *logout) Get(c echo.Context) error {
	if err := l.ctr.Container.Auth.Logout(c); err == nil {

	} else {
		msg.Danger(c, "An error occurred. Please try again.")
	}
	return l.ctr.Redirect(c, routeNames.RouteNameLandingPage)
}
