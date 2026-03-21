//go:build integration

package controllers_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"testing"
	"time"

	frameworktests "github.com/leomorpho/goship/framework/tests"
	routeNames "github.com/leomorpho/goship/framework/web/routenames"
	profilesvc "github.com/leomorpho/goship/modules/profile"
	"github.com/stretchr/testify/require"
)

type smokeClientState string

const (
	smokeClientAnonymous  smokeClientState = "anonymous"
	smokeClientOnboarding smokeClientState = "onboarding"
	smokeClientOnboarded  smokeClientState = "onboarded"
)

func TestRouteSmoke_PublicAndAuthenticatedGets(t *testing.T) {
	onboardingClient, _, _ := loginSmokeUser(t, false)
	onboardedClient, _, _ := loginSmokeUser(t, true)

	tests := []struct {
		name             string
		routeName        string
		client           smokeClientState
		wantStatus       int
		wantBodyContains string
	}{
		{name: "landing", routeName: routeNames.RouteNameLandingPage, client: smokeClientAnonymous, wantStatus: http.StatusOK, wantBodyContains: "GoShip"},
		{name: "healthcheck", routeName: routeNames.RouteNameHealthcheck, client: smokeClientAnonymous, wantStatus: http.StatusOK, wantBodyContains: "ok"},
		{name: "health liveness", routeName: routeNames.RouteNameHealthLiveness, client: smokeClientAnonymous, wantStatus: http.StatusOK, wantBodyContains: "alive"},
		{name: "health readiness", routeName: routeNames.RouteNameHealthReadiness, client: smokeClientAnonymous, wantStatus: http.StatusOK, wantBodyContains: "ready"},
		{name: "install app", routeName: routeNames.RouteNameInstallApp, client: smokeClientAnonymous, wantStatus: http.StatusOK},
		{name: "login", routeName: routeNames.RouteNameLogin, client: smokeClientAnonymous, wantStatus: http.StatusOK},
		{name: "register", routeName: routeNames.RouteNameRegister, client: smokeClientAnonymous, wantStatus: http.StatusOK},
		{name: "forgot password", routeName: routeNames.RouteNameForgotPassword, client: smokeClientAnonymous, wantStatus: http.StatusOK},
		{name: "preferences", routeName: routeNames.RouteNamePreferences, client: smokeClientOnboarding, wantStatus: http.StatusOK},
		{name: "preferences phone", routeName: routeNames.RouteNameGetPhone, client: smokeClientOnboarding, wantStatus: http.StatusOK},
		{name: "preferences phone verification", routeName: routeNames.RouteNameGetPhoneVerification, client: smokeClientOnboarding, wantStatus: http.StatusOK},
		{name: "preferences display name", routeName: routeNames.RouteNameGetDisplayName, client: smokeClientOnboarding, wantStatus: http.StatusOK},
		{name: "delete account", routeName: routeNames.RouteNameDeleteAccountPage, client: smokeClientOnboarding, wantStatus: http.StatusOK},
		{name: "finish onboarding", routeName: routeNames.RouteNameFinishOnboarding, client: smokeClientOnboarding, wantStatus: http.StatusOK},
		{name: "profile bio", routeName: routeNames.RouteNameGetBio, client: smokeClientOnboarding, wantStatus: http.StatusOK},
		{name: "home feed", routeName: routeNames.RouteNameHomeFeed, client: smokeClientOnboarded, wantStatus: http.StatusOK, wantBodyContains: "Home"},
		{name: "home feed buttons", routeName: routeNames.RouteNameGetHomeFeedButtons, client: smokeClientOnboarded, wantStatus: http.StatusOK},
		{name: "profile", routeName: routeNames.RouteNameProfile, client: smokeClientOnboarded, wantStatus: http.StatusOK},
		{name: "upload photo", routeName: routeNames.RouteNameUploadPhoto, client: smokeClientOnboarded, wantStatus: http.StatusOK},
		{name: "current profile photo", routeName: routeNames.RouteNameCurrentProfilePhoto, client: smokeClientOnboarded, wantStatus: http.StatusOK},
		{name: "normal notifications count", routeName: routeNames.RouteNameNormalNotificationsCount, client: smokeClientOnboarded, wantStatus: http.StatusOK},
		{name: "push subscriptions", routeName: routeNames.RouteNameGetPushSubscriptions, client: smokeClientOnboarded, wantStatus: http.StatusOK},
		{name: "pricing", routeName: routeNames.RouteNamePricingPage, client: smokeClientOnboarded, wantStatus: http.StatusOK},
		{name: "payment public key", routeName: routeNames.RouteNamePaymentProcessorGetPublicKey, client: smokeClientOnboarded, wantStatus: http.StatusOK},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := request(t).setRoute(tc.routeName)
			switch tc.client {
			case smokeClientOnboarding:
				req = req.setClient(onboardingClient)
			case smokeClientOnboarded:
				req = req.setClient(onboardedClient)
			}

			resp := req.get().assertStatusCode(tc.wantStatus)
			if tc.wantBodyContains != "" {
				body, err := io.ReadAll(resp.Body)
				require.NoError(t, err)
				require.NoError(t, resp.Body.Close())
				require.Contains(t, string(body), tc.wantBodyContains)
			}
		})
	}
}

func loginSmokeUser(t *testing.T, onboarded bool) (http.Client, int, string) {
	t.Helper()

	ctx := context.Background()
	email := fmt.Sprintf("smoke-%d@example.com", time.Now().UnixNano())
	password := "password123"
	passwordHash, err := c.Auth.HashPassword(password)
	require.NoError(t, err)

	user, err := frameworktests.CreateUserDB(ctx, c.Database, "Smoke User", email, passwordHash, true)
	require.NoError(t, err)

	profileService := profilesvc.NewProfileServiceWithDBDeps(c.Database, c.Config.Adapters.DB, nil, nil, nil)
	profileID, err := profileService.CreateProfile(
		ctx,
		user.ID,
		"",
		time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
		nil,
		nil,
	)
	require.NoError(t, err)
	require.NoError(t, initializeSmokeProfile(profileID))
	if onboarded {
		require.NoError(t, markSmokeUserOnboarded(profileID))
	}

	req := request(t)
	resp := req.
		setRoute(routeNames.RouteNameLogin).
		setBody(url.Values{
			"email":    []string{email},
			"password": []string{password},
		}).
		post()

	require.Equal(t, http.StatusOK, resp.StatusCode)
	return req.client, profileID, email
}

func markSmokeUserOnboarded(profileID int) error {
	_, err := c.Database.Exec(`
		UPDATE profiles
		SET fully_onboarded = 1
		WHERE id = ?
	`, profileID)
	return err
}

func initializeSmokeProfile(profileID int) error {
	_, err := c.Database.Exec(`
		UPDATE profiles
		SET phone_verified = 0
		WHERE id = ?
	`, profileID)
	return err
}
