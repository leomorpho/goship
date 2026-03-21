package controllers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/leomorpho/goship/app/foundation"
	"github.com/leomorpho/goship/config"
	"github.com/leomorpho/goship/framework/web/ui"
)

func newMailPreviewControllerForTest() MailPreviewRoute {
	cfg := &config.Config{
		App: config.AppConfig{
			Name:         "GoShip",
			SupportEmail: "support@example.com",
			Environment:  config.EnvDevelop,
		},
		HTTP: config.HTTPConfig{
			Domain: "https://example.test",
		},
	}
	ctr := ui.NewController(&foundation.Container{Config: cfg})
	return NewMailPreviewRoute(ctr)
}

func TestMailPreviewIndex(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/dev/mail", nil)
	rec := httptest.NewRecorder()
	ctx := e.NewContext(req, rec)

	route := newMailPreviewControllerForTest()
	if err := route.Index(ctx); err != nil {
		t.Fatalf("index: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{"/dev/mail/welcome", "/dev/mail/password-reset", "/dev/mail/verify-email"} {
		if !strings.Contains(body, want) {
			t.Fatalf("index body missing %q: %s", want, body)
		}
	}
}

func TestMailPreviewTemplateRoutes(t *testing.T) {
	tests := []struct {
		name string
		path string
		run  func(route *MailPreviewRoute, ctx echo.Context) error
	}{
		{name: "welcome", path: "/dev/mail/welcome", run: func(route *MailPreviewRoute, ctx echo.Context) error { return route.Welcome(ctx) }},
		{name: "password_reset", path: "/dev/mail/password-reset", run: func(route *MailPreviewRoute, ctx echo.Context) error { return route.PasswordReset(ctx) }},
		{name: "verify_email", path: "/dev/mail/verify-email", run: func(route *MailPreviewRoute, ctx echo.Context) error { return route.VerifyEmail(ctx) }},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			rec := httptest.NewRecorder()
			ctx := e.NewContext(req, rec)

			route := newMailPreviewControllerForTest()
			if err := tc.run(&route, ctx); err != nil {
				t.Fatalf("render preview: %v", err)
			}
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200", rec.Code)
			}
			if strings.TrimSpace(rec.Body.String()) == "" {
				t.Fatal("expected non-empty preview body")
			}
		})
	}
}
