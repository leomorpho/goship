package ship

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type fakeCall struct {
	name string
	args []string
}

type fakeRunner struct {
	calls    []fakeCall
	code     int
	err      error
	nextCode map[string]int
	nextErr  map[string]error
}

func (f *fakeRunner) Run(name string, args ...string) (int, error) {
	f.calls = append(f.calls, fakeCall{name: name, args: args})
	key := name + " " + strings.Join(args, " ")
	if err, ok := f.nextErr[key]; ok {
		return 1, err
	}
	if code, ok := f.nextCode[key]; ok {
		return code, nil
	}
	return f.code, f.err
}

func TestRun_DispatchAndArgs(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantCode   int
		wantCalls  []fakeCall
		wantOut    string
		wantErr    string
		runnerCode int
		runnerErr  error
	}{
		{
			name:      "no args prints root help",
			args:      nil,
			wantCode:  0,
			wantOut:   "ship - GoShip CLI",
			wantCalls: nil,
		},
		{
			name:      "unknown command",
			args:      []string{"wat"},
			wantCode:  1,
			wantErr:   "unknown command: wat",
			wantCalls: nil,
		},
		{
			name:     "new missing app name",
			args:     []string{"new"},
			wantCode: 1,
			wantErr:  "usage: ship new <app>",
		},
		{
			name:      "dev default",
			args:      []string{"dev"},
			wantCode:  0,
			wantCalls: []fakeCall{{name: "make", args: []string{"dev"}}},
		},
		{
			name:      "shipdev alias",
			args:      []string{"shipdev"},
			wantCode:  0,
			wantCalls: []fakeCall{{name: "make", args: []string{"dev"}}},
		},
		{
			name:      "dev worker positional",
			args:      []string{"dev", "worker"},
			wantCode:  0,
			wantCalls: []fakeCall{{name: "make", args: []string{"dev-worker"}}},
		},
		{
			name:      "dev all positional",
			args:      []string{"dev", "all"},
			wantCode:  0,
			wantCalls: []fakeCall{{name: "make", args: []string{"dev-full"}}},
		},
		{
			name:      "dev worker flag",
			args:      []string{"dev", "--worker"},
			wantCode:  0,
			wantCalls: []fakeCall{{name: "make", args: []string{"dev-worker"}}},
		},
		{
			name:      "dev all flag",
			args:      []string{"dev", "--all"},
			wantCode:  0,
			wantCalls: []fakeCall{{name: "make", args: []string{"dev-full"}}},
		},
		{
			name:     "dev both flags invalid",
			args:     []string{"dev", "--all", "--worker"},
			wantCode: 1,
			wantErr:  "cannot set both --worker and --all",
		},
		{
			name:     "dev unexpected arg invalid",
			args:     []string{"dev", "worker", "extra"},
			wantCode: 1,
			wantErr:  "unexpected dev arguments",
		},
		{
			name:     "dev help",
			args:     []string{"dev", "--help"},
			wantCode: 0,
			wantOut:  "ship dev commands:",
		},
		{
			name:      "test default",
			args:      []string{"test"},
			wantCode:  0,
			wantCalls: []fakeCall{{name: "make", args: []string{"test"}}},
		},
		{
			name:      "test integration",
			args:      []string{"test", "--integration"},
			wantCode:  0,
			wantCalls: []fakeCall{{name: "make", args: []string{"test-integration"}}},
		},
		{
			name:     "test invalid arg",
			args:     []string{"test", "extra"},
			wantCode: 1,
			wantErr:  "unexpected test arguments",
		},
		{
			name:     "test help",
			args:     []string{"test", "--help"},
			wantCode: 0,
			wantOut:  "ship test commands:",
		},
		{
			name:      "db create",
			args:      []string{"db", "create"},
			wantCode:  0,
			wantCalls: []fakeCall{{name: "make", args: []string{"up"}}},
		},
		{
			name:      "db migrate",
			args:      []string{"db", "migrate"},
			wantCode:  0,
			wantCalls: []fakeCall{{name: "make", args: []string{"migrate"}}},
		},
		{
			name:      "db seed",
			args:      []string{"db", "seed"},
			wantCode:  0,
			wantCalls: []fakeCall{{name: "make", args: []string{"seed"}}},
		},
		{
			name:     "db rollback default amount",
			args:     []string{"db", "rollback"},
			wantCode: 0,
			wantCalls: []fakeCall{{
				name: "atlas",
				args: []string{"migrate", "down", "--dir", atlasDir, "--url", atlasURL, "1"},
			}},
		},
		{
			name:     "db rollback explicit amount",
			args:     []string{"db", "rollback", "3"},
			wantCode: 0,
			wantCalls: []fakeCall{{
				name: "atlas",
				args: []string{"migrate", "down", "--dir", atlasDir, "--url", atlasURL, "3"},
			}},
		},
		{
			name:     "db rollback invalid amount",
			args:     []string{"db", "rollback", "x"},
			wantCode: 1,
			wantErr:  "invalid rollback amount",
		},
		{
			name:     "db rollback too many args",
			args:     []string{"db", "rollback", "1", "2"},
			wantCode: 1,
			wantErr:  "usage: ship db rollback [amount]",
		},
		{
			name:     "db missing subcommand",
			args:     []string{"db"},
			wantCode: 1,
			wantErr:  "ship db commands:",
		},
		{
			name:     "db help",
			args:     []string{"db", "help"},
			wantCode: 0,
			wantOut:  "ship db commands:",
		},
		{
			name:      "templ generate default path",
			args:      []string{"templ", "generate"},
			wantCode:  0,
			wantCalls: []fakeCall{{name: "templ", args: []string{"generate", "-path", "."}}},
		},
		{
			name:      "templ generate custom path",
			args:      []string{"templ", "generate", "--path", "app"},
			wantCode:  0,
			wantCalls: []fakeCall{{name: "templ", args: []string{"generate", "-path", "app"}}},
		},
		{
			name:      "templ generate single file",
			args:      []string{"templ", "generate", "--file", "app/goship/views/web/pages/home.templ"},
			wantCode:  0,
			wantCalls: []fakeCall{{name: "templ", args: []string{"generate", "-f", "app/goship/views/web/pages/home.templ"}}},
		},
		{
			name:     "templ generate invalid flag",
			args:     []string{"templ", "generate", "--watch"},
			wantCode: 1,
			wantErr:  "invalid templ generate arguments",
		},
		{
			name:     "templ generate invalid extra arg",
			args:     []string{"templ", "generate", "extra"},
			wantCode: 1,
			wantErr:  "unexpected templ generate arguments",
		},
		{
			name:     "templ help",
			args:     []string{"templ", "help"},
			wantCode: 0,
			wantOut:  "ship templ commands:",
		},
		{
			name:     "templ missing subcommand",
			args:     []string{"templ"},
			wantCode: 1,
			wantErr:  "ship templ commands:",
		},
		{
			name:     "generate help",
			args:     []string{"generate", "help"},
			wantCode: 0,
			wantOut:  "ship generate commands:",
		},
		{
			name:     "generate missing subcommand",
			args:     []string{"generate"},
			wantCode: 1,
			wantErr:  "ship generate commands:",
		},
		{
			name:     "generate unknown subcommand",
			args:     []string{"generate", "model"},
			wantCode: 1,
			wantErr:  "unknown generate command",
		},
		{
			name:     "generate resource missing name",
			args:     []string{"generate", "resource"},
			wantCode: 1,
			wantErr:  "usage: ship generate resource",
		},
		{
			name:     "check help",
			args:     []string{"check", "--help"},
			wantCode: 0,
			wantOut:  "ship check commands:",
		},
		{
			name:       "runner exit code is propagated",
			args:       []string{"dev"},
			wantCode:   7,
			wantCalls:  []fakeCall{{name: "make", args: []string{"dev"}}},
			runnerCode: 7,
		},
		{
			name:      "runner error prints message",
			args:      []string{"dev"},
			wantCode:  1,
			wantCalls: []fakeCall{{name: "make", args: []string{"dev"}}},
			wantErr:   "failed to run command",
			runnerErr: errors.New("boom"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := &bytes.Buffer{}
			errOut := &bytes.Buffer{}
			runner := &fakeRunner{code: tt.runnerCode, err: tt.runnerErr}
			cli := CLI{Out: out, Err: errOut, Runner: runner}

			got := cli.Run(tt.args)
			if got != tt.wantCode {
				t.Fatalf("exit code = %d, want %d", got, tt.wantCode)
			}
			if tt.wantOut != "" && !strings.Contains(out.String(), tt.wantOut) {
				t.Fatalf("stdout = %q, want contains %q", out.String(), tt.wantOut)
			}
			if tt.wantErr != "" && !strings.Contains(errOut.String(), tt.wantErr) {
				t.Fatalf("stderr = %q, want contains %q", errOut.String(), tt.wantErr)
			}
			if len(runner.calls) != len(tt.wantCalls) {
				t.Fatalf("calls len = %d, want %d", len(runner.calls), len(tt.wantCalls))
			}
			for i := range tt.wantCalls {
				if runner.calls[i].name != tt.wantCalls[i].name {
					t.Fatalf("call[%d] name = %q, want %q", i, runner.calls[i].name, tt.wantCalls[i].name)
				}
				if strings.Join(runner.calls[i].args, " ") != strings.Join(tt.wantCalls[i].args, " ") {
					t.Fatalf("call[%d] args = %v, want %v", i, runner.calls[i].args, tt.wantCalls[i].args)
				}
			}
		})
	}
}

