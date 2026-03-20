package controllers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/leomorpho/goship/framework/health"
)

func TestHealthcheckLiveness(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	ctx := e.NewContext(req, rec)

	h := healthcheck{startedAt: time.Now(), version: "dev", registry: health.NewRegistry()}
	if err := h.GetLiveness(ctx); err != nil {
		t.Fatalf("GetLiveness() error = %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status code: got %d want %d", rec.Code, http.StatusOK)
	}

	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if payload["status"] != health.StatusOK {
		t.Fatalf("unexpected status payload: %v", payload["status"])
	}
}

func TestHealthcheckReadiness503WhenAnyCheckFails(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	rec := httptest.NewRecorder()
	ctx := e.NewContext(req, rec)

	registry := health.NewRegistry()
	registry.Register(testChecker{name: "db", result: health.CheckResult{Status: health.StatusOK}})
	registry.Register(testChecker{name: "cache", result: health.CheckResult{Status: health.StatusError, Error: "down"}})

	h := healthcheck{
		startedAt: time.Now().Add(-2 * time.Minute),
		version:   "dev",
		registry:  registry,
	}
	if err := h.GetReadiness(ctx); err != nil {
		t.Fatalf("GetReadiness() error = %v", err)
	}
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("unexpected status code: got %d want %d", rec.Code, http.StatusServiceUnavailable)
	}
}

type testChecker struct {
	name   string
	result health.CheckResult
}

func (t testChecker) Name() string {
	return t.name
}

func (t testChecker) Check(_ context.Context) health.CheckResult {
	return t.result
}
