package foundation

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func TestBobAuthStore_GetUserRecordByEmail_SQLite(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	_, err = db.Exec(`CREATE TABLE users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		email TEXT NOT NULL UNIQUE,
		password TEXT NOT NULL,
		verified BOOLEAN NOT NULL DEFAULT 0
	)`)
	if err != nil {
		t.Fatalf("create users table: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO users (name, email, password, verified) VALUES
		('Alice', 'alice@example.com', 'hash1', 1)`); err != nil {
		t.Fatalf("seed users: %v", err)
	}

	store := newBobAuthStore(db, "sqlite")
	user, err := store.GetUserRecordByEmail(context.Background(), "ALICE@example.com")
	if err != nil {
		t.Fatalf("GetUserRecordByEmail: %v", err)
	}
	if user.UserID <= 0 || user.Name != "Alice" || user.Email != "alice@example.com" || user.Password != "hash1" || !user.IsVerified {
		t.Fatalf("unexpected user: %#v", user)
	}
}

func TestBobAuthStore_GetIdentityByUserID_SQLite(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	_, err = db.Exec(`CREATE TABLE users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		email TEXT NOT NULL UNIQUE,
		password TEXT NOT NULL,
		verified BOOLEAN NOT NULL DEFAULT 0
	)`)
	if err != nil {
		t.Fatalf("create users table: %v", err)
	}
	_, err = db.Exec(`CREATE TABLE profiles (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_profile INTEGER NOT NULL UNIQUE,
		fully_onboarded BOOLEAN NOT NULL DEFAULT 0
	)`)
	if err != nil {
		t.Fatalf("create profiles table: %v", err)
	}
	res, err := db.Exec(`INSERT INTO users (name, email, password, verified) VALUES
		('Bob', 'bob@example.com', 'hash2', 0)`)
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}
	userID, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("last insert id: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO profiles (user_profile, fully_onboarded) VALUES (?, 1)`, userID); err != nil {
		t.Fatalf("seed profile: %v", err)
	}

	store := newBobAuthStore(db, "sqlite")
	identity, err := store.GetIdentityByUserID(context.Background(), int(userID))
	if err != nil {
		t.Fatalf("GetIdentityByUserID: %v", err)
	}
	if identity.UserID != int(userID) || identity.UserName != "Bob" || identity.UserEmail != "bob@example.com" {
		t.Fatalf("unexpected identity user fields: %#v", identity)
	}
	if !identity.HasProfile || identity.ProfileID <= 0 || !identity.ProfileFullyOnboarded {
		t.Fatalf("unexpected identity profile fields: %#v", identity)
	}
}

func TestBobAuthStore_WritesNotImplemented(t *testing.T) {
	store := newBobAuthStore(nil, "sqlite")
	if err := store.CreateLastSeenOnline(context.Background(), 1, time.Now()); err == nil {
		t.Fatal("expected error")
	}
	if err := store.UpdateUserPasswordHashByUserID(context.Background(), 1, "hash"); err == nil {
		t.Fatal("expected error")
	}
	if _, err := store.GetUserDisplayNameByUserID(context.Background(), 1); err == nil {
		t.Fatal("expected error")
	}
	if err := store.UpdateUserDisplayNameByUserID(context.Background(), 1, "name"); err == nil {
		t.Fatal("expected error")
	}
	if err := store.MarkUserVerifiedByUserID(context.Background(), 1); err == nil {
		t.Fatal("expected error")
	}
	if _, err := store.CreatePasswordToken(context.Background(), 1, "hash"); err == nil {
		t.Fatal("expected error")
	}
	if _, err := store.GetPasswordTokenHash(context.Background(), 1, 1, time.Now()); err == nil {
		t.Fatal("expected error")
	}
	if err := store.DeletePasswordTokens(context.Background(), 1); err == nil {
		t.Fatal("expected error")
	}
}

