package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRuntimeMetadataSQLitePromotionContract(t *testing.T) {
	cfg := defaultConfig()

	md := cfg.RuntimeMetadata().Database
	assert.Equal(t, string(DBModeEmbedded), md.Mode)
	assert.Equal(t, string(DBDriverSQLite), md.Driver)
	assert.Equal(t, DefaultSchemaMigrationsTable, md.MigrationTrackingTable)
	assert.Equal(t, string(DBDriverSQLite), md.MigrationDialect)
	assert.Equal(t, MigrationPortabilitySQLCoreV1, md.MigrationPortability)
	assert.Equal(t, []string{string(DBDriverPostgres)}, md.CompatibleTargets)
	assert.Equal(t, PromotionPathSQLiteToPostgresManualV1, md.PromotionPath)

	managed := cfg.RuntimeMetadata().Managed
	assert.Equal(t, "standalone", managed.Mode)
	assert.Equal(t, "otter", managed.Keys["adapters.cache"].Value)
	assert.Equal(t, "framework-default", managed.Keys["adapters.cache"].Source)
}

func TestRuntimeMetadataPostgresHasNoPromotionPath(t *testing.T) {
	cfg := defaultConfig()
	cfg.Database.Driver = "pgx"
	cfg.Database.DbMode = DBModeStandalone

	md := cfg.RuntimeMetadata().Database
	assert.Equal(t, string(DBModeStandalone), md.Mode)
	assert.Equal(t, string(DBDriverPostgres), md.Driver)
	assert.Equal(t, string(DBDriverPostgres), md.MigrationDialect)
	assert.Empty(t, md.CompatibleTargets)
	assert.Empty(t, md.PromotionPath)
}
