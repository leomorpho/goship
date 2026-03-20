package controllers

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/leomorpho/goship/app/web/ui"
	"github.com/leomorpho/goship/config"
	"github.com/leomorpho/goship/framework/backup"
	"github.com/leomorpho/goship/framework/core"
)

type managedHooks struct {
	ctr           ui.Controller
	backupDriver  core.BackupDriver
	restoreDriver core.RestoreDriver
	now           func() time.Time
}

type ManagedHooksDeps struct {
	BackupDriver  core.BackupDriver
	RestoreDriver core.RestoreDriver
	Now           func() time.Time
}

type managedBackupRequest struct {
	ObjectKey string `json:"object_key"`
}

type managedRestoreRequest struct {
	Manifest backup.Manifest        `json:"manifest"`
	Linkage  backup.RecoveryLinkage `json:"linkage"`
}

func NewManagedHooksRoute(ctr ui.Controller, deps ManagedHooksDeps) managedHooks {
	nowFn := deps.Now
	if nowFn == nil {
		nowFn = time.Now
	}
	return managedHooks{
		ctr:           ctr,
		backupDriver:  deps.BackupDriver,
		restoreDriver: deps.RestoreDriver,
		now:           nowFn,
	}
}

func (m managedHooks) GetRuntimeStatus(ctx echo.Context) error {
	if err := m.requireManagedMode(); err != nil {
		return err
	}

	cfg := m.ctr.Container.Config
	return ctx.JSON(http.StatusOK, map[string]any{
		"status":            "ok",
		"timestamp":         m.now().UTC(),
		"runtime_metadata":  cfg.RuntimeMetadata(),
		"managed_settings":  cfg.ManagedSettingStatuses(),
		"managed_authority": cfg.Managed.RuntimeReport.Authority,
	})
}

func (m managedHooks) StartBackup(ctx echo.Context) error {
	if err := m.requireManagedMode(); err != nil {
		return err
	}
	if m.backupDriver == nil {
		return echo.NewHTTPError(http.StatusServiceUnavailable, "backup driver is not configured")
	}

	req := managedBackupRequest{}
	if ctx.Request().ContentLength > 0 {
		if err := ctx.Bind(&req); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid backup request payload")
		}
	}

	cfg := m.ctr.Container.Config
	sqlitePath := strings.TrimSpace(cfg.Backup.SQLitePath)
	if sqlitePath == "" {
		sqlitePath = strings.TrimSpace(cfg.Database.Path)
	}

	storage, err := managedBackupStorage(cfg, req.ObjectKey, m.now().UTC())
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	manifest, err := m.backupDriver.Create(ctx.Request().Context(), backup.CreateRequest{
		SQLitePath:    sqlitePath,
		SchemaVersion: cfg.Backup.SchemaVersion,
		Storage:       storage,
		CreatedAt:     m.now().UTC(),
	})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return ctx.JSON(http.StatusAccepted, map[string]any{
		"status":   "accepted",
		"manifest": manifest,
	})
}

func (m managedHooks) StartRestore(ctx echo.Context) error {
	if err := m.requireManagedMode(); err != nil {
		return err
	}
	if m.restoreDriver == nil {
		return echo.NewHTTPError(http.StatusServiceUnavailable, "restore driver is not configured")
	}

	req := managedRestoreRequest{}
	if err := ctx.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid restore request payload")
	}

	if err := m.restoreDriver.Restore(ctx.Request().Context(), backup.RestoreRequest{
		Manifest: req.Manifest,
		Linkage:  req.Linkage,
	}); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	return ctx.JSON(http.StatusAccepted, map[string]any{
		"status":           "accepted",
		"restore_evidence": backup.BuildRestoreEvidence(req.Manifest, req.Linkage),
	})
}

func (m managedHooks) requireManagedMode() error {
	if m.ctr.Container == nil || m.ctr.Container.Config == nil || !m.ctr.Container.Config.Managed.Enabled {
		return echo.NewHTTPError(http.StatusNotFound, "managed hooks are not enabled")
	}
	return nil
}

func managedBackupStorage(cfg *config.Config, objectKey string, now time.Time) (backup.StorageLocation, error) {
	if cfg == nil {
		return backup.StorageLocation{}, fmt.Errorf("config is not initialized")
	}

	if cfg.Backup.S3.Enabled {
		key := strings.TrimSpace(objectKey)
		if key == "" {
			key = "managed/" + now.UTC().Format("20060102T150405Z") + ".db"
		}
		return backup.S3ProviderConfig{
			Endpoint:  cfg.Backup.S3.Endpoint,
			Region:    cfg.Backup.S3.Region,
			Bucket:    cfg.Backup.S3.Bucket,
			Prefix:    cfg.Backup.S3.Prefix,
			AccessKey: cfg.Backup.S3.AccessKey,
			SecretKey: cfg.Backup.S3.SecretKey,
			UseSSL:    cfg.Backup.S3.UseSSL,
		}.StorageLocation(key)
	}

	uri := strings.TrimSpace(cfg.Backup.SQLitePath)
	if uri == "" {
		uri = strings.TrimSpace(cfg.Database.Path)
	}
	if uri == "" {
		return backup.StorageLocation{}, fmt.Errorf("local backup URI is empty")
	}
	return backup.StorageLocation{
		Provider: backup.ProviderLocal,
		URI:      "file://" + uri,
	}, nil
}
