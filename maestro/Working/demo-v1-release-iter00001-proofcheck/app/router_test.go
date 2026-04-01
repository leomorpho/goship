package goship

import (
	"testing"

	"example.com/demo-v1-release-iter00001-proofcheck/app/web/routenames"
)

func TestBuildRouterIncludesDefaultRoutes(t *testing.T) {
	routes := BuildRouter(nil)
	if len(routes) != 5 {
		t.Fatalf("expected 5 starter routes, got %d", len(routes))
	}

	want := []struct {
		name string
		path string
	}{
		{name: routenames.RouteNameLandingPage, path: "/"},
		{name: routenames.RouteNameLogin, path: "/auth/login"},
		{name: routenames.RouteNameRegister, path: "/auth/register"},
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
	}
}
