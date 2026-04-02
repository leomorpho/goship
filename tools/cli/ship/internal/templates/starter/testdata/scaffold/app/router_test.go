package goship

import (
	"testing"

	"github.com/leomorpho/goship/starter/app/web/routenames"
)

func TestBuildRouterIncludesDefaultRoutes(t *testing.T) {
	routes := BuildRouter(nil)
	if len(routes) != 11 {
		t.Fatalf("expected 11 starter routes, got %d", len(routes))
	}

	want := []struct {
		name string
		path string
	}{
		{name: routenames.RouteNameLandingPage, path: "/"},
		{name: routenames.RouteNameLogin, path: "/auth/login"},
		{name: routenames.RouteNameRegister, path: "/auth/register"},
		{name: routenames.RouteNamePasswordReset, path: "/auth/password/reset"},
		{name: routenames.RouteNamePasswordResetConfirm, path: "/auth/password/reset/confirm"},
		{name: routenames.RouteNameSession, path: "/auth/session"},
		{name: routenames.RouteNameSettings, path: "/auth/settings"},
		{name: routenames.RouteNameAdmin, path: "/auth/admin"},
		{name: routenames.RouteNameDeleteAccount, path: "/auth/delete-account"},
		{name: routenames.RouteNameHomeFeed, path: "/auth/homeFeed"},
		{name: routenames.RouteNameProfile, path: "/auth/profile"},
	}

	for i, route := range routes {
		if route.Name != want[i].name {
			t.Fatalf("route %d name = %q, want %q", i, route.Name, want[i].name)
		}
		if route.Path != want[i].path {
			t.Fatalf("route %d path = %q, want %q", i, route.Path, want[i].path)
		}
		if route.Page == "" {
			t.Fatalf("route %d has empty page", i)
		}
		if route.Kind != RouteKindPage {
			t.Fatalf("route %d kind = %q, want %q", i, route.Kind, RouteKindPage)
		}
		if len(route.Actions) != 0 {
			t.Fatalf("route %d actions = %v, want empty for page routes", i, route.Actions)
		}
	}
}
