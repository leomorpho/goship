package goship

import (
	"github.com/leomorpho/goship/starter/app/foundation"
	templates "github.com/leomorpho/goship/starter/app/views"
	pages "github.com/leomorpho/goship/starter/app/views/web/pages/gen"
	"github.com/leomorpho/goship/starter/app/web/routenames"
)

type Route struct {
	Name         string
	Path         string
	Page         templates.Page
	Kind         RouteKind
	Actions      []string
	StorageTable string
	Fields       []RouteField
}

type RouteField struct {
	Name string
	Type string
}

type RouteKind string

const (
	RouteKindPage     RouteKind = "page"
	RouteKindResource RouteKind = "resource"
)

func BuildRouter(c *foundation.Container) []Route {
	if c == nil {
		c = foundation.NewContainer()
	}

	// Keep the generated page packages linked into the starter scaffold.
	_ = pages.HomeFeed
	_ = pages.Landing

	return []Route{
		{Name: routenames.RouteNameLandingPage, Path: "/", Page: templates.PageLanding, Kind: RouteKindPage},
		// ship:routes:public:start
		// ship:routes:public:end
		{Name: routenames.RouteNameLogin, Path: "/auth/login", Page: templates.PageLanding, Kind: RouteKindPage},
		{Name: routenames.RouteNameRegister, Path: "/auth/register", Page: templates.PageLanding, Kind: RouteKindPage},
		{Name: routenames.RouteNamePasswordReset, Path: "/auth/password/reset", Page: templates.PageLanding, Kind: RouteKindPage},
		{Name: routenames.RouteNamePasswordResetConfirm, Path: "/auth/password/reset/confirm", Page: templates.PageLanding, Kind: RouteKindPage},
		// ship:routes:auth:start
		// ship:routes:auth:end
		{Name: routenames.RouteNameSession, Path: "/auth/session", Page: templates.PageLanding, Kind: RouteKindPage},
		{Name: routenames.RouteNameSettings, Path: "/auth/settings", Page: templates.PageLanding, Kind: RouteKindPage},
		{Name: routenames.RouteNameAdmin, Path: "/auth/admin", Page: templates.PageLanding, Kind: RouteKindPage},
		{Name: routenames.RouteNameDeleteAccount, Path: "/auth/delete-account", Page: templates.PageLanding, Kind: RouteKindPage},
		{Name: routenames.RouteNameHomeFeed, Path: "/auth/homeFeed", Page: templates.PageHomeFeed, Kind: RouteKindPage},
		{Name: routenames.RouteNameProfile, Path: "/auth/profile", Page: templates.PageProfile, Kind: RouteKindPage},
		// ship:routes:external:start
		// ship:routes:external:end
	}
}
