package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestOKWritesTypedEnvelope(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/posts/1", nil)
	rec := httptest.NewRecorder()
	ctx := e.NewContext(req, rec)

	if err := OK(ctx, map[string]any{"id": 1, "title": "Hello"}); err != nil {
		t.Fatalf("OK() error = %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var payload Response[map[string]any]
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload.Data["title"] != "Hello" {
		t.Fatalf("title = %v, want Hello", payload.Data["title"])
	}
}

func TestFailWritesErrorsEnvelope(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/posts", nil)
	rec := httptest.NewRecorder()
	ctx := e.NewContext(req, rec)

	if err := Fail(ctx, http.StatusUnprocessableEntity, Validation("title", "is required")); err != nil {
		t.Fatalf("Fail() error = %v", err)
	}
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnprocessableEntity)
	}

	var payload Response[struct{}]
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if len(payload.Errors) != 1 || payload.Errors[0].Code != "validation_error" {
		t.Fatalf("errors = %+v", payload.Errors)
	}
}
