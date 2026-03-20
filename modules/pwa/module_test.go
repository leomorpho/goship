package pwa

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
)

func TestModuleRegisterRoutes_RegistersInstallPage(t *testing.T) {
	e := echo.New()
	module := NewModule(&RouteService{})

	if err := module.RegisterRoutes(e); err != nil {
		t.Fatalf("RegisterRoutes() error = %v", err)
	}

	found := false
	for _, route := range e.Routes() {
		if route.Method == http.MethodGet && route.Path == "/install-app" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected GET /install-app route to be registered")
	}
}

func TestModuleRegisterStaticRoutes_ServesManifestAndServiceWorker(t *testing.T) {
	e := echo.New()
	module := NewModule(&RouteService{})

	if err := module.RegisterStaticRoutes(e, time.Hour); err != nil {
		t.Fatalf("RegisterStaticRoutes() error = %v", err)
	}

	testCases := []struct {
		name        string
		path        string
		contentType string
	}{
		{name: "manifest", path: "/files/manifest.json", contentType: "application/manifest+json"},
		{name: "service worker", path: "/service-worker.js", contentType: "application/javascript"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
			}
			if got := rec.Header().Get(echo.HeaderContentType); got != tc.contentType {
				t.Fatalf("content-type = %q, want %q", got, tc.contentType)
			}
		})
	}
}
