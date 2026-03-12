package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/leomorpho/goship/framework/runtimeconfig"
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
	assert.True(t, cfg.Security.Headers.Enabled)
	assert.False(t, cfg.Security.Headers.HSTS)
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

func TestGetConfig_APPENVCompatibilityAlias(t *testing.T) {
	useIsolatedWorkingDir(t)
	t.Setenv("PAGODA_APP_ENVIRONMENT", "")
	t.Setenv("APP_ENV", "production")

	cfg, err := GetConfig()
	require.NoError(t, err)
	assert.Equal(t, EnvProduction, cfg.App.Environment)
}

func TestGetConfig_EnvironmentNormalization(t *testing.T) {
	useIsolatedWorkingDir(t)
	t.Setenv("PAGODA_APP_ENVIRONMENT", "development")

	cfg, err := GetConfig()
	require.NoError(t, err)
	assert.Equal(t, EnvDevelop, cfg.App.Environment)
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

func TestGetConfig_SecurityHeaderOverrides(t *testing.T) {
	useIsolatedWorkingDir(t)
	t.Setenv("PAGODA_SECURITY_HEADERS_ENABLED", "false")
	t.Setenv("PAGODA_SECURITY_HEADERS_HSTS", "true")
	t.Setenv("PAGODA_SECURITY_HEADERS_CSP", "default-src 'self'")

	cfg, err := GetConfig()
	require.NoError(t, err)
	assert.False(t, cfg.Security.Headers.Enabled)
	assert.True(t, cfg.Security.Headers.HSTS)
	assert.Equal(t, "default-src 'self'", cfg.Security.Headers.CSP)
}

func TestGetConfig_ManagedOverridePrecedenceAndReporting(t *testing.T) {
	useIsolatedWorkingDir(t)
	require.NoError(t, os.WriteFile(".env", []byte("PAGODA_ADAPTERS_CACHE=repo-cache\n"), 0o644))

	t.Setenv("PAGODA_ADAPTERS_CACHE", "env-cache")
	t.Setenv("PAGODA_MANAGED_MODE", "true")
	t.Setenv("PAGODA_MANAGED_AUTHORITY", "control-plane")
	t.Setenv("PAGODA_MANAGED_OVERRIDES", `{"adapters.cache":"managed-cache"}`)

	cfg, err := GetConfig()
	require.NoError(t, err)
	assert.Equal(t, "managed-cache", cfg.Adapters.Cache)
	assert.Equal(t, runtimeconfig.ModeManaged, cfg.Managed.RuntimeReport.Mode)
	assert.Equal(t, "control-plane", cfg.Managed.RuntimeReport.Authority)
	assert.Equal(t, runtimeconfig.SourceManagedOverride, cfg.Managed.RuntimeReport.Keys["adapters.cache"].Source)
	assert.Equal(t, "managed-cache", cfg.Managed.RuntimeReport.Keys["adapters.cache"].Value)
}

func TestGetConfig_ManagedOverridesRequireAllowlistedKeys(t *testing.T) {
	useIsolatedWorkingDir(t)
	t.Setenv("PAGODA_MANAGED_MODE", "true")
	t.Setenv("PAGODA_MANAGED_OVERRIDES", `{"not.allowed":"value"}`)

	_, err := GetConfig()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "non-allowlisted")
	assert.Contains(t, err.Error(), "not.allowed")
}

func TestGetConfig_ManagedRuntimeProfileAppliesWhenProcessesUnset(t *testing.T) {
	useIsolatedWorkingDir(t)
	t.Setenv("PAGODA_MANAGED_MODE", "true")
	t.Setenv("PAGODA_MANAGED_OVERRIDES", `{"runtime.profile":"single-node"}`)

	cfg, err := GetConfig()
	require.NoError(t, err)
	assert.Equal(t, RuntimeProfileSingleNode, cfg.Runtime.Profile)
	assert.True(t, cfg.Processes.Web)
	assert.True(t, cfg.Processes.Worker)
	assert.True(t, cfg.Processes.Scheduler)
	assert.True(t, cfg.Processes.CoLocated)
	assert.Equal(t, runtimeconfig.SourceManagedOverride, cfg.Managed.RuntimeReport.Keys["runtime.profile"].Source)
}

func useIsolatedWorkingDir(t *testing.T) {
	t.Helper()

	root := t.TempDir()
	prevWD, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(prevWD) })
	require.NoError(t, os.Chdir(root))
}
