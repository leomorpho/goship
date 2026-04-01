package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
)

type stubI18n struct {
	message string
}

func (s stubI18n) DefaultLanguage() string                       { return "en" }
func (s stubI18n) SupportedLanguages() []string                  { return []string{"en"} }
func (s stubI18n) NormalizeLanguage(raw string) string           { return raw }
func (s stubI18n) T(context.Context, string, ...map[string]any) string {
	return s.message
}
func (s stubI18n) TC(context.Context, string, any, ...map[string]any) string { return s.message }
func (s stubI18n) TS(context.Context, string, string, ...map[string]any) string {
	return s.message
}

func TestOKWrapsData(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/status", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := OK(c, map[string]string{"status": "ok"}); err != nil {
		t.Fatalf("OK() error = %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if !strings.Contains(rec.Body.String(), `"data"`) || !strings.Contains(rec.Body.String(), `"status":"ok"`) {
		t.Fatalf("body = %s", rec.Body.String())
	}
}

func TestFailWrapsErrors(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/status", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := Fail(c, http.StatusUnauthorized, Unauthorized("Unauthorized")); err != nil {
		t.Fatalf("Fail() error = %v", err)
	}
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if _, ok := body["errors"]; !ok {
		t.Fatalf("body missing errors: %v", body)
	}
}

func TestIsAPIRequestDetectsPathOrAccept(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/status", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/v1/status")
	if !IsAPIRequest(c) {
		t.Fatal("expected path-based API request")
	}

	req = httptest.NewRequest(http.MethodGet, "/posts/1", nil)
	req.Header.Set(echo.HeaderAccept, "application/json")
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)
	c.SetPath("/posts/1")
	if !IsAPIRequest(c) {
		t.Fatal("expected accept-based API request")
	}
}

func TestLocalizedHelpersUseFallbackOrI18n(t *testing.T) {
	err := UnauthorizedLocalized(context.Background(), stubI18n{message: "Translated"}, "api.errors.unauthorized", "Unauthorized")
	if err.Message != "Translated" || err.Code != "unauthorized" {
		t.Fatalf("got %+v", err)
	}

	err = NotFoundLocalized(context.Background(), nil, "missing", "Fallback")
	if err.Message != "Fallback" || err.Code != "not_found" {
		t.Fatalf("got %+v", err)
	}
}
