package config

import "strings"

const (
	// DefaultSchemaMigrationsTable is the canonical migration tracking table for core SQL migrations.
	DefaultSchemaMigrationsTable = "goship_schema_migrations"
	// MigrationPortabilitySQLCoreV1 is the first portability profile for SQLite-first promotion.
	MigrationPortabilitySQLCoreV1 = "sql-core-v1"
	// PromotionPathSQLiteToPostgresManualV1 identifies the first supported promotion workflow.
	PromotionPathSQLiteToPostgresManualV1 = "sqlite-to-postgres-manual-v1"
)

// RuntimeMetadata reports normalized runtime capability metadata for status/reporting surfaces.
type RuntimeMetadata struct {
	Database DatabaseRuntimeMetadata `json:"database"`
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
	return RuntimeMetadata{
		Database: c.Database.RuntimeMetadata(),
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
