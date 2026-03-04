package jobs

import (
	"context"
	"testing"

	"github.com/leomorpho/goship/framework/core"
)

func TestRedisInspectorNotImplemented(t *testing.T) {
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
	inspector := mod.Inspector()
	if inspector == nil {
		t.Fatal("expected inspector")
	}
	if _, err := inspector.List(context.Background(), core.JobListFilter{}); err == nil {
		t.Fatal("expected redis inspector list to return not implemented error")
	}
	if _, _, err := inspector.Get(context.Background(), "id"); err == nil {
		t.Fatal("expected redis inspector get to return not implemented error")
	}
}
