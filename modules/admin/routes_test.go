package admin

import (
	"context"
	"database/sql"
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

func TestAdminRoutes_AdminManagedSettings(t *testing.T) {
	c := newContainerForAdminRoutes(t, true)

	req := httptest.NewRequest(http.MethodGet, "/admin/managed-settings", nil)
	rec := httptest.NewRecorder()
	c.Web.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if !strings.Contains(rec.Body.String(), "Managed Runtime Settings") {
		t.Fatalf("body = %q, want managed settings heading", rec.Body.String())
	}
}

func TestAdminRoutes_AdminTrashView(t *testing.T) {
	c := newContainerForAdminRoutes(t, true)

	_, err := c.Database.Exec(`
CREATE TABLE IF NOT EXISTS test_soft_delete_rows (
	id INTEGER PRIMARY KEY,
	deleted_at DATETIME
);
INSERT INTO test_soft_delete_rows (id, deleted_at) VALUES (1, CURRENT_TIMESTAMP);
`)
	if err != nil {
		t.Fatalf("seed soft delete table: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/admin/trash", nil)
	rec := httptest.NewRecorder()
	c.Web.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Admin - Trash") {
		t.Fatalf("body = %q, want trash heading", body)
	}
	if !strings.Contains(body, "test_soft_delete_rows") {
		t.Fatalf("body = %q, want soft delete table name", body)
	}
}

func TestAdminRoutes_AdminFlagsToggle(t *testing.T) {
	c := newContainerForAdminRoutes(t, true)

	_, err := c.Database.Exec(`
CREATE TABLE IF NOT EXISTS feature_flags (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    key TEXT NOT NULL UNIQUE,
    enabled INTEGER NOT NULL DEFAULT 0,
    rollout_pct INTEGER NOT NULL DEFAULT 0,
    user_ids TEXT,
    description TEXT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
INSERT INTO feature_flags (key, enabled, rollout_pct, description) VALUES ('new_checkout_flow', 0, 100, 'checkout rollout');
`)
	if err != nil {
		t.Fatalf("seed feature_flags: %v", err)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/admin/flags", nil)
	listRec := httptest.NewRecorder()
	c.Web.ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("list status = %d, want %d", listRec.Code, http.StatusOK)
	}
	if !strings.Contains(listRec.Body.String(), "new_checkout_flow") {
		t.Fatalf("list body = %q, want seeded key", listRec.Body.String())
	}

	toggleReq := httptest.NewRequest(http.MethodPost, "/admin/flags/new_checkout_flow/toggle", nil)
	toggleRec := httptest.NewRecorder()
	c.Web.ServeHTTP(toggleRec, toggleReq)
	if toggleRec.Code != http.StatusFound {
		t.Fatalf("toggle status = %d, want %d", toggleRec.Code, http.StatusFound)
	}

	var enabled int
	if err := c.Database.QueryRow(`SELECT enabled FROM feature_flags WHERE key = 'new_checkout_flow'`).Scan(&enabled); err != nil {
		t.Fatalf("select enabled: %v", err)
	}
	if enabled != 1 {
		t.Fatalf("enabled = %d, want 1", enabled)
	}
}

func newContainerForAdminRoutes(t *testing.T, admin bool) *foundation.Container {
	t.Helper()
	if err := chdirRepoRoot(); err != nil {
		t.Fatalf("chdir repo root: %v", err)
	}
	config.SwitchEnvironment(config.EnvTest)
	c := foundation.NewContainer()
	if err := ensureBackliteSchema(c.Database); err != nil {
		t.Fatalf("ensure backlite schema: %v", err)
	}
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
		Flags:      c.Flags,
	})
	if err := module.RegisterRoutes(c.Web); err != nil {
		t.Fatalf("register routes: %v", err)
	}
	return c
}

func ensureBackliteSchema(db *sql.DB) error {
	if db == nil {
		return nil
	}

	schema := `
CREATE TABLE IF NOT EXISTS backlite_tasks (
    id text PRIMARY KEY,
    created_at integer NOT NULL,
    queue text NOT NULL,
    task blob NOT NULL,
    wait_until integer,
    claimed_at integer,
    last_executed_at integer,
    attempts integer NOT NULL DEFAULT 0
) STRICT;

CREATE TABLE IF NOT EXISTS backlite_tasks_completed (
    id text PRIMARY KEY NOT NULL,
    created_at integer NOT NULL,
    queue text NOT NULL,
    last_executed_at integer,
    attempts integer NOT NULL,
    last_duration_micro integer,
    succeeded integer,
    task blob,
    expires_at integer,
    error text
) STRICT;

CREATE INDEX IF NOT EXISTS backlite_tasks_wait_until ON backlite_tasks (wait_until) WHERE wait_until IS NOT NULL;
`
	_, err := db.Exec(schema)
	return err
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
