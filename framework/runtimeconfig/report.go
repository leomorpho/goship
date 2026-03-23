package runtimeconfig

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// Source identifies which config layer produced a managed key's effective value.
type Source string

const (
	SourceFrameworkDefault Source = "framework-default"
	SourceAppRepo          Source = "app-repo"
	SourceEnvironment      Source = "environment"
	SourceManagedOverride  Source = "managed-override"
)

// Mode identifies whether the runtime is self-managed or externally managed.
type Mode string

const (
	ModeStandalone Mode = "standalone"
	ModeManaged    Mode = "managed"
)

// KeyState reports the effective value and source for a managed key.
type KeyState struct {
	Value          string `json:"value"`
	Source         Source `json:"source"`
	RollbackTarget Source `json:"rollback_target,omitempty"`
}

// Report captures source-of-truth metadata for allowlisted managed keys.
type Report struct {
	Mode      Mode                `json:"mode"`
	Authority string              `json:"authority,omitempty"`
	Keys      map[string]KeyState `json:"keys"`
}

// ProcessDefaults holds fallback process values when report keys are absent.
type ProcessDefaults struct {
	Web       bool
	Worker    bool
	Scheduler bool
	CoLocated bool
}

// ProcessTopologyEntry reports one process flag with its source layer.
type ProcessTopologyEntry struct {
	Enabled bool   `json:"enabled"`
	Source  Source `json:"source"`
}

// ProcessTopology reports process topology with managed-source provenance.
type ProcessTopology struct {
	Web       ProcessTopologyEntry `json:"web"`
	Worker    ProcessTopologyEntry `json:"worker"`
	Scheduler ProcessTopologyEntry `json:"scheduler"`
	CoLocated ProcessTopologyEntry `json:"co_located"`
}

// LayerInputs contains the layer state needed to compute a managed config report.
type LayerInputs struct {
	EffectiveValues map[string]string
	Defaults        map[string]string
	RepoSet         map[string]bool
	EnvSet          map[string]bool
	ManagedSet      map[string]bool
	ManagedEnabled  bool
	Authority       string
}

// BuildReport computes per-key source precedence using the standard layer order.
func BuildReport(input LayerInputs) Report {
	report := Report{
		Mode: ModeStandalone,
		Keys: map[string]KeyState{},
	}
	if input.ManagedEnabled {
		report.Mode = ModeManaged
		report.Authority = strings.TrimSpace(input.Authority)
	}

	keys := keyUnion(input.Defaults, input.EffectiveValues)
	for _, key := range keys {
		value := strings.TrimSpace(input.Defaults[key])
		if v, ok := input.EffectiveValues[key]; ok {
			value = strings.TrimSpace(v)
		}

		source := SourceFrameworkDefault
		if input.RepoSet[key] {
			source = SourceAppRepo
		}
		if input.EnvSet[key] {
			source = SourceEnvironment
		}
		if input.ManagedEnabled && input.ManagedSet[key] {
			source = SourceManagedOverride
		}

		rollbackTarget := Source("")
		if input.ManagedEnabled && input.ManagedSet[key] {
			switch {
			case input.EnvSet[key]:
				rollbackTarget = SourceEnvironment
			case input.RepoSet[key]:
				rollbackTarget = SourceAppRepo
			default:
				rollbackTarget = SourceFrameworkDefault
			}
		}

		report.Keys[key] = KeyState{
			Value:          value,
			Source:         source,
			RollbackTarget: rollbackTarget,
		}
	}

	return report
}

// BuildProcessTopology maps process keys from a managed report into typed topology entries.
func BuildProcessTopology(report Report, defaults ProcessDefaults) ProcessTopology {
	return ProcessTopology{
		Web:       processTopologyEntry(report.Keys, "processes.web", defaults.Web),
		Worker:    processTopologyEntry(report.Keys, "processes.worker", defaults.Worker),
		Scheduler: processTopologyEntry(report.Keys, "processes.scheduler", defaults.Scheduler),
		CoLocated: processTopologyEntry(report.Keys, "processes.colocated", defaults.CoLocated),
	}
}

// ParseManagedOverrides parses a JSON object of managed overrides into normalized strings.
func ParseManagedOverrides(raw string) (map[string]string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return map[string]string{}, nil
	}

	decoder := json.NewDecoder(strings.NewReader(trimmed))
	decoder.UseNumber()

	var payload map[string]any
	if err := decoder.Decode(&payload); err != nil {
		return nil, fmt.Errorf("parse managed overrides: %w", err)
	}

	overrides := make(map[string]string, len(payload))
	for key, value := range payload {
		normalizedKey := strings.TrimSpace(key)
		if normalizedKey == "" {
			return nil, fmt.Errorf("managed override key cannot be empty")
		}

		normalizedValue, err := normalizeOverrideValue(value)
		if err != nil {
			return nil, fmt.Errorf("managed override %q: %w", normalizedKey, err)
		}
		overrides[normalizedKey] = normalizedValue
	}

	return overrides, nil
}

// RejectUnknownKeys returns sorted keys that are not in the provided allowlist.
func RejectUnknownKeys(overrides map[string]string, allowlist map[string]struct{}) []string {
	rejected := make([]string, 0)
	for key := range overrides {
		if _, ok := allowlist[key]; !ok {
			rejected = append(rejected, key)
		}
	}
	sort.Strings(rejected)
	return rejected
}

func normalizeOverrideValue(value any) (string, error) {
	switch v := value.(type) {
	case nil:
		return "", nil
	case string:
		return v, nil
	case bool:
		return strconv.FormatBool(v), nil
	case json.Number:
		return v.String(), nil
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64), nil
	case int:
		return strconv.Itoa(v), nil
	case int64:
		return strconv.FormatInt(v, 10), nil
	default:
		return "", fmt.Errorf("unsupported value type %T", value)
	}
}

func keyUnion(sets ...map[string]string) []string {
	seen := map[string]struct{}{}
	for _, set := range sets {
		for key := range set {
			seen[key] = struct{}{}
		}
	}

	keys := make([]string, 0, len(seen))
	for key := range seen {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func processTopologyEntry(keys map[string]KeyState, key string, fallback bool) ProcessTopologyEntry {
	entry := ProcessTopologyEntry{
		Enabled: fallback,
		Source:  SourceFrameworkDefault,
	}
	if keys == nil {
		return entry
	}

	state, ok := keys[key]
	if !ok {
		return entry
	}
	parsed, err := strconv.ParseBool(strings.TrimSpace(state.Value))
	if err == nil {
		entry.Enabled = parsed
	}
	if state.Source != "" {
		entry.Source = state.Source
	}
	return entry
}
