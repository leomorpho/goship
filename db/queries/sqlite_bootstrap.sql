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
	preferred_language TEXT,
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

-- name: sqlite_bootstrap_create_ai_conversations
CREATE TABLE IF NOT EXISTS ai_conversations (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	user_id INTEGER NOT NULL,
	model TEXT NOT NULL,
	title TEXT,
	created_at DATETIME NOT NULL,
	updated_at DATETIME NOT NULL,
	FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
)

-- name: sqlite_bootstrap_create_ai_messages
CREATE TABLE IF NOT EXISTS ai_messages (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	conversation_id INTEGER NOT NULL,
	role TEXT NOT NULL,
	content TEXT NOT NULL,
	input_tokens INTEGER,
	output_tokens INTEGER,
	model TEXT,
	created_at DATETIME NOT NULL,
	FOREIGN KEY(conversation_id) REFERENCES ai_conversations(id) ON DELETE CASCADE
)

-- name: sqlite_bootstrap_create_audit_logs
CREATE TABLE IF NOT EXISTS audit_logs (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	user_id INTEGER,
	action TEXT NOT NULL,
	resource_type TEXT,
	resource_id TEXT,
	changes TEXT,
	ip_address TEXT,
	user_agent TEXT,
	created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_audit_logs_user_id ON audit_logs(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_logs_resource ON audit_logs(resource_type, resource_id)

-- name: sqlite_bootstrap_create_feature_flags
CREATE TABLE IF NOT EXISTS feature_flags (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	key TEXT NOT NULL UNIQUE,
	enabled INTEGER NOT NULL DEFAULT 0,
	rollout_pct INTEGER NOT NULL DEFAULT 0,
	user_ids TEXT,
	description TEXT,
	created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_feature_flags_key ON feature_flags(key)
