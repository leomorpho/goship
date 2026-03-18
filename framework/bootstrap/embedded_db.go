package bootstrap

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

var sqliteConnectionPragmas = []string{
	"PRAGMA journal_mode=WAL",
	"PRAGMA synchronous=NORMAL",
	"PRAGMA busy_timeout=5000",
	"PRAGMA foreign_keys=ON",
	"PRAGMA cache_size=-64000",
	"PRAGMA temp_store=MEMORY",
}

// OpenEmbeddedDB opens an embedded database connection.
// For SQLite we apply WAL-mode safety pragmas and use a single pooled connection
// to avoid SQLITE_BUSY lock contention in concurrent app workloads.
func OpenEmbeddedDB(driver, connection string) (*sql.DB, error) {
	if IsSQLiteDriver(driver) {
		if err := ensureSQLiteDataDir(connection); err != nil {
			return nil, err
		}
	}

	db, err := sql.Open(driver, connection)
	if err != nil {
		return nil, err
	}
	if !IsSQLiteDriver(driver) {
		return db, nil
	}
	if err := configureSQLiteConnection(db); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

func ensureSQLiteDataDir(connection string) error {
	path := strings.TrimSpace(connection)
	if path == "" {
		return nil
	}
	path = strings.SplitN(path, "?", 2)[0]
	if path == "" || path == ":memory:" {
		return nil
	}
	dir := filepath.Dir(path)
	if dir == "." || dir == "" {
		return nil
	}
	return os.MkdirAll(dir, 0o755)
}

func configureSQLiteConnection(db *sql.DB) error {
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0)

	for _, pragma := range sqliteConnectionPragmas {
		if _, err := db.Exec(pragma); err != nil {
			return fmt.Errorf("apply %s: %w", pragma, err)
		}
	}
	return nil
}
