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

	var report map[string]any
	if err := json.Unmarshal(out.Bytes(), &report); err != nil {
		t.Fatalf("stdout should be valid JSON: %v\n%s", err, out.String())
	}
	if got := report["schema_version"]; got != "upgrade-readiness-v1" {
		t.Fatalf("schema_version=%v want upgrade-readiness-v1", got)
	}
	if got := report["target_version"]; got != "v3.27.0" {
		t.Fatalf("target_version=%v want v3.27.0", got)
	}
	if got := report["ready"]; got != true {
		t.Fatalf("ready=%v want true", got)
	}
	if got := report["blockers"]; got == nil {
		t.Fatalf("expected blockers array in report")
	}
	if got := report["remediation_hints"]; got == nil {
		t.Fatalf("expected remediation_hints array in report")
	}
	if got := report["planned_changes"]; got == nil {
		t.Fatalf("expected planned_changes array in report")
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

	for _, field := range []string{"rollback_target", "canary", "verification"} {
		if got := report[field]; got == nil {
			t.Fatalf("expected upgrade readiness report to include %q contract field", field)
		}
	}
}
