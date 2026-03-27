package bootstrap

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/leomorpho/goship/config"
	"github.com/leomorpho/goship/framework/core/adapters"
	"github.com/leomorpho/goship/framework/health"
)

type testChecker struct {
	name   string
	result health.CheckResult
}

func (t testChecker) Name() string {
	return t.name
}

func (t testChecker) Check(context.Context) health.CheckResult {
	return t.result
}

func TestOpenEmbeddedDBConfiguresSQLitePragmas(t *testing.T) {
	conn := filepath.Join(t.TempDir(), "db", "app.db") + "?_journal=WAL&_timeout=5000&_fk=true"
	db, err := OpenEmbeddedDB("sqlite", conn)
	if err != nil {
		t.Fatalf("open embedded sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	stats := db.Stats()
	if stats.MaxOpenConnections != 1 {
		t.Fatalf("max open conns = %d, want 1", stats.MaxOpenConnections)
	}

	var journalMode string
	if err := db.QueryRow("PRAGMA journal_mode").Scan(&journalMode); err != nil {
		t.Fatalf("query journal_mode: %v", err)
	}
	if strings.ToLower(journalMode) != "wal" {
		t.Fatalf("journal_mode = %q, want wal", journalMode)
	}

	var synchronous int
	if err := db.QueryRow("PRAGMA synchronous").Scan(&synchronous); err != nil {
		t.Fatalf("query synchronous: %v", err)
	}
	if synchronous != 1 {
		t.Fatalf("synchronous = %d, want 1 (NORMAL)", synchronous)
	}

	var busyTimeout int
	if err := db.QueryRow("PRAGMA busy_timeout").Scan(&busyTimeout); err != nil {
		t.Fatalf("query busy_timeout: %v", err)
	}
	if busyTimeout != 5000 {
		t.Fatalf("busy_timeout = %d, want 5000", busyTimeout)
	}

	var foreignKeys int
	if err := db.QueryRow("PRAGMA foreign_keys").Scan(&foreignKeys); err != nil {
		t.Fatalf("query foreign_keys: %v", err)
	}
	if foreignKeys != 1 {
		t.Fatalf("foreign_keys = %d, want 1 (ON)", foreignKeys)
	}

	var cacheSize int
	if err := db.QueryRow("PRAGMA cache_size").Scan(&cacheSize); err != nil {
		t.Fatalf("query cache_size: %v", err)
	}
	if cacheSize != -64000 {
		t.Fatalf("cache_size = %d, want -64000", cacheSize)
	}

	var tempStore int
	if err := db.QueryRow("PRAGMA temp_store").Scan(&tempStore); err != nil {
		t.Fatalf("query temp_store: %v", err)
	}
	if tempStore != 2 {
		t.Fatalf("temp_store = %d, want 2 (MEMORY)", tempStore)
	}
}

func TestResetEmbeddedTestDB_RemovesFileBackedState(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "db", "test.db")
	conn := dbPath + "?_journal=WAL&_timeout=5000&_fk=true"
	db, err := OpenEmbeddedDB("sqlite", conn)
	if err != nil {
		t.Fatalf("open embedded sqlite: %v", err)
	}
	if _, err := db.Exec(`CREATE TABLE stateful (id INTEGER PRIMARY KEY, value TEXT)`); err != nil {
		t.Fatalf("create table: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO stateful(id, value) VALUES (1, 'before-reset')`); err != nil {
		t.Fatalf("insert row: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close db: %v", err)
	}

	if err := resetEmbeddedTestDB(conn); err != nil {
		t.Fatalf("resetEmbeddedTestDB error: %v", err)
	}

	if _, err := os.Stat(dbPath); !os.IsNotExist(err) {
		t.Fatalf("expected sqlite file to be removed, stat err=%v", err)
	}
}

func TestResetEmbeddedTestDB_NoOpForMemoryConnection(t *testing.T) {
	if err := resetEmbeddedTestDB(":memory:?_journal=WAL"); err != nil {
		t.Fatalf("reset memory sqlite should be no-op: %v", err)
	}
	if err := resetEmbeddedTestDB("file::memory:?cache=shared"); err != nil {
		t.Fatalf("reset file::memory sqlite should be no-op: %v", err)
	}
}

func TestFileBackedEmbeddedDBLifecycleIsDeterministic(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "deterministic", "test.db")
	conn := dbPath + "?_journal=WAL&_timeout=5000&_fk=true"

	openAndSeed := func() {
		if err := resetEmbeddedTestDB(conn); err != nil {
			t.Fatalf("resetEmbeddedTestDB error: %v", err)
		}
		db, err := OpenEmbeddedDB("sqlite", conn)
		if err != nil {
			t.Fatalf("open embedded sqlite: %v", err)
		}
		t.Cleanup(func() { _ = db.Close() })
		if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS deterministic_state (id INTEGER PRIMARY KEY, value TEXT)`); err != nil {
			t.Fatalf("create table: %v", err)
		}
		if _, err := db.Exec(`INSERT INTO deterministic_state(id, value) VALUES (1, 'seeded')`); err != nil {
			t.Fatalf("insert row: %v", err)
		}
		if err := db.Close(); err != nil {
			t.Fatalf("close db: %v", err)
		}
	}

	openAndSeed()
	openAndSeed()

	verifyDB, err := sql.Open("sqlite", conn)
	if err != nil {
		t.Fatalf("open sqlite verify: %v", err)
	}
	defer verifyDB.Close()
	var count int
	if err := verifyDB.QueryRow(`SELECT COUNT(*) FROM deterministic_state`).Scan(&count); err != nil {
		t.Fatalf("count deterministic_state: %v", err)
	}
	if count != 1 {
		t.Fatalf("deterministic_state count=%d, want 1", count)
	}
}

func TestOpenEmbeddedDBConcurrentWritesAvoidLockErrors(t *testing.T) {
	conn := filepath.Join(t.TempDir(), "db", "writes.db") + "?_journal=WAL&_timeout=5000&_fk=true"
	db, err := OpenEmbeddedDB("sqlite", conn)
	if err != nil {
		t.Fatalf("open embedded sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if _, err := db.Exec(`CREATE TABLE writes (id INTEGER PRIMARY KEY AUTOINCREMENT, value TEXT NOT NULL)`); err != nil {
		t.Fatalf("create table: %v", err)
	}

	const workers = 50
	errCh := make(chan error, workers)
	var wg sync.WaitGroup
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func(i int) {
			defer wg.Done()
			_, execErr := db.Exec(`INSERT INTO writes(value) VALUES (?)`, fmt.Sprintf("v-%d", i))
			if execErr != nil {
				errCh <- execErr
			}
		}(i)
	}
	wg.Wait()
	close(errCh)

	for err := range errCh {
		if strings.Contains(strings.ToLower(err.Error()), "database is locked") {
			t.Fatalf("database lock detected: %v", err)
		}
		t.Fatalf("unexpected insert error: %v", err)
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM writes`).Scan(&count); err != nil {
		t.Fatalf("count rows: %v", err)
	}
	if count != workers {
		t.Fatalf("writes count = %d, want %d", count, workers)
	}
}

func TestNewContainerEmbeddedBootHasHealthyReadiness(t *testing.T) {
	t.Setenv("PAGODA_APP_ENVIRONMENT", "test")
	t.Setenv("PAGODA_DB_PATH", filepath.Join(t.TempDir(), "app.db"))

	container := NewContainer(nil)
	t.Cleanup(func() {
		_ = container.Shutdown()
	})

	if container.Health == nil {
		t.Fatal("expected health registry to be initialized")
	}

	results, allOK := container.Health.Run(context.Background())
	if !allOK {
		t.Fatalf("expected healthy readiness, got %#v", results)
	}
	if results["db"].Status != health.StatusOK {
		t.Fatalf("expected db status ok, got %q", results["db"].Status)
	}
}

func TestContainerValidateStartupContractPanicsWhenHealthChecksMissing(t *testing.T) {
	container := &Container{
		Health: health.NewRegistry(),
	}

	var panicValue any
	defer func() {
		panicValue = recover()
		if panicValue == nil {
			t.Fatal("expected panic when startup health contract is invalid")
		}
		message := panicValue.(string)
		if !strings.Contains(message, "health startup contract") {
			t.Fatalf("panic = %q, want startup contract summary", message)
		}
		if !strings.Contains(message, "missing=[db cache jobs env]") {
			t.Fatalf("panic = %q, want missing checks summary", message)
		}
	}()

	container.validateStartupContract()
}

func TestContainerValidateStartupContractPanicsWhenRuntimeEnvIsMissing(t *testing.T) {
	container := &Container{
		Health: health.NewRegistry(
			testChecker{name: "db", result: health.CheckResult{Status: health.StatusOK}},
			testChecker{name: "cache", result: health.CheckResult{Status: health.StatusOK}},
			testChecker{name: "jobs", result: health.CheckResult{Status: health.StatusOK}},
			health.NewEnvChecker(
				health.EnvRequirement{Name: "PAGODA_APP_ENVIRONMENT", Value: "test"},
				health.EnvRequirement{Name: "PAGODA_ADAPTERS_DB", Value: ""},
			),
		),
	}

	var panicValue any
	defer func() {
		panicValue = recover()
		if panicValue == nil {
			t.Fatal("expected panic when required runtime env values are missing")
		}
		message := panicValue.(string)
		if !strings.Contains(message, "required runtime environment variables are missing") {
			t.Fatalf("panic = %q, want missing runtime env error", message)
		}
		if !strings.Contains(message, "PAGODA_ADAPTERS_DB") {
			t.Fatalf("panic = %q, want missing env variable name", message)
		}
	}()

	container.validateStartupContract()
}

func TestContainerValidateStartupContractPanicsForStandaloneDBMissingRuntimeFields(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		hostname    string
		port        uint16
		wantMissing string
	}{
		{
			name:        "missing hostname",
			hostname:    "",
			port:        5432,
			wantMissing: "PAGODA_DATABASE_HOSTNAME",
		},
		{
			name:        "missing port",
			hostname:    "db.internal",
			port:        0,
			wantMissing: "PAGODA_DATABASE_PORT",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			container := &Container{
				Config: &config.Config{
					App: config.AppConfig{
						Environment: config.EnvDevelop,
					},
					Database: config.DatabaseConfig{
						DbMode:   config.DBModeStandalone,
						Hostname: tc.hostname,
						Port:     tc.port,
					},
				},
				Adapters: adapters.Resolved{
					Selection: adapters.Selection{
						DB:     "postgres",
						Cache:  "otter",
						Jobs:   "backlite",
						PubSub: "inproc",
					},
				},
			}
			container.initHealth()

			var panicValue any
			defer func() {
				panicValue = recover()
				if panicValue == nil {
					t.Fatal("expected panic when standalone runtime db fields are missing")
				}
				message := panicValue.(string)
				if !strings.Contains(message, "required runtime environment variables are missing") {
					t.Fatalf("panic = %q, want missing runtime env error", message)
				}
				if !strings.Contains(message, tc.wantMissing) {
					t.Fatalf("panic = %q, want missing env variable name %q", message, tc.wantMissing)
				}
			}()

			container.validateStartupContract()
		})
	}
}
