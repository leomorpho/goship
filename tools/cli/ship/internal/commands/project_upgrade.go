package commands

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	gooseRefPattern = regexp.MustCompile(`(?m)^(\s*(?:const\s+)?gooseGoRunRef\s*=\s*"github\.com/pressly/goose/v3/cmd/goose@)v[^"]+("\s*)$`)
)

type UpgradeDeps struct {
	Out          io.Writer
	Err          io.Writer
	FindGoModule func(start string) (string, string, error)
}

type UpgradeReadinessReport struct {
	SchemaVersion    string                 `json:"schema_version"`
	TargetVersion    string                 `json:"target_version"`
	Ready            bool                   `json:"ready"`
	RollbackTarget   string                 `json:"rollback_target"`
	Canary           UpgradeCanaryPlan      `json:"canary"`
	Verification     UpgradeVerification    `json:"verification"`
	Blockers         []UpgradeReadinessItem `json:"blockers"`
	RemediationHints []string               `json:"remediation_hints"`
	PlannedChanges   []UpgradePlannedChange `json:"planned_changes"`
}

type UpgradeReadinessItem struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Remediation string `json:"remediation"`
}

type UpgradePlannedChange struct {
	File    string `json:"file"`
	Current string `json:"current"`
	Target  string `json:"target"`
}

type UpgradeCanaryPlan struct {
	Strategy string `json:"strategy"`
	Scope    string `json:"scope"`
}

type UpgradeVerification struct {
	Command string `json:"command"`
	Note    string `json:"note"`
}

func RunUpgrade(args []string, d UpgradeDeps) int {
	for _, arg := range args {
		if arg == "-h" || arg == "--help" || arg == "help" {
			PrintUpgradeHelp(d.Out)
			return 0
		}
	}
	fs := flag.NewFlagSet("upgrade", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	to := fs.String("to", "", "target pinned version, e.g. v0.3.1001")
	contractVersion := fs.String("contract-version", upgradeReadinessSchemaVersion, "required upgrade readiness contract version")
	dryRun := fs.Bool("dry-run", false, "print planned file changes without writing")
	jsonOut := fs.Bool("json", false, "emit machine-readable upgrade readiness report without writing")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(d.Err, "invalid upgrade arguments: %v\n", err)
		return 1
	}
	if fs.NArg() > 0 {
		fmt.Fprintf(d.Err, "unexpected upgrade arguments: %v\n", fs.Args())
		PrintUpgradeHelp(d.Err)
		return 1
	}
	if strings.TrimSpace(*to) == "" {
		fmt.Fprintln(d.Err, "missing required --to version")
		return 1
	}
	if !strings.HasPrefix(*to, "v") {
		fmt.Fprintln(d.Err, "version must start with 'v' (example: v0.3.1001)")
		return 1
	}
	if !isSupportedUpgradeContractVersion(strings.TrimSpace(*contractVersion)) {
		fmt.Fprintf(d.Err, "unsupported upgrade contract version %q (supported: %s)\n", strings.TrimSpace(*contractVersion), upgradeReadinessSchemaVersion)
		return 1
	}

	wd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(d.Err, "failed to resolve working directory: %v\n", err)
		return 1
	}
	root, _, err := d.FindGoModule(wd)
	if err != nil {
		fmt.Fprintf(d.Err, "failed to resolve project root (go.mod): %v\n", err)
		return 1
	}

	return upgradeGoose(d, root, *to, *dryRun, *jsonOut)
}

func upgradeGoose(d UpgradeDeps, root, version string, dryRun, jsonOut bool) int {
	path := filepath.Join(root, "tools", "cli", "ship", "internal", "cli", "cli.go")
	old, newText, changed, err := RewriteGooseVersion(path, version)
	if err != nil {
		fmt.Fprintf(d.Err, "failed to update goose version: %v\n", err)
		return 1
	}
	report := buildUpgradeReadinessReport(path, old, version, changed)
	if jsonOut {
		if err := json.NewEncoder(d.Out).Encode(report); err != nil {
			fmt.Fprintf(d.Err, "failed to encode upgrade readiness report: %v\n", err)
			return 1
		}
		return 0
	}
	if !changed {
		fmt.Fprintf(d.Out, "goose already pinned to %s in %s\n", version, path)
		return 0
	}
	if dryRun {
		fmt.Fprintf(d.Out, "dry-run: would update goose in %s: %s -> %s\n", path, old, version)
		return 0
	}
	if err := os.WriteFile(path, []byte(newText), 0o644); err != nil {
		fmt.Fprintf(d.Err, "failed to write %s: %v\n", path, err)
		return 1
	}
	fmt.Fprintf(d.Out, "updated goose pin in %s: %s -> %s\n", path, old, version)
	return 0
}

func RewriteGooseVersion(path, target string) (oldVersion string, rewritten string, changed bool, err error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", "", false, err
	}
	text := string(b)
	match := gooseRefPattern.FindStringSubmatch(text)
	if len(match) == 0 {
		return "", "", false, fmt.Errorf("gooseGoRunRef constant not found in %s", path)
	}
	full := match[0]
	prefix := match[1]
	suffix := match[2]
	old := strings.TrimSuffix(strings.TrimPrefix(full, prefix), suffix)
	if old == target {
		return old, text, false, nil
	}
	replacement := prefix + target + suffix
	updated := gooseRefPattern.ReplaceAllString(text, replacement)
	return old, updated, true, nil
}

func buildUpgradeReadinessReport(path, currentVersion, targetVersion string, changed bool) UpgradeReadinessReport {
	dryRunCommand := fmt.Sprintf("ship upgrade --to %s --dry-run", targetVersion)
	applyCommand := fmt.Sprintf("ship upgrade --to %s", targetVersion)

	report := UpgradeReadinessReport{
		SchemaVersion:  upgradeReadinessSchemaVersion,
		TargetVersion:  targetVersion,
		Ready:          true,
		RollbackTarget: currentVersion,
		Canary: UpgradeCanaryPlan{
			Strategy: "cli-pin-preflight",
			Scope:    "single pinned goose reference",
		},
		Verification: UpgradeVerification{
			Command: dryRunCommand,
			Note:    "Review the readiness report and planned mutation before writing the new pin.",
		},
		Blockers: []UpgradeReadinessItem{},
		RemediationHints: []string{
			"Review the readiness report before mutating pinned versions.",
			fmt.Sprintf("Use %s to preview the text mutation plan.", dryRunCommand),
			fmt.Sprintf("Run %s after the readiness report is accepted.", applyCommand),
		},
		PlannedChanges: []UpgradePlannedChange{},
	}
	if changed {
		report.PlannedChanges = append(report.PlannedChanges, UpgradePlannedChange{
			File:    path,
			Current: currentVersion,
			Target:  targetVersion,
		})
	}
	return report
}

func PrintUpgradeHelp(w io.Writer) {
	fmt.Fprintln(w, "ship upgrade commands:")
	fmt.Fprintln(w, "  ship upgrade --to <version> [--contract-version <schema>] [--dry-run] [--json]  Update pinned CLI tooling references and surface the upgrade readiness report/blocker schema contract (currently Goose only; no auto-latest)")
}
