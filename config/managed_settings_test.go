package config

import (
	"encoding/json"
	"testing"

	"github.com/leomorpho/goship/config/runtimeconfig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestManagedSettingStatuses_StandaloneDefaultsToEditable(t *testing.T) {
	cfg := defaultConfig()

	statuses := cfg.ManagedSettingStatuses()
	require.NotEmpty(t, statuses)

	for _, status := range statuses {
		assert.Equal(t, SettingAccessEditable, status.Access)
	}

	cacheStatus := findManagedSettingStatus(t, statuses, "adapters.cache")
	assert.Equal(t, "Cache adapter", cacheStatus.Label)
	assert.Equal(t, "otter", cacheStatus.Value)
}

func TestManagedSettingStatuses_ManagedModeMapsReadOnlyAndExternallyManaged(t *testing.T) {
	cfg := defaultConfig()
	cfg.Managed.Enabled = true
	cfg.Managed.Authority = "control-plane"
	cfg.Adapters.Cache = "managed-cache"

	cfg.Managed.RuntimeReport = runtimeconfig.BuildReport(runtimeconfig.LayerInputs{
		Defaults:        managedKeyValues(defaultConfig()),
		EffectiveValues: managedKeyValues(cfg),
		RepoSet:         map[string]bool{},
		EnvSet:          map[string]bool{},
		ManagedSet: map[string]bool{
			"adapters.cache": true,
		},
		ManagedEnabled: true,
		Authority:      cfg.Managed.Authority,
	})

	statuses := cfg.ManagedSettingStatuses()

	cacheStatus := findManagedSettingStatus(t, statuses, "adapters.cache")
	assert.Equal(t, SettingAccessExternallyManaged, cacheStatus.Access)
	assert.Equal(t, runtimeconfig.SourceManagedOverride, cacheStatus.Source)
	assert.Equal(t, "managed-cache", cacheStatus.Value)

	jobsStatus := findManagedSettingStatus(t, statuses, "adapters.jobs")
	assert.Equal(t, SettingAccessReadOnly, jobsStatus.Access)
	assert.NotEqual(t, runtimeconfig.SourceManagedOverride, jobsStatus.Source)
}

func TestManagedSettingStatuses_DetectsDriftAndRollbackContract_RedSpec(t *testing.T) {
	cfg := defaultConfig()
	cfg.Managed.Enabled = true
	cfg.Managed.Authority = "control-plane"
	cfg.Adapters.Cache = "managed-cache"

	cfg.Managed.RuntimeReport = runtimeconfig.BuildReport(runtimeconfig.LayerInputs{
		Defaults:        managedKeyValues(defaultConfig()),
		EffectiveValues: managedKeyValues(cfg),
		RepoSet:         map[string]bool{},
		EnvSet:          map[string]bool{},
		ManagedSet: map[string]bool{
			"adapters.cache": true,
		},
		ManagedEnabled: true,
		Authority:      cfg.Managed.Authority,
	})

	cfg.Adapters.Cache = "rolled-back-cache"

	raw, err := json.Marshal(cfg.ManagedSettingStatuses())
	require.NoError(t, err)

	assert.Contains(t, string(raw), `"drift":true`)
	assert.Contains(t, string(raw), `"rollback_target":"framework-default"`)
}

func findManagedSettingStatus(t *testing.T, statuses []ManagedSettingStatus, key string) ManagedSettingStatus {
	t.Helper()
	for _, status := range statuses {
		if status.Key == key {
			return status
		}
	}
	t.Fatalf("managed setting status %q not found", key)
	return ManagedSettingStatus{}
}
