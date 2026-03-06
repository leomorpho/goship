package foundation

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func resolveMigrationsDir() (string, error) {
	relative := filepath.Join("db", "migrate", "migrations")
	if _, err := os.Stat(relative); err == nil {
		return relative, nil
	}

	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("runtime caller unavailable")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", ".."))
	absolute := filepath.Join(repoRoot, "db", "migrate", "migrations")
	if _, err := os.Stat(absolute); err != nil {
		return "", err
	}
	return absolute, nil
}

func applySQLMigrations(db *sql.DB, migrationsDir string, driver string) error {
	if _, err := os.Stat(migrationsDir); err != nil {
		return fmt.Errorf("find migrations directory %q: %w", migrationsDir, err)
	}

	trackingDDL := `
CREATE TABLE IF NOT EXISTS goship_schema_migrations (
	version VARCHAR(255) PRIMARY KEY,
	applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
)`
	if driver == "sqlite3" {
		trackingDDL = `
CREATE TABLE IF NOT EXISTS goship_schema_migrations (
	version TEXT PRIMARY KEY,
	applied_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
)`
	}
	if _, err := db.Exec(trackingDDL); err != nil {
		return fmt.Errorf("ensure goship_schema_migrations table: %w", err)
	}

	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("read migrations directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		version := migrationVersion(entry)
		if version == "" {
			continue
		}

		applied, err := hasAppliedMigration(db, driver, version)
		if err != nil {
			return err
		}
		if applied {
			continue
		}

		content, err := os.ReadFile(filepath.Join(migrationsDir, entry.Name()))
		if err != nil {
			return fmt.Errorf("read migration %s: %w", entry.Name(), err)
		}

		if err := applySingleMigration(db, driver, version, string(content)); err != nil {
			return fmt.Errorf("apply migration %s: %w", entry.Name(), err)
		}
	}

	return nil
}

func migrationVersion(entry os.DirEntry) string {
	name := entry.Name()
	if idx := strings.Index(name, "."); idx > 0 {
		return name[:idx]
	}
	return ""
}

func hasAppliedMigration(db *sql.DB, driver string, version string) (bool, error) {
	var marker int
	query := `SELECT 1 FROM goship_schema_migrations WHERE version = $1`
	if driver == "sqlite3" {
		query = `SELECT 1 FROM goship_schema_migrations WHERE version = ?`
	}
	err := db.QueryRow(query, version).Scan(&marker)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return false, fmt.Errorf("check migration version %q: %w", version, err)
}

func applySingleMigration(db *sql.DB, driver string, version string, sqlText string) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.Exec(sqlText); err != nil {
		return fmt.Errorf("execute SQL: %w", err)
	}
	insert := `INSERT INTO goship_schema_migrations (version) VALUES ($1)`
	if driver == "sqlite3" {
		insert = `INSERT INTO goship_schema_migrations (version) VALUES (?)`
	}
	if _, err := tx.Exec(insert, version); err != nil {
		return fmt.Errorf("record migration version %q: %w", version, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit migration %q: %w", version, err)
	}
	return nil
}

// ensureEmbeddedSQLiteSchema creates the minimal schema required by DB-first auth/container paths.
// Embedded full migration parity is tracked separately in the Bob transition plan.
func ensureEmbeddedSQLiteSchema(db *sql.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			name TEXT NOT NULL,
			email TEXT NOT NULL UNIQUE,
			password TEXT NOT NULL,
			verified INTEGER NOT NULL DEFAULT 0,
			last_online DATETIME NULL
		)`,
		`CREATE TABLE IF NOT EXISTS profiles (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			user_profile INTEGER NOT NULL UNIQUE,
			fully_onboarded INTEGER NOT NULL DEFAULT 0,
			FOREIGN KEY(user_profile) REFERENCES users(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS password_tokens (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			hash TEXT NOT NULL,
			created_at DATETIME NOT NULL,
			password_token_user INTEGER NOT NULL,
			FOREIGN KEY(password_token_user) REFERENCES users(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS last_seen_onlines (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			seen_at DATETIME NOT NULL,
			user_last_seen_at INTEGER NOT NULL,
			FOREIGN KEY(user_last_seen_at) REFERENCES users(id) ON DELETE CASCADE
		)`,
	}

	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			return err
		}
	}
	return nil
}
