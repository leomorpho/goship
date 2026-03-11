package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetConfig(t *testing.T) {
	useIsolatedWorkingDir(t)
	t.Setenv("PAGODA_APP_ENVIRONMENT", "")

	_, err := GetConfig()
	require.NoError(t, err)

	cfg, err := GetConfig()
	require.NoError(t, err)
	assert.Equal(t, EnvLocal, cfg.App.Environment)
	assert.Equal(t, RuntimeProfileServerDB, cfg.Runtime.Profile)
	assert.True(t, cfg.Processes.Web)
	assert.Equal(t, "sqlite", cfg.Adapters.DB)
	assert.Equal(t, "otter", cfg.Adapters.Cache)
	assert.Equal(t, "backlite", cfg.Adapters.Jobs)
	assert.Equal(t, DBDriverSQLite, cfg.Database.Driver)
}

func TestGetConfig_EnvironmentOverrides(t *testing.T) {
	useIsolatedWorkingDir(t)
	t.Setenv("PAGODA_APP_ENVIRONMENT", string(EnvProduction))

	cfg, err := GetConfig()
	require.NoError(t, err)
	assert.Equal(t, EnvProduction, cfg.App.Environment)
	assert.Equal(t, RuntimeProfileDistributed, cfg.Runtime.Profile)
	assert.True(t, cfg.Processes.Web)
	assert.False(t, cfg.Processes.Worker)
}

func TestGetConfig_UsesProcessProfile(t *testing.T) {
	useIsolatedWorkingDir(t)
	t.Setenv("PAGODA_APP_ENVIRONMENT", string(EnvLocal))

	cfg, err := GetConfig()
	require.NoError(t, err)
	assert.Equal(t, RuntimeProfileServerDB, cfg.Runtime.Profile)
	assert.True(t, cfg.Processes.Web)
	assert.False(t, cfg.Processes.Worker)
	assert.False(t, cfg.Processes.Scheduler)
}

func TestGetConfig_LoadsDotEnv(t *testing.T) {
	root := t.TempDir()
	prevWD, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(prevWD) })
	require.NoError(t, os.Chdir(root))

	dotEnv := strings.Join([]string{
		"PAGODA_APP_NAME=Env Loaded App",
		"PAGODA_DATABASE_DBMODE=standalone",
		"PAGODA_DATABASE_HOSTNAME=db.internal",
		"PAGODA_DATABASE_PORT=5433",
		"",
	}, "\n")
	require.NoError(t, os.WriteFile(filepath.Join(root, ".env"), []byte(dotEnv), 0o644))

	t.Setenv("PAGODA_APP_NAME", "")
	t.Setenv("PAGODA_DATABASE_DBMODE", "")
	t.Setenv("PAGODA_DATABASE_HOSTNAME", "")
	t.Setenv("PAGODA_DATABASE_PORT", "")

	cfg, err := GetConfig()
	require.NoError(t, err)
	assert.Equal(t, app("Env Loaded App"), cfg.App.Name)
	assert.Equal(t, DBModeStandalone, cfg.Database.DbMode)
	assert.Equal(t, "db.internal", cfg.Database.Hostname)
	assert.Equal(t, uint16(5433), cfg.Database.Port)
}

func TestGetConfig_DBDriverOverridesLegacyEmbeddedSettings(t *testing.T) {
	useIsolatedWorkingDir(t)
	t.Setenv("PAGODA_DB_DRIVER", "sqlite")
	t.Setenv("PAGODA_DB_PATH", "dbs/app.db")
	t.Setenv("PAGODA_DATABASE_DBMODE", string(DBModeStandalone))
	t.Setenv("PAGODA_DATABASE_EMBEDDEDDRIVER", "sqlite3")

	cfg, err := GetConfig()
	require.NoError(t, err)
	assert.Equal(t, DBDriverSQLite, cfg.Database.Driver)
	assert.Equal(t, DBModeEmbedded, cfg.Database.DbMode)
	assert.Equal(t, "sqlite", cfg.Database.EmbeddedDriver)
	assert.Equal(t, "dbs/app.db?_journal=WAL&_timeout=5000&_fk=true", cfg.Database.EmbeddedConnection)
}

func TestGetConfig_JobsDriverAliasOverridesDefault(t *testing.T) {
	useIsolatedWorkingDir(t)
	t.Setenv("PAGODA_JOBS_DRIVER", "asynq")
	t.Setenv("PAGODA_ADAPTERS_JOBS", "")

	cfg, err := GetConfig()
	require.NoError(t, err)
	assert.Equal(t, "asynq", cfg.Adapters.Jobs)
}

func TestGetConfig_CacheDriverAliasOverridesDefault(t *testing.T) {
	useIsolatedWorkingDir(t)
	t.Setenv("PAGODA_CACHE_DRIVER", "redis")
	t.Setenv("PAGODA_ADAPTERS_CACHE", "")

	cfg, err := GetConfig()
	require.NoError(t, err)
	assert.Equal(t, "redis", cfg.Adapters.Cache)
}

func TestGetConfig_StandaloneModeDefaultsToPostgresDriver(t *testing.T) {
	useIsolatedWorkingDir(t)
	t.Setenv("PAGODA_DB_DRIVER", "")
	t.Setenv("PAGODA_DATABASE_DBMODE", string(DBModeStandalone))

	cfg, err := GetConfig()
	require.NoError(t, err)
	assert.Equal(t, DBDriverPostgres, cfg.Database.Driver)
	assert.Equal(t, DBModeStandalone, cfg.Database.DbMode)
}

func useIsolatedWorkingDir(t *testing.T) {
	t.Helper()

	root := t.TempDir()
	prevWD, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(prevWD) })
	require.NoError(t, os.Chdir(root))
}
