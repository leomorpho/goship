package controllers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/leomorpho/goship/app/foundation"
	webmiddleware "github.com/leomorpho/goship/app/web/middleware"
	"github.com/leomorpho/goship/app/web/ui"
	"github.com/leomorpho/goship/config"
	"github.com/leomorpho/goship/framework/backup"
	frameworksecurity "github.com/leomorpho/goship/framework/security"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestManagedHooksRejectUnsignedRequest(t *testing.T) {
	e := newManagedHooksTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/managed/status", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestManagedHooksStatusSignedAndReplayProtected(t *testing.T) {
	e := newManagedHooksTestServer(t)

	ts := time.Now().UTC().Unix()
	nonce := "managed-status-nonce"

	req1 := httptest.NewRequest(http.MethodGet, "/managed/status", nil)
	signManagedRequest(req1, "secret", ts, nonce, []byte{})
	rec1 := httptest.NewRecorder()
	e.ServeHTTP(rec1, req1)
	assert.Equal(t, http.StatusOK, rec1.Code)

	req2 := httptest.NewRequest(http.MethodGet, "/managed/status", nil)
	signManagedRequest(req2, "secret", ts, nonce, []byte{})
	rec2 := httptest.NewRecorder()
	e.ServeHTTP(rec2, req2)
	assert.Equal(t, http.StatusConflict, rec2.Code)
}

func TestManagedHooksBackupAndRestoreSigned(t *testing.T) {
	backupDriver := &fakeBackupDriver{
		manifest: backup.Manifest{
			Version:   backup.ManifestVersionV1,
			CreatedAt: time.Now().UTC(),
			Database: backup.DatabaseDescriptor{
				Mode:          backup.DBModeEmbedded,
				Driver:        backup.DBDriverSQLite,
				SchemaVersion: "v1",
				SourcePath:    ".local/db/main.db",
			},
			Artifact: backup.ArtifactDescriptor{
				ChecksumSHA256: "abc123",
				SizeBytes:      10,
			},
			Storage: backup.StorageLocation{
				Provider: backup.ProviderLocal,
				URI:      "file://.local/db/main.db",
			},
		},
	}
	restoreDriver := &fakeRestoreDriver{}
	e := newManagedHooksTestServerWithDrivers(t, backupDriver, restoreDriver)

	backupBody := []byte(`{"object_key":"snapshots/latest.db"}`)
	backupReq := httptest.NewRequest(http.MethodPost, "/managed/backup", bytes.NewReader(backupBody))
	backupReq.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	signManagedRequest(backupReq, "secret", time.Now().UTC().Unix(), "backup-nonce", backupBody)
	backupRec := httptest.NewRecorder()
	e.ServeHTTP(backupRec, backupReq)
	assert.Equal(t, http.StatusAccepted, backupRec.Code)
	assert.True(t, backupDriver.called)

	restoreBody, err := json.Marshal(map[string]any{"manifest": backupDriver.manifest})
	require.NoError(t, err)
	restoreReq := httptest.NewRequest(http.MethodPost, "/managed/restore", bytes.NewReader(restoreBody))
	restoreReq.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	signManagedRequest(restoreReq, "secret", time.Now().UTC().Unix(), "restore-nonce", restoreBody)
	restoreRec := httptest.NewRecorder()
	e.ServeHTTP(restoreRec, restoreReq)
	assert.Equal(t, http.StatusAccepted, restoreRec.Code)
	assert.True(t, restoreDriver.called)
}

func newManagedHooksTestServer(t *testing.T) *echo.Echo {
	t.Helper()
	return newManagedHooksTestServerWithDrivers(t, &fakeBackupDriver{
		manifest: backup.Manifest{
			Version:   backup.ManifestVersionV1,
			CreatedAt: time.Now().UTC(),
			Database: backup.DatabaseDescriptor{
				Mode:          backup.DBModeEmbedded,
				Driver:        backup.DBDriverSQLite,
				SchemaVersion: "v1",
				SourcePath:    ".local/db/main.db",
			},
			Artifact: backup.ArtifactDescriptor{
				ChecksumSHA256: "abc123",
				SizeBytes:      10,
			},
			Storage: backup.StorageLocation{
				Provider: backup.ProviderLocal,
				URI:      "file://.local/db/main.db",
			},
		},
	}, &fakeRestoreDriver{})
}

func newManagedHooksTestServerWithDrivers(t *testing.T, backupDriver *fakeBackupDriver, restoreDriver *fakeRestoreDriver) *echo.Echo {
	t.Helper()

	cfg := config.Config{}
	cfg.Managed.Enabled = true
	cfg.Managed.HooksSecret = "secret"
	cfg.Managed.HooksMaxSkewSeconds = 300
	cfg.Managed.HooksNonceTTLSeconds = 300
	cfg.Backup.SchemaVersion = "v1"
	cfg.Backup.SQLitePath = ".local/db/main.db"
	cfg.Database.Path = ".local/db/main.db"

	ctr := ui.NewController(&foundation.Container{
		Config: &cfg,
	})
	route := NewManagedHooksRoute(ctr, ManagedHooksDeps{
		BackupDriver:  backupDriver,
		RestoreDriver: restoreDriver,
		Now:           time.Now,
	})

	verifier := frameworksecurity.NewManagedHookVerifier(
		cfg.Managed.HooksSecret,
		time.Duration(cfg.Managed.HooksMaxSkewSeconds)*time.Second,
		time.Duration(cfg.Managed.HooksNonceTTLSeconds)*time.Second,
	)

	e := echo.New()
	g := e.Group("/managed", webmiddleware.RequireManagedHookSignature(verifier))
	g.GET("/status", route.GetRuntimeStatus)
	g.POST("/backup", route.StartBackup)
	g.POST("/restore", route.StartRestore)
	return e
}

func signManagedRequest(req *http.Request, secret string, timestamp int64, nonce string, body []byte) {
	path := req.URL.Path
	if req.URL.RawQuery != "" {
		path += "?" + req.URL.RawQuery
	}
	signature := frameworksecurity.SignManagedHookRequest(secret, req.Method, path, timestamp, nonce, body)
	req.Header.Set(frameworksecurity.HeaderManagedTimestamp, strconv.FormatInt(timestamp, 10))
	req.Header.Set(frameworksecurity.HeaderManagedNonce, nonce)
	req.Header.Set(frameworksecurity.HeaderManagedSignature, signature)
}

type fakeBackupDriver struct {
	called   bool
	manifest backup.Manifest
	err      error
}

func (f *fakeBackupDriver) Create(_ context.Context, _ backup.CreateRequest) (backup.Manifest, error) {
	f.called = true
	if f.err != nil {
		return backup.Manifest{}, f.err
	}
	return f.manifest, nil
}

type fakeRestoreDriver struct {
	called bool
	err    error
}

func (f *fakeRestoreDriver) Restore(_ context.Context, _ backup.RestoreRequest) error {
	f.called = true
	return f.err
}
