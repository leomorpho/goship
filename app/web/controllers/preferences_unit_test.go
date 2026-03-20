package controllers

import (
	"testing"

	"github.com/leomorpho/goship/config"
	"github.com/leomorpho/goship/framework/runtimeconfig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestManagedSettingsViewData_MapsManagedStatus(t *testing.T) {
	cfg := config.Config{}
	cfg.Managed.RuntimeReport = runtimeconfig.Report{
		Mode:      runtimeconfig.ModeManaged,
		Authority: "control-plane",
		Keys: map[string]runtimeconfig.KeyState{
			"adapters.cache": {
				Value:  "managed-cache",
				Source: runtimeconfig.SourceManagedOverride,
			},
		},
	}

	settings := managedSettingsViewData(&cfg)
	require.NotEmpty(t, settings)

	var cacheSettingFound bool
	for _, setting := range settings {
		if setting.Key != "adapters.cache" {
			continue
		}
		cacheSettingFound = true
		assert.Equal(t, "managed-cache", setting.Value)
		assert.Equal(t, string(runtimeconfig.SourceManagedOverride), setting.Source)
		assert.Equal(t, string(config.SettingAccessExternallyManaged), setting.Access)
	}
	assert.True(t, cacheSettingFound, "expected adapters.cache setting in mapped view data")
}
