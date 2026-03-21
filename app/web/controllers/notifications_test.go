package controllers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	modnotifications "github.com/leomorpho/goship-modules/notifications/routes"
	customctx "github.com/leomorpho/goship/framework/context"
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

func TestAuthenticatedProfileID(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	ctx := e.NewContext(req, rec)

	if _, err := authenticatedProfileID(ctx); err == nil {
		t.Fatal("expected error when profile id context key is missing")
	}

	ctx.Set(customctx.AuthenticatedProfileIDKey, 0)
	if _, err := authenticatedProfileID(ctx); err == nil {
		t.Fatal("expected error for non-positive profile id")
	}

	ctx.Set(customctx.AuthenticatedProfileIDKey, 42)
	got, err := authenticatedProfileID(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != 42 {
		t.Fatalf("profileID = %d, want 42", got)
	}
}

func TestNormalNotificationsCount_Get(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	ctx := e.NewContext(req, rec)
	ctx.Set(customctx.AuthenticatedProfileIDKey, 123)

	route := modnotifications.NewNormalNotificationsCountRoute(ui.Controller{}, fakeNotificationCountReader{count: 5})
	if err := route.Get(ctx); err != nil {
		t.Fatalf("route get returned error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Body.String(); got != "<span>5</span>" {
		t.Fatalf("body = %q", got)
	}
}

func TestNormalNotificationsCount_Get_HidesZero(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	ctx := e.NewContext(req, rec)
	ctx.Set(customctx.AuthenticatedProfileIDKey, 123)

	route := modnotifications.NewNormalNotificationsCountRoute(ui.Controller{}, fakeNotificationCountReader{count: 0})
	if err := route.Get(ctx); err != nil {
		t.Fatalf("route get returned error: %v", err)
	}
	if got := rec.Body.String(); got != "<span class='hidden'>0</span>" {
		t.Fatalf("body = %q", got)
	}
}

func TestNormalNotificationsCount_Get_PropagatesReaderError(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	ctx := e.NewContext(req, rec)
	ctx.Set(customctx.AuthenticatedProfileIDKey, 123)

	wantErr := errors.New("count failed")
	route := modnotifications.NewNormalNotificationsCountRoute(ui.Controller{}, fakeNotificationCountReader{err: wantErr})
	err := route.Get(ctx)
	if !errors.Is(err, wantErr) {
		t.Fatalf("error = %v, want %v", err, wantErr)
	}
}

func TestNormalNotificationsCount_Get_RequiresProfileID(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	ctx := e.NewContext(req, rec)

	route := modnotifications.NewNormalNotificationsCountRoute(ui.Controller{}, fakeNotificationCountReader{count: 1})
	if err := route.Get(ctx); err == nil {
		t.Fatal("expected error when profile id is missing")
	}
}
