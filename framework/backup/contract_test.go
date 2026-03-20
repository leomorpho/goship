package backup

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestS3ProviderConfigValidate(t *testing.T) {
	cfg := S3ProviderConfig{
		Endpoint:  "s3.us-west-002.backblazeb2.com",
		Region:    "us-west-002",
		Bucket:    "goship-backups",
		Prefix:    "nightly",
		AccessKey: "key",
		SecretKey: "secret",
		UseSSL:    true,
	}

	require.NoError(t, cfg.Validate())

	loc, err := cfg.StorageLocation("snapshot.db")
	require.NoError(t, err)
	assert.Equal(t, ProviderS3Compatible, loc.Provider)
	assert.Equal(t, "goship-backups", loc.Bucket)
	assert.Equal(t, "nightly/snapshot.db", loc.Key)
	assert.Equal(t, "s3://goship-backups/nightly/snapshot.db", loc.URI)
}

func TestS3ProviderConfigValidateMissingBucket(t *testing.T) {
	cfg := S3ProviderConfig{
		Endpoint:  "s3.us-west-002.backblazeb2.com",
		AccessKey: "key",
		SecretKey: "secret",
	}

	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bucket")
}

func TestStorageLocationValidateLocal(t *testing.T) {
	loc := StorageLocation{Provider: ProviderLocal, URI: "file:///tmp/snapshot.db"}
	require.NoError(t, loc.Validate())
}

func TestStorageLocationValidateS3MissingKey(t *testing.T) {
	loc := StorageLocation{Provider: ProviderS3Compatible, Bucket: "backups"}
	err := loc.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "storage key")
}

func TestManifestValidate_V1SchemaAndChecksumInvariant_RedSpec(t *testing.T) {
	valid := Manifest{
		Version:   ManifestVersionV1,
		CreatedAt: time.Date(2026, time.March, 18, 12, 0, 0, 0, time.UTC),
		Database: DatabaseDescriptor{
			Mode:          DBModeEmbedded,
			Driver:        DBDriverSQLite,
			SchemaVersion: "20260318_001",
			SourcePath:    ".local/db/main.db",
		},
		Artifact: ArtifactDescriptor{
			ChecksumSHA256: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
			SizeBytes:      4096,
		},
		Storage: StorageLocation{
			Provider: ProviderLocal,
			URI:      "file:///tmp/main.db",
		},
	}
	require.NoError(t, valid.Validate())

	invalidVersion := valid
	invalidVersion.Version = "backup-manifest-v0"
	err := invalidVersion.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported manifest version")

	invalidChecksum := valid
	invalidChecksum.Artifact.ChecksumSHA256 = "not-hex"
	err = invalidChecksum.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "artifact checksum")
}

func TestRestoreEvidenceJSONContract_RedSpec(t *testing.T) {
	evidence := RestoreEvidence{
		Status:                  "accepted",
		AcceptedManifestVersion: ManifestVersionV1,
		ArtifactChecksumSHA256:  "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		Database: DatabaseDescriptor{
			Mode:          DBModeEmbedded,
			Driver:        DBDriverSQLite,
			SchemaVersion: "20260318_001",
			SourcePath:    ".local/db/main.db",
		},
		RecordLinks: RecordLinks{
			IncidentID: "inc-100",
			RecoveryID: "recovery-200",
			DeployID:   "deploy-300",
		},
		PostRestoreChecks: []string{
			"manifest.validated",
			"artifact.checksum.sha256",
			"database.schema_version.present",
		},
	}

	payload, err := json.Marshal(evidence)
	require.NoError(t, err)
	text := string(payload)
	assert.Contains(t, text, `"accepted_manifest_version":"backup-manifest-v1"`)
	assert.Contains(t, text, `"record_links":{"incident_id":"inc-100","recovery_id":"recovery-200","deploy_id":"deploy-300"}`)
	assert.Contains(t, text, `"post_restore_checks":["manifest.validated","artifact.checksum.sha256","database.schema_version.present"]`)
}

func TestRestoreEvidenceJSONContract_UsesAcceptedManifestVersionField_RedSpec(t *testing.T) {
	evidence := BuildRestoreEvidence(Manifest{
		Version: ManifestVersionV1,
		Database: DatabaseDescriptor{
			Mode:          DBModeEmbedded,
			Driver:        DBDriverSQLite,
			SchemaVersion: "20260318_001",
			SourcePath:    ".local/db/main.db",
		},
		Artifact: ArtifactDescriptor{
			ChecksumSHA256: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		},
	}, RecordLinks{
		IncidentID: "inc-100",
		RecoveryID: "recovery-200",
	})

	payload, err := json.Marshal(evidence)
	require.NoError(t, err)
	text := string(payload)
	assert.Contains(t, text, `"accepted_manifest_version":"backup-manifest-v1"`)
	assert.Contains(t, text, `"record_links":{"incident_id":"inc-100","recovery_id":"recovery-200"}`)
}
