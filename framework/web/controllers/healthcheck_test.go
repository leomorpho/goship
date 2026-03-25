package controllers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	frameworkbootstrap "github.com/leomorpho/goship/framework/bootstrap"
	"github.com/leomorpho/goship/framework/health"
	"github.com/leomorpho/goship/framework/web/ui"
)

func TestHealthcheckLiveness(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	ctx := e.NewContext(req, rec)

	h := HealthcheckRoute{startedAt: time.Now(), version: "dev", registry: health.NewRegistry()}
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

	h := HealthcheckRoute{
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

func TestNewHealthCheckRouteUsesContainerHealthRegistry(t *testing.T) {
	registry := health.NewRegistry(
		testChecker{name: "db", result: health.CheckResult{Status: health.StatusOK}},
		testChecker{name: "cache", result: health.CheckResult{Status: health.StatusOK}},
		testChecker{name: "jobs", result: health.CheckResult{Status: health.StatusOK}},
		testChecker{name: "env", result: health.CheckResult{Status: health.StatusOK}},
	)

	route := NewHealthCheckRoute(ui.NewController(&frameworkbootstrap.Container{Health: registry}))
	if route.registry != registry {
		t.Fatal("expected route to use container health registry")
	}
}

func TestNewHealthCheckRoutePanicsWithoutHealthRegistry(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic when health registry is missing")
		}
	}()

	_ = NewHealthCheckRoute(ui.NewController(&frameworkbootstrap.Container{}))
}

func TestNewHealthCheckRoutePanicsWhenHealthContractIsInvalid(t *testing.T) {
	var panicValue any
	defer func() {
		panicValue = recover()
		if panicValue == nil {
			t.Fatal("expected panic when health contract is invalid")
		}
		message := panicValue.(string)
		if !strings.Contains(message, "health startup contract") {
			t.Fatalf("panic = %q, want startup contract summary", message)
		}
		if !strings.Contains(message, "missing=[cache jobs env]") {
			t.Fatalf("panic = %q, want missing checks summary", message)
		}
	}()

	brokenRegistry := health.NewRegistry(testChecker{
		name:   "db",
		result: health.CheckResult{Status: health.StatusOK},
	})
	_ = NewHealthCheckRoute(ui.NewController(&frameworkbootstrap.Container{Health: brokenRegistry}))
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
