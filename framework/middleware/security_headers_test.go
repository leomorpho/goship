package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/leomorpho/goship/config"
)

func TestSecurityHeaders_Disabled(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	cfg := config.SecurityConfig{}
	cfg.Headers.Enabled = false

	handler := SecurityHeaders(cfg, string(config.EnvLocal))(func(c echo.Context) error {
		return c.NoContent(http.StatusNoContent)
	})
	if err := handler(c); err != nil {
		t.Fatalf("handler error: %v", err)
	}

	if got := rec.Header().Get("Content-Security-Policy"); got != "" {
		t.Fatalf("expected no CSP when disabled, got %q", got)
	}
}

func TestSecurityHeaders_EnabledAddsHeadersAndNonce(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	cfg := config.SecurityConfig{}
	cfg.Headers.Enabled = true
	cfg.Headers.HSTS = true

	handler := SecurityHeaders(cfg, string(config.EnvDevelop))(func(c echo.Context) error {
		nonce := CSPNonce(c)
		if nonce == "" {
			t.Fatal("expected nonce in context")
		}
		return c.NoContent(http.StatusNoContent)
	})
	if err := handler(c); err != nil {
		t.Fatalf("handler error: %v", err)
	}

	headers := rec.Header()
	assertHeader(t, headers.Get("X-Content-Type-Options"), "nosniff")
	assertHeader(t, headers.Get("X-Frame-Options"), "SAMEORIGIN")
	assertHeader(t, headers.Get("Referrer-Policy"), "strict-origin-when-cross-origin")
	assertHeader(t, headers.Get("Permissions-Policy"), "camera=(), microphone=(), geolocation=()")
	assertHeader(t, headers.Get("X-XSS-Protection"), "0")
	assertHeader(t, headers.Get("Strict-Transport-Security"), "max-age=31536000; includeSubDomains")

	csp := headers.Get("Content-Security-Policy")
	if csp == "" {
		t.Fatal("expected Content-Security-Policy header")
	}
	if !strings.Contains(csp, "script-src") {
		t.Fatalf("expected script-src in CSP, got %q", csp)
	}
	if !strings.Contains(csp, "nonce-") {
		t.Fatalf("expected nonce in CSP, got %q", csp)
	}
	if !strings.Contains(csp, "ws://localhost:5173") {
		t.Fatalf("expected dev websocket source in CSP, got %q", csp)
	}
}

func TestSecurityHeaders_CSPOverride(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	cfg := config.SecurityConfig{}
	cfg.Headers.Enabled = true
	cfg.Headers.CSP = "default-src 'self'"

	handler := SecurityHeaders(cfg, string(config.EnvProduction))(func(c echo.Context) error {
		return c.NoContent(http.StatusNoContent)
	})
	if err := handler(c); err != nil {
		t.Fatalf("handler error: %v", err)
	}

	assertHeader(t, rec.Header().Get("Content-Security-Policy"), "default-src 'self'")
}

func TestCSPNonce_Missing(t *testing.T) {
	if got := CSPNonce(nil); got != "" {
		t.Fatalf("expected empty nonce, got %q", got)
	}
}

func assertHeader(t *testing.T, got, want string) {
	t.Helper()
	if got != want {
		t.Fatalf("unexpected header value: got %q want %q", got, want)
	}
}
