package i18n

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestDetectLanguage_UsesAcceptLanguageHeader(t *testing.T) {
	service := testService(t)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Language", "fr, en;q=0.8")
	rec := httptest.NewRecorder()
	ctx := e.NewContext(req, rec)

	handler := DetectLanguage(service, nil)(func(c echo.Context) error {
		return c.String(http.StatusOK, service.T(c.Request().Context(), "auth.login.title"))
	})

	if err := handler(ctx); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if got := rec.Body.String(); got != "Connectez-vous a votre compte" {
		t.Fatalf("response body = %q", got)
	}
	if got := rec.Header().Get("Content-Language"); got != "fr" {
		t.Fatalf("content-language = %q, want fr", got)
	}
}

func TestDetectLanguage_QueryParamWinsAndSetsCookie(t *testing.T) {
	service := testService(t)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/?lang=fr", nil)
	rec := httptest.NewRecorder()
	ctx := e.NewContext(req, rec)

	handler := DetectLanguage(service, nil)(func(c echo.Context) error {
		return c.NoContent(http.StatusNoContent)
	})

	if err := handler(ctx); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if got := rec.Header().Get("Set-Cookie"); got == "" || !strings.Contains(got, "lang=fr") {
		t.Fatalf("set-cookie = %q, want lang cookie", got)
	}
}

func TestDetectLanguage_NilServiceFallsBackToEnglish(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	ctx := e.NewContext(req, rec)

	handler := DetectLanguage(nil, nil)(func(c echo.Context) error {
		return c.NoContent(http.StatusNoContent)
	})

	if err := handler(ctx); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if got := rec.Header().Get("Content-Language"); got != "en" {
		t.Fatalf("content-language = %q, want en", got)
	}
}

func testService(t *testing.T) *Service {
	t.Helper()

	dir := t.TempDir()
	writeTestLocale(t, dir, "en.yaml", `
auth:
  login:
    title: "Sign in to your account"
`)
	writeTestLocale(t, dir, "fr.yaml", `
auth:
  login:
    title: "Connectez-vous a votre compte"
`)

	service, err := NewService(Options{
		LocaleDir:       dir,
		DefaultLanguage: "en",
	})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	return service
}

func writeTestLocale(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir locales: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatalf("write locale: %v", err)
	}
}
