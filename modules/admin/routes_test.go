package admin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/leomorpho/goship/app/foundation"
	"github.com/leomorpho/goship/app/web/ui"
	"github.com/leomorpho/goship/config"
	ctxkeys "github.com/leomorpho/goship/framework/context"
)

func TestAdminRoutes_NonAdminForbidden(t *testing.T) {
	c := newContainerForAdminRoutes(t, false)

	req := httptest.NewRequest(http.MethodGet, "/admin/users", nil)
	rec := httptest.NewRecorder()
	c.Web.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

func TestAdminRoutes_AdminCanList(t *testing.T) {
	c := newContainerForAdminRoutes(t, true)

	req := httptest.NewRequest(http.MethodGet, "/admin/users", nil)
	rec := httptest.NewRecorder()
	c.Web.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestAdminRoutes_AdminQueueMonitor(t *testing.T) {
	c := newContainerForAdminRoutes(t, true)

	req := httptest.NewRequest(http.MethodGet, "/admin/queues", nil)
	rec := httptest.NewRecorder()
	c.Web.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestAdminRoutes_AdminAuditLogs(t *testing.T) {
	c := newContainerForAdminRoutes(t, true)

	if err := c.AuditLogs.Record(context.Background(), "user.login", "user", "1", nil); err != nil {
		t.Fatalf("Record() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/admin/audit-logs?action=user.login", nil)
	rec := httptest.NewRecorder()
	c.Web.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if !strings.Contains(rec.Body.String(), "user.login") {
		t.Fatalf("body = %q, want audit action", rec.Body.String())
	}
}

func newContainerForAdminRoutes(t *testing.T, admin bool) *foundation.Container {
	t.Helper()
	if err := chdirRepoRoot(); err != nil {
		t.Fatalf("chdir repo root: %v", err)
	}
	config.SwitchEnvironment(config.EnvTest)
	c := foundation.NewContainer()
	t.Cleanup(func() { _ = c.Shutdown() })

	c.Web.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(ctx echo.Context) error {
			ctx.Set(ctxkeys.AuthenticatedUserIDKey, 1)
			ctx.Set(ctxkeys.AuthenticatedUserIsAdminKey, admin)
			ctx.Set(ctxkeys.ProfileFullyOnboarded, true)
			return next(ctx)
		}
	})

	module := New(ModuleDeps{
		Controller: ui.NewController(c),
		DB:         c.Database,
		AuditLogs:  c.AuditLogs,
	})
	if err := module.RegisterRoutes(c.Web); err != nil {
		t.Fatalf("register routes: %v", err)
	}
	return c
}

func chdirRepoRoot() error {
	dir, err := os.Getwd()
	if err != nil {
		return err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return os.Chdir(dir)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return os.ErrNotExist
		}
		dir = parent
	}
}
