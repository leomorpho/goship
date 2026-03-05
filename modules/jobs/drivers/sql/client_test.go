package sql

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func TestNewRequiresSQLDB(t *testing.T) {
	t.Parallel()

	_, err := New(Config{})
	if err == nil {
		t.Fatal("expected error for missing SQLDB")
	}
}

func TestNewWithSQLDB(t *testing.T) {
	t.Parallel()

	db, err := sql.Open("sqlite3", "file:jobsmod_driver_sqldb?mode=memory&_fk=1")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	client, err := New(Config{SQLDB: db})
	if err != nil {
		t.Fatalf("new client failed: %v", err)
	}
	if client == nil {
		t.Fatal("expected non-nil client")
	}
}

func TestClaimNextAndMarkDone(t *testing.T) {
	t.Parallel()

	db, err := sql.Open("sqlite3", "file:jobsmod_driver?mode=memory&_fk=1")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	client, err := New(Config{SQLDB: db})
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
