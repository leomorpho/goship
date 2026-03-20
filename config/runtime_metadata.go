package config

import (
	"strings"

	"github.com/leomorpho/goship/framework/runtimeconfig"
)

const (
	// DefaultSchemaMigrationsTable is the canonical migration tracking table for core SQL migrations.
	DefaultSchemaMigrationsTable = "goship_schema_migrations"
	// MigrationPortabilitySQLCoreV1 is the first portability profile for SQLite-first promotion.
	MigrationPortabilitySQLCoreV1 = "sql-core-v1"
	// PromotionPathSQLiteToPostgresManualV1 identifies the first supported promotion workflow.
	PromotionPathSQLiteToPostgresManualV1 = "sqlite-to-postgres-manual-v1"
	// ManagedKeyRegistryVersion is the versioned contract identifier for managed-key metadata.
	ManagedKeyRegistryVersion = "managed-key-registry-v1"
	// ManagedKeySchemaVersion is the schema version for the managed-key registry payload.
	ManagedKeySchemaVersion = "managed-key-schema-v1"
	// ManagedDivergenceSchemaVersion is the schema version for managed divergence metadata.
	ManagedDivergenceSchemaVersion = "managed-divergence-v1"
)

const (
	ManagedDivergenceClassificationDrift                 = "drift"
	ManagedDivergenceActionRollback                      = "rollback"
	ManagedDivergenceActionUpstreamModuleCandidateReview = "upstream-module-candidate-review"
)

// RuntimeMetadata reports normalized runtime capability metadata for status/reporting surfaces.
type RuntimeMetadata struct {
	Database DatabaseRuntimeMetadata `json:"database"`
	Managed  ManagedRuntimeMetadata  `json:"managed"`
}

// ManagedRuntimeMetadata reports effective managed keys and their source layers.
type ManagedRuntimeMetadata struct {
	Mode            string                        `json:"mode"`
	Authority       string                        `json:"authority,omitempty"`
	RegistryVersion string                        `json:"registry_version"`
	SchemaVersion   string                        `json:"schema_version"`
	Divergence      ManagedDivergenceMetadata     `json:"divergence"`
	Keys            map[string]ManagedKeyMetadata `json:"keys"`
}

// ManagedKeyMetadata reports the effective value and source for one managed key.
type ManagedKeyMetadata struct {
	Value  string `json:"value"`
	Source string `json:"source"`
}

// ManagedDivergenceMetadata reports machine-readable divergence items for managed keys.
type ManagedDivergenceMetadata struct {
	SchemaVersion string                  `json:"schema_version"`
	Items         []ManagedDivergenceItem `json:"items"`
}

// ManagedDivergenceItem classifies one managed-key divergence and its escalation path.
type ManagedDivergenceItem struct {
	Key             string `json:"key"`
	Classification  string `json:"classification"`
	ImmediateAction string `json:"immediate_action"`
	RepeatedAction  string `json:"repeated_action"`
	RollbackTarget  string `json:"rollback_target,omitempty"`
}

// DatabaseRuntimeMetadata reports DB mode/driver and promotion compatibility details.
type DatabaseRuntimeMetadata struct {
	Mode                   string   `json:"mode"`
	Driver                 string   `json:"driver"`
	MigrationTrackingTable string   `json:"migration_tracking_table"`
	MigrationDialect       string   `json:"migration_dialect"`
	MigrationPortability   string   `json:"migration_portability"`
	CompatibleTargets      []string `json:"compatible_targets"`
	PromotionPath          string   `json:"promotion_path,omitempty"`
}

