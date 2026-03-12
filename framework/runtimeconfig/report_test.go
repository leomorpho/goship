package runtimeconfig

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildReportPrecedence(t *testing.T) {
	report := BuildReport(LayerInputs{
		Defaults: map[string]string{
			"adapters.cache": "otter",
			"adapters.jobs":  "backlite",
		},
		EffectiveValues: map[string]string{
			"adapters.cache": "redis",
			"adapters.jobs":  "asynq",
		},
		RepoSet: map[string]bool{
			"adapters.cache": true,
		},
		EnvSet: map[string]bool{
			"adapters.cache": true,
		},
		ManagedSet: map[string]bool{
			"adapters.jobs": true,
		},
		ManagedEnabled: true,
		Authority:      "control-plane",
	})

	assert.Equal(t, ModeManaged, report.Mode)
	assert.Equal(t, "control-plane", report.Authority)
	assert.Equal(t, KeyState{Value: "redis", Source: SourceEnvironment}, report.Keys["adapters.cache"])
	assert.Equal(t, KeyState{Value: "asynq", Source: SourceManagedOverride}, report.Keys["adapters.jobs"])
}

func TestBuildReportStandaloneIgnoresManagedSet(t *testing.T) {
	report := BuildReport(LayerInputs{
		Defaults: map[string]string{
			"database.driver": "sqlite",
		},
		EffectiveValues: map[string]string{
			"database.driver": "postgres",
		},
		ManagedSet: map[string]bool{
			"database.driver": true,
		},
		ManagedEnabled: false,
	})

	assert.Equal(t, ModeStandalone, report.Mode)
	assert.Empty(t, report.Authority)
	assert.Equal(t, KeyState{Value: "postgres", Source: SourceFrameworkDefault}, report.Keys["database.driver"])
}

func TestParseManagedOverrides(t *testing.T) {
	overrides, err := ParseManagedOverrides(`{"processes.worker":true,"processes.scheduler":false,"database.path":"dbs/managed.db","limits.page_size":25}`)
	require.NoError(t, err)
	assert.Equal(t, "true", overrides["processes.worker"])
	assert.Equal(t, "false", overrides["processes.scheduler"])
	assert.Equal(t, "dbs/managed.db", overrides["database.path"])
	assert.Equal(t, "25", overrides["limits.page_size"])
}

func TestParseManagedOverridesInvalidJSON(t *testing.T) {
	_, err := ParseManagedOverrides(`{"bad":`)
	require.Error(t, err)
}

func TestRejectUnknownKeys(t *testing.T) {
	rejected := RejectUnknownKeys(map[string]string{
		"adapters.cache": "redis",
		"invalid.key":    "x",
	}, map[string]struct{}{
		"adapters.cache": {},
	})

	assert.Equal(t, []string{"invalid.key"}, rejected)
}
