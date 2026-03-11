//go:build integration

package controllers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	modnotifications "github.com/leomorpho/goship-modules/notifications/routes"
	"github.com/leomorpho/goship/app/web/ui"
	customctx "github.com/leomorpho/goship/framework/context"
)

type integrationNotificationCountReader struct{}

func (integrationNotificationCountReader) GetCountOfUnseenNotifications(context.Context, int) (int, error) {
	return 3, nil
}

func TestNormalNotificationsCountRoute_APIReadPath(t *testing.T) {
	e := echo.New()

	route := modnotifications.NewNormalNotificationsCountRoute(ui.Controller{}, integrationNotificationCountReader{})
	e.GET("/auth/notifications/normalNotificationsCount", route.Get, func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set(customctx.AuthenticatedProfileIDKey, 77)
			return next(c)
		}
	})

	req := httptest.NewRequest(http.MethodGet, "/auth/notifications/normalNotificationsCount", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if body := rec.Body.String(); body != "<span>3</span>" {
		t.Fatalf("body = %q, want %q", body, "<span>3</span>")
	}
}
