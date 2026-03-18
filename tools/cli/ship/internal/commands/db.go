package commands

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/leomorpho/goship/config"
	"github.com/leomorpho/goship/framework/backup"
	rt "github.com/leomorpho/goship/tools/cli/ship/internal/runtime"
)

type DBDeps struct {
	Out             io.Writer
	Err             io.Writer
	LoadConfig      func() (config.Config, error)
	ResolveDBURL    func() (string, error)
	ResolveDBDriver func() (string, error)
	RunGoose        func(args ...string) int
	RunCmd          func(name string, args ...string) int
	GooseDir        string
	FindGoModule    func(start string) (string, string, error)
}

func RunDB(args []string, d DBDeps) int {
	if len(args) == 0 {
		PrintDBHelp(d.Err)
		return 1
	}

	switch args[0] {
	case "create":
		return runCreate(args[1:], d)
	case "generate":
		return runGenerate(args[1:], d)
	case "export":
		return runExport(args[1:], d)
	case "import":
		return runImport(args[1:], d)
	case "promote":
		return runPromote(args[1:], d)
	case "make":
		return runMake(args[1:], d)
	case "migrate":
		if len(args) != 1 {
			fmt.Fprintln(d.Err, "usage: ship db:migrate")
			return 1
		}
		dbURL, err := d.ResolveDBURL()
		if err != nil {
			fmt.Fprintf(d.Err, "failed to resolve database URL: %v\n", err)
			return 1
		}
		return runGooseUpAll(d, dbURL)
	case "status":
		return runStatus(args[1:], d)
	case "verify-import":
		return runVerifyImport(args[1:], d)
	case "reset":
		return runReset(args[1:], d)
	case "drop":
		return runDrop(args[1:], d)
	case "rollback":
		return runRollback(args[1:], d)
	case "seed":
		if len(args) != 1 {
			fmt.Fprintln(d.Err, "usage: ship db:seed")
			return 1
		}
		return d.RunCmd("go", "run", "./cmd/seed/main.go")
	case "console":
		return runConsole(args[1:], d)
	case "help", "-h", "--help":
		PrintDBHelp(d.Out)
		return 0
	default:
		fmt.Fprintf(d.Err, "unknown db command: %s\n\n", args[0])
		PrintDBHelp(d.Err)
		return 1
	}
}

func runStatus(args []string, d DBDeps) int {
	if len(args) != 0 {
		fmt.Fprintln(d.Err, "usage: ship db:status")
		return 1
	}

	dbURL, err := d.ResolveDBURL()
	if err != nil {
		fmt.Fprintf(d.Err, "failed to resolve database URL: %v\n", err)
		return 1
	}

	return runGooseStatusAll(d, dbURL)
}

func runReset(args []string, d DBDeps) int {
	fs := flag.NewFlagSet("db:reset", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	seed := fs.Bool("seed", false, "seed after reset+migrate")
	force := fs.Bool("force", false, "allow reset on non-local database URLs")
	yes := fs.Bool("yes", false, "confirm destructive reset")
	dryRun := fs.Bool("dry-run", false, "print planned actions without executing")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(d.Err, "invalid db:reset arguments: %v\n", err)
		fmt.Fprintln(d.Err, "usage: ship db:reset [--seed] [--force] [--yes] [--dry-run]")
		return 1
	}
	if fs.NArg() > 0 {
		fmt.Fprintln(d.Err, "usage: ship db:reset [--seed] [--force] [--yes] [--dry-run]")
		return 1
	}

	dbURL, err := d.ResolveDBURL()
	if err != nil {
		fmt.Fprintf(d.Err, "failed to resolve database URL: %v\n", err)
		return 1
	}
	local := IsLocalDBURL(dbURL)
	if isProductionEnv() && !(*force && *yes) {
		fmt.Fprintln(d.Err, "refusing to reset in production without both --force and --yes")
		return 1
	}
	if !local && !*force {
		fmt.Fprintln(d.Err, "refusing to reset a non-local database without --force")
		return 1
	}
	if !*yes && !*dryRun {
		fmt.Fprintln(d.Err, "refusing destructive reset without --yes (or use --dry-run)")
		return 1
	}

	printPlan(d.Out, "reset", dbURL, local, []string{
		"goose reset",
		"goose up",
	}, *seed, *dryRun)
	if *dryRun {
		return 0
	}

	if code := runGooseResetAll(d, dbURL); code != 0 {
		return code
	}
	if code := runGooseUpAll(d, dbURL); code != 0 {
		return code
	}
	if *seed {
		return d.RunCmd("go", "run", "./cmd/seed/main.go")
	}
	return 0
}

