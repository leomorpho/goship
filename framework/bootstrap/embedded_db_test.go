package bootstrap

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/leomorpho/goship/framework/health"
)

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

	defer func() {
		if recover() == nil {
			t.Fatal("expected panic when startup health contract is invalid")
		}
	}()

	container.validateStartupContract()
}
