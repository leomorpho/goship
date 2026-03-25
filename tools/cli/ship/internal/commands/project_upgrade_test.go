package commands

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestUpgradeRewriteGooseVersion_CodemodFixtures(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		beforeFile  string
		afterFile   string
		target      string
		wantOld     string
		wantChanged bool
	}{
		{
			name:        "canonical v3 ref version bump",
			beforeFile:  "testdata/upgrade_codemods/goose_v3_before.go",
			afterFile:   "testdata/upgrade_codemods/goose_v3_after.go",
			target:      "v3.27.0",
			wantOld:     "v3.26.0",
			wantChanged: true,
		},
		{
			name:        "legacy ref canonicalization and version bump",
			beforeFile:  "testdata/upgrade_codemods/goose_legacy_before.go",
			afterFile:   "testdata/upgrade_codemods/goose_legacy_after.go",
			target:      "v3.27.0",
			wantOld:     "v3.25.1",
			wantChanged: true,
		},
		{
			name:        "legacy ref canonicalization with same version",
			beforeFile:  "testdata/upgrade_codemods/goose_legacy_same_version_before.go",
			afterFile:   "testdata/upgrade_codemods/goose_legacy_same_version_after.go",
			target:      "v3.26.0",
			wantOld:     "v3.26.0",
			wantChanged: true,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			path := filepath.Join(root, "cli.go")
			if err := os.WriteFile(path, fixtureText(t, tc.beforeFile), 0o644); err != nil {
				t.Fatal(err)
			}

			old, updated, changed, err := RewriteGooseVersion(path, tc.target)
			if err != nil {
				t.Fatalf("RewriteGooseVersion failed: %v", err)
			}
			if old != tc.wantOld {
				t.Fatalf("old=%q want %q", old, tc.wantOld)
			}
			if changed != tc.wantChanged {
				t.Fatalf("changed=%v want %v", changed, tc.wantChanged)
			}

			want := string(fixtureText(t, tc.afterFile))
			if updated != want {
				t.Fatalf("updated text mismatch for %s", tc.name)
			}
		})
	}
}

func fixtureText(t *testing.T, relPath string) []byte {
	t.Helper()
	b, err := os.ReadFile(relPath)
	if err != nil {
		t.Fatalf("read fixture %s: %v", relPath, err)
	}
	return b
}

func TestUpgradeApplyRewrite_RollsBackOnVerificationFailure(t *testing.T) {
	t.Parallel()

	type writeCall struct {
		path string
		body string
	}
	var writes []writeCall
	writeFile := func(path string, data []byte, _ os.FileMode) error {
		writes = append(writes, writeCall{path: path, body: string(data)})
		return nil
	}
	readFile := func(string) ([]byte, error) {
		return []byte("unexpected"), nil
	}

	err := applyUpgradeRewrite("/tmp/cli.go", "new content", "old content", writeFile, readFile)
	if err == nil {
		t.Fatal("expected rollback error")
	}
	if !strings.Contains(err.Error(), "rolled back") {
		t.Fatalf("error should mention rollback, got: %v", err)
	}
	if len(writes) != 2 {
		t.Fatalf("writes=%d want 2", len(writes))
	}
	if writes[0].body != "new content" {
		t.Fatalf("first write=%q want new content", writes[0].body)
	}
	if writes[1].body != "old content" {
		t.Fatalf("second write=%q want old content rollback", writes[1].body)
	}
}

func TestUpgradeApplyRewrite_NoRollbackWhenInitialWriteFails(t *testing.T) {
	t.Parallel()

	var writes int
	writeFile := func(string, []byte, os.FileMode) error {
		writes++
		return os.ErrPermission
	}
	readFile := func(string) ([]byte, error) {
		t.Fatal("readFile should not be called when initial write fails")
		return nil, nil
	}

	err := applyUpgradeRewrite("/tmp/cli.go", "new content", "old content", writeFile, readFile)
	if err == nil {
		t.Fatal("expected write failure")
	}
	if writes != 1 {
		t.Fatalf("writes=%d want 1", writes)
	}
}

