package jobs

import (
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestNew(t *testing.T) {
	t.Parallel()

	mod, err := New(Config{
		Backend: BackendRedis,
		Redis: RedisConfig{
			Addr: "localhost:6379",
		},
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if mod == nil {
		t.Fatalf("expected module, got nil")
	}
	if got := mod.Backend(); got != BackendRedis {
		t.Fatalf("expected backend %q, got %q", BackendRedis, got)
	}
	if mod.Jobs() == nil {
		t.Fatalf("expected core jobs implementation, got nil")
	}
	if mod.Inspector() == nil {
		t.Fatalf("expected jobs inspector, got nil")
	}
}

func TestNewSQLProvidesInspector(t *testing.T) {
	t.Parallel()

	db, err := sql.Open("sqlite3", "file:jobs_module_sql?mode=memory&_fk=1")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	mod, err := New(Config{
		Backend: BackendSQL,
		SQLDB:   db,
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if mod.Inspector() == nil {
		t.Fatal("expected SQL inspector, got nil")
	}
}

func TestNewSQLProvidesInspectorWithSQLDB(t *testing.T) {
	t.Parallel()

	db, err := sql.Open("sqlite3", "file:jobs_module_sql_db?mode=memory&_fk=1")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	mod, err := New(Config{
		Backend: BackendSQL,
		SQLDB:   db,
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if mod.Inspector() == nil {
		t.Fatal("expected SQL inspector, got nil")
	}
}
