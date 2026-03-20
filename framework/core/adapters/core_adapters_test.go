package adapters

import (
	"context"
	"testing"

	"github.com/leomorpho/goship/config"
	"github.com/leomorpho/goship/framework/core"
)

func TestCoreCacheAdapterNilClient(t *testing.T) {
	t.Parallel()

	adapter := NewCoreCacheAdapter(nil)
	_, _, err := adapter.Get(context.Background(), "k")
	if err == nil {
		t.Fatal("expected error for uninitialized cache client")
	}
	if err := adapter.Set(context.Background(), "k", []byte("v"), 0); err == nil {
		t.Fatal("expected error for uninitialized cache client")
	}
	if err := adapter.Delete(context.Background(), "k"); err == nil {
		t.Fatal("expected error for uninitialized cache client")
	}
	if err := adapter.InvalidatePrefix(context.Background(), "prefix"); err == nil {
		t.Fatal("expected error for uninitialized cache client")
	}
	if err := adapter.Close(); err != nil {
		t.Fatalf("expected nil error on close, got %v", err)
	}
}

func TestCoreJobsAdapterBasics(t *testing.T) {
	t.Parallel()

	caps := core.JobCapabilities{Delayed: true, Retries: true}
	adapter := NewCoreJobsAdapter(nil, caps)

	if got := adapter.Capabilities(); got != caps {
		t.Fatalf("capabilities mismatch: got=%+v want=%+v", got, caps)
	}

	if err := adapter.Register("", func(context.Context, []byte) error { return nil }); err == nil {
		t.Fatal("expected validation error for empty job name")
	}
	if err := adapter.Register("job", nil); err == nil {
		t.Fatal("expected validation error for nil handler")
	}
	if err := adapter.Register("job", func(context.Context, []byte) error { return nil }); err != nil {
		t.Fatalf("expected register success, got %v", err)
	}

	if _, err := adapter.Enqueue(context.Background(), "job", nil, core.EnqueueOptions{}); err == nil {
		t.Fatal("expected enqueue error for uninitialized client")
	}
	if err := adapter.StartScheduler(context.Background()); err == nil {
		t.Fatal("expected scheduler error for uninitialized client")
	}
	if err := adapter.StartWorker(context.Background()); err != nil {
		t.Fatalf("expected no-op worker start, got %v", err)
	}
	if err := adapter.Stop(context.Background()); err != nil {
		t.Fatalf("expected nil stop error, got %v", err)
	}
}

func TestNewCoreJobsAdapterFromConfig(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Adapters: config.AdaptersConfig{
			DB:     "postgres",
			Cache:  "redis",
			Jobs:   "asynq",
			PubSub: "redis",
		},
	}

	adapter, err := NewCoreJobsAdapterFromConfig(nil, cfg)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if !adapter.Capabilities().Dashboard {
		t.Fatal("expected asynq capabilities to include dashboard")
	}

	cfg.Adapters.Jobs = "unknown"
	if _, err := NewCoreJobsAdapterFromConfig(nil, cfg); err == nil {
		t.Fatal("expected error for unknown jobs adapter")
	}
}

func TestCoreJobsInspectorAdapterNilDelegate(t *testing.T) {
	t.Parallel()

	adapter := NewCoreJobsInspectorAdapter(nil)
	if _, err := adapter.List(context.Background(), core.JobListFilter{}); err == nil {
		t.Fatal("expected list error for uninitialized inspector")
	}
	if _, _, err := adapter.Get(context.Background(), "missing"); err == nil {
		t.Fatal("expected get error for uninitialized inspector")
	}
}
