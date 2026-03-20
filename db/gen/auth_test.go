package gen

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func TestGetAuthUserRecordByEmail_SQLite(t *testing.T) {
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
		('Alice', 'alice@example.com', 'hashed_pw', 1)`); err != nil {
		t.Fatalf("seed users: %v", err)
	}

	user, err := GetAuthUserRecordByEmail(context.Background(), db, "sqlite", "ALICE@example.com")
	if err != nil {
		t.Fatalf("get auth user record: %v", err)
	}
	if user.UserID <= 0 {
		t.Fatalf("user id = %d", user.UserID)
	}
	if user.Name != "Alice" || user.Email != "alice@example.com" || user.Password != "hashed_pw" || !user.IsVerified {
		t.Fatalf("unexpected user: %#v", user)
	}
}

func TestGetAuthIdentityByUserID_SQLite(t *testing.T) {
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
		('Bob', 'bob@example.com', 'hashed_pw2', 0)`)
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}
	userID, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("read last insert id: %v", err)
	}
	_, err = db.Exec(`INSERT INTO profiles (user_profile, fully_onboarded) VALUES (?, 1)`, userID)
	if err != nil {
		t.Fatalf("seed profile: %v", err)
	}

	identity, err := GetAuthIdentityByUserID(context.Background(), db, "sqlite", int(userID))
	if err != nil {
		t.Fatalf("get auth identity: %v", err)
	}
	if identity.UserID != int(userID) || identity.UserName != "Bob" || identity.UserEmail != "bob@example.com" {
		t.Fatalf("unexpected identity user fields: %#v", identity)
	}
	if !identity.HasProfile || identity.ProfileID <= 0 || !identity.ProfileFullyOnboarded {
		t.Fatalf("unexpected identity profile fields: %#v", identity)
	}
}

func TestGetAuthUserRecordByEmailQuery_PostgresPlaceholders(t *testing.T) {
	query, args := getAuthUserRecordByEmailQuery("postgres", " Alice@Example.com ")
	if !strings.Contains(query, "WHERE email = $1") {
		t.Fatalf("query = %q", query)
	}
	if len(args) != 1 || args[0] != "alice@example.com" {
		t.Fatalf("args = %#v", args)
	}
}

func TestInsertLastSeenOnline_SQLite(t *testing.T) {
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
		('Casey', 'casey@example.com', 'hash3', 0)`)
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}
	userID, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("last insert id: %v", err)
	}
	now := time.Now().UTC().Truncate(time.Second)
	if err := InsertLastSeenOnline(context.Background(), db, "sqlite", int(userID), now); err != nil {
		t.Fatalf("insert last seen: %v", err)
	}

	var gotCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM last_seen_onlines WHERE user_last_seen_at = ?`, userID).Scan(&gotCount); err != nil {
		t.Fatalf("count last seen rows: %v", err)
	}
	if gotCount != 1 {
		t.Fatalf("count = %d, want 1", gotCount)
	}
}

func TestPasswordTokenHelpers_SQLite(t *testing.T) {
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
		('Evan', 'evan@example.com', 'hash5', 0)`)
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}
	userID64, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("last insert id: %v", err)
	}
	userID := int(userID64)

	createdAt := time.Now().UTC().Truncate(time.Second)
	tokenID, err := InsertPasswordToken(context.Background(), db, "sqlite", userID, "pt_hash", createdAt)
	if err != nil {
		t.Fatalf("insert password token: %v", err)
	}
	if tokenID <= 0 {
		t.Fatalf("token id = %d", tokenID)
	}

	hash, err := GetPasswordTokenHash(context.Background(), db, "sqlite", userID, tokenID, createdAt.Add(-time.Minute))
	if err != nil {
		t.Fatalf("get password token hash: %v", err)
	}
	if hash != "pt_hash" {
		t.Fatalf("hash = %q", hash)
	}

	if err := DeletePasswordTokensByUserID(context.Background(), db, "sqlite", userID); err != nil {
		t.Fatalf("delete password tokens: %v", err)
	}
	var gotCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM password_tokens WHERE password_token_user = ?`, userID).Scan(&gotCount); err != nil {
		t.Fatalf("count password tokens: %v", err)
	}
	if gotCount != 0 {
		t.Fatalf("count = %d, want 0", gotCount)
	}
}

func TestUserUpdateHelpers_SQLite(t *testing.T) {
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
		('Greg', 'greg@example.com', 'old_hash', 0)`)
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}
	userID64, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("last insert id: %v", err)
	}
	userID := int(userID64)

	if err := UpdateUserPasswordHashByUserID(context.Background(), db, "sqlite", userID, "new_hash"); err != nil {
		t.Fatalf("UpdateUserPasswordHashByUserID: %v", err)
	}
	if err := MarkUserVerifiedByUserID(context.Background(), db, "sqlite", userID); err != nil {
		t.Fatalf("MarkUserVerifiedByUserID: %v", err)
	}

	var gotHash string
	var gotVerified bool
	if err := db.QueryRow(`SELECT password, verified FROM users WHERE id = ?`, userID).Scan(&gotHash, &gotVerified); err != nil {
		t.Fatalf("read user: %v", err)
	}
	if gotHash != "new_hash" || !gotVerified {
		t.Fatalf("unexpected user values hash=%q verified=%v", gotHash, gotVerified)
	}
}

func TestDisplayNameHelpers_SQLite(t *testing.T) {
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
		('Iris', 'iris@example.com', 'hash9', 0)`)
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}
	userID64, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("last insert id: %v", err)
	}
	userID := int(userID64)

	name, err := GetUserDisplayNameByUserID(context.Background(), db, "sqlite", userID)
	if err != nil {
		t.Fatalf("GetUserDisplayNameByUserID: %v", err)
	}
	if name != "Iris" {
		t.Fatalf("name = %q", name)
	}

	if err := UpdateUserDisplayNameByUserID(context.Background(), db, "sqlite", userID, "Iris B"); err != nil {
		t.Fatalf("UpdateUserDisplayNameByUserID: %v", err)
	}
	name, err = GetUserDisplayNameByUserID(context.Background(), db, "sqlite", userID)
	if err != nil {
		t.Fatalf("GetUserDisplayNameByUserID updated: %v", err)
	}
	if name != "Iris B" {
		t.Fatalf("updated name = %q", name)
	}
}
