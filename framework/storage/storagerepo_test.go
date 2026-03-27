package storagerepo

import (
	"database/sql"
	"strings"
	"testing"
	"time"

	"github.com/leomorpho/goship/config"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func TestParseBucket(t *testing.T) {
	t.Run("main app aliases", func(t *testing.T) {
		for _, raw := range []string{"main-app", "main", "app"} {
			b, err := ParseBucket(raw)
			require.NoError(t, err)
			assert.Equal(t, BucketMainApp, b)
		}
	})

	t.Run("static aliases", func(t *testing.T) {
		for _, raw := range []string{"static-files", "static"} {
			b, err := ParseBucket(raw)
			require.NoError(t, err)
			assert.Equal(t, BucketStaticFiles, b)
		}
	})

	t.Run("invalid", func(t *testing.T) {
		_, err := ParseBucket("unknown")
		require.ErrorIs(t, err, ErrBucketDoesNotExist)
	})
}

func TestStorageClient_Local_TestEnv(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	// Create table
	_, err = db.Exec(`
		CREATE TABLE file_storages (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			bucket_name TEXT NOT NULL,
			object_key TEXT NOT NULL,
			file_size INTEGER NOT NULL,
			file_hash TEXT NOT NULL
		)
	`)
	require.NoError(t, err)

	cfg := &config.Config{
		App: config.AppConfig{
			Environment: config.EnvTest,
		},
		Storage: config.StorageConfig{
			Driver:                config.StorageDriverLocal,
			LocalStoragePath:      "./test_uploads",
			AppBucketName:         "test-bucket",
			StaticFilesBucketName: "test-static",
		},
	}

	sc := NewStorageClient(cfg, db, "sqlite")
	require.NotNil(t, sc.fs)
	// In EnvTest, it should be MemMapFs
	_, ok := sc.fs.(*afero.MemMapFs)
	assert.True(t, ok, "should use MemMapFs in test environment")

	// Test CreateBucket
	err = sc.CreateBucket("mybucket", "us-east-1")
	require.NoError(t, err)

	bucketName := "mybucket" + string(config.EnvTest)
	exists, err := afero.DirExists(sc.fs, bucketName)
	require.NoError(t, err)
	assert.True(t, exists)

	// Test UploadFile
	content := "hello world"
	reader := strings.NewReader(content)
	objectName := "test.txt"
	fileID, err := sc.UploadFile(BucketMainApp, objectName, reader)
	require.NoError(t, err)
	assert.NotNil(t, fileID)

	// Verify file exists in MemMapFs
	appBucket, _ := sc.getBucketName(BucketMainApp)
	exists, err = afero.Exists(sc.fs, appBucket+"/"+objectName)
	require.NoError(t, err)
	assert.True(t, exists)

	// Verify file content
	data, err := afero.ReadFile(sc.fs, appBucket+"/"+objectName)
	require.NoError(t, err)
	assert.Equal(t, content, string(data))

	// Test GetPresignedURL (should return local path)
	url, err := sc.GetPresignedURL(BucketMainApp, objectName, 1*time.Hour)
	require.NoError(t, err)
	assert.Equal(t, "/uploads/"+appBucket+"/"+objectName, url)
	assert.Equal(t, "main-app", BucketMainApp.String())

	// Test DeleteFile
	err = sc.DeleteFile(BucketMainApp, objectName)
	require.NoError(t, err)

	exists, err = afero.Exists(sc.fs, appBucket+"/"+objectName)
	require.NoError(t, err)
	assert.False(t, exists)

	// Verify DB entry is gone
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM file_storages WHERE object_key = ?", objectName).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}
