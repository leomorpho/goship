package profiles

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func TestRegisterUserWithProfile_RequiresDB(t *testing.T) {
	svc := NewProfileServiceWithDBDeps(nil, "sqlite", nil, nil, nil)
	_, err := svc.RegisterUserWithProfile(
		context.Background(),
		"Alice",
		"alice@example.com",
		"hash",
		time.Now().UTC().AddDate(-30, 0, 0),
		nil,
	)
	if err != ErrProfileDBNotConfigured {
		t.Fatalf("expected ErrProfileDBNotConfigured, got %v", err)
	}
}

func TestMarkPhoneVerified_SQLite(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()

	if _, err := db.Exec(`
CREATE TABLE profiles (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  phone_verified BOOLEAN NOT NULL DEFAULT 0
);
INSERT INTO profiles (id, phone_verified) VALUES (1, 0);
`); err != nil {
		t.Fatalf("seed sqlite: %v", err)
	}

	svc := NewProfileServiceWithDBDeps(db, "sqlite", nil, nil, nil)
	if err := svc.MarkPhoneVerified(context.Background(), 1); err != nil {
		t.Fatalf("MarkPhoneVerified error: %v", err)
	}

	var verified bool
	if err := db.QueryRow(`SELECT phone_verified FROM profiles WHERE id = 1`).Scan(&verified); err != nil {
		t.Fatalf("query phone_verified: %v", err)
	}
	if !verified {
		t.Fatal("expected phone_verified=true")
	}
}
