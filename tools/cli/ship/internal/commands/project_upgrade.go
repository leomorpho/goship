package commands

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var (
	canonicalGooseRefPrefix = "github.com/pressly/goose/v3/cmd/goose@"
	gooseRefCodemods        = []struct {
		pattern         *regexp.Regexp
		rewritePrefix   string
		versionSubmatch int
	}{
		{
			pattern:         regexp.MustCompile(`(?m)^(\s*(?:const\s+)?gooseGoRunRef\s*=\s*")github\.com/pressly/goose/v3/cmd/goose@(v[^"]+)("\s*)$`),
			rewritePrefix:   canonicalGooseRefPrefix,
			versionSubmatch: 2,
		},
		{
			pattern:         regexp.MustCompile(`(?m)^(\s*(?:const\s+)?gooseGoRunRef\s*=\s*")github\.com/pressly/goose/cmd/goose@(v[^"]+)("\s*)$`),
			rewritePrefix:   canonicalGooseRefPrefix,
			versionSubmatch: 2,
		},
	}
)

type UpgradeDeps struct {
	Out          io.Writer
	Err          io.Writer
	FindGoModule func(start string) (string, string, error)
}

type UpgradeReadinessReport struct {
	SchemaVersion         string                  `json:"schema_version"`
	BlockerClassification string                  `json:"blocker_classification"`
	TargetVersion         string                  `json:"target_version"`
	Ready                 bool                    `json:"ready"`
	RollbackTarget        string                  `json:"rollback_target"`
	Canary                UpgradeCanaryPlan       `json:"canary"`
	Verification          UpgradeVerification     `json:"verification"`
	Plan                  UpgradePlan             `json:"plan"`
	Result                UpgradeResult           `json:"result"`
	Blockers              []UpgradeReadinessItem  `json:"blockers"`
	ManualFollowUps       []UpgradeManualFollowUp `json:"manual_follow_ups"`
	RemediationHints      []string                `json:"remediation_hints"`
	PlannedChanges        []UpgradePlannedChange  `json:"planned_changes"`
}

type UpgradePlan struct {
	Strategy  string            `json:"strategy"`
	SafeSteps []UpgradeSafeStep `json:"safe_steps"`
}

type UpgradeSafeStep struct {
	From    string `json:"from"`
	To      string `json:"to"`
	Command string `json:"command"`
}

type UpgradeReadinessItem struct {
	ID             string `json:"id"`
	Classification string `json:"classification"`
	Title          string `json:"title"`
	Remediation    string `json:"remediation"`
}

type UpgradePlannedChange struct {
	File    string `json:"file"`
	Current string `json:"current"`
	Target  string `json:"target"`
}

type UpgradeManualFollowUp struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	Command     string `json:"command"`
}

type UpgradeCanaryPlan struct {
	Strategy string `json:"strategy"`
	Scope    string `json:"scope"`
}

type UpgradeVerification struct {
	Command string `json:"command"`
	Note    string `json:"note"`
}

type UpgradeResult struct {
	Mode    string `json:"mode"`
	Outcome string `json:"outcome"`
	Changed bool   `json:"changed"`
	Applied bool   `json:"applied"`
}

type upgradeConventionDriftError struct {
	path   string
	detail string
}

func (e upgradeConventionDriftError) Error() string {
	return fmt.Sprintf("canonical generated conventions drifted in %s: %s", e.path, e.detail)
}

func RunUpgrade(args []string, d UpgradeDeps) int {
	for _, arg := range args {
		if arg == "-h" || arg == "--help" || arg == "help" {
			PrintUpgradeHelp(d.Out)
			return 0
		}
	}
	applyMode := false
	if len(args) > 0 && strings.TrimSpace(args[0]) == "apply" {
		applyMode = true
		args = args[1:]
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

	return upgradeGoose(d, root, *to, *dryRun, *jsonOut, applyMode)
}

func upgradeGoose(d UpgradeDeps, root, version string, dryRun, jsonOut, applyMode bool) int {
	displayPath := filepath.ToSlash(filepath.Join("tools", "cli", "ship", "internal", "cli", "cli.go"))
	path := filepath.Join(root, filepath.FromSlash(displayPath))
	originalTextBytes, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(d.Err, "failed to read %s: %v\n", path, err)
		return 1
	}
	originalText := string(originalTextBytes)
	old, newText, changed, err := RewriteGooseVersion(path, version)
	if err != nil {
		var driftErr upgradeConventionDriftError
		if errors.As(err, &driftErr) {
			report := buildUpgradeDriftReport(displayPath, version, driftErr.detail)
			if jsonOut {
				if encodeErr := json.NewEncoder(d.Out).Encode(report); encodeErr != nil {
					fmt.Fprintf(d.Err, "failed to encode upgrade readiness report: %v\n", encodeErr)
					return 1
				}
				return 1
			}
			fmt.Fprintln(d.Err, "upgrade blocked: canonical generated conventions drifted")
			fmt.Fprintf(d.Err, "- %s: %s\n", displayPath, driftErr.detail)
			fmt.Fprintln(d.Err, "run `ship verify --profile strict` and align generated conventions before retrying upgrade apply")
			return 1
		}
		fmt.Fprintf(d.Err, "failed to update goose version: %v\n", err)
		return 1
	}
	report := buildUpgradeReadinessReport(displayPath, old, version, changed)
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
	if !applyMode {
		fmt.Fprintln(d.Out, "preflight: no files were written")
		fmt.Fprintf(d.Out, "planned rewrite: %s: %s -> %s\n", displayPath, old, version)
		fmt.Fprintf(d.Out, "next: ship upgrade apply --to %s\n", version)
		return 0
	}
	if err := applyUpgradeRewrite(path, newText, originalText, os.WriteFile, os.ReadFile); err != nil {
		fmt.Fprintf(d.Err, "failed to apply upgrade rewrite in %s: %v\n", path, err)
		return 1
	}
	fmt.Fprintln(d.Out, "applied upgrade rewrites:")
	fmt.Fprintf(d.Out, "- %s: %s -> %s\n", displayPath, old, version)
	return 0
}

