package commands

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/leomorpho/goship/config"
	"github.com/leomorpho/goship/framework/backup"
)

type dbPromoteReport struct {
	Database          config.DatabaseRuntimeMetadata `json:"database"`
	Steps             []string                       `json:"steps"`
	SuggestedCommands []string                       `json:"suggested_commands,omitempty"`
	StateMachine      dbPromotionStateMachine        `json:"state_machine"`
	MutationPlan      *dbPromoteMutationPlan         `json:"mutation_plan,omitempty"`
	Note              string                         `json:"note,omitempty"`
}

type dbPromoteMutationPlan struct {
	DryRun  bool              `json:"dry_run"`
	EnvPath string            `json:"env_path,omitempty"`
	Values  map[string]string `json:"values"`
	Order   []string          `json:"order,omitempty"`
}

type dbPromotionStateMachine struct {
	SchemaVersion string                 `json:"schema_version"`
	InitialState  string                 `json:"initial_state"`
	TerminalState string                 `json:"terminal_state"`
	UnsafeStates  []string               `json:"unsafe_states"`
	States        []dbPromotionStateSpec `json:"states"`
}

type dbPromotionStateSpec struct {
	Name           string   `json:"name"`
	Kind           string   `json:"kind"`
	AllowsCommands []string `json:"allows_commands,omitempty"`
	BlocksCommands []string `json:"blocks_commands,omitempty"`
}

type dbExportReport struct {
	Manifest          backup.Manifest `json:"manifest"`
	SuggestedCommands []string        `json:"suggested_commands,omitempty"`
	Note              string          `json:"note,omitempty"`
}

type dbImportReport struct {
	Database          config.DatabaseRuntimeMetadata `json:"database"`
	Steps             []string                       `json:"steps"`
	SuggestedCommands []string                       `json:"suggested_commands,omitempty"`
	Note              string                         `json:"note,omitempty"`
}

type dbVerifyImportReport struct {
	Database          config.DatabaseRuntimeMetadata `json:"database"`
	PostImportChecks  []string                       `json:"post_import_checks"`
	SuggestedCommands []string                       `json:"suggested_commands,omitempty"`
	Note              string                         `json:"note,omitempty"`
}

func buildDBPromoteReport(md config.DatabaseRuntimeMetadata, dryRun bool) dbPromoteReport {
	report := dbPromoteReport{
		Database: md,
		Steps: []string{
			"freeze writes for the source app",
			"record runtime metadata and migration baseline",
			"export data from SQLite through framework-supported export hooks",
			"provision Postgres target and run canonical migrations",
			"import exported data into Postgres through framework-supported import hooks",
			"run framework verification checks for row counts, migration baseline, and key integrity",
			"switch config to Postgres and unfreeze writes",
		},
		SuggestedCommands: []string{
			"ship db:migrate",
			"ship db:export --json",
			"ship db:import --json",
			"ship db:verify-import --json",
		},
		StateMachine: buildDBPromotionStateMachine(),
	}
	if md.PromotionPath == "" {
		report.Note = "no promotion path is defined for the current database driver"
		return report
	}
	report.MutationPlan = buildDBPromoteMutationPlan(md, dryRun)
	if report.MutationPlan == nil {
		report.Note = "no deterministic config mutation plan is defined for the current promotion path"
		return report
	}
	if dryRun {
		report.Note = "dry-run only; rerun without --dry-run to update .env"
		return report
	}
	report.Note = "config updated; export/import/verification steps remain manual"
	return report
}

func buildDBPromotionStateMachine() dbPromotionStateMachine {
	return dbPromotionStateMachine{
		SchemaVersion: "promotion-state-machine-v1",
		InitialState:  "source-ready",
		TerminalState: "completed",
		UnsafeStates: []string{
			"config-applied",
			"source-exported",
			"target-migrated",
			"target-imported",
			"verification-failed",
		},
		States: []dbPromotionStateSpec{
			{
				Name:           "source-ready",
				Kind:           "safe",
				AllowsCommands: []string{"ship db:promote", "ship db:export --json"},
			},
			{
				Name:           "config-applied",
				Kind:           "unsafe-partial",
				AllowsCommands: []string{"ship db:import --json", "ship db:verify-import --json"},
				BlocksCommands: []string{"ship db:promote"},
			},
			{
				Name:           "source-exported",
				Kind:           "unsafe-partial",
				AllowsCommands: []string{"ship db:import --json"},
				BlocksCommands: []string{"ship db:promote", "ship db:export --json"},
			},
			{
				Name:           "target-migrated",
				Kind:           "unsafe-partial",
				AllowsCommands: []string{"ship db:import --json"},
				BlocksCommands: []string{"ship db:promote", "ship db:export --json"},
			},
			{
				Name:           "target-imported",
				Kind:           "unsafe-partial",
				AllowsCommands: []string{"ship db:verify-import --json"},
				BlocksCommands: []string{"ship db:promote", "ship db:export --json", "ship db:import --json"},
			},
			{
				Name:           "verification-failed",
				Kind:           "unsafe-partial",
				AllowsCommands: []string{"ship db:verify-import --json"},
				BlocksCommands: []string{"ship db:promote", "ship db:export --json", "ship db:import --json"},
			},
			{
				Name:           "completed",
				Kind:           "terminal",
				BlocksCommands: []string{"ship db:promote", "ship db:export --json", "ship db:import --json", "ship db:verify-import --json"},
			},
		},
	}
}

