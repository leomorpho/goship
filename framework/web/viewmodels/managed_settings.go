package viewmodels

import (
	"github.com/leomorpho/goship/config"
)

func ManagedSettingsFromConfig(cfg *config.Config) []ManagedSettingControl {
	if cfg == nil {
		return []ManagedSettingControl{}
	}

	statuses := cfg.ManagedSettingStatuses()
	controls := make([]ManagedSettingControl, 0, len(statuses))
	for _, status := range statuses {
		control := NewManagedSettingControl()
		control.Key = status.Key
		control.Label = status.Label
		control.Value = status.Value
		control.Source = string(status.Source)
		control.Access = string(status.Access)
		controls = append(controls, control)
	}
	return controls
}