func runDrop(args []string, d DBDeps) int {
	fs := flag.NewFlagSet("db:drop", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	force := fs.Bool("force", false, "allow drop on non-local database URLs")
	yes := fs.Bool("yes", false, "confirm destructive drop")
	dryRun := fs.Bool("dry-run", false, "print planned actions without executing")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(d.Err, "invalid db:drop arguments: %v\n", err)
		fmt.Fprintln(d.Err, "usage: ship db:drop [--force] [--yes] [--dry-run]")
		return 1
	}
	if fs.NArg() > 0 {
		fmt.Fprintln(d.Err, "usage: ship db:drop [--force] [--yes] [--dry-run]")
		return 1
	}

	dbURL, err := d.ResolveDBURL()
	if err != nil {
		fmt.Fprintf(d.Err, "failed to resolve database URL: %v\n", err)
		return 1
	}
	local := IsLocalDBURL(dbURL)
	if isProductionEnv() && !(*force && *yes) {
		fmt.Fprintln(d.Err, "refusing to drop in production without both --force and --yes")
		return 1
	}
	if !local && !*force {
		fmt.Fprintln(d.Err, "refusing to drop a non-local database without --force")
		return 1
	}
	if !*yes && !*dryRun {
		fmt.Fprintln(d.Err, "refusing destructive drop without --yes (or use --dry-run)")
		return 1
	}
	printPlan(d.Out, "drop", dbURL, local, []string{"goose reset (revert all migrations; does not drop DB)"}, false, *dryRun)
	if *dryRun {
		return 0
	}
	return runGooseResetAll(d, dbURL)
}

func runCreate(args []string, d DBDeps) int {
	fs := flag.NewFlagSet("db:create", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	dryRun := fs.Bool("dry-run", false, "print planned actions without executing")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(d.Err, "invalid db:create arguments: %v\n", err)
		fmt.Fprintln(d.Err, "usage: ship db:create [--dry-run]")
		return 1
	}
	if fs.NArg() > 0 {
		fmt.Fprintln(d.Err, "usage: ship db:create [--dry-run]")
		return 1
	}

	dbURL, err := d.ResolveDBURL()
	if err != nil {
		fmt.Fprintf(d.Err, "failed to resolve database URL: %v\n", err)
		return 1
	}
	local := IsLocalDBURL(dbURL)
	printPlan(d.Out, "create", dbURL, local, []string{"verify target database is reachable"}, false, *dryRun)
	if *dryRun {
		return 0
	}

	if code := runGooseStatusAll(d, dbURL); code != 0 {
		fmt.Fprintln(d.Err, "database is not reachable or does not exist; create it with your DB provider and retry")
		return code
	}
	return 0
}

func runRollback(args []string, d DBDeps) int {
	amount := "1"
	if len(args) > 1 {
		fmt.Fprintln(d.Err, "usage: ship db:rollback [amount]")
		return 1
	}
	if len(args) == 1 {
		if _, err := strconv.Atoi(args[0]); err != nil {
			fmt.Fprintf(d.Err, "invalid rollback amount %q: must be an integer\n", args[0])
			return 1
		}
		amount = args[0]
	}

	dbURL, err := d.ResolveDBURL()
	if err != nil {
		fmt.Fprintf(d.Err, "failed to resolve database URL: %v\n", err)
		return 1
	}

	return runGooseDown(d, dbURL, amount)
}

