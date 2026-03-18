package goship

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/leomorpho/goship-modules/notifications"
	paidsubscriptions "github.com/leomorpho/goship-modules/paidsubscriptions"
)

func TestBuildRouter_RequiresContainer_RedSpec(t *testing.T) {
	t.Skip("red spec: enable once BuildRouter rejects a nil container before route/module wiring")

	err := BuildRouter(nil, RouterModules{
		PaidSubscriptions: &paidsubscriptions.Service{},
		Notifications:     &notifications.Services{},
	})
	if err == nil {
		t.Fatal("expected explicit error for nil container")
	}
	if err.Error() != "invalid router container: nil" {
		t.Fatalf("error = %q, want %q", err.Error(), "invalid router container: nil")
	}
}

func TestRouterComposition_UsesSingleStaticRegistrationPath_RedSpec(t *testing.T) {
	t.Skip("red spec: enable once app/router.go routes all static registration through appweb.RegisterStaticRoutes")

	content, err := os.ReadFile(filepath.Join("router.go"))
	if err != nil {
		t.Fatalf("read router.go: %v", err)
	}
	text := string(content)

	if strings.Count(text, "RegisterStaticRoutes(") != 1 {
		t.Fatalf("router.go should contain exactly one static registration call, got %d", strings.Count(text, "RegisterStaticRoutes("))
	}
	if strings.Contains(text, "RegisterStaticRoutes(c.Web") {
		t.Fatal("router.go still contains module-level static route registration instead of the canonical appweb.RegisterStaticRoutes(c) path")
	}
}
