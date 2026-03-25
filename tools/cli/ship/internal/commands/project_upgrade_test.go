package commands

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

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
