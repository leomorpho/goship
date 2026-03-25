package commands

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPrintUpgradeHelp_ListsReadinessReportContract_RedSpec(t *testing.T) {
	out := captureHelp(t, PrintUpgradeHelp)

	for _, want := range []string{
		"upgrade readiness report",
		"blocker schema",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("upgrade help should mention %q\n%s", want, out)
		}
	}
}

func TestRunUpgrade_JSONReadinessReport_RedSpec(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/demo\n\ngo 1.25\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cliPath := filepath.Join(root, "tools", "cli", "ship", "internal", "cli", "cli.go")
	if err := os.MkdirAll(filepath.Dir(cliPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cliPath, []byte(`package ship
const gooseGoRunRef = "github.com/pressly/goose/v3/cmd/goose@v3.26.0"
`), 0o644); err != nil {
		t.Fatal(err)
	}

	prevWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(prevWD) })
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	code := RunUpgrade([]string{"--to", "v3.27.0", "--json"}, UpgradeDeps{Out: out, Err: errOut, FindGoModule: findGoModuleTest})
	if code != 0 {
		t.Fatalf("code=%d stderr=%s", code, errOut.String())
	}

	var report UpgradeReadinessReport
	decoder := json.NewDecoder(bytes.NewReader(out.Bytes()))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&report); err != nil {
		t.Fatalf("stdout should be valid JSON: %v\n%s", err, out.String())
	}
	if got := report.SchemaVersion; got != "upgrade-readiness-v1" {
		t.Fatalf("schema_version=%v want upgrade-readiness-v1", got)
	}
	if got := report.BlockerClassification; got != "upgrade-blocker-classification-v1" {
		t.Fatalf("blocker_classification=%v want upgrade-blocker-classification-v1", got)
	}
	if got := report.TargetVersion; got != "v3.27.0" {
		t.Fatalf("target_version=%v want v3.27.0", got)
	}
	if got := report.Ready; got != true {
		t.Fatalf("ready=%v want true", got)
	}
	if report.RollbackTarget != "v3.26.0" {
		t.Fatalf("rollback_target=%q want v3.26.0", report.RollbackTarget)
	}
	if report.Canary.Strategy != "cli-pin-preflight" {
		t.Fatalf("canary.strategy=%q want cli-pin-preflight", report.Canary.Strategy)
	}
	if report.Canary.Scope != "single pinned goose reference" {
		t.Fatalf("canary.scope=%q want single pinned goose reference", report.Canary.Scope)
	}
	if report.Verification.Command != "ship upgrade --to v3.27.0 --dry-run" {
		t.Fatalf("verification.command=%q want concrete dry-run command", report.Verification.Command)
	}
	if report.Plan.Strategy != "minor-boundary-bridge-v1" {
		t.Fatalf("plan.strategy=%q want minor-boundary-bridge-v1", report.Plan.Strategy)
	}
	if len(report.Plan.SafeSteps) != 1 {
		t.Fatalf("plan.safe_steps=%d want 1", len(report.Plan.SafeSteps))
	}
	if report.Plan.SafeSteps[0].From != "v3.26.0" {
		t.Fatalf("plan.safe_steps[0].from=%q want v3.26.0", report.Plan.SafeSteps[0].From)
	}
	if report.Plan.SafeSteps[0].To != "v3.27.0" {
		t.Fatalf("plan.safe_steps[0].to=%q want v3.27.0", report.Plan.SafeSteps[0].To)
	}
	if report.Plan.SafeSteps[0].Command != "ship upgrade apply --to v3.27.0" {
		t.Fatalf("plan.safe_steps[0].command=%q want ship upgrade apply --to v3.27.0", report.Plan.SafeSteps[0].Command)
	}
	if len(report.Blockers) != 0 {
		t.Fatalf("blockers=%+v want empty", report.Blockers)
	}
	if len(report.ManualFollowUps) != 2 {
		t.Fatalf("manual_follow_ups=%d want 2", len(report.ManualFollowUps))
	}
	if got := report.ManualFollowUps[0].Command; got != "ship upgrade --to v3.27.0 --dry-run" {
		t.Fatalf("manual_follow_ups[0].command=%q want ship upgrade --to v3.27.0 --dry-run", got)
	}
	if got := report.ManualFollowUps[1].Command; got != "ship upgrade apply --to v3.27.0" {
		t.Fatalf("manual_follow_ups[1].command=%q want ship upgrade apply --to v3.27.0", got)
	}
	if len(report.RemediationHints) != 3 {
		t.Fatalf("remediation_hints=%d want 3", len(report.RemediationHints))
	}
	if got := report.RemediationHints[1]; got != "Use ship upgrade --to v3.27.0 --dry-run to preview the text mutation plan." {
		t.Fatalf("remediation_hints[1]=%q", got)
	}
	if got := report.Result.Mode; got != "preflight" {
		t.Fatalf("result.mode=%q want preflight", got)
	}
	if got := report.Result.Outcome; got != "planned-change" {
		t.Fatalf("result.outcome=%q want planned-change", got)
	}
	if got := report.Result.Changed; got != true {
		t.Fatalf("result.changed=%v want true", got)
	}
	if got := report.Result.Applied; got != false {
		t.Fatalf("result.applied=%v want false", got)
	}
	if len(report.PlannedChanges) != 1 {
		t.Fatalf("planned_changes=%d want 1", len(report.PlannedChanges))
	}
	change := report.PlannedChanges[0]
	if !strings.HasSuffix(change.File, filepath.Join("tools", "cli", "ship", "internal", "cli", "cli.go")) {
		t.Fatalf("planned_changes[0].file=%q should point at ship internal cli.go", change.File)
	}
	if change.Current != "v3.26.0" {
		t.Fatalf("planned_changes[0].current=%q want v3.26.0", change.Current)
	}
	if change.Target != "v3.27.0" {
		t.Fatalf("planned_changes[0].target=%q want v3.27.0", change.Target)
	}

	b, err := os.ReadFile(cliPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "@v3.26.0") {
		t.Fatalf("cli.go should remain unchanged for --json preflight, got:\n%s", string(b))
	}
}

