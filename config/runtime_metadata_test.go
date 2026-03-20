package config

import (
	"encoding/json"
	"testing"

	"github.com/leomorpho/goship/framework/runtimeconfig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestRuntimeMetadataManagedRegistryContract(t *testing.T) {
	cfg := defaultConfig()
	cfg.Managed.Enabled = true
	cfg.Managed.Authority = "control-plane"
	cfg.Managed.RuntimeReport = runtimeconfig.BuildReport(runtimeconfig.LayerInputs{
		Defaults:        managedKeyValues(defaultConfig()),
		EffectiveValues: managedKeyValues(cfg),
		RepoSet:         map[string]bool{},
		EnvSet:          map[string]bool{},
		ManagedSet:      map[string]bool{},
		ManagedEnabled:  true,
		Authority:       cfg.Managed.Authority,
	})

	metadata := cfg.RuntimeMetadata().Managed
	statuses := cfg.ManagedSettingStatuses()

	assert.Equal(t, "managed", metadata.Mode)
	assert.Equal(t, "control-plane", metadata.Authority)
	assert.Equal(t, ManagedKeyRegistryVersion, metadata.RegistryVersion)
	assert.Equal(t, ManagedKeySchemaVersion, metadata.SchemaVersion)
	assert.ElementsMatch(t, managedSettingRegistryKeys(), managedMetadataKeys(metadata.Keys))
	assert.ElementsMatch(t, managedSettingRegistryKeys(), managedStatusKeys(statuses))
	assert.Equal(t, "otter", metadata.Keys["adapters.cache"].Value)
	assert.Equal(t, "framework-default", metadata.Keys["adapters.cache"].Source)
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

func TestRuntimeMetadataManagedKeyRegistryVersionContract(t *testing.T) {
	cfg := defaultConfig()

	raw, err := json.Marshal(cfg.RuntimeMetadata().Managed)
	require.NoError(t, err)

	assert.Contains(t, string(raw), `"registry_version":"managed-key-registry-v1"`)
	assert.Contains(t, string(raw), `"schema_version":"managed-key-schema-v1"`)
}

func managedMetadataKeys(metadata map[string]ManagedKeyMetadata) []string {
	keys := make([]string, 0, len(metadata))
	for key := range metadata {
		keys = append(keys, key)
	}
	return keys
}

func managedStatusKeys(statuses []ManagedSettingStatus) []string {
	keys := make([]string, 0, len(statuses))
	for _, status := range statuses {
		keys = append(keys, status.Key)
	}
	return keys
}
