package goship

import (
	"testing"

	"github.com/labstack/echo/v4"
	paidsubscriptions "github.com/leomorpho/goship-modules/paidsubscriptions"
	"github.com/leomorpho/goship/app/foundation"
	"github.com/leomorpho/goship/app/web/ui"
	"github.com/stretchr/testify/require"
)

func TestBuildRouter_RequiresPaidSubscriptionsModule(t *testing.T) {
	t.Parallel()

	err := BuildRouter(nil, RouterModules{})
	require.EqualError(t, err, "missing paid subscriptions module")
}

func TestBuildRouter_RequiresNotificationsModule(t *testing.T) {
	t.Parallel()

	err := BuildRouter(nil, RouterModules{
		PaidSubscriptions: &paidsubscriptions.Service{},
	})
	require.EqualError(t, err, "missing notifications module")
}

func TestRegisterRealtimeRoutes_RequiresNotifier(t *testing.T) {
	t.Parallel()

	c := &foundation.Container{}
	e := echo.New()
	s := e.Group("")
	ctr := ui.NewController(&foundation.Container{Config: testConfigForDevelop()})

	err := registerRealtimeRoutes(c, s, ctr)
	require.EqualError(t, err, "cannot register realtime routes: notifier is nil")
}

func TestBuildRouter_InvalidRuntimePlanFailsStartup_RedSpec(t *testing.T) {
	t.Skip("red-spec only (TKT-257): enable when invalid runtime plans return startup errors instead of safe fallback")
}

func TestBuildRouter_RealtimeDependencyMismatchFailsStartup_RedSpec(t *testing.T) {
	t.Skip("red-spec only (TKT-257): enable when missing realtime dependencies fail startup instead of silently disabling routes")
}
