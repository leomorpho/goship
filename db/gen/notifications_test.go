package gen

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestCountUnseenNotificationsByProfile_SQLite(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	_, err = db.Exec(`CREATE TABLE notifications (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		profile_notifications INTEGER NOT NULL,
		read BOOLEAN NOT NULL DEFAULT 0
	)`)
	if err != nil {
		t.Fatalf("create table: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO notifications (profile_notifications, read) VALUES
		(1, 0),
		(1, 0),
		(1, 1),
		(2, 0)`); err != nil {
		t.Fatalf("seed notifications: %v", err)
	}

	got, err := CountUnseenNotificationsByProfile(context.Background(), db, "sqlite", 1)
	if err != nil {
		t.Fatalf("count unseen notifications: %v", err)
	}
	if got != 2 {
		t.Fatalf("count = %d, want 2", got)
	}
}

func TestCountUnseenNotificationsByProfileQuery_PostgresPlaceholders(t *testing.T) {
	query, args := countUnseenNotificationsByProfileQuery("postgres", 12)
	if query != "SELECT COUNT(*) FROM notifications WHERE profile_notifications = $1 AND read = $2" {
		t.Fatalf("query = %q", query)
	}
	if len(args) != 2 || args[0] != 12 || args[1] != false {
		t.Fatalf("args = %#v", args)
	}
}
