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
	assert.Equal(t, "postgres", cfg.Adapters.DB)
	assert.Equal(t, "inproc", cfg.Adapters.Jobs)
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

func useIsolatedWorkingDir(t *testing.T) {
	t.Helper()

	root := t.TempDir()
	prevWD, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(prevWD) })
	require.NoError(t, os.Chdir(root))
}
