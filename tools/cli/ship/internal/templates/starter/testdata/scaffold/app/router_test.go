package goship

import (
	"testing"

	"github.com/leomorpho/goship/v2/starter/app/web/routenames"
)

func TestBuildRouterIncludesCoreRoutes(t *testing.T) {
	routes := BuildRouter(nil)
	if len(routes) < 5 {
		t.Fatalf("expected starter router to expose core routes, got %d", len(routes))
	}

	expected := map[string]string{
		routenames.RouteNameLandingPage: "/",
		routenames.RouteNameLogin:       "/auth/login",
		routenames.RouteNameRegister:    "/auth/register",
		routenames.RouteNameHomeFeed:    "/auth/homeFeed",
		routenames.RouteNameProfile:     "/auth/profile",
	}

	for _, route := range routes {
		wantPath, ok := expected[route.Name]
		if !ok {
			continue
		}
		if route.Path != wantPath {
			t.Fatalf("route %s path = %q, want %q", route.Name, route.Path, wantPath)
		}
		delete(expected, route.Name)
	}

	if len(expected) > 0 {
		t.Fatalf("starter router missing expected routes: %v", expected)
	}
}
