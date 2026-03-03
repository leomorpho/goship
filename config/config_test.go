package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetConfig(t *testing.T) {
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
	t.Setenv("PAGODA_APP_ENVIRONMENT", string(EnvProduction))

	cfg, err := GetConfig()
	require.NoError(t, err)
	assert.Equal(t, EnvProduction, cfg.App.Environment)
	assert.Equal(t, RuntimeProfileDistributed, cfg.Runtime.Profile)
	assert.True(t, cfg.Processes.Web)
	assert.False(t, cfg.Processes.Worker)
}

func TestGetConfig_UsesProcessProfile(t *testing.T) {
	t.Setenv("PAGODA_APP_ENVIRONMENT", string(EnvLocal))

	cfg, err := GetConfig()
	require.NoError(t, err)
	assert.Equal(t, RuntimeProfileServerDB, cfg.Runtime.Profile)
	assert.True(t, cfg.Processes.Web)
	assert.False(t, cfg.Processes.Worker)
	assert.False(t, cfg.Processes.Scheduler)
}