func TestRelocateTemplGenerated(t *testing.T) {
	root := t.TempDir()
	moduleRoot := filepath.Join(root, "repo")
	if err := os.MkdirAll(moduleRoot, 0o755); err != nil {
		t.Fatal(err)
	}

	goMod := "module example.com/test\n\ngo 1.25\n"
	if err := os.WriteFile(filepath.Join(moduleRoot, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatal(err)
	}

	templDir := filepath.Join(moduleRoot, "app", "demo", "views", "web", "components")
	if err := os.MkdirAll(templDir, 0o755); err != nil {
		t.Fatal(err)
	}

	srcPath := filepath.Join(templDir, "foo_templ.go")
	src := `package components

import "example.com/test/app/demo/views/web/components"
import "example.com/test/app/demo/views/web/helpers"
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}

	helperDir := filepath.Join(moduleRoot, "app", "demo", "views", "web", "helpers")
	if err := os.MkdirAll(helperDir, 0o755); err != nil {
		t.Fatal(err)
	}
	helperSrcPath := filepath.Join(helperDir, "helpers_templ.go")
	helperSrc := `package helpers

import "example.com/test/app/demo/views/web/components"
`
	if err := os.WriteFile(helperSrcPath, []byte(helperSrc), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := relocateTemplGenerated(filepath.Join(moduleRoot, "app")); err != nil {
		t.Fatalf("relocateTemplGenerated returned error: %v", err)
	}

	if _, err := os.Stat(srcPath); !os.IsNotExist(err) {
		t.Fatalf("expected source file to be moved, stat err=%v", err)
	}

	dstPath := filepath.Join(templDir, "gen", "foo_templ.go")
	b, err := os.ReadFile(dstPath)
	if err != nil {
		t.Fatalf("reading moved file: %v", err)
	}
	content := string(b)
	if !strings.Contains(content, `"example.com/test/app/demo/views/web/components/gen"`) {
		t.Fatalf("moved file missing rewritten self import: %s", content)
	}
	if !strings.Contains(content, `"example.com/test/app/demo/views/web/helpers/gen"`) {
		t.Fatalf("moved file missing rewritten helper import: %s", content)
	}

	helperDstPath := filepath.Join(helperDir, "gen", "helpers_templ.go")
	helperContent, err := os.ReadFile(helperDstPath)
	if err != nil {
		t.Fatalf("reading moved helper file: %v", err)
	}
	if !strings.Contains(string(helperContent), `"example.com/test/app/demo/views/web/components/gen"`) {
		t.Fatalf("helper moved file missing rewritten component import: %s", string(helperContent))
	}
}

func TestRunCheck_UsesProjectPackageLists(t *testing.T) {
	root := t.TempDir()
	prevWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(prevWD) })
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}

	if err := os.MkdirAll(filepath.Join(root, "scripts", "test"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "app", "goship", "web", "routes"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "scripts", "test", "unit-packages.txt"), []byte("./pkg/a\n#c\n./pkg/b\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "scripts", "test", "compile-packages.txt"), []byte("./app/x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "app", "goship", "web", "routes", "routes_test.go"), []byte("package routes_test\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	runner := &fakeRunner{}
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	cli := CLI{Out: out, Err: errOut, Runner: runner}

	code := cli.Run([]string{"check"})
	if code != 0 {
		t.Fatalf("check exit code = %d, stderr=%s", code, errOut.String())
	}

	want := []fakeCall{
		{name: "go", args: []string{"test", "./pkg/a"}},
		{name: "go", args: []string{"test", "./pkg/b"}},
		{name: "go", args: []string{"test", "-run", "^$", "./app/x"}},
		{name: "go", args: []string{"test", "-c", "./app/goship/web/routes"}},
	}
	if len(runner.calls) != len(want) {
		t.Fatalf("calls len=%d want=%d calls=%v", len(runner.calls), len(want), runner.calls)
	}
	for i := range want {
		if runner.calls[i].name != want[i].name || strings.Join(runner.calls[i].args, " ") != strings.Join(want[i].args, " ") {
			t.Fatalf("call[%d]=%s %v want %s %v", i, runner.calls[i].name, runner.calls[i].args, want[i].name, want[i].args)
		}
	}
}

func TestRunCheck_FallbackToGoTestAll(t *testing.T) {
	root := t.TempDir()
	prevWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(prevWD) })
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}

	runner := &fakeRunner{}
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	cli := CLI{Out: out, Err: errOut, Runner: runner}
	code := cli.Run([]string{"check"})
	if code != 0 {
		t.Fatalf("check exit code = %d, stderr=%s", code, errOut.String())
	}
	if len(runner.calls) != 1 {
		t.Fatalf("calls len=%d want=1 calls=%v", len(runner.calls), runner.calls)
	}
	if runner.calls[0].name != "go" || strings.Join(runner.calls[0].args, " ") != "test ./..." {
		t.Fatalf("unexpected call: %s %v", runner.calls[0].name, runner.calls[0].args)
	}
}
