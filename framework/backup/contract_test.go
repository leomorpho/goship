package backup

import (
	"testing"

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
