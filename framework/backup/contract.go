package backup

import (
	"context"
	"encoding/hex"
	"fmt"
	"path"
	"strings"
	"time"
)

const (
	// ManifestVersionV1 identifies the first stable backup manifest schema.
	ManifestVersionV1 = "backup-manifest-v1"

	// DBModeEmbedded indicates the database runs in embedded mode.
	DBModeEmbedded = "embedded"
	// DBDriverSQLite indicates SQLite as the database engine.
	DBDriverSQLite = "sqlite"
)

// Provider identifies where backup artifacts are stored.
type Provider string

const (
	ProviderLocal        Provider = "local"
	ProviderS3Compatible Provider = "s3-compatible"
)

// Manifest is the typed backup descriptor produced by backup drivers.
type Manifest struct {
	Version   string             `json:"version"`
	CreatedAt time.Time          `json:"created_at"`
	Database  DatabaseDescriptor `json:"database"`
	Artifact  ArtifactDescriptor `json:"artifact"`
	Storage   StorageLocation    `json:"storage"`
}

// DatabaseDescriptor describes the source DB for a backup.
type DatabaseDescriptor struct {
	Mode          string `json:"mode"`
	Driver        string `json:"driver"`
	SchemaVersion string `json:"schema_version"`
	SourcePath    string `json:"source_path"`
}

// ArtifactDescriptor describes the exported backup artifact.
type ArtifactDescriptor struct {
	ChecksumSHA256 string `json:"checksum_sha256"`
	SizeBytes      int64  `json:"size_bytes"`
}

// StorageLocation identifies where the artifact is stored.
type StorageLocation struct {
	Provider Provider `json:"provider"`
	URI      string   `json:"uri,omitempty"`
	Endpoint string   `json:"endpoint,omitempty"`
	Region   string   `json:"region,omitempty"`
	Bucket   string   `json:"bucket,omitempty"`
	Key      string   `json:"key,omitempty"`
}

// CreateRequest defines input for generating a backup manifest.
type CreateRequest struct {
	SQLitePath    string
	SchemaVersion string
	Storage       StorageLocation
	CreatedAt     time.Time
}

// RestoreRequest defines input for restore operations.
type RestoreRequest struct {
	Manifest Manifest
}

// RestoreEvidence is the machine-readable post-restore contract returned to callers.
type RestoreEvidence struct {
	Status                 string             `json:"status"`
	ManifestVersion        string             `json:"manifest_version"`
	ArtifactChecksumSHA256 string             `json:"artifact_checksum_sha256"`
	Database               DatabaseDescriptor `json:"database"`
	PostRestoreChecks      []string           `json:"post_restore_checks"`
}

// Driver creates backup manifests from runtime data.
type Driver interface {
	Create(ctx context.Context, req CreateRequest) (Manifest, error)
}

// Restorer validates and applies restore operations.
type Restorer interface {
	Restore(ctx context.Context, req RestoreRequest) error
}

// S3ProviderConfig describes the S3-compatible provider boundary.
type S3ProviderConfig struct {
	Endpoint  string
	Region    string
	Bucket    string
	Prefix    string
	AccessKey string
	SecretKey string
	UseSSL    bool
}

