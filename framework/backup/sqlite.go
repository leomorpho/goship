package backup

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"time"
)

// SQLiteDriver creates SQLite backup manifests from on-disk DB files.
type SQLiteDriver struct {
	Now func() time.Time
}

// NewSQLiteDriver returns a SQLite driver using the current clock.
func NewSQLiteDriver() SQLiteDriver {
	return SQLiteDriver{Now: time.Now}
}

// Create builds a typed manifest for a SQLite backup artifact.
func (d SQLiteDriver) Create(_ context.Context, req CreateRequest) (Manifest, error) {
	sqlitePath := strings.TrimSpace(req.SQLitePath)
	if sqlitePath == "" {
		return Manifest{}, fmt.Errorf("sqlite path is required")
	}

	schemaVersion := strings.TrimSpace(req.SchemaVersion)
	if schemaVersion == "" {
		return Manifest{}, fmt.Errorf("schema version is required")
	}

	bytes, err := os.ReadFile(sqlitePath)
	if err != nil {
		return Manifest{}, fmt.Errorf("read sqlite file: %w", err)
	}
	stat, err := os.Stat(sqlitePath)
	if err != nil {
		return Manifest{}, fmt.Errorf("stat sqlite file: %w", err)
	}

	createdAt := req.CreatedAt
	if createdAt.IsZero() {
		nowFn := d.Now
		if nowFn == nil {
			nowFn = time.Now
		}
		createdAt = nowFn().UTC()
	}

	sum := sha256.Sum256(bytes)
	manifest := Manifest{
		Version:   ManifestVersionV1,
		CreatedAt: createdAt,
		Database: DatabaseDescriptor{
			Mode:          DBModeEmbedded,
			Driver:        DBDriverSQLite,
			SchemaVersion: schemaVersion,
			SourcePath:    sqlitePath,
		},
		Artifact: ArtifactDescriptor{
			ChecksumSHA256: hex.EncodeToString(sum[:]),
			SizeBytes:      stat.Size(),
		},
		Storage: req.Storage,
	}

	if err := manifest.Validate(); err != nil {
		return Manifest{}, err
	}

	return manifest, nil
}
