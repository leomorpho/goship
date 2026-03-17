package factories

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

func TestUserFactoryCreateAndTraits(t *testing.T) {
	db, err := sql.Open("sqlite", "file:user_factory?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if _, err := db.Exec(`CREATE TABLE users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		email TEXT NOT NULL UNIQUE,
		password TEXT NOT NULL,
		role TEXT NOT NULL,
		created_at DATETIME NOT NULL
	)`); err != nil {
		t.Fatalf("create users table: %v", err)
	}

	user := User.Create(t, db)
	admin := User.Create(t, db, WithAdminRole)

	if user.ID == 0 || admin.ID == 0 {
		t.Fatalf("expected inserted IDs, got user=%d admin=%d", user.ID, admin.ID)
	}
	if user.Role != "member" {
		t.Fatalf("user role = %q, want member", user.Role)
	}
	if admin.Role != "admin" {
		t.Fatalf("admin role = %q, want admin", admin.Role)
	}
	if user.Email == admin.Email {
		t.Fatalf("emails must be unique, got %q", user.Email)
	}
}
