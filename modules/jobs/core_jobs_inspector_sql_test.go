package jobs

import (
	"context"
	"testing"
	"time"

	"github.com/leomorpho/goship/db/ent/enttest"
	"github.com/leomorpho/goship/framework/core"
	_ "github.com/mattn/go-sqlite3"
)

func TestSQLInspectorListAndGet(t *testing.T) {
	t.Parallel()

	client := enttest.Open(t, "sqlite3", "file:jobs_inspector?mode=memory&_fk=1")
	t.Cleanup(func() { _ = client.Close() })
	if err := client.Schema.Create(context.Background()); err != nil {
		t.Fatalf("failed to create schema: %v", err)
	}
	mod, err := New(Config{Backend: BackendSQL, EntClient: client})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	jobs := mod.Jobs()
	inspector := mod.Inspector()
	if inspector == nil {
		t.Fatal("expected inspector")
	}

	jobID, err := jobs.Enqueue(context.Background(), "job.inspect", []byte(`{"p":1}`), core.EnqueueOptions{
		Queue:      "default",
		RunAt:      time.Now().UTC(),
		MaxRetries: 2,
	})
	if err != nil {
		t.Fatalf("enqueue failed: %v", err)
	}

	list, err := inspector.List(context.Background(), core.JobListFilter{
		Queue:    "default",
		Statuses: []core.JobStatus{core.JobStatusQueued},
		Limit:    10,
	})
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected exactly one job, got %d", len(list))
	}
	if list[0].ID != jobID {
		t.Fatalf("expected listed id %s, got %s", jobID, list[0].ID)
	}

	got, found, err := inspector.Get(context.Background(), jobID)
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if !found {
		t.Fatal("expected job to exist")
	}
	if got.Name != "job.inspect" {
		t.Fatalf("expected name job.inspect, got %s", got.Name)
	}
	if got.Status != core.JobStatusQueued {
		t.Fatalf("expected queued status, got %s", got.Status)
	}
}