func runMake(args []string, d DBDeps) int {
	var (
		name       string
		tableName  string
		softDelete bool
	)

	for i := 0; i < len(args); i++ {
		arg := strings.TrimSpace(args[i])
		switch arg {
		case "--soft-delete":
			softDelete = true
		case "--table":
			if i+1 >= len(args) {
				fmt.Fprintln(d.Err, "db:make --soft-delete requires --table <table>")
				return 1
			}
			i++
			tableName = strings.TrimSpace(args[i])
		default:
			if strings.HasPrefix(arg, "--table=") {
				tableName = strings.TrimSpace(strings.TrimPrefix(arg, "--table="))
				continue
			}
			if strings.HasPrefix(arg, "-") {
				fmt.Fprintf(d.Err, "invalid db:make arguments: unknown flag %s\n", arg)
				fmt.Fprintln(d.Err, "usage: ship db:make <migration_name> [--soft-delete --table <table>]")
				return 1
			}
			if name != "" {
				fmt.Fprintln(d.Err, "usage: ship db:make <migration_name> [--soft-delete --table <table>]")
				return 1
			}
			name = arg
		}
	}

	if name == "" {
		fmt.Fprintln(d.Err, "usage: ship db:make <migration_name> [--soft-delete --table <table>]")
		return 1
	}

	if softDelete {
		tableName = strings.TrimSpace(tableName)
		if tableName == "" {
			fmt.Fprintln(d.Err, "db:make --soft-delete requires --table <table>")
			return 1
		}
		if err := writeSoftDeleteMigration(d.GooseDir, name, tableName, time.Now().UTC()); err != nil {
			fmt.Fprintf(d.Err, "failed to write soft-delete migration: %v\n", err)
			return 1
		}
		fmt.Fprintf(d.Out, "created soft-delete migration for %s in %s\n", tableName, d.GooseDir)
		return 0
	}

	return d.RunGoose("-dir", d.GooseDir, "create", name, "sql")
}

func writeSoftDeleteMigration(dir string, migrationName string, tableName string, now time.Time) error {
	filename := fmt.Sprintf("%s_%s.sql", now.UTC().Format("20060102150405"), migrationName)
	path := filepath.Join(dir, filename)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	body := fmt.Sprintf(`-- +goose Up
ALTER TABLE %s ADD COLUMN deleted_at DATETIME;
CREATE INDEX idx_%s_deleted_at ON %s(deleted_at);

-- +goose Down
DROP INDEX IF EXISTS idx_%s_deleted_at;
`, tableName, tableName, tableName, tableName)

	return os.WriteFile(path, []byte(body), 0o644)
}

func runGenerate(args []string, d DBDeps) int {
	fs := flag.NewFlagSet("db:generate", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	configPath := fs.String("config", filepath.ToSlash(filepath.Join("db", "bobgen.yaml")), "path to bobgen config")
	dryRun := fs.Bool("dry-run", false, "print planned generation command without executing")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(d.Err, "invalid db:generate arguments: %v\n", err)
		fmt.Fprintln(d.Err, "usage: ship db:generate [--config <path>] [--dry-run]")
		return 1
	}
	if fs.NArg() > 0 {
		fmt.Fprintln(d.Err, "usage: ship db:generate [--config <path>] [--dry-run]")
		return 1
	}

	cfg := strings.TrimSpace(*configPath)
	if cfg == "" {
		fmt.Fprintln(d.Err, "usage: ship db:generate [--config <path>] [--dry-run]")
		return 1
	}

	configs, err := resolveBobgenConfigs(d, cfg)
	if err != nil {
		fmt.Fprintf(d.Err, "failed to resolve bobgen config paths: %v\n", err)
		return 1
	}

	fmt.Fprintln(d.Out, "DB generate plan:")
	for _, config := range configs {
		fmt.Fprintf(d.Out, "- config: %s\n", config)
		fmt.Fprintf(d.Out, "- command: bobgen-sql -c %s\n", config)
	}
	if *dryRun {
		fmt.Fprintln(d.Out, "- mode: dry-run (no commands executed)")
		return 0
	}

	for _, config := range configs {
		if code := d.RunCmd("bobgen-sql", "-c", config); code != 0 {
			return code
		}
	}
	return 0
}

