package profiles

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestPreferredLanguageAndSetPreferredLanguage_SQLite(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()

	if _, err := db.Exec(`
CREATE TABLE profiles (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_profile INTEGER NOT NULL UNIQUE,
  preferred_language TEXT
);
INSERT INTO profiles (user_profile, preferred_language) VALUES (1, NULL);
`); err != nil {
		t.Fatalf("seed sqlite: %v", err)
	}

	svc := NewProfileServiceWithDBDeps(db, "sqlite", nil, nil, nil)

	if _, ok, err := svc.PreferredLanguage(context.Background(), 1); err != nil || ok {
		t.Fatalf("expected no preferred language before update, ok=%v err=%v", ok, err)
	}

	if err := svc.SetPreferredLanguage(context.Background(), 1, "FR"); err != nil {
		t.Fatalf("SetPreferredLanguage error: %v", err)
	}

	lang, ok, err := svc.PreferredLanguage(context.Background(), 1)
	if err != nil {
		t.Fatalf("PreferredLanguage error: %v", err)
	}
	if !ok {
		t.Fatalf("expected preferred language to exist after update")
	}
	if lang != "fr" {
		t.Fatalf("lang = %q, want fr", lang)
	}
}

func TestSetPreferredLanguage_NoProfileReturnsError(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()

	if _, err := db.Exec(`
CREATE TABLE profiles (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_profile INTEGER NOT NULL UNIQUE,
  preferred_language TEXT
);
`); err != nil {
		t.Fatalf("seed sqlite: %v", err)
	}

	svc := NewProfileServiceWithDBDeps(db, "sqlite", nil, nil, nil)
	if err := svc.SetPreferredLanguage(context.Background(), 99, "en"); err == nil {
		t.Fatal("expected error when profile row does not exist")
	}
}
