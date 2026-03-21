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

func TestStartupFailurePathContract_RedSpec(t *testing.T) {
	t.Run("build router failure paths stay explicit", func(t *testing.T) {
		t.Parallel()

		cases := []struct {
			name    string
			c       *frameworkbootstrap.Container
			modules RouterModules
			wantErr string
		}{
			{
				name: "nil container",
				c:    nil,
				modules: RouterModules{
					PaidSubscriptions: &paidsubscriptions.Service{},
					Notifications:     &notifications.Services{},
				},
				wantErr: "invalid router container: nil",
			},
			{
				name:    "missing paid subscriptions module",
				c:       &frameworkbootstrap.Container{},
				modules: RouterModules{},
				wantErr: "missing paid subscriptions module",
			},
			{
				name: "missing notifications module",
				c:    &frameworkbootstrap.Container{},
				modules: RouterModules{
					PaidSubscriptions: &paidsubscriptions.Service{},
				},
				wantErr: "missing notifications module",
			},
		}

		for _, tc := range cases {
			tc := tc
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				require.EqualError(t, BuildRouter(tc.c, tc.modules), tc.wantErr)
			})
		}
	})

	t.Run("runtime startup capability failures stay explicit", func(t *testing.T) {
		t.Parallel()

		brokenProfile := testConfigForDevelop()
		brokenProfile.Runtime.Profile = "broken"
		brokenProfile.Processes.Web = true
		brokenProfile.Adapters.PubSub = "inproc"

		realtimeMismatch := testConfigForDevelop()
		realtimeMismatch.Runtime.Profile = config.RuntimeProfileServerDB
		realtimeMismatch.Processes.Web = true
		realtimeMismatch.Adapters.PubSub = "inproc"

		cases := []struct {
			name    string
			c       *frameworkbootstrap.Container
			wantErr string
		}{
			{
				name:    "nil config",
				c:       &frameworkbootstrap.Container{},
				wantErr: "invalid runtime container: config is nil",
			},
			{
				name:    "invalid runtime profile",
				c:       &frameworkbootstrap.Container{Config: brokenProfile},
				wantErr: "invalid runtime plan: unknown runtime profile: broken",
			},
			{
				name:    "realtime dependency mismatch",
				c:       &frameworkbootstrap.Container{Config: realtimeMismatch},
				wantErr: "invalid startup capability contract: realtime requires notifier service",
			},
		}

		for _, tc := range cases {
			tc := tc
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				_, _, err := resolveStartupWebFeatures(tc.c)
				require.EqualError(t, err, tc.wantErr)
			})
		}
	})

	t.Run("realtime route registration requires notifier", func(t *testing.T) {
		t.Parallel()

		e := echo.New()
		group := e.Group("")
		ctr := ui.NewController(&frameworkbootstrap.Container{Config: testConfigForDevelop()})

		err := registerRealtimeRoutes(&frameworkbootstrap.Container{}, group, ctr)
		require.EqualError(t, err, "cannot register realtime routes: notifier is nil")
	})
}