func TestRewriteGooseVersion(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cli.go")
	input := `package ship

const (
	gooseGoRunRef = "github.com/pressly/goose/v3/cmd/goose@v3.26.0"
)
`
	if err := os.WriteFile(path, []byte(input), 0o644); err != nil {
		t.Fatal(err)
	}

	old, updated, changed, err := RewriteGooseVersion(path, "v3.27.0")
	if err != nil {
		t.Fatalf("rewriteGooseVersion failed: %v", err)
	}
	if !changed {
		t.Fatal("expected changed=true")
	}
	if old != "v3.26.0" {
		t.Fatalf("old=%q want %q", old, "v3.26.0")
	}
	if !strings.Contains(updated, `gooseGoRunRef = "github.com/pressly/goose/v3/cmd/goose@v3.27.0"`) {
		t.Fatalf("updated text missing target version:\n%s", updated)
	}
}

func TestRunUpgrade(t *testing.T) {
	t.Run("missing version", func(t *testing.T) {
		out := &bytes.Buffer{}
		errOut := &bytes.Buffer{}
		code := RunUpgrade([]string{}, UpgradeDeps{Out: out, Err: errOut, FindGoModule: findGoModuleTest})
		if code != 1 {
			t.Fatalf("code=%d want=1", code)
		}
		if !strings.Contains(errOut.String(), "missing required --to version") {
			t.Fatalf("stderr=%q", errOut.String())
		}
	})

	t.Run("invalid version format", func(t *testing.T) {
		out := &bytes.Buffer{}
		errOut := &bytes.Buffer{}
		code := RunUpgrade([]string{"--to", "3.27.0"}, UpgradeDeps{Out: out, Err: errOut, FindGoModule: findGoModuleTest})
		if code != 1 {
			t.Fatalf("code=%d want=1", code)
		}
		if !strings.Contains(errOut.String(), "version must start with 'v'") {
			t.Fatalf("stderr=%q", errOut.String())
		}
	})

	t.Run("unexpected positional args", func(t *testing.T) {
		out := &bytes.Buffer{}
		errOut := &bytes.Buffer{}
		code := RunUpgrade([]string{"goose", "--to", "v3.27.0"}, UpgradeDeps{Out: out, Err: errOut, FindGoModule: findGoModuleTest})
		if code != 1 {
			t.Fatalf("code=%d want=1", code)
		}
		if !strings.Contains(errOut.String(), "unexpected upgrade arguments") {
			t.Fatalf("stderr=%q", errOut.String())
		}
	})

	t.Run("goose dry run", func(t *testing.T) {
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
		code := RunUpgrade([]string{"--to", "v3.27.0", "--dry-run"}, UpgradeDeps{Out: out, Err: errOut, FindGoModule: findGoModuleTest})
		if code != 0 {
			t.Fatalf("code=%d stderr=%s", code, errOut.String())
		}
		if !strings.Contains(out.String(), "dry-run: would update goose") {
			t.Fatalf("stdout=%q", out.String())
		}

		b, err := os.ReadFile(cliPath)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(b), "@v3.26.0") {
			t.Fatalf("cli.go should be unchanged in dry-run, got:\n%s", string(b))
		}
	})

	t.Run("goose preflight without apply does not write file", func(t *testing.T) {
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
		code := RunUpgrade([]string{"--to", "v3.27.0"}, UpgradeDeps{Out: out, Err: errOut, FindGoModule: findGoModuleTest})
		if code != 0 {
			t.Fatalf("code=%d stderr=%s", code, errOut.String())
		}
		b, err := os.ReadFile(cliPath)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(b), "@v3.26.0") {
			t.Fatalf("cli.go should remain unchanged until apply mode, got:\n%s", string(b))
		}
		if !strings.Contains(out.String(), "preflight: no files were written") {
			t.Fatalf("stdout=%q", out.String())
		}
		if !strings.Contains(out.String(), "ship upgrade apply --to v3.27.0") {
			t.Fatalf("stdout should include explicit apply follow-up command, got:\n%s", out.String())
		}
	})

	t.Run("goose apply writes file", func(t *testing.T) {
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
		code := RunUpgrade([]string{"apply", "--to", "v3.27.0"}, UpgradeDeps{Out: out, Err: errOut, FindGoModule: findGoModuleTest})
		if code != 0 {
			t.Fatalf("code=%d stderr=%s", code, errOut.String())
		}
		b, err := os.ReadFile(cliPath)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(b), "@v3.27.0") {
			t.Fatalf("expected goose version update in cli.go, got:\n%s", string(b))
		}
	})

	t.Run("stale convention emits blocker report", func(t *testing.T) {
		root := t.TempDir()
		if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/demo\n\ngo 1.25\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		cliPath := filepath.Join(root, "tools", "cli", "ship", "internal", "cli", "cli.go")
		if err := os.MkdirAll(filepath.Dir(cliPath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(cliPath, fixtureText(t, "testdata/upgrade_codemods/goose_stale_convention.go"), 0o644); err != nil {
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
		if code != 1 {
			t.Fatalf("code=%d want 1", code)
		}

		var report UpgradeReadinessReport
		if err := json.Unmarshal(out.Bytes(), &report); err != nil {
			t.Fatalf("stdout should be valid JSON report: %v\n%s", err, out.String())
		}
		if report.Ready {
			t.Fatalf("ready=%v want false", report.Ready)
		}
		if got := report.Result.Outcome; got != "blocked" {
			t.Fatalf("result.outcome=%q want blocked", got)
		}
		if len(report.Blockers) != 1 {
			t.Fatalf("blockers=%d want 1", len(report.Blockers))
		}
		if got := report.Blockers[0].ID; got != "upgrade.convention_drift" {
			t.Fatalf("blockers[0].id=%q want upgrade.convention_drift", got)
		}
		if got := report.Blockers[0].Classification; got != "convention-drift" {
			t.Fatalf("blockers[0].classification=%q want convention-drift", got)
		}
		if got := len(report.PlannedChanges); got != 0 {
			t.Fatalf("planned_changes=%d want 0 when blocked", got)
		}
	})
}

func TestRunUpgradeApply_OutputGolden(t *testing.T) {
	packageDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

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
	code := RunUpgrade([]string{"apply", "--to", "v3.27.0"}, UpgradeDeps{Out: out, Err: errOut, FindGoModule: findGoModuleTest})
	if code != 0 {
		t.Fatalf("code=%d stderr=%s", code, errOut.String())
	}
	normalized := strings.ReplaceAll(out.String(), root, "<root>")
	assertCLIGoldenSnapshot(t, packageDir, "upgrade_apply_human.golden", normalized)
}

func TestComputeSafeUpgradeSteps_RepresentativeVersions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		current string
		target  string
		wantTo  []string
	}{
		{
			name:    "single patch hop",
			current: "v3.26.0",
			target:  "v3.26.3",
			wantTo:  []string{"v3.26.3"},
		},
		{
			name:    "multi minor hop",
			current: "v3.26.0",
			target:  "v3.29.2",
			wantTo:  []string{"v3.27.0", "v3.28.0", "v3.29.2"},
		},
		{
			name:    "major and minor hop",
			current: "v3.26.4",
			target:  "v4.2.1",
			wantTo:  []string{"v4.0.0", "v4.1.0", "v4.2.1"},
		},
		{
			name:    "already pinned",
			current: "v3.27.0",
			target:  "v3.27.0",
			wantTo:  []string{},
		},
		{
			name:    "fallback when current cannot be parsed",
			current: "legacy",
			target:  "v3.27.0",
			wantTo:  []string{"v3.27.0"},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			steps := computeSafeUpgradeSteps(tt.current, tt.target)
			gotTo := make([]string, 0, len(steps))
			for i, step := range steps {
				gotTo = append(gotTo, step.To)
				if step.Command != "ship upgrade apply --to "+step.To {
					t.Fatalf("step[%d].command=%q want %q", i, step.Command, "ship upgrade apply --to "+step.To)
				}
				if i == 0 {
					if step.From != tt.current {
						t.Fatalf("step[0].from=%q want %q", step.From, tt.current)
					}
					continue
				}
				if step.From != steps[i-1].To {
					t.Fatalf("step[%d].from=%q should chain from prior to=%q", i, step.From, steps[i-1].To)
				}
			}
			if !reflect.DeepEqual(gotTo, tt.wantTo) {
				t.Fatalf("planned step targets=%v want %v", gotTo, tt.wantTo)
			}
		})
	}
}
