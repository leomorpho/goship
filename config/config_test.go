package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetConfig(t *testing.T) {
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
