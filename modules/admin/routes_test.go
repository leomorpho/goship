package admin

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/leomorpho/goship/config"
	frameworkbootstrap "github.com/leomorpho/goship/framework/bootstrap"
	ctxkeys "github.com/leomorpho/goship/framework/context"
	"github.com/leomorpho/goship/framework/core"
	"github.com/leomorpho/goship/framework/runtimeconfig"
	"github.com/leomorpho/goship/framework/web/ui"
	"github.com/leomorpho/goship/modules/flags"
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

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
	if !strings.Contains(rec.Body.String(), "Queue monitor unavailable") {
		t.Fatalf("body = %q, want unavailable banner", rec.Body.String())
	}
}

func TestAdminRoutes_AdminQueueMonitorUsesCoreJobsInspector(t *testing.T) {
	c := newContainerForAdminRoutes(t, true)
	c.CoreJobsInspector = fakeJobsInspector{
		records: []core.JobRecord{
			{
				ID:         "job-1",
				Name:       "emails.send",
				Queue:      "default",
				Status:     core.JobStatusQueued,
				Attempt:    1,
				MaxRetries: 5,
				Payload:    []byte(`{"user_id":1}`),
				CreatedAt:  time.Unix(10, 0).UTC(),
				UpdatedAt:  time.Unix(20, 0).UTC(),
				RunAt:      time.Unix(30, 0).UTC(),
			},
		},
	}

	listReq := httptest.NewRequest(http.MethodGet, "/admin/queues", nil)
	listRec := httptest.NewRecorder()
	c.Web.ServeHTTP(listRec, listReq)

	if listRec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", listRec.Code, http.StatusOK)
	}
	if !strings.Contains(listRec.Body.String(), "emails.send") || !strings.Contains(listRec.Body.String(), "/admin/queues/job-1") {
		t.Fatalf("body = %q, want job list output", listRec.Body.String())
	}

	detailReq := httptest.NewRequest(http.MethodGet, "/admin/queues/job-1", nil)
	detailRec := httptest.NewRecorder()
	c.Web.ServeHTTP(detailRec, detailReq)

	if detailRec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", detailRec.Code, http.StatusOK)
	}
	body := detailRec.Body.String()
	if !strings.Contains(body, "emails.send") || !strings.Contains(body, "user_id") {
		t.Fatalf("body = %q, want job detail payload", body)
	}
}

