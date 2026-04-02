package goship

import (
	"github.com/leomorpho/goship/starter/app/foundation"
	templates "github.com/leomorpho/goship/starter/app/views"
	pages "github.com/leomorpho/goship/starter/app/views/web/pages/gen"
	"github.com/leomorpho/goship/starter/app/web/routenames"
)

type Route struct {
	Name string
	Path string
	Page templates.Page
}

func BuildRouter(c *foundation.Container) []Route {
	if c == nil {
		c = foundation.NewContainer()
	}

	// Keep the generated page packages linked into the starter scaffold.
	_ = pages.HomeFeed
	_ = pages.Landing

	return []Route{
		{Name: routenames.RouteNameLandingPage, Path: "/", Page: templates.PageLanding},
		// ship:routes:public:start
		// ship:routes:public:end
		{Name: routenames.RouteNameLogin, Path: "/auth/login", Page: templates.PageLanding},
		{Name: routenames.RouteNameRegister, Path: "/auth/register", Page: templates.PageLanding},
		{Name: routenames.RouteNamePasswordReset, Path: "/auth/password/reset", Page: templates.PageLanding},
		{Name: routenames.RouteNamePasswordResetConfirm, Path: "/auth/password/reset/confirm", Page: templates.PageLanding},
		// ship:routes:auth:start
		// ship:routes:auth:end
		{Name: routenames.RouteNameSession, Path: "/auth/session", Page: templates.PageLanding},
		{Name: routenames.RouteNameSettings, Path: "/auth/settings", Page: templates.PageLanding},
		{Name: routenames.RouteNameDeleteAccount, Path: "/auth/delete-account", Page: templates.PageLanding},
		{Name: routenames.RouteNameHomeFeed, Path: "/auth/homeFeed", Page: templates.PageHomeFeed},
		{Name: routenames.RouteNameProfile, Path: "/auth/profile", Page: templates.PageProfile},
		// ship:routes:external:start
		// ship:routes:external:end
	}
}
