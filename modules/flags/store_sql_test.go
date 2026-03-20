package flags

import (
	"context"
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

func TestSQLStoreCRUD(t *testing.T) {
	db := newFlagsTestDB(t)
	store := NewSQLStore(db)

	seed := Flag{
		Key:         "new_checkout_flow",
		Enabled:     true,
		RolloutPct:  25,
		UserIDs:     []int64{42, 99},
		Description: "staged checkout rollout",
	}
	if err := store.Create(context.Background(), seed); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	got, err := store.Find(context.Background(), "new_checkout_flow")
	if err != nil {
		t.Fatalf("Find() error = %v", err)
	}
	if !got.Enabled || got.RolloutPct != 25 || len(got.UserIDs) != 2 {
		t.Fatalf("Find() = %+v", got)
	}

	seed.Enabled = false
	seed.RolloutPct = 0
	seed.UserIDs = nil
	if err := store.Update(context.Background(), seed); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	got, err = store.Find(context.Background(), "new_checkout_flow")
	if err != nil {
		t.Fatalf("Find() after update error = %v", err)
	}
	if got.Enabled || got.RolloutPct != 0 || len(got.UserIDs) != 0 {
		t.Fatalf("updated flag = %+v", got)
	}

	flags, err := store.List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(flags) != 1 || flags[0].Key != "new_checkout_flow" {
		t.Fatalf("List() = %+v", flags)
	}

	if err := store.Delete(context.Background(), "new_checkout_flow"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if _, err := store.Find(context.Background(), "new_checkout_flow"); err == nil {
		t.Fatal("expected missing flag error after Delete()")
	}
}

func newFlagsTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite", "file:flags_store_test?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	_, err = db.Exec(`
CREATE TABLE feature_flags (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    key TEXT NOT NULL UNIQUE,
    enabled INTEGER NOT NULL DEFAULT 0,
    rollout_pct INTEGER NOT NULL DEFAULT 0,
    user_ids TEXT,
    description TEXT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);`)
	if err != nil {
		t.Fatalf("create feature_flags: %v", err)
	}

	return db
}
