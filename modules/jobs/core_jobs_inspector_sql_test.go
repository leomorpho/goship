package jobs

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func TestSQLInspectorListAndGet(t *testing.T) {
	t.Parallel()

	client, err := sql.Open("sqlite3", "file:jobs_inspector?mode=memory&_fk=1")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = client.Close() })
	mod, err := New(Config{Backend: BackendSQL, SQLDB: client})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	jobs := mod.Jobs()
	inspector := mod.Inspector()
	if inspector == nil {
		t.Fatal("expected inspector")
	}

	jobID, err := jobs.Enqueue(context.Background(), "job.inspect", []byte(`{"p":1}`), EnqueueOptions{
		Queue:      "default",
		RunAt:      time.Now().UTC(),
		MaxRetries: 2,
	})
	if err != nil {
		t.Fatalf("enqueue failed: %v", err)
	}

	list, err := inspector.List(context.Background(), JobListFilter{
		Queue:    "default",
		Statuses: []JobStatus{JobStatusQueued},
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
	if got.Status != JobStatusQueued {
		t.Fatalf("expected queued status, got %s", got.Status)
	}
}
