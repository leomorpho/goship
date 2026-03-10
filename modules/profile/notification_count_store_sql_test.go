package profiles

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestSQLNotificationCountStore_CountUnseenNotifications_SQLite(t *testing.T) {
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

	store := NewSQLNotificationCountStore(db, "sqlite")
	got, err := store.CountUnseenNotifications(context.Background(), 1)
	if err != nil {
		t.Fatalf("count unseen notifications: %v", err)
	}
	if got != 2 {
		t.Fatalf("count = %d, want 2", got)
	}
}

func TestSQLNotificationCountStore_CountQuery_PostgresPlaceholders(t *testing.T) {
	store := NewSQLNotificationCountStore(nil, "postgres")
	query, args := store.countQuery(12)
	normalized := strings.Join(strings.Fields(query), " ")
	if normalized != "SELECT COUNT(*) FROM notifications WHERE profile_notifications = $1 AND read = $2;" {
		t.Fatalf("query = %q", query)
	}
	if len(args) != 2 || args[0] != 12 || args[1] != false {
		t.Fatalf("args = %#v", args)
	}
}
