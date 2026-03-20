package backup

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSQLiteDriverCreateManifest(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "main.db")
	dbBytes := []byte("sqlite-backup-content")
	require.NoError(t, os.WriteFile(dbPath, dbBytes, 0o644))

	createdAt := time.Date(2026, time.March, 12, 12, 0, 0, 0, time.UTC)
	driver := SQLiteDriver{Now: func() time.Time { return createdAt }}

	manifest, err := driver.Create(context.Background(), CreateRequest{
		SQLitePath:    dbPath,
		SchemaVersion: "20260312_001",
		Storage: StorageLocation{
			Provider: ProviderLocal,
			URI:      "file:///tmp/backups/main.db",
		},
	})
	require.NoError(t, err)

	sum := sha256.Sum256(dbBytes)
	assert.Equal(t, ManifestVersionV1, manifest.Version)
	assert.Equal(t, createdAt, manifest.CreatedAt)
	assert.Equal(t, DBModeEmbedded, manifest.Database.Mode)
	assert.Equal(t, DBDriverSQLite, manifest.Database.Driver)
	assert.Equal(t, "20260312_001", manifest.Database.SchemaVersion)
	assert.Equal(t, dbPath, manifest.Database.SourcePath)
	assert.Equal(t, hex.EncodeToString(sum[:]), manifest.Artifact.ChecksumSHA256)
	assert.Equal(t, int64(len(dbBytes)), manifest.Artifact.SizeBytes)
	assert.Equal(t, ProviderLocal, manifest.Storage.Provider)
	assert.Equal(t, "file:///tmp/backups/main.db", manifest.Storage.URI)
}

func TestSQLiteDriverCreateRequiresPathAndSchemaVersion(t *testing.T) {
	driver := NewSQLiteDriver()
	_, err := driver.Create(context.Background(), CreateRequest{
		SchemaVersion: "v1",
		Storage:       StorageLocation{Provider: ProviderLocal, URI: "file:///tmp/main.db"},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "sqlite path")

	_, err = driver.Create(context.Background(), CreateRequest{
		SQLitePath: t.TempDir(),
		Storage:    StorageLocation{Provider: ProviderLocal, URI: "file:///tmp/main.db"},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "schema version")
}

func TestNoopRestorerValidatesManifest(t *testing.T) {
	restorer := NoopRestorer{}
	err := restorer.Restore(context.Background(), RestoreRequest{Manifest: Manifest{}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "manifest version")
}
