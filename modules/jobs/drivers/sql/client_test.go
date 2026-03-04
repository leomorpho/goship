package sql

import (
	"context"
	"testing"
	"time"

	"github.com/leomorpho/goship/db/ent/enttest"
	_ "github.com/mattn/go-sqlite3"
)

func TestNewRequiresEntClient(t *testing.T) {
	t.Parallel()

	_, err := New(Config{})
	if err == nil {
		t.Fatal("expected error for nil Ent client")
	}
}

func TestClaimNextAndMarkDone(t *testing.T) {
	t.Parallel()

	entClient := enttest.Open(t, "sqlite3", "file:jobsmod_driver?mode=memory&_fk=1")
	t.Cleanup(func() { _ = entClient.Close() })
	if err := entClient.Schema.Create(context.Background()); err != nil {
		t.Fatalf("failed to create schema: %v", err)
	}
	client, err := New(Config{EntClient: entClient})
	if err != nil {
		t.Fatalf("new client failed: %v", err)
	}

	if err := client.Enqueue(context.Background(), "j1", "default", "job.test", `{"ok":true}`, time.Now().UTC(), 1); err != nil {
		t.Fatalf("enqueue failed: %v", err)
	}
	job, found, err := client.ClaimNext(context.Background(), "worker-1", time.Now().UTC().Add(30*time.Second))
	if err != nil {
		t.Fatalf("claim failed: %v", err)
	}
	if !found {
		t.Fatal("expected claimed job")
	}
	if job.ID != "j1" {
		t.Fatalf("expected claimed id j1, got %s", job.ID)
	}
	if err := client.MarkDone(context.Background(), job.ID); err != nil {
		t.Fatalf("mark done failed: %v", err)
	}
}
