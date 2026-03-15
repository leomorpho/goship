package goship

import (
	"testing"

	"github.com/leomorpho/goship/starter/app/foundation"
)

func TestBuildRouter_StarterScope(t *testing.T) {
	container := foundation.NewContainer()
	routes := BuildRouter(container)

	if len(routes) != 5 {
		t.Fatalf("route count = %d, want 5", len(routes))
	}
	if !container.SupportsModule("auth") || !container.SupportsModule("profile") {
		t.Fatal("starter must include auth and profile modules")
	}
	if container.SupportsModule("pwa") || container.SupportsModule("paidsubscriptions") {
		t.Fatal("starter must exclude pwa and paidsubscriptions modules")
	}
}
