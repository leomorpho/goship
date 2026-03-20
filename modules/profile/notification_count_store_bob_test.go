package profiles

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestBobNotificationCountStore_CountUnseenNotifications(t *testing.T) {
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
		(5, 0),
		(5, 1),
		(5, 0),
		(7, 0)`); err != nil {
		t.Fatalf("seed notifications: %v", err)
	}

	store := NewBobNotificationCountStore(db, "sqlite")
	got, err := store.CountUnseenNotifications(context.Background(), 5)
	if err != nil {
		t.Fatalf("count unseen notifications: %v", err)
	}
	if got != 2 {
		t.Fatalf("count = %d, want 2", got)
	}
}
