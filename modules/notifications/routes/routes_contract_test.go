package routes

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	routeNames "github.com/leomorpho/goship/framework/web/routenames"
)

func TestRouteModule_RegisterRoutes_CanonicalNotificationSurface(t *testing.T) {
	t.Parallel()

	e := echo.New()
	onboarding := e.Group("/welcome")
	onboarded := e.Group("/auth")

	module := NewRouteModule(RouteModuleDeps{})
	if err := module.RegisterOnboardingRoutes(onboarding); err != nil {
		t.Fatalf("RegisterOnboardingRoutes() error = %v", err)
	}
	if err := module.RegisterRoutes(onboarded); err != nil {
		t.Fatalf("RegisterRoutes() error = %v", err)
	}

	want := map[string]*string{
		"GET /welcome/subscription/push":                                 stringPtr(routeNames.RouteNameGetPushSubscriptions),
		"POST /welcome/subscription/:platform":                           stringPtr(routeNames.RouteNameRegisterSubscription),
		"DELETE /welcome/subscription/:platform":                         stringPtr(routeNames.RouteNameDeleteSubscription),
		"GET /welcome/email-subscription/unsubscribe/:permission/:token": stringPtr(routeNames.RouteNameDeleteEmailSubscriptionWithToken),
		"GET /auth/notifications":                                        stringPtr(routeNames.RouteNameNotifications),
		"GET /auth/notifications/mark-all-read":                          stringPtr(routeNames.RouteNameMarkAllNotificationsAsRead),
		"DELETE /auth/notifications/:notification_id":                    stringPtr(routeNames.RouteNameDeleteNotification),
		"GET /auth/notifications/normalNotificationsCount":               stringPtr(routeNames.RouteNameNormalNotificationsCount),
		"POST /auth/notifications/:notification_id/read":                 stringPtr(routeNames.RouteNameMarkNotificationsAsRead),
		"POST /auth/notifications/unread":                                stringPtr(routeNames.RouteNameMarkNotificationsAsUnread),
	}

	got := make(map[string]string, len(e.Routes()))
	for _, route := range e.Routes() {
		got[route.Method+" "+route.Path] = route.Name
	}

	for key, wantName := range want {
		gotName, ok := got[key]
		if !ok {
			t.Fatalf("missing notification route %q; got routes: %#v", key, got)
		}
		if wantName == nil {
			continue
		}
		if gotName != *wantName {
			t.Fatalf("route %q name = %q, want %q", key, gotName, *wantName)
		}
	}
}

func TestNotificationRouteSurface_UsesCanonicalRouteNameConstants(t *testing.T) {
	for _, rel := range []string{
		filepath.Join("modules", "notifications", "routes", "routes.go"),
		filepath.Join("framework", "web", "components", "gen", "navbar_templ.go"),
		filepath.Join("framework", "web", "components", "gen", "drawer_templ.go"),
		filepath.Join("framework", "web", "components", "gen", "bottom_nav_templ.go"),
	} {
		content, err := os.ReadFile(filepath.Join("..", "..", "..", rel))
		if err != nil {
			t.Fatalf("read %s: %v", rel, err)
		}
		text := string(content)
		for _, legacy := range []string{
			"\"normalNotifications\"",
			"\"normalNotificationsCount\"",
			"\"markNormalNotificationUnread\"",
		} {
			if strings.Contains(text, legacy) {
				t.Fatalf("%s still contains raw notification route name %s", rel, legacy)
			}
		}
	}
}

func TestNotificationRouteSurface_WiresCanonicalNotificationsView(t *testing.T) {
	t.Parallel()

	path := filepath.Join("routes.go")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	text := string(content)
	for _, token := range []string{
		"templates.PageNotifications",
		"pages.NotificationsPage",
		"domain.BottomNavbarItemNotifications",
		"viewmodels.NewNormalNotificationsPageData",
	} {
		if !strings.Contains(text, token) {
			t.Fatalf("routes.go missing notifications view wiring token %q", token)
		}
	}
}

func stringPtr(value string) *string {
	return &value
}