// RuntimeMetadata builds a runtime metadata snapshot using normalized configuration values.
func (c Config) RuntimeMetadata() RuntimeMetadata {
	normalized := normalizedConfigForReporting(c)
	report := c.Managed.RuntimeReport
	if report.Mode == "" {
		report = runtimeconfig.BuildReport(runtimeconfig.LayerInputs{
			Defaults:        managedKeyValues(normalizedDefaultConfigForReporting()),
			EffectiveValues: managedKeyValues(normalized),
			RepoSet:         map[string]bool{},
			EnvSet:          map[string]bool{},
			ManagedSet:      map[string]bool{},
			ManagedEnabled:  normalized.Managed.Enabled,
			Authority:       normalized.Managed.Authority,
		})
	}

	keys := map[string]ManagedKeyMetadata{}
	effective := managedKeyValues(normalized)
	for key, state := range report.Keys {
		keys[key] = ManagedKeyMetadata{
			Value:  state.Value,
			Source: string(state.Source),
		}
	}

	return RuntimeMetadata{
		Database: normalized.Database.RuntimeMetadata(),
		Managed: ManagedRuntimeMetadata{
			Mode:            string(report.Mode),
			Authority:       report.Authority,
			RegistryVersion: ManagedKeyRegistryVersion,
			SchemaVersion:   ManagedKeySchemaVersion,
			Divergence:      buildManagedDivergenceMetadata(report, effective),
			Keys:            keys,
		},
	}
}

func buildManagedDivergenceMetadata(report runtimeconfig.Report, effective map[string]string) ManagedDivergenceMetadata {
	items := make([]ManagedDivergenceItem, 0)
	if report.Mode != runtimeconfig.ModeManaged {
		return ManagedDivergenceMetadata{
			SchemaVersion: ManagedDivergenceSchemaVersion,
			Items:         items,
		}
	}

	for _, key := range managedSettingRegistryKeys() {
		state, ok := report.Keys[key]
		if !ok {
			continue
		}

		effectiveValue := strings.TrimSpace(effective[key])
		reportedValue := strings.TrimSpace(state.Value)
		if effectiveValue == reportedValue {
			continue
		}

		items = append(items, ManagedDivergenceItem{
			Key:             key,
			Classification:  ManagedDivergenceClassificationDrift,
			ImmediateAction: ManagedDivergenceActionRollback,
			RepeatedAction:  ManagedDivergenceActionUpstreamModuleCandidateReview,
			RollbackTarget:  managedSettingRollbackTarget(report.Mode, state, true),
		})
	}

	return ManagedDivergenceMetadata{
		SchemaVersion: ManagedDivergenceSchemaVersion,
		Items:         items,
	}
}

// RuntimeMetadata returns normalized metadata for DB mode, migration compatibility, and promotion path.
func (d DatabaseConfig) RuntimeMetadata() DatabaseRuntimeMetadata {
	mode := normalizeRuntimeMode(d.DbMode)
	driver := normalizeDBDriver(string(d.Driver))

	if mode == "" {
		switch driver {
		case string(DBDriverPostgres):
			mode = string(DBModeStandalone)
		default:
			mode = string(DBModeEmbedded)
		}
	}
	if driver == "" {
		if mode == string(DBModeStandalone) {
			driver = string(DBDriverPostgres)
		} else {
			driver = string(DBDriverSQLite)
		}
	}

	metadata := DatabaseRuntimeMetadata{
		Mode:                   mode,
		Driver:                 driver,
		MigrationTrackingTable: DefaultSchemaMigrationsTable,
		MigrationDialect:       driver,
		MigrationPortability:   MigrationPortabilitySQLCoreV1,
		CompatibleTargets:      []string{},
	}

	// v1 contract: SQLite is the primary source mode with manual promotion support to Postgres.
	if driver == string(DBDriverSQLite) {
		metadata.CompatibleTargets = []string{string(DBDriverPostgres)}
		metadata.PromotionPath = PromotionPathSQLiteToPostgresManualV1
	}

	return metadata
}

func normalizeRuntimeMode(raw dbmode) string {
	switch strings.ToLower(strings.TrimSpace(string(raw))) {
	case string(DBModeEmbedded):
		return string(DBModeEmbedded)
	case string(DBModeStandalone):
		return string(DBModeStandalone)
	default:
		return ""
	}
}
