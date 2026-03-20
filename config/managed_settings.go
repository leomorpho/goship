package config

import (
	"sort"
	"strings"

	"github.com/leomorpho/goship/framework/runtimeconfig"
)

// SettingAccess defines whether a setting is locally editable, locked, or controlled externally.
type SettingAccess string

const (
	SettingAccessEditable          SettingAccess = "editable"
	SettingAccessReadOnly          SettingAccess = "read-only"
	SettingAccessExternallyManaged SettingAccess = "externally-managed"
)

// ManagedSettingStatus reports effective state for one managed-capable setting key.
type ManagedSettingStatus struct {
	Key    string
	Label  string
	Value  string
	Source runtimeconfig.Source
	Access SettingAccess
}

var managedSettingLabels = map[string]string{
	"runtime.profile":     "Runtime profile",
	"processes.web":       "Web process enabled",
	"processes.worker":    "Worker process enabled",
	"processes.scheduler": "Scheduler process enabled",
	"processes.colocated": "Processes co-located",
	"adapters.db":         "Database adapter",
	"adapters.cache":      "Cache adapter",
	"adapters.jobs":       "Jobs adapter",
	"adapters.pubsub":     "PubSub adapter",
	"database.driver":     "Database driver",
	"database.path":       "Database path",
	"storage.driver":      "Storage driver",
}

// ManagedSettingStatuses returns the allowlisted managed settings with explicit access state.
func (c Config) ManagedSettingStatuses() []ManagedSettingStatus {
	report := c.Managed.RuntimeReport
	if report.Mode == "" {
		report = runtimeconfig.BuildReport(runtimeconfig.LayerInputs{
			Defaults:        managedKeyValues(defaultConfig()),
			EffectiveValues: managedKeyValues(c),
			RepoSet:         map[string]bool{},
			EnvSet:          map[string]bool{},
			ManagedSet:      map[string]bool{},
			ManagedEnabled:  c.Managed.Enabled,
			Authority:       c.Managed.Authority,
		})
	}

	keys := make([]string, 0, len(managedOverrideSpecs))
	for key := range managedOverrideSpecs {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	effective := managedKeyValues(c)
	statuses := make([]ManagedSettingStatus, 0, len(keys))
	for _, key := range keys {
		keyState, ok := report.Keys[key]
		if !ok {
			keyState = runtimeconfig.KeyState{
				Value:  strings.TrimSpace(effective[key]),
				Source: runtimeconfig.SourceFrameworkDefault,
			}
		}

		statuses = append(statuses, ManagedSettingStatus{
			Key:    key,
			Label:  managedSettingLabel(key),
			Value:  strings.TrimSpace(keyState.Value),
			Source: keyState.Source,
			Access: managedSettingAccess(report.Mode, keyState.Source),
		})
	}

	return statuses
}

func managedSettingAccess(mode runtimeconfig.Mode, source runtimeconfig.Source) SettingAccess {
	if mode != runtimeconfig.ModeManaged {
		return SettingAccessEditable
	}
	if source == runtimeconfig.SourceManagedOverride {
		return SettingAccessExternallyManaged
	}
	return SettingAccessReadOnly
}

func managedSettingLabel(key string) string {
	if label, ok := managedSettingLabels[key]; ok {
		return label
	}
	return key
}