func buildDBPromoteMutationPlan(md config.DatabaseRuntimeMetadata, dryRun bool) *dbPromoteMutationPlan {
	if md.PromotionPath != config.PromotionPathSQLiteToPostgresManualV1 {
		return nil
	}
	values := map[string]string{}
	order := make([]string, 0, len(profilePresets["standard"].keyOrder)+8)
	for _, key := range profilePresets["standard"].keyOrder {
		values[key] = profilePresets["standard"].values[key]
		order = append(order, key)
	}
	adapterDesired := map[string]string{
		"db":    "postgres",
		"cache": "redis",
		"jobs":  "asynq",
	}
	for key, value := range adapterEnvValues(adapterDesired) {
		values[key] = value
	}
	order = append(order, adapterEnvOrder(adapterDesired)...)
	return &dbPromoteMutationPlan{
		DryRun: dryRun,
		Values: values,
		Order:  order,
	}
}

func buildDBExportReport(cfg config.Config) (dbExportReport, error) {
	md := cfg.RuntimeMetadata().Database
	if md.Driver != string(config.DBDriverSQLite) {
		return dbExportReport{}, fmt.Errorf("db:export requires a sqlite source, got %s", md.Driver)
	}

	sqlitePath := strings.TrimSpace(cfg.Backup.SQLitePath)
	if sqlitePath == "" {
		sqlitePath = strings.TrimSpace(cfg.Database.Path)
	}
	if sqlitePath == "" {
		return dbExportReport{}, fmt.Errorf("db:export requires a sqlite source path")
	}

	driver := backup.NewSQLiteDriver()
	manifest, err := driver.Create(context.Background(), backup.CreateRequest{
		SQLitePath:    sqlitePath,
		SchemaVersion: cfg.Backup.SchemaVersion,
		Storage: backup.StorageLocation{
			Provider: backup.ProviderLocal,
			URI:      "file://" + filepath.ToSlash(sqlitePath),
		},
	})
	if err != nil {
		return dbExportReport{}, err
	}

	return dbExportReport{
		Manifest: manifest,
		SuggestedCommands: []string{
			"ship db:import --json",
		},
		Note: "planning only; db:export reports manifest checksums and does not mutate runtime state yet",
	}, nil
}

func printDBExportReport(w io.Writer, report dbExportReport) {
	fmt.Fprintln(w, "DB export manifest:")
	fmt.Fprintf(w, "- version: %s\n", report.Manifest.Version)
	fmt.Fprintf(w, "- created_at: %s\n", report.Manifest.CreatedAt.UTC().Format(time.RFC3339))
	fmt.Fprintf(w, "- database_mode: %s\n", report.Manifest.Database.Mode)
	fmt.Fprintf(w, "- database_driver: %s\n", report.Manifest.Database.Driver)
	fmt.Fprintf(w, "- schema_version: %s\n", report.Manifest.Database.SchemaVersion)
	fmt.Fprintf(w, "- source_path: %s\n", report.Manifest.Database.SourcePath)
	fmt.Fprintf(w, "- checksum_sha256: %s\n", report.Manifest.Artifact.ChecksumSHA256)
	fmt.Fprintf(w, "- artifact_size_bytes: %d\n", report.Manifest.Artifact.SizeBytes)
	fmt.Fprintf(w, "- storage_provider: %s\n", report.Manifest.Storage.Provider)
	if report.Manifest.Storage.URI != "" {
		fmt.Fprintf(w, "- storage_uri: %s\n", report.Manifest.Storage.URI)
	}
	for _, cmd := range report.SuggestedCommands {
		fmt.Fprintf(w, "- next: %s\n", cmd)
	}
	if report.Note != "" {
		fmt.Fprintf(w, "- note: %s\n", report.Note)
	}
}

func buildDBImportReport(md config.DatabaseRuntimeMetadata) dbImportReport {
	report := dbImportReport{
		Database: md,
		Steps: []string{
			"load export manifest and validate version, driver, and checksums",
			"import exported data into Postgres through framework-supported import hooks",
			"record import evidence for post-import verification",
		},
		SuggestedCommands: []string{
			"ship db:verify-import --json",
		},
	}
	if md.PromotionPath == "" {
		report.Note = "no import path is defined for the current database driver"
		return report
	}
	report.Note = "planning only; db:import does not mutate files or import data yet"
	return report
}

