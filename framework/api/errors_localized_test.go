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

type localizedAPIRequestOpts struct {
	path           string
	acceptLanguage string
	cookies        []*http.Cookie
}

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

func TestUnauthorizedLocalized_LanguageResolutionPriority(t *testing.T) {
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

	queryPayload, queryHeader := runLocalizedAPIRequestWithOptions(t, e, handler, localizedAPIRequestOpts{
		path:           "/api/v1/demo?lang=fr",
		acceptLanguage: "en, fr;q=0.8",
		cookies:        []*http.Cookie{{Name: "lang", Value: "en"}},
	})
	if queryHeader != "fr" {
		t.Fatalf("query-driven content-language = %q, want fr", queryHeader)
	}
	if queryPayload.Errors[0].Code != ErrorCodeUnauthorized {
		t.Fatalf("query-driven code = %q, want %q", queryPayload.Errors[0].Code, ErrorCodeUnauthorized)
	}

	cookiePayload, cookieHeader := runLocalizedAPIRequestWithOptions(t, e, handler, localizedAPIRequestOpts{
		path:           "/api/v1/demo",
		acceptLanguage: "en, fr;q=0.8",
		cookies:        []*http.Cookie{{Name: "lang", Value: "fr"}},
	})
	if cookieHeader != "fr" {
		t.Fatalf("cookie content-language = %q, want fr", cookieHeader)
	}
	if cookiePayload.Errors[0].Message != "Non autorise" {
		t.Fatalf("cookie message = %q, want %q", cookiePayload.Errors[0].Message, "Non autorise")
	}

	headerPayload, headerHeader := runLocalizedAPIRequestWithOptions(t, e, handler, localizedAPIRequestOpts{
		path:           "/api/v1/demo",
		acceptLanguage: "fr, en;q=0.8",
	})
	if headerHeader != "fr" {
		t.Fatalf("header content-language = %q, want fr", headerHeader)
	}
	if headerPayload.Errors[0].Code != ErrorCodeUnauthorized {
		t.Fatalf("header code = %q, want %q", headerPayload.Errors[0].Code, ErrorCodeUnauthorized)
	}

	defaultPayload, defaultHeader := runLocalizedAPIRequestWithOptions(t, e, handler, localizedAPIRequestOpts{
		path:           "/api/v1/demo",
		acceptLanguage: "es-MX,es;q=0.9",
	})
	if defaultHeader != "en" {
		t.Fatalf("default content-language = %q, want en", defaultHeader)
	}
	if defaultPayload.Errors[0].Message != "Unauthorized" {
		t.Fatalf("default message = %q, want %q", defaultPayload.Errors[0].Message, "Unauthorized")
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
	return runLocalizedAPIRequestWithOptions(t, e, handler, localizedAPIRequestOpts{
		path:           path,
		acceptLanguage: acceptLanguage,
	})
}

func runLocalizedAPIRequestWithOptions(t *testing.T, e *echo.Echo, handler echo.HandlerFunc, opts localizedAPIRequestOpts) (Response[struct{}], string) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, opts.path, nil)
	if opts.acceptLanguage != "" {
		req.Header.Set("Accept-Language", opts.acceptLanguage)
	}
	for _, cookie := range opts.cookies {
		req.AddCookie(cookie)
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