type dbPromoteReport struct {
	Database          config.DatabaseRuntimeMetadata `json:"database"`
	Steps             []string                       `json:"steps"`
	SuggestedCommands []string                       `json:"suggested_commands,omitempty"`
	Note              string                         `json:"note,omitempty"`
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

func runPromote(args []string, d DBDeps) int {
	for _, arg := range args {
		if arg == "help" || arg == "-h" || arg == "--help" {
			PrintDBHelp(d.Out)
			return 0
		}
	}

	fs := flag.NewFlagSet("db:promote", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	jsonOutput := fs.Bool("json", false, "print JSON output")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(d.Err, "invalid db:promote arguments: %v\n", err)
		fmt.Fprintln(d.Err, "usage: ship db:promote [--json]")
		return 1
	}
	if fs.NArg() > 0 {
		fmt.Fprintln(d.Err, "usage: ship db:promote [--json]")
		return 1
	}
	if d.LoadConfig == nil {
		fmt.Fprintln(d.Err, "db:promote requires config loader dependency")
		return 1
	}

	cfg, err := d.LoadConfig()
	if err != nil {
		fmt.Fprintf(d.Err, "failed to load config: %v\n", err)
		return 1
	}

	report := buildDBPromoteReport(cfg.RuntimeMetadata().Database)
	if *jsonOutput {
		enc := json.NewEncoder(d.Out)
		enc.SetIndent("", "  ")
		if err := enc.Encode(report); err != nil {
			fmt.Fprintf(d.Err, "failed to encode db:promote output: %v\n", err)
			return 1
		}
		return 0
	}

	printDBPromoteReport(d.Out, report)
	return 0
}

func buildDBPromoteReport(md config.DatabaseRuntimeMetadata) dbPromoteReport {
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
			"ship runtime:report --json",
			"ship profile:set standard",
			"ship adapter:set db=postgres cache=redis jobs=asynq",
			"ship db:migrate",
			"ship db:export --json",
			"ship db:import --json",
			"ship db:verify-import --json",
		},
	}
	if md.PromotionPath == "" {
		report.Note = "no promotion path is defined for the current database driver"
		return report
	}
	report.Note = "planning only; db:promote does not mutate files or run migrations yet"
	return report
}