func printDBImportReport(w io.Writer, report dbImportReport) {
	fmt.Fprintln(w, "DB import plan:")
	fmt.Fprintf(w, "- mode: %s\n", report.Database.Mode)
	fmt.Fprintf(w, "- driver: %s\n", report.Database.Driver)
	fmt.Fprintf(w, "- migration_portability: %s\n", report.Database.MigrationPortability)
	if report.Database.PromotionPath != "" {
		fmt.Fprintf(w, "- promotion_path: %s\n", report.Database.PromotionPath)
	}
	if len(report.Database.CompatibleTargets) > 0 {
		fmt.Fprintf(w, "- compatible_targets: %s\n", strings.Join(report.Database.CompatibleTargets, ", "))
	}
	for _, step := range report.Steps {
		fmt.Fprintf(w, "- step: %s\n", step)
	}
	for _, cmd := range report.SuggestedCommands {
		fmt.Fprintf(w, "- next: %s\n", cmd)
	}
	if report.Note != "" {
		fmt.Fprintf(w, "- note: %s\n", report.Note)
	}
}

func buildDBVerifyImportReport(md config.DatabaseRuntimeMetadata) dbVerifyImportReport {
	report := dbVerifyImportReport{
		Database: md,
		PostImportChecks: []string{
			"manifest.validated",
			"row.counts.checked",
			"migration.baseline.checked",
			"key.integrity.checked",
		},
	}
	if md.PromotionPath == "" {
		report.Note = "no verification path is defined for the current database driver"
		return report
	}
	report.Note = "planning only; db:verify-import does not mutate files or databases yet"
	return report
}

func printDBVerifyImportReport(w io.Writer, report dbVerifyImportReport) {
	fmt.Fprintln(w, "DB verify-import plan:")
	fmt.Fprintf(w, "- mode: %s\n", report.Database.Mode)
	fmt.Fprintf(w, "- driver: %s\n", report.Database.Driver)
	fmt.Fprintf(w, "- migration_portability: %s\n", report.Database.MigrationPortability)
	if report.Database.PromotionPath != "" {
		fmt.Fprintf(w, "- promotion_path: %s\n", report.Database.PromotionPath)
	}
	if len(report.Database.CompatibleTargets) > 0 {
		fmt.Fprintf(w, "- compatible_targets: %s\n", strings.Join(report.Database.CompatibleTargets, ", "))
	}
	for _, check := range report.PostImportChecks {
		fmt.Fprintf(w, "- check: %s\n", check)
	}
	if report.Note != "" {
		fmt.Fprintf(w, "- note: %s\n", report.Note)
	}
}

func printDBPromoteReport(w io.Writer, report dbPromoteReport) {
	fmt.Fprintln(w, "DB promote plan:")
	if report.MutationPlan != nil {
		if report.MutationPlan.DryRun {
			fmt.Fprintln(w, "- mode: dry-run (no files changed)")
		} else {
			fmt.Fprintln(w, "- mode: apply")
		}
	}
	fmt.Fprintf(w, "- database_mode: %s\n", report.Database.Mode)
	fmt.Fprintf(w, "- driver: %s\n", report.Database.Driver)
	fmt.Fprintf(w, "- migration_portability: %s\n", report.Database.MigrationPortability)
	if report.Database.PromotionPath != "" {
		fmt.Fprintf(w, "- promotion_path: %s\n", report.Database.PromotionPath)
	}
	fmt.Fprintf(w, "- state_machine: %s\n", report.StateMachine.SchemaVersion)
	fmt.Fprintf(w, "- unsafe_states: %s\n", strings.Join(report.StateMachine.UnsafeStates, ", "))
	if len(report.Database.CompatibleTargets) > 0 {
		fmt.Fprintf(w, "- compatible_targets: %s\n", strings.Join(report.Database.CompatibleTargets, ", "))
	}
	for _, step := range report.Steps {
		fmt.Fprintf(w, "- step: %s\n", step)
	}
	for _, state := range report.StateMachine.States {
		for _, cmd := range state.BlocksCommands {
			fmt.Fprintf(w, "- blocked: %s -> %s\n", state.Name, cmd)
		}
	}
	for _, cmd := range report.SuggestedCommands {
		fmt.Fprintf(w, "- next: %s\n", cmd)
	}
	if report.MutationPlan != nil {
		for _, key := range report.MutationPlan.Order {
			fmt.Fprintf(w, "- set: %s=%s\n", key, report.MutationPlan.Values[key])
		}
	}
	if report.Note != "" {
		fmt.Fprintf(w, "- note: %s\n", report.Note)
	}
}
