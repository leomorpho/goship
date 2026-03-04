package jobs

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/leomorpho/goship/db/ent/enttest"
	"github.com/leomorpho/goship/framework/core"
	_ "github.com/mattn/go-sqlite3"
)

func TestSQLJobsEnqueue(t *testing.T) {
	t.Parallel()

	client := enttest.Open(t, "sqlite3", "file:jobsmod?mode=memory&_fk=1")
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

	id, err := mod.Jobs().Enqueue(context.Background(), "job.test", []byte(`{"k":"v"}`), core.EnqueueOptions{
		Queue:      "default",
		RunAt:      time.Now().UTC().Add(2 * time.Second),
		MaxRetries: 3,
	})
	if err != nil {
		t.Fatalf("expected enqueue success, got %v", err)
	}
	if id == "" {
		t.Fatal("expected non-empty job id")
	}

	rows, err := client.QueryContext(context.Background(), "SELECT count(*) FROM goship_jobs WHERE id = ?", id)
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}
	defer rows.Close()

	var count int
	if !rows.Next() {
		t.Fatal("expected one row from count query")
	}
	if err := rows.Scan(&count); err != nil {
		t.Fatalf("scan failed: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 inserted job row, got %d", count)
	}
}

func TestSQLJobsNewRequiresEnt(t *testing.T) {
	t.Parallel()

	_, err := New(Config{Backend: BackendSQL})
	if err == nil {
		t.Fatal("expected error for sql backend without Ent client")
	}
}

func TestSQLJobsWorkerMarksDone(t *testing.T) {
	t.Parallel()

	client := enttest.Open(t, "sqlite3", "file:jobsmod_worker_done?mode=memory&_fk=1")
	t.Cleanup(func() { _ = client.Close() })
	if err := client.Schema.Create(context.Background()); err != nil {
		t.Fatalf("failed to create schema: %v", err)
	}
	mod, err := New(Config{Backend: BackendSQL, EntClient: client})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	jobs := mod.Jobs()
	processed := make(chan struct{}, 1)
	if err := jobs.Register("job.done", func(context.Context, []byte) error {
		processed <- struct{}{}
		return nil
	}); err != nil {
		t.Fatalf("register failed: %v", err)
	}
	if _, err := jobs.Enqueue(context.Background(), "job.done", []byte(`{"v":1}`), core.EnqueueOptions{}); err != nil {
		t.Fatalf("enqueue failed: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- jobs.StartWorker(ctx) }()

	select {
	case <-processed:
		cancel()
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for worker to process job")
	}
	if err := <-done; err != nil {
		t.Fatalf("worker returned unexpected error: %v", err)
	}

	rows, err := client.QueryContext(context.Background(), "SELECT status FROM goship_jobs WHERE name = ?", "job.done")
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}
	defer rows.Close()
	if !rows.Next() {
		t.Fatal("expected one row")
	}
	var status string
	if err := rows.Scan(&status); err != nil {
		t.Fatalf("scan failed: %v", err)
	}
	if status != "done" {
		t.Fatalf("expected status done, got %s", status)
	}
}

func TestSQLJobsWorkerRetriesThenFails(t *testing.T) {
	t.Parallel()

	client := enttest.Open(t, "sqlite3", "file:jobsmod_worker_retry?mode=memory&_fk=1")
	t.Cleanup(func() { _ = client.Close() })
	if err := client.Schema.Create(context.Background()); err != nil {
		t.Fatalf("failed to create schema: %v", err)
	}
	mod, err := New(Config{Backend: BackendSQL, EntClient: client})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	jobs := mod.Jobs()

	attempts := 0
	processed := make(chan struct{}, 2)
	if err := jobs.Register("job.retry", func(context.Context, []byte) error {
		attempts++
		processed <- struct{}{}
		return errors.New("boom")
	}); err != nil {
		t.Fatalf("register failed: %v", err)
	}
	if _, err := jobs.Enqueue(context.Background(), "job.retry", []byte(`{"v":1}`), core.EnqueueOptions{
		MaxRetries: 1,
	}); err != nil {
		t.Fatalf("enqueue failed: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- jobs.StartWorker(ctx) }()

	for i := 0; i < 2; i++ {
		select {
		case <-processed:
		case <-time.After(5 * time.Second):
			t.Fatal("timed out waiting for worker retry cycle")
		}
	}
	cancel()
	if err := <-done; err != nil {
		t.Fatalf("worker returned unexpected error: %v", err)
	}
	if attempts != 2 {
		t.Fatalf("expected 2 attempts, got %d", attempts)
	}

	rows, err := client.QueryContext(context.Background(), "SELECT status, attempt FROM goship_jobs WHERE name = ?", "job.retry")
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}
	defer rows.Close()
	if !rows.Next() {
		t.Fatal("expected one row")
	}
	var status string
	var attempt int
	if err := rows.Scan(&status, &attempt); err != nil {
		t.Fatalf("scan failed: %v", err)
	}
	if status != "failed" {
		t.Fatalf("expected status failed, got %s", status)
	}
	if attempt != 2 {
		t.Fatalf("expected attempt 2, got %d", attempt)
	}
}
