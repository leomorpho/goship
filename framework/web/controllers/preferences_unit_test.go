package controllers

import (
	"bytes"
	"context"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/leomorpho/goship/config"
	"github.com/leomorpho/goship/framework/runtimeconfig"
	"github.com/leomorpho/goship/framework/tests"
	"github.com/leomorpho/goship/framework/web/pages/gen"
	"github.com/leomorpho/goship/framework/web/ui"
	viewmodels "github.com/leomorpho/goship/framework/web/viewmodels"
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

	settings := viewmodels.ManagedSettingsFromConfig(&cfg)
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

func TestManagedSettingsPreferencesPage_RendersAccessAndSourceStates(t *testing.T) {
	cfg := config.Config{}
	cfg.Managed.Enabled = true
	cfg.Managed.RuntimeReport = runtimeconfig.Report{
		Mode:      runtimeconfig.ModeManaged,
		Authority: "control-plane",
		Keys: map[string]runtimeconfig.KeyState{
			"adapters.cache": {
				Value:  "managed-cache",
				Source: runtimeconfig.SourceManagedOverride,
			},
			"adapters.jobs": {
				Value:  "backlite",
				Source: runtimeconfig.SourceEnvironment,
			},
		},
	}

	pageData := viewmodels.NewPreferencesData()
	pageData.ManagedMode = true
	pageData.ManagedAuthority = cfg.Managed.RuntimeReport.Authority
	pageData.ManagedSettings = viewmodels.ManagedSettingsFromConfig(&cfg)

	ctx, _ := tests.NewContext(echo.New(), "/auth/admin/managed-settings")
	page := ui.NewPage(ctx)
	page.ToURL = func(name string, _ ...any) string {
		return "/" + name
	}
	page.Data = pageData

	var rendered bytes.Buffer
	require.NoError(t, pages.Settings(&page).Render(context.Background(), &rendered))

	out := rendered.String()
	assert.Contains(t, out, "Managed mode is enabled.")
	assert.Contains(t, out, "Managed keys are locked locally.")
	assert.Contains(t, out, "externally managed")
	assert.Contains(t, out, "read only")
	assert.Contains(t, out, string(runtimeconfig.SourceManagedOverride))
	assert.Contains(t, out, string(runtimeconfig.SourceEnvironment))
}