func runExport(args []string, d DBDeps) int {
	for _, arg := range args {
		if arg == "help" || arg == "-h" || arg == "--help" {
			PrintDBHelp(d.Out)
			return 0
		}
	}

	fs := flag.NewFlagSet("db:export", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	jsonOutput := fs.Bool("json", false, "print JSON output")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(d.Err, "invalid db:export arguments: %v\n", err)
		fmt.Fprintln(d.Err, "usage: ship db:export [--json]")
		return 1
	}
	if fs.NArg() > 0 {
		fmt.Fprintln(d.Err, "usage: ship db:export [--json]")
		return 1
	}
	if d.LoadConfig == nil {
		fmt.Fprintln(d.Err, "db:export requires config loader dependency")
		return 1
	}

	cfg, err := d.LoadConfig()
	if err != nil {
		fmt.Fprintf(d.Err, "failed to load config: %v\n", err)
		return 1
	}

	report, err := buildDBExportReport(cfg)
	if err != nil {
		fmt.Fprintf(d.Err, "failed to build db export manifest: %v\n", err)
		return 1
	}

	if *jsonOutput {
		enc := json.NewEncoder(d.Out)
		enc.SetIndent("", "  ")
		if err := enc.Encode(report); err != nil {
			fmt.Fprintf(d.Err, "failed to encode db:export output: %v\n", err)
			return 1
		}
		return 0
	}

	printDBExportReport(d.Out, report)
	return 0
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

func runImport(args []string, d DBDeps) int {
	for _, arg := range args {
		if arg == "help" || arg == "-h" || arg == "--help" {
			PrintDBHelp(d.Out)
			return 0
		}
	}

	fs := flag.NewFlagSet("db:import", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	jsonOutput := fs.Bool("json", false, "print JSON output")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(d.Err, "invalid db:import arguments: %v\n", err)
		fmt.Fprintln(d.Err, "usage: ship db:import [--json]")
		return 1
	}
	if fs.NArg() > 0 {
		fmt.Fprintln(d.Err, "usage: ship db:import [--json]")
		return 1
	}
	if d.LoadConfig == nil {
		fmt.Fprintln(d.Err, "db:import requires config loader dependency")
		return 1
	}

	cfg, err := d.LoadConfig()
	if err != nil {
		fmt.Fprintf(d.Err, "failed to load config: %v\n", err)
		return 1
	}

	report := buildDBImportReport(cfg.RuntimeMetadata().Database)
	if *jsonOutput {
		enc := json.NewEncoder(d.Out)
		enc.SetIndent("", "  ")
		if err := enc.Encode(report); err != nil {
			fmt.Fprintf(d.Err, "failed to encode db:import output: %v\n", err)
			return 1
		}
		return 0
	}

	printDBImportReport(d.Out, report)
	return 0
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

func runVerifyImport(args []string, d DBDeps) int {
	for _, arg := range args {
		if arg == "help" || arg == "-h" || arg == "--help" {
			PrintDBHelp(d.Out)
			return 0
		}
	}

	fs := flag.NewFlagSet("db:verify-import", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	jsonOutput := fs.Bool("json", false, "print JSON output")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(d.Err, "invalid db:verify-import arguments: %v\n", err)
		fmt.Fprintln(d.Err, "usage: ship db:verify-import [--json]")
		return 1
	}
	if fs.NArg() > 0 {
		fmt.Fprintln(d.Err, "usage: ship db:verify-import [--json]")
		return 1
	}
	if d.LoadConfig == nil {
		fmt.Fprintln(d.Err, "db:verify-import requires config loader dependency")
		return 1
	}

	cfg, err := d.LoadConfig()
	if err != nil {
		fmt.Fprintf(d.Err, "failed to load config: %v\n", err)
		return 1
	}

	report := buildDBVerifyImportReport(cfg.RuntimeMetadata().Database)
	if *jsonOutput {
		enc := json.NewEncoder(d.Out)
		enc.SetIndent("", "  ")
		if err := enc.Encode(report); err != nil {
			fmt.Fprintf(d.Err, "failed to encode db:verify-import output: %v\n", err)
			return 1
		}
		return 0
	}

	printDBVerifyImportReport(d.Out, report)
	return 0
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

func runConsole(args []string, d DBDeps) int {
	if len(args) != 0 {
		fmt.Fprintln(d.Err, "usage: ship db:console")
		return 1
	}

	dbURL, err := d.ResolveDBURL()
	if err != nil {
		fmt.Fprintf(d.Err, "failed to resolve database URL: %v\n", err)
		return 1
	}

	driver, err := resolveConsoleDriver(d, dbURL)
	if err != nil {
		fmt.Fprintf(d.Err, "failed to resolve database driver: %v\n", err)
		return 1
	}

	name, cmdArgs, err := dbConsoleCommand(driver, dbURL)
	if err != nil {
		fmt.Fprintf(d.Err, "failed to build db shell command: %v\n", err)
		return 1
	}
	return d.RunCmd(name, cmdArgs...)
}

func resolveConsoleDriver(d DBDeps, dbURL string) (string, error) {
	if d.ResolveDBDriver != nil {
		driver, err := d.ResolveDBDriver()
		if err != nil {
			return "", err
		}
		normalized := normalizeConsoleDriver(driver)
		if normalized != "" {
			return normalized, nil
		}
		if strings.TrimSpace(driver) != "" {
			return "", fmt.Errorf("unsupported DB driver %q (supported: postgres, mysql, sqlite)", driver)
		}
	}

	if inferred := inferConsoleDriverFromURL(dbURL); inferred != "" {
		return inferred, nil
	}
	return "", fmt.Errorf("unable to determine DB driver from URL")
}

func normalizeConsoleDriver(driver string) string {
	switch strings.ToLower(strings.TrimSpace(driver)) {
	case "postgres", "postgresql", "pgx":
		return "postgres"
	case "mysql", "mariadb":
		return "mysql"
	case "sqlite", "sqlite3":
		return "sqlite"
	default:
		return ""
	}
}

func inferConsoleDriverFromURL(dbURL string) string {
	if strings.HasPrefix(dbURL, "sqlite://") || strings.HasPrefix(dbURL, "sqlite3://") {
		return "sqlite"
	}
	u, err := url.Parse(dbURL)
	if err != nil {
		return ""
	}
	return normalizeConsoleDriver(u.Scheme)
}

func dbConsoleCommand(driver string, dbURL string) (string, []string, error) {
	switch normalizeConsoleDriver(driver) {
	case "postgres":
		return "psql", []string{dbURL}, nil
	case "mysql":
		args, err := mysqlConsoleArgs(dbURL)
		if err != nil {
			return "", nil, err
		}
		return "mysql", args, nil
	case "sqlite":
		path, err := sqliteConsolePath(dbURL)
		if err != nil {
			return "", nil, err
		}
		return "sqlite3", []string{path}, nil
	default:
		return "", nil, fmt.Errorf("unsupported DB driver %q", driver)
	}
}

func mysqlConsoleArgs(dbURL string) ([]string, error) {
	u, err := url.Parse(dbURL)
	if err != nil {
		return nil, err
	}
	if normalizeConsoleDriver(u.Scheme) != "mysql" {
		return nil, fmt.Errorf("expected mysql URL, got %q", u.Scheme)
	}
	host := strings.TrimSpace(u.Hostname())
	if host == "" {
		return nil, fmt.Errorf("mysql URL is missing host")
	}

	args := []string{"--host", host}
	if port := strings.TrimSpace(u.Port()); port != "" {
		args = append(args, "--port", port)
	}
	if user := strings.TrimSpace(u.User.Username()); user != "" {
		args = append(args, "--user", user)
	}
	if pass, ok := u.User.Password(); ok && strings.TrimSpace(pass) != "" {
		args = append(args, "--password="+pass)
	}
	if dbName := strings.TrimSpace(strings.TrimPrefix(u.Path, "/")); dbName != "" {
		args = append(args, dbName)
	}
	return args, nil
}

func sqliteConsolePath(dbURL string) (string, error) {
	switch {
	case strings.HasPrefix(dbURL, "sqlite://"):
		return normalizeSQLitePath(strings.TrimPrefix(dbURL, "sqlite://"))
	case strings.HasPrefix(dbURL, "sqlite3://"):
		return normalizeSQLitePath(strings.TrimPrefix(dbURL, "sqlite3://"))
	default:
		u, err := url.Parse(dbURL)
		if err != nil {
			return "", err
		}
		if normalizeConsoleDriver(u.Scheme) != "sqlite" {
			return "", fmt.Errorf("expected sqlite URL, got %q", u.Scheme)
		}
		return normalizeSQLitePath(strings.TrimPrefix(dbURL, u.Scheme+"://"))
	}
}

func normalizeSQLitePath(raw string) (string, error) {
	dsn := strings.TrimSpace(raw)
	if idx := strings.Index(dsn, "?"); idx >= 0 {
		dsn = dsn[:idx]
	}
	dsn = strings.TrimSpace(dsn)
	if dsn == "" {
		return "", fmt.Errorf("sqlite URL is missing database path")
	}
	return dsn, nil
}

func resolveBobgenConfigs(d DBDeps, explicitConfig string) ([]string, error) {
	if strings.TrimSpace(explicitConfig) != "" && explicitConfig != filepath.ToSlash(filepath.Join("db", "bobgen.yaml")) {
		return []string{explicitConfig}, nil
	}
	configs := []string{filepath.ToSlash(filepath.Join("db", "bobgen.yaml"))}
	if d.FindGoModule == nil {
		return configs, nil
	}

	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	root, _, err := d.FindGoModule(wd)
	if err != nil {
		return nil, err
	}
	manifestPath := filepath.Join(root, "config", "modules.yaml")
	if !pathExists(manifestPath) {
		return configs, nil
	}

	manifest, err := rt.LoadModulesManifest(manifestPath)
	if err != nil {
		return nil, err
	}
	for _, module := range manifest.Modules {
		configRel := filepath.ToSlash(filepath.Join("modules", module, "db", "bobgen.yaml"))
		configAbs := filepath.Join(root, filepath.FromSlash(configRel))
		if !pathExists(configAbs) {
			return nil, fmt.Errorf("enabled module %q missing bobgen config: %s", module, configRel)
		}
		configs = append(configs, configRel)
	}
	return configs, nil
}

func IsLocalDBURL(dbURL string) bool {
	if strings.HasPrefix(dbURL, "sqlite://") {
		return true
	}
	u, err := url.Parse(dbURL)
	if err != nil {
		return false
	}
	host := strings.ToLower(u.Hostname())
	if host == "" {
		return false
	}
	for _, allowed := range localDBHosts() {
		if host == allowed {
			return true
		}
	}
	return false
}

func localDBHosts() []string {
	raw := strings.TrimSpace(os.Getenv("SHIP_LOCAL_DB_HOSTS"))
	if raw == "" {
		return []string{"localhost", "127.0.0.1", "::1", "db", "postgres", "mysql"}
	}
	var out []string
	for _, part := range strings.Split(raw, ",") {
		v := strings.ToLower(strings.TrimSpace(part))
		if v != "" {
			out = append(out, v)
		}
	}
	if len(out) == 0 {
		return []string{"localhost", "127.0.0.1", "::1"}
	}
	return out
}

func isProductionEnv() bool {
	env := strings.ToLower(strings.TrimSpace(rt.ResolveAppEnvironment()))
	return env == "production" || env == "prod"
}

func printPlan(w io.Writer, action, dbURL string, local bool, steps []string, seed, dryRun bool) {
	fmt.Fprintf(w, "DB %s plan:\n", action)
	fmt.Fprintf(w, "- url: %s\n", dbURL)
	fmt.Fprintf(w, "- local: %t\n", local)
	for _, step := range steps {
		fmt.Fprintf(w, "- step: %s\n", step)
	}
	if seed {
		fmt.Fprintln(w, "- step: go run ./cmd/seed/main.go")
	}
	if dryRun {
		fmt.Fprintln(w, "- mode: dry-run (no commands executed)")
	}
}

func PrintDBHelp(w io.Writer) {
	fmt.Fprintln(w, "ship db commands:")
	fmt.Fprintln(w, "  ship db:create [--dry-run]                                 Validate DB connectivity and migration table reachability")
	fmt.Fprintln(w, "  ship db:generate [--config <path>] [--dry-run]            Generate DB access code from bobgen config")
	fmt.Fprintln(w, "  ship db:export [--json]                                   Show the SQLite export manifest")
	fmt.Fprintln(w, "  ship db:import [--json]                                   Show the manual SQLite export import plan")
	fmt.Fprintln(w, "  ship db:promote [--json]                                  Show the manual SQLite-to-Postgres promotion plan")
	fmt.Fprintln(w, "  ship db:make <migration_name>                              Create a new SQL migration file")
	fmt.Fprintln(w, "  ship db:migrate                                            Apply pending migrations")
	fmt.Fprintln(w, "  ship db:status                                             Show migration status")
	fmt.Fprintln(w, "  ship db:verify-import [--json]                             Show post-import verification checks")
	fmt.Fprintln(w, "  ship db:console                                            Open database shell client")
	fmt.Fprintln(w, "  ship db:reset [--seed] [--force] [--yes] [--dry-run]      Reset and re-apply migrations (destructive)")
	fmt.Fprintln(w, "  ship db:drop [--force] [--yes] [--dry-run]                Revert all migrations (destructive)")
	fmt.Fprintln(w, "  ship db:rollback [amount]                                  Roll back one or more migration steps")
	fmt.Fprintln(w, "  ship db:seed                                               Run database seed command")
}
