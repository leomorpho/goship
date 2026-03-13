-- name: sqlite_bootstrap_create_users
CREATE TABLE IF NOT EXISTS users (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	created_at DATETIME NOT NULL,
	updated_at DATETIME NOT NULL,
	name TEXT NOT NULL,
	email TEXT NOT NULL UNIQUE,
	password TEXT NOT NULL,
	verified INTEGER NOT NULL DEFAULT 0,
	last_online DATETIME NULL,
	totp_secret TEXT,
	totp_enabled INTEGER NOT NULL DEFAULT 0,
	totp_backup_codes TEXT
)

-- name: sqlite_bootstrap_create_profiles
CREATE TABLE IF NOT EXISTS profiles (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	created_at DATETIME NOT NULL,
	updated_at DATETIME NOT NULL,
	user_profile INTEGER NOT NULL UNIQUE,
	fully_onboarded INTEGER NOT NULL DEFAULT 0,
	FOREIGN KEY(user_profile) REFERENCES users(id) ON DELETE CASCADE
)

-- name: sqlite_bootstrap_create_password_tokens
CREATE TABLE IF NOT EXISTS password_tokens (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	hash TEXT NOT NULL,
	created_at DATETIME NOT NULL,
	password_token_user INTEGER NOT NULL,
	FOREIGN KEY(password_token_user) REFERENCES users(id) ON DELETE CASCADE
)

-- name: sqlite_bootstrap_create_last_seen_onlines
CREATE TABLE IF NOT EXISTS last_seen_onlines (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	seen_at DATETIME NOT NULL,
	user_last_seen_at INTEGER NOT NULL,
	FOREIGN KEY(user_last_seen_at) REFERENCES users(id) ON DELETE CASCADE
)