func TestRunUpgrade_JSONIncludesRollbackAndCanaryContract_RedSpec(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/demo\n\ngo 1.25\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cliPath := filepath.Join(root, "tools", "cli", "ship", "internal", "cli", "cli.go")
	if err := os.MkdirAll(filepath.Dir(cliPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cliPath, []byte(`package ship
const gooseGoRunRef = "github.com/pressly/goose/v3/cmd/goose@v3.26.0"
`), 0o644); err != nil {
		t.Fatal(err)
	}

	prevWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(prevWD) })
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	code := RunUpgrade([]string{"--to", "v3.27.0", "--json"}, UpgradeDeps{Out: out, Err: errOut, FindGoModule: findGoModuleTest})
	if code != 0 {
		t.Fatalf("code=%d stderr=%s", code, errOut.String())
	}

	var report map[string]any
	if err := json.Unmarshal(out.Bytes(), &report); err != nil {
		t.Fatalf("stdout should be valid JSON: %v\n%s", err, out.String())
	}

	for _, field := range []string{"rollback_target", "canary", "verification", "plan", "blocker_classification", "manual_follow_ups", "result"} {
		if got := report[field]; got == nil {
			t.Fatalf("expected upgrade readiness report to include %q contract field", field)
		}
	}
}

func TestBuildUpgradeReadinessReport_UsesConcreteCommands_RedSpec(t *testing.T) {
	report := buildUpgradeReadinessReport(
		"/repo/tools/cli/ship/internal/cli/cli.go",
		"v3.26.0",
		"v3.27.0",
		true,
	)

	if report.Verification.Command != "ship upgrade --to v3.27.0 --dry-run" {
		t.Fatalf("verification.command=%q want concrete dry-run command", report.Verification.Command)
	}
	if got := report.RemediationHints[1]; got != "Use ship upgrade --to v3.27.0 --dry-run to preview the text mutation plan." {
		t.Fatalf("remediation_hints[1]=%q", got)
	}
	if got := report.RemediationHints[2]; got != "Run ship upgrade apply --to v3.27.0 after the readiness report is accepted." {
		t.Fatalf("remediation_hints[2]=%q", got)
	}
	if got := len(report.Plan.SafeSteps); got != 1 {
		t.Fatalf("plan.safe_steps=%d want 1", got)
	}
	if got := report.Plan.SafeSteps[0].Command; got != "ship upgrade apply --to v3.27.0" {
		t.Fatalf("plan.safe_steps[0].command=%q want ship upgrade apply --to v3.27.0", got)
	}
}

func TestRunUpgrade_RejectsUnsupportedContractVersion_RedSpec(t *testing.T) {
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	code := RunUpgrade([]string{"--to", "v3.27.0", "--contract-version", "upgrade-readiness-v9"}, UpgradeDeps{
		Out:          out,
		Err:          errOut,
		FindGoModule: findGoModuleTest,
	})
	if code != 1 {
		t.Fatalf("code=%d want 1", code)
	}
	if !strings.Contains(errOut.String(), "unsupported upgrade contract version") {
		t.Fatalf("stderr=%q", errOut.String())
	}
}
