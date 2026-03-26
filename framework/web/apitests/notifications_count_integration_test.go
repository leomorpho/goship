//go:build integration

package apitests

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	modnotifications "github.com/leomorpho/goship-modules/notifications/routes"
	customctx "github.com/leomorpho/goship/framework/appcontext"
	"github.com/leomorpho/goship/framework/web/ui"
)

type fakeNotificationCountReader struct {
	count int
	err   error
}

func (f fakeNotificationCountReader) GetCountOfUnseenNotifications(context.Context, int) (int, error) {
	if f.err != nil {
		return 0, f.err
	}
	return f.count, nil
}

func TestNotificationsCountAPIReadPath(t *testing.T) {
	e := echo.New()
	route := modnotifications.NewNormalNotificationsCountRoute(ui.Controller{}, fakeNotificationCountReader{count: 3})
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

func TestNotificationsCountAPIReadPath_MissingProfileID(t *testing.T) {
	e := echo.New()
	route := modnotifications.NewNormalNotificationsCountRoute(ui.Controller{}, fakeNotificationCountReader{count: 3})
	e.GET("/auth/notifications/normalNotificationsCount", route.Get)

	req := httptest.NewRequest(http.MethodGet, "/auth/notifications/normalNotificationsCount", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestNotificationsCountAPIReadPath_StoreError(t *testing.T) {
	e := echo.New()
	route := modnotifications.NewNormalNotificationsCountRoute(ui.Controller{}, fakeNotificationCountReader{err: errors.New("store failed")})
	e.GET("/auth/notifications/normalNotificationsCount", route.Get, func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set(customctx.AuthenticatedProfileIDKey, 77)
			return next(c)
		}
	})

	req := httptest.NewRequest(http.MethodGet, "/auth/notifications/normalNotificationsCount", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}
