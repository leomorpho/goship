package jobs

import (
	"context"
	"testing"

	"github.com/leomorpho/goship/db/ent/enttest"
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

	client := enttest.Open(t, "sqlite3", "file:jobs_module_sql?mode=memory&_fk=1")
	t.Cleanup(func() { _ = client.Close() })
	if err := client.Schema.Create(context.Background()); err != nil {
		t.Fatalf("failed to create schema: %v", err)
	}
	mod, err := New(Config{
		Backend:   BackendSQL,
		EntClient: client,
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if mod.Inspector() == nil {
		t.Fatal("expected SQL inspector, got nil")
	}
}
