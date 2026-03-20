package jobs

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func TestSQLJobsEnqueue(t *testing.T) {
	t.Parallel()

	client := openSQLJobsTestDB(t, "jobsmod")

	mod, err := New(Config{Backend: BackendSQL, SQLDB: client})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	id, err := mod.Jobs().Enqueue(context.Background(), "job.test", []byte(`{"k":"v"}`), EnqueueOptions{
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

func TestSQLJobsNewRequiresSQLDB(t *testing.T) {
	t.Parallel()

	_, err := New(Config{Backend: BackendSQL})
	if err == nil {
		t.Fatal("expected error for sql backend without SQL DB")
	}
}

func TestSQLJobsWorkerMarksDone(t *testing.T) {
	t.Parallel()

	client := openSQLJobsTestDB(t, "jobsmod_worker_done")
	var err error
	mod, err := New(Config{Backend: BackendSQL, SQLDB: client})
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
	if _, err := jobs.Enqueue(context.Background(), "job.done", []byte(`{"v":1}`), EnqueueOptions{}); err != nil {
		t.Fatalf("enqueue failed: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- jobs.StartWorker(ctx) }()

	select {
	case <-processed:
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for worker to process job")
	}
	waitForJobStatus(t, client, "job.done", "done", -1)
	cancel()
	if err := <-done; err != nil && !errors.Is(err, context.Canceled) {
		t.Fatalf("worker returned unexpected error: %v", err)
	}
}

func TestSQLJobsWorkerRetriesThenFails(t *testing.T) {
	t.Parallel()

	client := openSQLJobsTestDB(t, "jobsmod_worker_retry")
	var err error
	mod, err := New(Config{Backend: BackendSQL, SQLDB: client})
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
	if _, err := jobs.Enqueue(context.Background(), "job.retry", []byte(`{"v":1}`), EnqueueOptions{
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
	waitForJobStatus(t, client, "job.retry", "failed", 2)
	cancel()
	if err := <-done; err != nil && !errors.Is(err, context.Canceled) {
		t.Fatalf("worker returned unexpected error: %v", err)
	}
	if attempts != 2 {
		t.Fatalf("expected 2 attempts, got %d", attempts)
	}

}

func openSQLJobsTestDB(t *testing.T, name string) *sql.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared&_fk=1", name)
	client, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	client.SetMaxOpenConns(1)
	client.SetMaxIdleConns(1)
	t.Cleanup(func() { _ = client.Close() })
	return client
}

func waitForJobStatus(t *testing.T, db *sql.DB, name, wantStatus string, wantAttempt int) {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		var status string
		var attempt int
		err := db.QueryRowContext(context.Background(), "SELECT status, attempt FROM goship_jobs WHERE name = ?", name).Scan(&status, &attempt)
		if err == nil && status == wantStatus && (wantAttempt < 0 || attempt == wantAttempt) {
			return
		}
		time.Sleep(25 * time.Millisecond)
	}

	t.Fatalf("timed out waiting for status %q (attempt=%d) for %s", wantStatus, wantAttempt, name)
}
