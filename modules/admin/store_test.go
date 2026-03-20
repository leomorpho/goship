package admin

import (
	"context"
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

func TestStoreCRUD(t *testing.T) {
	db := newAdminTestDB(t)
	ctx := context.Background()
	res := AdminResource{
		Name:       "Post",
		PluralName: "Posts",
		TableName:  "posts",
		IDField:    "id",
	}

	if err := Create(ctx, db, res, map[string]any{
		"title":     "hello",
		"published": true,
	}); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := Create(ctx, db, res, map[string]any{
		"title":     "world",
		"published": false,
	}); err != nil {
		t.Fatalf("create second: %v", err)
	}

	rows, total, err := List(ctx, db, res, 1, 10)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if total != 2 {
		t.Fatalf("total = %d, want 2", total)
	}
	if len(rows) != 2 {
		t.Fatalf("rows len = %d, want 2", len(rows))
	}

	row, err := Get(ctx, db, res, int64(1))
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if row["title"] != "hello" {
		t.Fatalf("get title = %v, want hello", row["title"])
	}

	if err := Update(ctx, db, res, int64(1), map[string]any{
		"title": "edited",
	}); err != nil {
		t.Fatalf("update: %v", err)
	}

	updated, err := Get(ctx, db, res, int64(1))
	if err != nil {
		t.Fatalf("get updated: %v", err)
	}
	if updated["title"] != "edited" {
		t.Fatalf("updated title = %v, want edited", updated["title"])
	}

	if err := Delete(ctx, db, res, int64(2)); err != nil {
		t.Fatalf("delete: %v", err)
	}
	rows, total, err = List(ctx, db, res, 1, 10)
	if err != nil {
		t.Fatalf("list after delete: %v", err)
	}
	if total != 1 {
		t.Fatalf("total after delete = %d, want 1", total)
	}
	if len(rows) != 1 {
		t.Fatalf("rows after delete = %d, want 1", len(rows))
	}
}

func newAdminTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite", "file:admin_store_test?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS posts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			title TEXT NOT NULL,
			published BOOLEAN NOT NULL DEFAULT 0
		);
	`); err != nil {
		t.Fatalf("create schema: %v", err)
	}

	if _, err := db.Exec(`DELETE FROM posts`); err != nil {
		t.Fatalf("clean schema: %v", err)
	}
	return db
}