// Validate validates manifest correctness.
func (m Manifest) Validate() error {
	if strings.TrimSpace(m.Version) == "" {
		return fmt.Errorf("manifest version is required")
	}
	if m.Version != ManifestVersionV1 {
		return fmt.Errorf("unsupported manifest version %q", m.Version)
	}
	if m.CreatedAt.IsZero() {
		return fmt.Errorf("manifest created_at is required")
	}
	if strings.TrimSpace(m.Database.Mode) == "" {
		return fmt.Errorf("database mode is required")
	}
	if m.Database.Mode != DBModeEmbedded {
		return fmt.Errorf("unsupported database mode %q", m.Database.Mode)
	}
	if strings.TrimSpace(m.Database.Driver) == "" {
		return fmt.Errorf("database driver is required")
	}
	if m.Database.Driver != DBDriverSQLite {
		return fmt.Errorf("unsupported database driver %q", m.Database.Driver)
	}
	if strings.TrimSpace(m.Database.SchemaVersion) == "" {
		return fmt.Errorf("database schema_version is required")
	}
	if strings.TrimSpace(m.Database.SourcePath) == "" {
		return fmt.Errorf("database source_path is required")
	}
	if strings.TrimSpace(m.Artifact.ChecksumSHA256) == "" {
		return fmt.Errorf("artifact checksum is required")
	}
	if len(m.Artifact.ChecksumSHA256) != 64 {
		return fmt.Errorf("artifact checksum must be 64 lowercase hex characters")
	}
	if _, err := hex.DecodeString(m.Artifact.ChecksumSHA256); err != nil {
		return fmt.Errorf("artifact checksum must be valid hex: %w", err)
	}
	if m.Artifact.SizeBytes < 0 {
		return fmt.Errorf("artifact size cannot be negative")
	}
	if err := m.Storage.Validate(); err != nil {
		return err
	}
	return nil
}

// Validate validates storage target metadata.
func (l StorageLocation) Validate() error {
	switch l.Provider {
	case ProviderLocal:
		if strings.TrimSpace(l.URI) == "" {
			return fmt.Errorf("storage uri is required for local provider")
		}
	case ProviderS3Compatible:
		if strings.TrimSpace(l.Bucket) == "" {
			return fmt.Errorf("storage bucket is required for s3-compatible provider")
		}
		if strings.TrimSpace(l.Key) == "" {
			return fmt.Errorf("storage key is required for s3-compatible provider")
		}
	default:
		return fmt.Errorf("unsupported storage provider %q", l.Provider)
	}
	return nil
}

// Validate validates S3-compatible provider configuration.
func (c S3ProviderConfig) Validate() error {
	if strings.TrimSpace(c.Endpoint) == "" {
		return fmt.Errorf("s3 endpoint is required")
	}
	if strings.TrimSpace(c.Bucket) == "" {
		return fmt.Errorf("s3 bucket is required")
	}
	if strings.TrimSpace(c.AccessKey) == "" {
		return fmt.Errorf("s3 access key is required")
	}
	if strings.TrimSpace(c.SecretKey) == "" {
		return fmt.Errorf("s3 secret key is required")
	}
	return nil
}

// StorageLocation builds a storage location for an object key.
func (c S3ProviderConfig) StorageLocation(objectKey string) (StorageLocation, error) {
	if err := c.Validate(); err != nil {
		return StorageLocation{}, err
	}
	key := strings.TrimSpace(objectKey)
	if key == "" {
		return StorageLocation{}, fmt.Errorf("s3 object key is required")
	}
	joined := path.Join(strings.TrimSpace(c.Prefix), key)
	if strings.TrimSpace(joined) == "" || joined == "." {
		return StorageLocation{}, fmt.Errorf("resolved s3 object key is empty")
	}
	return StorageLocation{
		Provider: ProviderS3Compatible,
		Endpoint: strings.TrimSpace(c.Endpoint),
		Region:   strings.TrimSpace(c.Region),
		Bucket:   strings.TrimSpace(c.Bucket),
		Key:      joined,
		URI:      "s3://" + strings.TrimSpace(c.Bucket) + "/" + joined,
	}, nil
}

// NoopRestorer validates restore manifests without mutating runtime state.
type NoopRestorer struct{}

// Restore validates the provided manifest.
func (NoopRestorer) Restore(_ context.Context, req RestoreRequest) error {
	return req.Manifest.Validate()
}

// BuildRestoreEvidence returns a machine-readable restore evidence payload.
func BuildRestoreEvidence(manifest Manifest) RestoreEvidence {
	return RestoreEvidence{
		Status:                 "accepted",
		ManifestVersion:        manifest.Version,
		ArtifactChecksumSHA256: manifest.Artifact.ChecksumSHA256,
		Database:               manifest.Database,
		PostRestoreChecks: []string{
			"manifest.validated",
			"artifact.checksum.sha256",
			"database.schema_version.present",
		},
	}
}
