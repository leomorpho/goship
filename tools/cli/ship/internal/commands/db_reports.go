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
	StateMachine      dbPromotionStateMachine        `json:"state_machine"`
	Steps             []string                       `json:"steps"`
	SuggestedCommands []string                       `json:"suggested_commands,omitempty"`
	MutationPlan      *dbPromoteMutationPlan         `json:"mutation_plan,omitempty"`
	Note              string                         `json:"note,omitempty"`
}

type dbPromotionStateMachine struct {
	SchemaVersion  string                    `json:"schema_version"`
	CurrentState   string                    `json:"current_state"`
	NextState      string                    `json:"next_state,omitempty"`
	BlockingStates []string                  `json:"blocking_states"`
	States         []dbPromotionState        `json:"states"`
	Blockers       []dbPromotionStateBlocker `json:"blockers,omitempty"`
}

type dbPromotionState struct {
	ID           string `json:"id"`
	Class        string `json:"class"`
	Description  string `json:"description"`
	AllowPromote bool   `json:"allow_promote"`
}

type dbPromotionStateBlocker struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Remediation string `json:"remediation"`
}

type dbPromoteMutationPlan struct {
	DryRun  bool              `json:"dry_run"`
	EnvPath string            `json:"env_path,omitempty"`
	Values  map[string]string `json:"values"`
	Order   []string          `json:"order,omitempty"`
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
		StateMachine: buildDBPromotionStateMachine(md),
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
	}
	if len(report.StateMachine.Blockers) > 0 {
		report.Note = "promotion blocked until the current partial or inconsistent state is resolved"
		return report
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

func buildDBPromotionStateMachine(md config.DatabaseRuntimeMetadata) dbPromotionStateMachine {
	machine := dbPromotionStateMachine{
		SchemaVersion:  "promotion-state-machine-v1",
		BlockingStates: []string{"config-mutated-awaiting-import", "import-complete-awaiting-verify", "inconsistent-runtime-state"},
		States: []dbPromotionState{
			{
				ID:           "sqlite-source-ready",
				Class:        "safe",
				Description:  "SQLite source runtime is ready for the first config-mutation step.",
				AllowPromote: true,
			},
			{
				ID:           "config-mutated-awaiting-import",
				Class:        "partial",
				Description:  "Runtime config already points at the target topology and still requires import or verification follow-up.",
				AllowPromote: false,
			},
			{
				ID:           "import-complete-awaiting-verify",
				Class:        "partial",
				Description:  "Data import is complete but verification evidence has not been recorded yet.",
				AllowPromote: false,
			},
			{
				ID:           "inconsistent-runtime-state",
				Class:        "unsafe",
				Description:  "Runtime metadata does not match a promotable SQLite source or a recognized partial state.",
				AllowPromote: false,
			},
		},
	}

	switch {
	case md.Mode == string(config.DBModeEmbedded) && md.Driver == string(config.DBDriverSQLite) && md.PromotionPath == config.PromotionPathSQLiteToPostgresManualV1:
		machine.CurrentState = "sqlite-source-ready"
		machine.NextState = "config-mutated-awaiting-import"
	case md.Mode == string(config.DBModeStandalone) && md.Driver == string(config.DBDriverPostgres):
		machine.CurrentState = "config-mutated-awaiting-import"
		machine.Blockers = []dbPromotionStateBlocker{{
			ID:          "promotion-state.partial-transition-blocked",
			Title:       "promotion is already in a partial post-config state",
			Remediation: "Run the import and verification follow-up for the Postgres target instead of re-running ship db:promote.",
		}}
	default:
		machine.CurrentState = "inconsistent-runtime-state"
		machine.Blockers = []dbPromotionStateBlocker{{
			ID:          "promotion-state.inconsistent-runtime-state",
			Title:       "runtime metadata is not in a promotable SQLite source state",
			Remediation: "Return the runtime to an embedded SQLite source or complete the existing promotion workflow before retrying.",
		}}
	}

	return machine
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
	fmt.Fprintf(w, "- promotion_state_schema: %s\n", report.StateMachine.SchemaVersion)
	fmt.Fprintf(w, "- current_state: %s\n", report.StateMachine.CurrentState)
	if report.StateMachine.NextState != "" {
		fmt.Fprintf(w, "- next_state: %s\n", report.StateMachine.NextState)
	}
	if report.Database.PromotionPath != "" {
		fmt.Fprintf(w, "- promotion_path: %s\n", report.Database.PromotionPath)
	}
	if len(report.Database.CompatibleTargets) > 0 {
		fmt.Fprintf(w, "- compatible_targets: %s\n", strings.Join(report.Database.CompatibleTargets, ", "))
	}
	for _, state := range report.StateMachine.States {
		fmt.Fprintf(w, "- state: %s (%s) allow_promote=%t\n", state.ID, state.Class, state.AllowPromote)
	}
	for _, blocker := range report.StateMachine.Blockers {
		fmt.Fprintf(w, "- blocker: %s\n", blocker.ID)
		fmt.Fprintf(w, "- blocker_title: %s\n", blocker.Title)
		fmt.Fprintf(w, "- remediation: %s\n", blocker.Remediation)
	}
	for _, step := range report.Steps {
		fmt.Fprintf(w, "- step: %s\n", step)
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