func TestBobAuthStore_CreateLastSeenOnline_SQLite(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	_, err = db.Exec(`CREATE TABLE users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		email TEXT NOT NULL UNIQUE,
		password TEXT NOT NULL,
		verified BOOLEAN NOT NULL DEFAULT 0
	)`)
	if err != nil {
		t.Fatalf("create users table: %v", err)
	}
	_, err = db.Exec(`CREATE TABLE last_seen_onlines (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		seen_at DATETIME NOT NULL,
		user_last_seen_at INTEGER NOT NULL
	)`)
	if err != nil {
		t.Fatalf("create last_seen_onlines table: %v", err)
	}
	res, err := db.Exec(`INSERT INTO users (name, email, password, verified) VALUES
		('Dana', 'dana@example.com', 'hash4', 0)`)
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}
	userID, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("last insert id: %v", err)
	}

	store := newBobAuthStore(db, "sqlite")
	if err := store.CreateLastSeenOnline(context.Background(), int(userID), time.Now().UTC()); err != nil {
		t.Fatalf("CreateLastSeenOnline: %v", err)
	}

	var gotCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM last_seen_onlines WHERE user_last_seen_at = ?`, userID).Scan(&gotCount); err != nil {
		t.Fatalf("count last seen rows: %v", err)
	}
	if gotCount != 1 {
		t.Fatalf("count = %d, want 1", gotCount)
	}
}

func TestBobAuthStore_PasswordTokenFlow_SQLite(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	_, err = db.Exec(`CREATE TABLE users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		email TEXT NOT NULL UNIQUE,
		password TEXT NOT NULL,
		verified BOOLEAN NOT NULL DEFAULT 0
	)`)
	if err != nil {
		t.Fatalf("create users table: %v", err)
	}
	_, err = db.Exec(`CREATE TABLE password_tokens (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		hash TEXT NOT NULL,
		created_at DATETIME NOT NULL,
		password_token_user INTEGER NOT NULL
	)`)
	if err != nil {
		t.Fatalf("create password_tokens table: %v", err)
	}
	res, err := db.Exec(`INSERT INTO users (name, email, password, verified) VALUES
		('Fran', 'fran@example.com', 'hash6', 0)`)
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}
	userID64, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("last insert id: %v", err)
	}
	userID := int(userID64)

	store := newBobAuthStore(db, "sqlite")
	tokenID, err := store.CreatePasswordToken(context.Background(), userID, "pw_hash")
	if err != nil {
		t.Fatalf("CreatePasswordToken: %v", err)
	}
	if tokenID <= 0 {
		t.Fatalf("token id = %d", tokenID)
	}

	hash, err := store.GetPasswordTokenHash(context.Background(), userID, tokenID, time.Now().UTC().Add(-time.Hour))
	if err != nil {
		t.Fatalf("GetPasswordTokenHash: %v", err)
	}
	if hash != "pw_hash" {
		t.Fatalf("hash = %q", hash)
	}

	if err := store.DeletePasswordTokens(context.Background(), userID); err != nil {
		t.Fatalf("DeletePasswordTokens: %v", err)
	}
	var gotCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM password_tokens WHERE password_token_user = ?`, userID).Scan(&gotCount); err != nil {
		t.Fatalf("count password tokens: %v", err)
	}
	if gotCount != 0 {
		t.Fatalf("count = %d, want 0", gotCount)
	}
}

func TestBobAuthStore_UserUpdateHelpers_SQLite(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	_, err = db.Exec(`CREATE TABLE users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		email TEXT NOT NULL UNIQUE,
		password TEXT NOT NULL,
		verified BOOLEAN NOT NULL DEFAULT 0
	)`)
	if err != nil {
		t.Fatalf("create users table: %v", err)
	}
	res, err := db.Exec(`INSERT INTO users (name, email, password, verified) VALUES
		('Hanna', 'hanna@example.com', 'hash7', 0)`)
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}
	userID64, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("last insert id: %v", err)
	}
	userID := int(userID64)

	store := newBobAuthStore(db, "sqlite")
	if err := store.UpdateUserPasswordHashByUserID(context.Background(), userID, "hash8"); err != nil {
		t.Fatalf("UpdateUserPasswordHashByUserID: %v", err)
	}
	if err := store.MarkUserVerifiedByUserID(context.Background(), userID); err != nil {
		t.Fatalf("MarkUserVerifiedByUserID: %v", err)
	}

	var gotHash string
	var gotVerified bool
	if err := db.QueryRow(`SELECT password, verified FROM users WHERE id = ?`, userID).Scan(&gotHash, &gotVerified); err != nil {
		t.Fatalf("read user: %v", err)
	}
	if gotHash != "hash8" || !gotVerified {
		t.Fatalf("unexpected user values hash=%q verified=%v", gotHash, gotVerified)
	}
}

func TestBobAuthStore_DisplayNameHelpers_SQLite(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	_, err = db.Exec(`CREATE TABLE users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		email TEXT NOT NULL UNIQUE,
		password TEXT NOT NULL,
		verified BOOLEAN NOT NULL DEFAULT 0
	)`)
	if err != nil {
		t.Fatalf("create users table: %v", err)
	}
	res, err := db.Exec(`INSERT INTO users (name, email, password, verified) VALUES
		('Jade', 'jade@example.com', 'hash10', 0)`)
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}
	userID64, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("last insert id: %v", err)
	}
	userID := int(userID64)

	store := newBobAuthStore(db, "sqlite")
	name, err := store.GetUserDisplayNameByUserID(context.Background(), userID)
	if err != nil {
		t.Fatalf("GetUserDisplayNameByUserID: %v", err)
	}
	if name != "Jade" {
		t.Fatalf("name = %q", name)
	}
	if err := store.UpdateUserDisplayNameByUserID(context.Background(), userID, "Jade K"); err != nil {
		t.Fatalf("UpdateUserDisplayNameByUserID: %v", err)
	}
	name, err = store.GetUserDisplayNameByUserID(context.Background(), userID)
	if err != nil {
		t.Fatalf("GetUserDisplayNameByUserID updated: %v", err)
	}
	if name != "Jade K" {
		t.Fatalf("updated name = %q", name)
	}
}