func applyUpgradeRewrite(
	path, newText, oldText string,
	writeFile func(string, []byte, os.FileMode) error,
	readFile func(string) ([]byte, error),
) error {
	if err := writeFile(path, []byte(newText), 0o644); err != nil {
		return err
	}
	after, err := readFile(path)
	if err != nil {
		if rollbackErr := writeFile(path, []byte(oldText), 0o644); rollbackErr != nil {
			return fmt.Errorf("post-write verification read failed: %w; rollback failed: %v", err, rollbackErr)
		}
		return fmt.Errorf("post-write verification read failed: %w; rolled back", err)
	}
	if string(after) != newText {
		if rollbackErr := writeFile(path, []byte(oldText), 0o644); rollbackErr != nil {
			return fmt.Errorf("post-write verification mismatch; rollback failed: %v", rollbackErr)
		}
		return fmt.Errorf("post-write verification mismatch; rolled back")
	}
	return nil
}

func RewriteGooseVersion(path, target string) (oldVersion string, rewritten string, changed bool, err error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", "", false, err
	}
	text := string(b)
	for _, codemod := range gooseRefCodemods {
		match := codemod.pattern.FindStringSubmatch(text)
		if len(match) == 0 {
			continue
		}
		full := match[0]
		quotePrefix := match[1]
		old := match[codemod.versionSubmatch]
		quoteSuffix := match[3]
		replacement := quotePrefix + codemod.rewritePrefix + target + quoteSuffix
		if full == replacement {
			return old, text, false, nil
		}
		updated := codemod.pattern.ReplaceAllString(text, replacement)
		return old, updated, true, nil
	}
	return "", "", false, upgradeConventionDriftError{
		path:   path,
		detail: "gooseGoRunRef constant not found",
	}
}

func buildUpgradeDriftReport(path, targetVersion, detail string) UpgradeReadinessReport {
	return UpgradeReadinessReport{
		SchemaVersion:         upgradeReadinessSchemaVersion,
		BlockerClassification: "upgrade-blocker-classification-v1",
		TargetVersion:         targetVersion,
		Ready:                 false,
		RollbackTarget:        "",
		Canary: UpgradeCanaryPlan{
			Strategy: "cli-pin-preflight",
			Scope:    "single pinned goose reference",
		},
		Verification: UpgradeVerification{
			Command: "ship verify --profile strict",
			Note:    "Canonical generated conventions must be aligned before applying upgrade rewrites.",
		},
		Plan: UpgradePlan{
			Strategy:  "minor-boundary-bridge-v1",
			SafeSteps: []UpgradeSafeStep{},
		},
		Result: UpgradeResult{
			Mode:    "preflight",
			Outcome: "blocked",
			Changed: false,
			Applied: false,
		},
		Blockers: []UpgradeReadinessItem{
			{
				ID:             "upgrade.convention_drift",
				Classification: "convention-drift",
				Title:          "Canonical generated conventions have drifted",
				Remediation:    fmt.Sprintf("%s; run `ship verify --profile strict` and update stale generated files before retrying.", detail),
			},
		},
		ManualFollowUps: []UpgradeManualFollowUp{
			{
				ID:          "upgrade.convention.align",
				Description: "Align stale generated conventions before applying rewrites.",
				Command:     "ship verify --profile strict",
			},
		},
		RemediationHints: []string{
			"Upgrade apply rewrites only run against canonical generated conventions.",
			"Run ship verify --profile strict and repair stale generated files first.",
		},
		PlannedChanges: []UpgradePlannedChange{},
	}
}


