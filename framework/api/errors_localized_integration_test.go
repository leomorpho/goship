//go:build integration

package api

import (
	"encoding/json"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	i18nmodule "github.com/leomorpho/goship/modules/i18n"
)

func TestLocalizedAPIIntegration_LanguageResolutionAndCodeStability(t *testing.T) {
	service := testI18nService(t)
	e := echo.New()
	e.Use(i18nmodule.DetectLanguage(service, nil))
	e.GET("/api/v1/demo", func(c echo.Context) error {
		return Fail(c, http.StatusUnauthorized, UnauthorizedLocalized(
			c.Request().Context(),
			service,
			"api.errors.unauthorized",
			"Unauthorized",
		))
	})

	server := httptest.NewServer(e)
	defer server.Close()

	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("cookie jar: %v", err)
	}
	client := &http.Client{Jar: jar}

	queryPayload, queryLang := runLocalizedHTTPRequest(t, client, server.URL+"/api/v1/demo?lang=fr", "en, fr;q=0.8")
	if queryLang != "fr" {
		t.Fatalf("query content-language = %q, want fr", queryLang)
	}
	if len(queryPayload.Errors) != 1 {
		t.Fatalf("query errors len = %d, want 1", len(queryPayload.Errors))
	}
	if queryPayload.Errors[0].Code != ErrorCodeUnauthorized {
		t.Fatalf("query code = %q, want %q", queryPayload.Errors[0].Code, ErrorCodeUnauthorized)
	}

	cookiePayload, cookieLang := runLocalizedHTTPRequest(t, client, server.URL+"/api/v1/demo", "en, fr;q=0.8")
	if cookieLang != "fr" {
		t.Fatalf("cookie content-language = %q, want fr", cookieLang)
	}
	if cookiePayload.Errors[0].Message != "Non autorise" {
		t.Fatalf("cookie message = %q, want %q", cookiePayload.Errors[0].Message, "Non autorise")
	}
	if cookiePayload.Errors[0].Code != ErrorCodeUnauthorized {
		t.Fatalf("cookie code = %q, want %q", cookiePayload.Errors[0].Code, ErrorCodeUnauthorized)
	}

	noCookieClient := &http.Client{}
	defaultPayload, defaultLang := runLocalizedHTTPRequest(t, noCookieClient, server.URL+"/api/v1/demo", "es-MX,es;q=0.9")
	if defaultLang != "en" {
		t.Fatalf("default content-language = %q, want en", defaultLang)
	}
	if defaultPayload.Errors[0].Code != ErrorCodeUnauthorized {
		t.Fatalf("default code = %q, want %q", defaultPayload.Errors[0].Code, ErrorCodeUnauthorized)
	}
}

func runLocalizedHTTPRequest(t *testing.T, client *http.Client, url, acceptLanguage string) (Response[struct{}], string) {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if acceptLanguage != "" {
		req.Header.Set("Accept-Language", acceptLanguage)
	}
	res, err := client.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	defer res.Body.Close()

	var payload Response[struct{}]
	if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	return payload, res.Header.Get("Content-Language")
}