func TestAdminRoutes_AdminQueueMonitorRedisCapabilityGap(t *testing.T) {
	c := newContainerForAdminRoutes(t, true)
	c.CoreJobsInspector = fakeJobsInspector{err: errors.New("redis jobs inspector is not implemented yet")}

	req := httptest.NewRequest(http.MethodGet, "/admin/queues", nil)
	rec := httptest.NewRecorder()
	c.Web.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
	if !strings.Contains(rec.Body.String(), "redis jobs inspector is not implemented yet") {
		t.Fatalf("body = %q, want capability-gap message", rec.Body.String())
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

func TestAdminRoutes_AdminManagedSettingsShowsSourceAndReadOnlyStates(t *testing.T) {
	c := newContainerForAdminRoutes(t, true)
	c.Config.Managed.Enabled = true
	c.Config.Managed.Authority = "control-plane"
	c.Config.Adapters.Cache = "managed-cache"
	c.Config.Managed.RuntimeReport = runtimeconfig.Report{
		Mode:      runtimeconfig.ModeManaged,
		Authority: "control-plane",
		Keys: map[string]runtimeconfig.KeyState{
			"adapters.cache": {
				Value:  "managed-cache",
				Source: runtimeconfig.SourceManagedOverride,
			},
			"adapters.jobs": {
				Value:  "backlite",
				Source: runtimeconfig.SourceEnvironment,
			},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/admin/managed-settings", nil)
	rec := httptest.NewRecorder()
	c.Web.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "Managed mode: enabled (authority: control-plane)") {
		t.Fatalf("body = %q, want managed authority banner", body)
	}
	if !strings.Contains(body, "Cache adapter") || !strings.Contains(body, "managed-cache") || !strings.Contains(body, "externally-managed") {
		t.Fatalf("body = %q, want externally managed cache row", body)
	}
	if !strings.Contains(body, "Jobs adapter") || !strings.Contains(body, "backlite") || !strings.Contains(body, "read-only") || !strings.Contains(body, "environment") {
		t.Fatalf("body = %q, want read-only jobs row with environment source", body)
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

func TestFlagsListHandler_ShowsRegisteredConstantKey(t *testing.T) {
	c := newContainerForAdminRoutes(t, true)
	seedFeatureFlagForAdminTest(t, c.Database, "admin_flags_constant_key", false, 100, "manual description")
	ensureFlagDefinition(t, "admin_flags_constant_key", "registry description", true)

	req := httptest.NewRequest(http.MethodGet, "/admin/flags", nil)
	rec := httptest.NewRecorder()
	c.Web.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if !strings.Contains(rec.Body.String(), "<code>admin_flags_constant_key</code>") {
		t.Fatalf("body = %q, want constant key code badge", rec.Body.String())
	}
}

func TestFlagsListHandler_ShowsCodeDefault(t *testing.T) {
	c := newContainerForAdminRoutes(t, true)
	seedFeatureFlagForAdminTest(t, c.Database, "admin_flags_default_on", false, 100, "manual description")
	ensureFlagDefinition(t, "admin_flags_default_on", "registry description", true)

	req := httptest.NewRequest(http.MethodGet, "/admin/flags", nil)
	rec := httptest.NewRecorder()
	c.Web.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if !strings.Contains(rec.Body.String(), "Code default: On") {
		t.Fatalf("body = %q, want code default badge", rec.Body.String())
	}
}

func TestFlagsListHandler_MergesRegistryAndDB(t *testing.T) {
	c := newContainerForAdminRoutes(t, true)
	seedFeatureFlagForAdminTest(t, c.Database, "admin_flags_merge", false, 100, "manual description")
	ensureFlagDefinition(t, "admin_flags_merge", "registry description", false)

	req := httptest.NewRequest(http.MethodGet, "/admin/flags", nil)
	rec := httptest.NewRecorder()
	c.Web.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "admin_flags_merge") {
		t.Fatalf("body = %q, want db key row", body)
	}
	if !strings.Contains(body, "<td>off</td>") {
		t.Fatalf("body = %q, want db toggle state", body)
	}
	if !strings.Contains(body, "registry description") {
		t.Fatalf("body = %q, want registry description", body)
	}
}

func seedFeatureFlagForAdminTest(t *testing.T, db *sql.DB, key string, enabled bool, rolloutPct int, description string) {
	t.Helper()
	if _, err := db.Exec(`
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
`); err != nil {
		t.Fatalf("create feature_flags: %v", err)
	}

	if _, err := db.Exec(
		`INSERT INTO feature_flags (key, enabled, rollout_pct, description) VALUES (?, ?, ?, ?)`,
		key,
		boolToInt(enabled),
		rolloutPct,
		description,
	); err != nil {
		t.Fatalf("seed feature_flags: %v", err)
	}
}

func ensureFlagDefinition(t *testing.T, key, description string, defaultEnabled bool) {
	t.Helper()
	flagKey := flags.FlagKey(key)
	if _, ok := flags.Lookup(flagKey); ok {
		return
	}
	flags.Register(flags.FlagDefinition{
		Key:         flagKey,
		Description: description,
		Default:     defaultEnabled,
	})
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func newContainerForAdminRoutes(t *testing.T, admin bool) *frameworkbootstrap.Container {
	t.Helper()
	if err := chdirRepoRoot(); err != nil {
		t.Fatalf("chdir repo root: %v", err)
	}
	config.SwitchEnvironment(config.EnvTest)
	c := frameworkbootstrap.NewContainer(nil)
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

type fakeJobsInspector struct {
	records []core.JobRecord
	err     error
}

func (f fakeJobsInspector) List(context.Context, core.JobListFilter) ([]core.JobRecord, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.records, nil
}

func (f fakeJobsInspector) Get(context.Context, string) (core.JobRecord, bool, error) {
	if f.err != nil {
		return core.JobRecord{}, false, f.err
	}
	if len(f.records) == 0 {
		return core.JobRecord{}, false, nil
	}
	return f.records[0], true, nil
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
