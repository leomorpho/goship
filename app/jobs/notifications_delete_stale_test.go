package tasks

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/leomorpho/goship/framework/domain"
	_ "github.com/mattn/go-sqlite3"
)

func TestDeleteStaleNotificationsProcessor_ProcessTask_SQLite(t *testing.T) {
	t.Parallel()

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	if _, err := db.ExecContext(ctx, `
		CREATE TABLE notifications (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			type TEXT NOT NULL,
			created_at DATETIME NOT NULL
		)
	`); err != nil {
		t.Fatalf("create schema: %v", err)
	}

	now := time.Now().UTC()
	rows := []struct {
		typeVal   string
		createdAt time.Time
	}{
		{typeVal: "other", createdAt: now.Add(-10 * 24 * time.Hour)},
		{typeVal: domain.NotificationTypeDailyConversationReminder.Value, createdAt: now.Add(-3 * 24 * time.Hour)},
		{typeVal: domain.NotificationTypeDailyConversationReminder.Value, createdAt: now.Add(-24 * time.Hour)},
		{typeVal: "other", createdAt: now.Add(-24 * time.Hour)},
	}
	for _, r := range rows {
		if _, err := db.ExecContext(ctx, `INSERT INTO notifications(type, created_at) VALUES(?, ?)`, r.typeVal, r.createdAt); err != nil {
			t.Fatalf("seed row: %v", err)
		}
	}

	p := NewDeleteStaleNotificationsProcessor(db, "sqlite", 7)
	if err := p.ProcessTask(ctx, nil); err != nil {
		t.Fatalf("process task: %v", err)
	}

	var count int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM notifications`).Scan(&count); err != nil {
		t.Fatalf("count rows: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected 2 rows after cleanup, got %d", count)
	}
}
