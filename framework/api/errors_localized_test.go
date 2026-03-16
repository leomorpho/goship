package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/labstack/echo/v4"
	i18nmodule "github.com/leomorpho/goship/modules/i18n"
)

func TestUnauthorizedLocalized_StableCodeAcrossLocales(t *testing.T) {
	service := testI18nService(t)
	e := echo.New()
	handler := i18nmodule.DetectLanguage(service, nil)(func(c echo.Context) error {
		return Fail(c, http.StatusUnauthorized, UnauthorizedLocalized(
			c.Request().Context(),
			service,
			"api.errors.unauthorized",
			"Unauthorized",
		))
	})

	frPayload, frHeader := runLocalizedAPIRequest(t, e, handler, "/api/v1/demo", "fr, en;q=0.8")
	if frHeader != "fr" {
		t.Fatalf("content-language = %q, want fr", frHeader)
	}
	if len(frPayload.Errors) != 1 {
		t.Fatalf("errors len = %d, want 1", len(frPayload.Errors))
	}
	if frPayload.Errors[0].Code != ErrorCodeUnauthorized {
		t.Fatalf("fr code = %q, want %q", frPayload.Errors[0].Code, ErrorCodeUnauthorized)
	}
	if frPayload.Errors[0].Message != "Non autorise" {
		t.Fatalf("fr message = %q, want %q", frPayload.Errors[0].Message, "Non autorise")
	}

	enPayload, enHeader := runLocalizedAPIRequest(t, e, handler, "/api/v1/demo", "en, fr;q=0.8")
	if enHeader != "en" {
		t.Fatalf("content-language = %q, want en", enHeader)
	}
	if len(enPayload.Errors) != 1 {
		t.Fatalf("errors len = %d, want 1", len(enPayload.Errors))
	}
	if enPayload.Errors[0].Code != ErrorCodeUnauthorized {
		t.Fatalf("en code = %q, want %q", enPayload.Errors[0].Code, ErrorCodeUnauthorized)
	}
	if enPayload.Errors[0].Message != "Unauthorized" {
		t.Fatalf("en message = %q, want %q", enPayload.Errors[0].Message, "Unauthorized")
	}
}

func TestNotFoundLocalized_FallsBackToDefaultLocaleAndFallbackText(t *testing.T) {
	service := testI18nService(t)
	e := echo.New()

	fallbackHandler := i18nmodule.DetectLanguage(service, nil)(func(c echo.Context) error {
		return Fail(c, http.StatusNotFound, NotFoundLocalized(
			c.Request().Context(),
			service,
			"api.errors.not_found",
			"Resource not found",
		))
	})

	payload, header := runLocalizedAPIRequest(t, e, fallbackHandler, "/api/v1/demo", "es-MX,es;q=0.9")
	if header != "en" {
		t.Fatalf("content-language = %q, want en", header)
	}
	if len(payload.Errors) != 1 {
		t.Fatalf("errors len = %d, want 1", len(payload.Errors))
	}
	if payload.Errors[0].Code != ErrorCodeNotFound {
		t.Fatalf("code = %q, want %q", payload.Errors[0].Code, ErrorCodeNotFound)
	}
	if payload.Errors[0].Message != "Resource not found" {
		t.Fatalf("message = %q, want %q", payload.Errors[0].Message, "Resource not found")
	}

	missingKeyHandler := i18nmodule.DetectLanguage(service, nil)(func(c echo.Context) error {
		return Fail(c, http.StatusNotFound, NotFoundLocalized(
			c.Request().Context(),
			service,
			"api.errors.missing",
			"Fallback text",
		))
	})
	missingPayload, _ := runLocalizedAPIRequest(t, e, missingKeyHandler, "/api/v1/demo", "fr, en;q=0.8")
	if len(missingPayload.Errors) != 1 {
		t.Fatalf("errors len = %d, want 1", len(missingPayload.Errors))
	}
	if missingPayload.Errors[0].Message != "Fallback text" {
		t.Fatalf("message = %q, want %q", missingPayload.Errors[0].Message, "Fallback text")
	}
}

func runLocalizedAPIRequest(t *testing.T, e *echo.Echo, handler echo.HandlerFunc, path string, acceptLanguage string) (Response[struct{}], string) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	if acceptLanguage != "" {
		req.Header.Set("Accept-Language", acceptLanguage)
	}
	rec := httptest.NewRecorder()
	ctx := e.NewContext(req, rec)
	if err := handler(ctx); err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if rec.Code == 0 {
		t.Fatal("response status code is zero")
	}

	var payload Response[struct{}]
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	return payload, rec.Header().Get("Content-Language")
}

func testI18nService(t *testing.T) *i18nmodule.Service {
	t.Helper()
	dir := t.TempDir()
	writeLocaleFile(t, dir, "en.toml", `
"api.errors.not_found" = "Resource not found"
"api.errors.unauthorized" = "Unauthorized"
`)
	writeLocaleFile(t, dir, "fr.toml", `
"api.errors.not_found" = "Ressource introuvable"
"api.errors.unauthorized" = "Non autorise"
`)

	service, err := i18nmodule.NewService(i18nmodule.Options{
		LocaleDir:       dir,
		DefaultLanguage: "en",
	})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	return service
}

func writeLocaleFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir locale dir: %v", err)
	}
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write locale file %s: %v", name, err)
	}
}
