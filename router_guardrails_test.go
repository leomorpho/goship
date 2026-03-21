package goship

import (
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/leomorpho/goship-modules/notifications"
	paidsubscriptions "github.com/leomorpho/goship-modules/paidsubscriptions"
	"github.com/leomorpho/goship/config"
	frameworkbootstrap "github.com/leomorpho/goship/framework/bootstrap"
	"github.com/leomorpho/goship/framework/web/ui"
	"github.com/stretchr/testify/require"
)

func TestBuildRouter_RequiresPaidSubscriptionsModule(t *testing.T) {
	t.Parallel()

	err := BuildRouter(&frameworkbootstrap.Container{}, RouterModules{})
	require.EqualError(t, err, "missing paid subscriptions module")
}

func TestBuildRouter_RequiresNotificationsModule(t *testing.T) {
	t.Parallel()

	err := BuildRouter(&frameworkbootstrap.Container{}, RouterModules{
		PaidSubscriptions: &paidsubscriptions.Service{},
	})
	require.EqualError(t, err, "missing notifications module")
}

func TestRegisterRealtimeRoutes_RequiresNotifier(t *testing.T) {
	t.Parallel()

	c := &frameworkbootstrap.Container{}
	e := echo.New()
	s := e.Group("")
	ctr := ui.NewController(&frameworkbootstrap.Container{Config: testConfigForDevelop()})

	err := registerRealtimeRoutes(c, s, ctr)
	require.EqualError(t, err, "cannot register realtime routes: notifier is nil")
}

func TestResolveStartupWebFeatures_InvalidRuntimePlanFailsStartup(t *testing.T) {
	t.Parallel()

	cfg := testConfigForDevelop()
	cfg.Runtime.Profile = "broken"
	cfg.Processes.Web = true
	cfg.Adapters.PubSub = "inproc"

	_, _, err := resolveStartupWebFeatures(&frameworkbootstrap.Container{Config: cfg})
	require.EqualError(t, err, "invalid runtime plan: unknown runtime profile: broken")
}

func TestResolveStartupWebFeatures_RealtimeDependencyMismatchFailsStartup(t *testing.T) {
	t.Parallel()

	cfg := testConfigForDevelop()
	cfg.Runtime.Profile = config.RuntimeProfileServerDB
	cfg.Processes.Web = true
	cfg.Adapters.PubSub = "inproc"

	_, _, err := resolveStartupWebFeatures(&frameworkbootstrap.Container{Config: cfg})
	require.EqualError(t, err, "invalid startup capability contract: realtime requires notifier service")
}

func TestResolveStartupWebFeatures_ValidRealtimeConfig(t *testing.T) {
	t.Parallel()

	cfg := testConfigForDevelop()
	cfg.Runtime.Profile = config.RuntimeProfileServerDB
	cfg.Processes.Web = true
	cfg.Adapters.PubSub = "inproc"

	plan, features, err := resolveStartupWebFeatures(&frameworkbootstrap.Container{
		Config:   cfg,
		Notifier: &notifications.NotifierService{},
	})
	require.NoError(t, err)
	require.True(t, plan.RunWeb)
	require.True(t, features.EnableRealtime)
}
