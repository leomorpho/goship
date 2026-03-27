package factory

import (
	"database/sql"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

type testUser struct {
	ID        int64     `db:"id"`
	Name      string    `db:"name"`
	Email     string    `db:"email"`
	Role      string    `db:"role"`
	CreatedAt time.Time `db:"created_at"`
}

func (testUser) TableName() string { return "users" }

func TestFactoryCreateInsertsAndAssignsID(t *testing.T) {
	db, err := sql.Open("sqlite", "file:factory_create?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if _, err := db.Exec(`CREATE TABLE users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		email TEXT NOT NULL UNIQUE,
		role TEXT NOT NULL,
		created_at DATETIME NOT NULL
	)`); err != nil {
		t.Fatalf("create table: %v", err)
	}

	f := New(func() testUser {
		return testUser{
			Name:      "Test User",
			Email:     Sequence("user") + "@example.com",
			Role:      "member",
			CreatedAt: time.Now().UTC(),
		}
	}).AfterBuild(func(u *testUser) {
		if u.Role == "" {
			u.Role = "member"
		}
	})

	got := f.Create(t, db, func(u *testUser) { u.Role = "admin" })
	if got.ID == 0 {
		t.Fatalf("expected auto ID to be assigned, got %+v", got)
	}

	var role string
	if err := db.QueryRow(`SELECT role FROM users WHERE id = ?`, got.ID).Scan(&role); err != nil {
		t.Fatalf("query role: %v", err)
	}
	if role != "admin" {
		t.Fatalf("role = %q, want admin", role)
	}
}

func TestSequenceGeneratesUniqueValues(t *testing.T) {
	first := Sequence("user")
	second := Sequence("user")
	if first == second {
		t.Fatalf("sequence values must be unique: %q == %q", first, second)
	}
}
