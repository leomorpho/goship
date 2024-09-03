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
}
