package generators

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseMakeModuleArgs(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr string
		check   func(t *testing.T, opts ModuleMakeOptions)
	}{
		{
			name: "defaults",
			args: []string{"EmailSubscriptions"},
			check: func(t *testing.T, opts ModuleMakeOptions) {
				if opts.Path != "modules" {
					t.Fatalf("path = %q", opts.Path)
				}
				if opts.ModuleBase != "github.com/leomorpho/goship-modules" {
					t.Fatalf("module-base = %q", opts.ModuleBase)
				}
			},
		},
		{
			name: "full flags",
			args: []string{"EmailSubscriptions", "--path", "pkg/custom", "--module-base", "example.com/mods", "--dry-run", "--force"},
			check: func(t *testing.T, opts ModuleMakeOptions) {
				if opts.Path != "pkg/custom" || opts.ModuleBase != "example.com/mods" {
					t.Fatalf("unexpected opts: %+v", opts)
				}
				if !opts.DryRun || !opts.Force {
					t.Fatalf("expected dry-run+force in %+v", opts)
				}
			},
		},
		{
			name:    "missing name",
			args:    nil,
			wantErr: "usage: ship make:module",
		},
		{
			name:    "unknown option",
			args:    []string{"EmailSubscriptions", "--wat"},
			wantErr: "unknown option",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts, err := ParseMakeModuleArgs(tt.args)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("err=%v want contains %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("parse err: %v", err)
			}
			if tt.check != nil {
				tt.check(t, opts)
			}
		})
	}
}

func TestRunMakeModule_DryRun(t *testing.T) {
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	code := RunMakeModule([]string{"EmailSubscriptions", "--dry-run"}, ModuleDeps{Out: out, Err: errOut, PathExists: testHasFile})
	if code != 0 {
		t.Fatalf("exit=%d stderr=%s", code, errOut.String())
	}
	if !strings.Contains(out.String(), "Module scaffold plan (dry-run):") {
		t.Fatalf("unexpected output: %s", out.String())
	}
	if !strings.Contains(out.String(), "modules/emailsubscriptions/go.mod") {
		t.Fatalf("missing go.mod path in output: %s", out.String())
	}
}

func TestRunMakeModule_Integration(t *testing.T) {
	root := t.TempDir()
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
	code := RunMakeModule([]string{"EmailSubscriptions"}, ModuleDeps{Out: out, Err: errOut, PathExists: testHasFile})
	if code != 0 {
		t.Fatalf("exit=%d stderr=%s", code, errOut.String())
	}

	moduleDir := filepath.Join(root, "modules", "emailsubscriptions")
	required := []string{
		filepath.Join(moduleDir, "go.mod"),
		filepath.Join(moduleDir, "module.go"),
		filepath.Join(moduleDir, "contracts.go"),
		filepath.Join(moduleDir, "types.go"),
		filepath.Join(moduleDir, "errors.go"),
		filepath.Join(moduleDir, "service.go"),
		filepath.Join(moduleDir, "service_test.go"),
		filepath.Join(moduleDir, "db", "bobgen.yaml"),
		filepath.Join(moduleDir, "db", "migrate", "migrations", ".gitkeep"),
		filepath.Join(moduleDir, "db", "queries", ".gitkeep"),
		filepath.Join(moduleDir, "db", "gen", ".gitkeep"),
	}
	for _, p := range required {
		if !testHasFile(p) {
			t.Fatalf("missing scaffolded file: %s", p)
		}
	}

	goMod, err := os.ReadFile(filepath.Join(moduleDir, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(goMod), "module github.com/leomorpho/goship-modules/emailsubscriptions") {
		t.Fatalf("unexpected go.mod:\n%s", string(goMod))
	}

	moduleFile, err := os.ReadFile(filepath.Join(moduleDir, "module.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(moduleFile), "package emailsubscriptions") {
		t.Fatalf("unexpected module.go package:\n%s", string(moduleFile))
	}

	bobgen, err := os.ReadFile(filepath.Join(moduleDir, "db", "bobgen.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(bobgen), "modules/emailsubscriptions/db/migrate/migrations/*.sql") {
		t.Fatalf("unexpected bobgen pattern:\n%s", string(bobgen))
	}
}
