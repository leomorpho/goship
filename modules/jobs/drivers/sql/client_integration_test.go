//go:build integration

package sql

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
)

func TestClient_WithModuleMigration_EndToEnd(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "jobs-driver-integration.db")
	db, err := sql.Open("sqlite3", dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	applyJobsModuleMigration(t, db)

	client, err := New(Config{SQLDB: db})
	require.NoError(t, err)

	ctx := context.Background()
	require.NoError(t, client.Enqueue(ctx, "job-1", "default", "job.test", `{"ok":true}`, time.Now().UTC(), 1))

	job, found, err := client.ClaimNext(ctx, "worker-1", time.Now().UTC().Add(15*time.Second))
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, "job-1", job.ID)

	require.NoError(t, client.MarkDone(ctx, job.ID))
	stored, ok, err := client.Get(ctx, job.ID)
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, "done", stored.Status)
}

func applyJobsModuleMigration(t *testing.T, db *sql.DB) {
	t.Helper()
	migrationPath := filepath.Join("..", "..", "db", "migrate", "migrations", "20260305195000_init_jobs.sql")
	migrationBytes, err := os.ReadFile(migrationPath)
	require.NoError(t, err)
	upSQL := sqliteCompatibleJobsDDL(extractJobsGooseUp(string(migrationBytes)))
	_, err = db.Exec(upSQL)
	require.NoError(t, err)
}

func extractJobsGooseUp(content string) string {
	lines := strings.Split(content, "\n")
	var b strings.Builder
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		switch trimmed {
		case "-- +goose Up":
			continue
		case "-- +goose Down":
			return b.String()
		default:
			b.WriteString(line)
			b.WriteString("\n")
		}
	}
	return b.String()
}

func sqliteCompatibleJobsDDL(v string) string {
	return strings.ReplaceAll(v, "TIMESTAMPTZ", "DATETIME")
}