func buildUpgradeReadinessReport(path, currentVersion, targetVersion string, changed bool) UpgradeReadinessReport {
	dryRunCommand := fmt.Sprintf("ship upgrade --to %s --dry-run", targetVersion)
	applyCommand := fmt.Sprintf("ship upgrade apply --to %s", targetVersion)
	planSteps := computeSafeUpgradeSteps(currentVersion, targetVersion)

	report := UpgradeReadinessReport{
		SchemaVersion:         upgradeReadinessSchemaVersion,
		BlockerClassification: "upgrade-blocker-classification-v1",
		TargetVersion:         targetVersion,
		Ready:                 true,
		RollbackTarget:        currentVersion,
		Canary: UpgradeCanaryPlan{
			Strategy: "cli-pin-preflight",
			Scope:    "single pinned goose reference",
		},
		Verification: UpgradeVerification{
			Command: dryRunCommand,
			Note:    "Review the readiness report and planned mutation before writing the new pin.",
		},
		Plan: UpgradePlan{
			Strategy:  "minor-boundary-bridge-v1",
			SafeSteps: []UpgradeSafeStep{},
		},
		Result: UpgradeResult{
			Mode:    "preflight",
			Outcome: "already-pinned",
			Changed: changed,
			Applied: false,
		},
		Blockers: []UpgradeReadinessItem{},
		ManualFollowUps: []UpgradeManualFollowUp{
			{
				ID:          "upgrade.readiness.review",
				Description: "Review the readiness report and planned mutation before writing the new pin.",
				Command:     dryRunCommand,
			},
			{
				ID:          "upgrade.pin.apply",
				Description: "Apply the pinned version mutation once readiness review is complete.",
				Command:     applyCommand,
			},
		},
		RemediationHints: []string{
			"Review the readiness report before mutating pinned versions.",
			fmt.Sprintf("Use %s to preview the text mutation plan.", dryRunCommand),
			fmt.Sprintf("Run %s after the readiness report is accepted.", applyCommand),
		},
		PlannedChanges: []UpgradePlannedChange{},
	}
	if changed {
		report.Result.Outcome = "planned-change"
		report.Plan.SafeSteps = planSteps
		report.PlannedChanges = append(report.PlannedChanges, UpgradePlannedChange{
			File:    path,
			Current: currentVersion,
			Target:  targetVersion,
		})
	}
	return report
}

type semverTriple struct {
	major int
	minor int
	patch int
}

func computeSafeUpgradeSteps(currentVersion, targetVersion string) []UpgradeSafeStep {
	current, okCurrent := parseSimpleSemver(currentVersion)
	target, okTarget := parseSimpleSemver(targetVersion)
	if !okCurrent || !okTarget {
		if currentVersion == targetVersion {
			return []UpgradeSafeStep{}
		}
		return []UpgradeSafeStep{
			{
				From:    currentVersion,
				To:      targetVersion,
				Command: fmt.Sprintf("ship upgrade apply --to %s", targetVersion),
			},
		}
	}
	if compareSemverTriple(current, target) >= 0 {
		return []UpgradeSafeStep{}
	}

	waypoints := make([]string, 0, 4)
	cursor := current
	for cursor.major < target.major {
		cursor.major++
		cursor.minor = 0
		cursor.patch = 0
		waypoints = append(waypoints, formatSimpleSemver(cursor))
	}
	for cursor.minor < target.minor {
		if cursor.minor+1 == target.minor && target.patch > 0 {
			break
		}
		cursor.minor++
		cursor.patch = 0
		waypoints = append(waypoints, formatSimpleSemver(cursor))
	}
	if cursor.patch != target.patch {
		waypoints = append(waypoints, formatSimpleSemver(target))
	} else if len(waypoints) == 0 || waypoints[len(waypoints)-1] != formatSimpleSemver(target) {
		waypoints = append(waypoints, formatSimpleSemver(target))
	}

	steps := make([]UpgradeSafeStep, 0, len(waypoints))
	from := currentVersion
	for _, to := range waypoints {
		if from == to {
			continue
		}
		steps = append(steps, UpgradeSafeStep{
			From:    from,
			To:      to,
			Command: fmt.Sprintf("ship upgrade apply --to %s", to),
		})
		from = to
	}
	return steps
}

func parseSimpleSemver(version string) (semverTriple, bool) {
	trimmed := strings.TrimSpace(version)
	if !strings.HasPrefix(trimmed, "v") {
		return semverTriple{}, false
	}
	parts := strings.Split(strings.TrimPrefix(trimmed, "v"), ".")
	if len(parts) != 3 {
		return semverTriple{}, false
	}
	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return semverTriple{}, false
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return semverTriple{}, false
	}
	patch, err := strconv.Atoi(parts[2])
	if err != nil {
		return semverTriple{}, false
	}
	return semverTriple{major: major, minor: minor, patch: patch}, true
}

func compareSemverTriple(left, right semverTriple) int {
	if left.major != right.major {
		return left.major - right.major
	}
	if left.minor != right.minor {
		return left.minor - right.minor
	}
	return left.patch - right.patch
}

func formatSimpleSemver(version semverTriple) string {
	return fmt.Sprintf("v%d.%d.%d", version.major, version.minor, version.patch)
}

func PrintUpgradeHelp(w io.Writer) {
	fmt.Fprintln(w, "ship upgrade commands:")
	fmt.Fprintln(w, "  ship upgrade --to <version> [--contract-version <schema>] [--dry-run] [--json]        Show the upgrade readiness report, blocker schema, and planned rewrites without writing")
	fmt.Fprintln(w, "  ship upgrade apply --to <version> [--contract-version <schema>] [--dry-run] [--json]  Apply the deterministic rewrite plan after review")
}
