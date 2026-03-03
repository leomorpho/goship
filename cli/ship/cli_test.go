package ship

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testDBURL = "postgres://test-user:test-pass@localhost:5432/test_db?sslmode=disable"

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
		name            string
		args            []string
		wantCode        int
		wantCalls       []fakeCall
		wantOut         string
		wantErr         string
		runnerCode      int
		runnerErr       error
		useDevAllRunner bool
		devAllCode      int
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
			wantCalls: []fakeCall{{name: "go", args: []string{"run", "./cmd/web"}}},
		},
		{
			name:      "shipdev alias",
			args:      []string{"shipdev"},
			wantCode:  0,
			wantCalls: []fakeCall{{name: "go", args: []string{"run", "./cmd/web"}}},
		},
		{
			name:      "dev worker positional",
			args:      []string{"dev", "worker"},
			wantCode:  0,
			wantCalls: []fakeCall{{name: "go", args: []string{"run", "./cmd/worker"}}},
		},
		{
			name:            "dev all positional",
			args:            []string{"dev", "all"},
			wantCode:        0,
			wantCalls:       nil,
			useDevAllRunner: true,
			devAllCode:      0,
		},
		{
			name:      "dev worker flag",
			args:      []string{"dev", "--worker"},
			wantCode:  0,
			wantCalls: []fakeCall{{name: "go", args: []string{"run", "./cmd/worker"}}},
		},
		{
			name:            "dev all flag",
			args:            []string{"dev", "--all"},
			wantCode:        0,
			wantCalls:       nil,
			useDevAllRunner: true,
			devAllCode:      0,
		},
		{
			name:            "dev all runner exit code is propagated",
			args:            []string{"dev", "all"},
			wantCode:        9,
			wantCalls:       nil,
			useDevAllRunner: true,
			devAllCode:      9,
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
			wantCalls: []fakeCall{{name: "go", args: []string{"test", "./..."}}},
		},
		{
			name:      "test integration",
			args:      []string{"test", "--integration"},
			wantCode:  0,
			wantCalls: []fakeCall{{name: "go", args: []string{"test", "-tags=integration", "./..."}}},
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
			name:     "db create removed",
			args:     []string{"db", "create"},
			wantCode: 1,
			wantErr:  "use namespaced DB commands",
		},
		{
			name:      "db migrate",
			args:      []string{"db:migrate"},
			wantCode:  0,
			wantCalls: []fakeCall{{name: "atlas", args: []string{"migrate", "apply", "--dir", atlasDir, "--url", testDBURL}}},
		},
		{
			name:      "db status",
			args:      []string{"db:status"},
			wantCode:  0,
			wantCalls: []fakeCall{{name: "atlas", args: []string{"migrate", "status", "--dir", atlasDir, "--url", testDBURL}}},
		},
		{
			name:      "db make",
			args:      []string{"db:make", "add_posts"},
			wantCode:  0,
			wantCalls: []fakeCall{{name: "atlas", args: []string{"migrate", "diff", "add_posts", "--dir", atlasDir, "--to", "ent://app/goship/db/schema", "--dev-url", "sqlite://file?mode=memory&_fk=1"}}},
		},
		{
			name:     "db make missing name",
			args:     []string{"db:make"},
			wantCode: 1,
			wantErr:  "usage: ship db:make <migration_name>",
		},
		{
			name:      "db seed",
			args:      []string{"db:seed"},
			wantCode:  0,
			wantCalls: []fakeCall{{name: "go", args: []string{"run", "./cmd/seed/main.go"}}},
		},
		{
			name:     "db rollback default amount",
			args:     []string{"db:rollback"},
			wantCode: 0,
			wantCalls: []fakeCall{{
				name: "atlas",
				args: []string{"migrate", "down", "--dir", atlasDir, "--url", testDBURL, "1"},
			}},
		},
		{
			name:     "db rollback explicit amount",
			args:     []string{"db:rollback", "3"},
			wantCode: 0,
			wantCalls: []fakeCall{{
				name: "atlas",
				args: []string{"migrate", "down", "--dir", atlasDir, "--url", testDBURL, "3"},
			}},
		},
		{
			name:     "db rollback invalid amount",
			args:     []string{"db:rollback", "x"},
			wantCode: 1,
			wantErr:  "invalid rollback amount",
		},
		{
			name:     "db rollback too many args",
			args:     []string{"db:rollback", "1", "2"},
			wantCode: 1,
			wantErr:  "usage: ship db:rollback [amount]",
		},
		{
			name:     "db status extra arg",
			args:     []string{"db:status", "extra"},
			wantCode: 1,
			wantErr:  "usage: ship db:status",
		},
		{
			name:     "db missing subcommand",
			args:     []string{"db"},
			wantCode: 0,
			wantOut:  "ship db commands:",
		},
		{
			name:     "db help",
			args:     []string{"db", "help"},
			wantCode: 0,
			wantOut:  "ship db commands:",
		},
		{
			name:     "infra up",
			args:     []string{"infra:up"},
			wantCode: 0,
			wantCalls: []fakeCall{
				{name: "docker-compose", args: []string{"up", "-d", "cache"}},
				{name: "docker-compose", args: []string{"up", "-d", "mailpit"}},
			},
		},
		{
			name:      "infra down",
			args:      []string{"infra:down"},
			wantCode:  0,
			wantCalls: []fakeCall{{name: "docker-compose", args: []string{"down"}}},
		},
		{
			name:     "infra help",
			args:     []string{"infra", "help"},
			wantCode: 0,
			wantOut:  "ship infra commands:",
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
			name:     "make help",
			args:     []string{"make", "help"},
			wantCode: 0,
			wantOut:  "ship make commands:",
		},
		{
			name:     "make missing subcommand",
			args:     []string{"make"},
			wantCode: 0,
			wantOut:  "ship make commands:",
		},
		{
			name:     "make unknown subcommand",
			args:     []string{"make:widget"},
			wantCode: 1,
			wantErr:  "unknown make command",
		},
		{
			name:     "make resource missing name",
			args:     []string{"make:resource"},
			wantCode: 1,
			wantErr:  "usage: ship make:resource",
		},
		{
			name:     "make model missing name",
			args:     []string{"make:model"},
			wantCode: 1,
			wantErr:  "usage: ship make:model <Name> [fields...]",
		},
		{
			name:     "make controller missing name",
			args:     []string{"make:controller"},
			wantCode: 1,
			wantErr:  "usage: ship make:controller",
		},
		{
			name:     "make model",
			args:     []string{"make:model", "Post"},
			wantCode: 0,
			wantCalls: []fakeCall{
				{name: "go", args: []string{"run", "-mod=mod", "entgo.io/ent/cmd/ent", "generate", "--feature", "sql/upsert,sql/execquery", "--target", "./ent", "./app/goship/db/schema"}},
			},
		},
		{
			name:     "make model with fields",
			args:     []string{"make:model", "Post", "title:string"},
			wantCode: 0,
			wantCalls: []fakeCall{
				{name: "go", args: []string{"run", "-mod=mod", "entgo.io/ent/cmd/ent", "generate", "--feature", "sql/upsert,sql/execquery", "--target", "./ent", "./app/goship/db/schema"}},
			},
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
			wantCalls:  []fakeCall{{name: "go", args: []string{"run", "./cmd/web"}}},
			runnerCode: 7,
		},
		{
			name:      "runner error prints message",
			args:      []string{"dev"},
			wantCode:  1,
			wantCalls: []fakeCall{{name: "go", args: []string{"run", "./cmd/web"}}},
			wantErr:   "failed to run command",
			runnerErr: errors.New("boom"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if len(tt.args) > 0 && (tt.args[0] == "dev" || tt.args[0] == "shipdev" || tt.args[0] == "test" || tt.args[0] == "check" || tt.args[0] == "make:model" || tt.args[0] == "make:resource") {
				prevWD, err := os.Getwd()
				if err != nil {
					t.Fatal(err)
				}
				tmp := t.TempDir()
				if err := os.Chdir(tmp); err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { _ = os.Chdir(prevWD) })
			}

			out := &bytes.Buffer{}
			errOut := &bytes.Buffer{}
			runner := &fakeRunner{code: tt.runnerCode, err: tt.runnerErr}
			devAllCalls := 0
			cli := CLI{Out: out, Err: errOut, Runner: runner}
			cli.ResolveCompose = func() ([]string, error) {
				return []string{"docker-compose"}, nil
			}
			cli.ResolveDBURL = func() (string, error) {
				return testDBURL, nil
			}
			if tt.useDevAllRunner {
				cli.RunDevAll = func() int {
					devAllCalls++
					return tt.devAllCode
				}
			}

			got := cli.Run(tt.args)
			if got != tt.wantCode {
				t.Fatalf("exit code = %d, want %d", got, tt.wantCode)
			}
			if tt.useDevAllRunner && devAllCalls != 1 {
				t.Fatalf("RunDevAll calls = %d, want 1", devAllCalls)
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

func TestRunInfraUp_ResolveComposeFailure(t *testing.T) {
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	runner := &fakeRunner{}
	cli := CLI{
		Out:    out,
		Err:    errOut,
		Runner: runner,
		ResolveCompose: func() ([]string, error) {
			return nil, errors.New("missing compose")
		},
	}

	code := cli.Run([]string{"infra:up"})
	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if !strings.Contains(errOut.String(), "failed to resolve docker compose") {
		t.Fatalf("stderr = %q, want compose failure message", errOut.String())
	}
	if len(runner.calls) != 0 {
		t.Fatalf("runner calls = %v, want none", runner.calls)
	}
}

func TestRunInfraUp_MailpitFailureIsNonFatal(t *testing.T) {
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	runner := &fakeRunner{
		nextCode: map[string]int{
			"docker-compose up -d mailpit": 1,
		},
	}
	cli := CLI{
		Out:    out,
		Err:    errOut,
		Runner: runner,
		ResolveCompose: func() ([]string, error) {
			return []string{"docker-compose"}, nil
		},
	}

	code := cli.Run([]string{"infra:up"})
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if !strings.Contains(errOut.String(), "warning: could not start mailpit") {
		t.Fatalf("stderr = %q, want mailpit warning", errOut.String())
	}
	want := []fakeCall{
		{name: "docker-compose", args: []string{"up", "-d", "cache"}},
		{name: "docker-compose", args: []string{"up", "-d", "mailpit"}},
	}
	if len(runner.calls) != len(want) {
		t.Fatalf("calls len=%d want=%d calls=%v", len(runner.calls), len(want), runner.calls)
	}
}

func TestRunInfraDown_ResolveComposeFailure(t *testing.T) {
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	runner := &fakeRunner{}
	cli := CLI{
		Out:    out,
		Err:    errOut,
		Runner: runner,
		ResolveCompose: func() ([]string, error) {
			return nil, errors.New("missing compose")
		},
	}

	code := cli.Run([]string{"infra:down"})
	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if !strings.Contains(errOut.String(), "failed to resolve docker compose") {
		t.Fatalf("stderr = %q, want compose failure message", errOut.String())
	}
	if len(runner.calls) != 0 {
		t.Fatalf("runner calls = %v, want none", runner.calls)
	}
}

func TestResolveAtlasDBURL_PrefersEnv(t *testing.T) {
	prev := os.Getenv("DATABASE_URL")
	t.Cleanup(func() { _ = os.Setenv("DATABASE_URL", prev) })
	if err := os.Setenv("DATABASE_URL", "postgres://env-only"); err != nil {
		t.Fatal(err)
	}

	got, err := resolveAtlasDBURL()
	if err != nil {
		t.Fatalf("resolveAtlasDBURL error = %v", err)
	}
	if got != "postgres://env-only" {
		t.Fatalf("db url = %q, want %q", got, "postgres://env-only")
	}
}

func TestResolveAtlasDBURL_PrefersDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://primary")
	t.Setenv("PAGODA_DATABASE_URL", "postgres://secondary")
	got, err := resolveAtlasDBURL()
	if err != nil {
		t.Fatalf("resolveAtlasDBURL error = %v", err)
	}
	if got != "postgres://primary" {
		t.Fatalf("db url = %q, want %q", got, "postgres://primary")
	}
}

func TestResolveAtlasDBURL_RejectsLegacyPagodaDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("PAGODA_DATABASE_URL", "postgres://pagoda-env")
	_, err := resolveAtlasDBURL()
	if err == nil {
		t.Fatal("expected error for PAGODA_DATABASE_URL, got nil")
	}
	if !strings.Contains(err.Error(), "PAGODA_DATABASE_URL is not supported") {
		t.Fatalf("error = %q, want explicit legacy var message", err.Error())
	}
}

func TestResolveAtlasDBURL_FromConfig(t *testing.T) {
	root := t.TempDir()
	prevWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(prevWD) })
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}

	t.Setenv("DATABASE_URL", "")
	t.Setenv("PAGODA_DATABASE_URL", "")
	t.Setenv("APP_ENV", "local")

	if err := os.MkdirAll(filepath.Join(root, "config", "environments"), 0o755); err != nil {
		t.Fatal(err)
	}
	application := `
app:
  environment: "local"
database:
  dbMode: "standalone"
  hostname: "db.local"
  port: 5432
  user: "app"
  password: "secret"
  databaseNameLocal: "goship_db"
  databaseNameProd: "goship_prod"
  testDatabase: "goship_test"
  sslMode: "disable"
`
	if err := os.WriteFile(filepath.Join(root, "config", "application.yaml"), []byte(application), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "config", "environments", "local.yaml"), []byte("app:\n  environment: local\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := resolveAtlasDBURL()
	if err != nil {
		t.Fatalf("resolveAtlasDBURL error = %v", err)
	}
	if !strings.Contains(got, "db.local:5432") {
		t.Fatalf("db url = %q, want host/port", got)
	}
	if !strings.Contains(got, "/goship_db") {
		t.Fatalf("db url = %q, want local database name", got)
	}
}

func TestResolveAtlasDBURL_EmbeddedModeError(t *testing.T) {
	root := t.TempDir()
	prevWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(prevWD) })
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}

	t.Setenv("DATABASE_URL", "")
	t.Setenv("PAGODA_DATABASE_URL", "")
	t.Setenv("APP_ENV", "local")

	if err := os.MkdirAll(filepath.Join(root, "config", "environments"), 0o755); err != nil {
		t.Fatal(err)
	}
	application := `
app:
  environment: "local"
database:
  dbMode: "embedded"
  embeddedConnection: "dbs/main.db"
`
	if err := os.WriteFile(filepath.Join(root, "config", "application.yaml"), []byte(application), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err = resolveAtlasDBURL()
	if err == nil {
		t.Fatal("expected error for embedded mode, got nil")
	}
	if !strings.Contains(err.Error(), "embedded") {
		t.Fatalf("error = %q, want embedded message", err.Error())
	}
}

func TestResolveAtlasDBURL_UsesProductionDatabaseName(t *testing.T) {
	root := t.TempDir()
	prevWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(prevWD) })
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DATABASE_URL", "")
	t.Setenv("PAGODA_DATABASE_URL", "")
	t.Setenv("APP_ENV", "production")

	if err := os.MkdirAll(filepath.Join(root, "config", "environments"), 0o755); err != nil {
		t.Fatal(err)
	}
	application := `
app:
  environment: "local"
database:
  dbMode: "standalone"
  hostname: "db.local"
  port: 5432
  user: "app"
  password: "secret"
  databaseNameLocal: "goship_db"
  databaseNameProd: "goship_prod"
  testDatabase: "goship_test"
  sslMode: "disable"
`
	if err := os.WriteFile(filepath.Join(root, "config", "application.yaml"), []byte(application), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := resolveAtlasDBURL()
	if err != nil {
		t.Fatalf("resolveAtlasDBURL error = %v", err)
	}
	if !strings.Contains(got, "/goship_prod") {
		t.Fatalf("db url = %q, want production database name", got)
	}
}

func TestResolveAtlasDBURL_UsesTestDatabaseName(t *testing.T) {
	root := t.TempDir()
	prevWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(prevWD) })
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DATABASE_URL", "")
	t.Setenv("PAGODA_DATABASE_URL", "")
	t.Setenv("APP_ENV", "test")

	if err := os.MkdirAll(filepath.Join(root, "config", "environments"), 0o755); err != nil {
		t.Fatal(err)
	}
	application := `
app:
  environment: "local"
database:
  dbMode: "standalone"
  hostname: "db.local"
  port: 5432
  user: "app"
  password: "secret"
  databaseNameLocal: "goship_db"
  databaseNameProd: "goship_prod"
  testDatabase: "goship_test"
  sslMode: "disable"
`
	if err := os.WriteFile(filepath.Join(root, "config", "application.yaml"), []byte(application), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := resolveAtlasDBURL()
	if err != nil {
		t.Fatalf("resolveAtlasDBURL error = %v", err)
	}
	if !strings.Contains(got, "/goship_test") {
		t.Fatalf("db url = %q, want test database name", got)
	}
}

func TestResolveComposeCommandWith_DockerComposeAvailable(t *testing.T) {
	lookPath := func(bin string) (string, error) {
		if bin == "docker-compose" {
			return "/usr/bin/docker-compose", nil
		}
		return "", errors.New("missing")
	}
	got, err := resolveComposeCommandWith(lookPath, func() error { return nil })
	if err != nil {
		t.Fatalf("resolveComposeCommandWith error = %v", err)
	}
	if strings.Join(got, " ") != "docker-compose" {
		t.Fatalf("compose command = %v, want docker-compose", got)
	}
}

func TestResolveComposeCommandWith_DockerComposeSubcommandAvailable(t *testing.T) {
	lookPath := func(bin string) (string, error) {
		if bin == "docker" {
			return "/usr/bin/docker", nil
		}
		return "", errors.New("missing")
	}
	got, err := resolveComposeCommandWith(lookPath, func() error { return nil })
	if err != nil {
		t.Fatalf("resolveComposeCommandWith error = %v", err)
	}
	if strings.Join(got, " ") != "docker compose" {
		t.Fatalf("compose command = %v, want docker compose", got)
	}
}

func TestResolveComposeCommandWith_NoComposeAvailable(t *testing.T) {
	lookPath := func(string) (string, error) {
		return "", errors.New("missing")
	}
	_, err := resolveComposeCommandWith(lookPath, func() error { return errors.New("no compose") })
	if err == nil {
		t.Fatal("expected compose resolution error, got nil")
	}
}

func TestRunDBMigrate_DBURLResolutionError(t *testing.T) {
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	runner := &fakeRunner{}
	cli := CLI{
		Out:    out,
		Err:    errOut,
		Runner: runner,
		ResolveDBURL: func() (string, error) {
			return "", errors.New("missing url")
		},
	}
	code := cli.Run([]string{"db:migrate"})
	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if !strings.Contains(errOut.String(), "failed to resolve database URL") {
		t.Fatalf("stderr = %q, want db url resolution failure", errOut.String())
	}
	if len(runner.calls) != 0 {
		t.Fatalf("runner calls = %v, want none", runner.calls)
	}
}

func TestRunDBRollback_DBURLResolutionError(t *testing.T) {
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	runner := &fakeRunner{}
	cli := CLI{
		Out:    out,
		Err:    errOut,
		Runner: runner,
		ResolveDBURL: func() (string, error) {
			return "", errors.New("missing url")
		},
	}
	code := cli.Run([]string{"db:rollback"})
	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if !strings.Contains(errOut.String(), "failed to resolve database URL") {
		t.Fatalf("stderr = %q, want db url resolution failure", errOut.String())
	}
	if len(runner.calls) != 0 {
		t.Fatalf("runner calls = %v, want none", runner.calls)
	}
}

func TestRunDBStatus_DBURLResolutionError(t *testing.T) {
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	runner := &fakeRunner{}
	cli := CLI{
		Out:    out,
		Err:    errOut,
		Runner: runner,
		ResolveDBURL: func() (string, error) {
			return "", errors.New("missing url")
		},
	}
	code := cli.Run([]string{"db:status"})
	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if !strings.Contains(errOut.String(), "failed to resolve database URL") {
		t.Fatalf("stderr = %q, want db url resolution failure", errOut.String())
	}
	if len(runner.calls) != 0 {
		t.Fatalf("runner calls = %v, want none", runner.calls)
	}
}

func TestRunAtlasCmd_UsesPathAtlasWhenAvailable(t *testing.T) {
	restore := stubAtlasResolution(
		func(CmdRunner) bool { return true },
		func(string) (string, error) { return "/usr/local/bin/atlas", nil },
		func(io.Writer, io.Writer) (string, error) { return "", errors.New("should not install") },
	)
	defer restore()

	runner := &fakeRunner{}
	cli := CLI{Out: &bytes.Buffer{}, Err: &bytes.Buffer{}, Runner: runner}
	code := cli.runAtlasCmd("migrate", "apply")
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if len(runner.calls) != 1 {
		t.Fatalf("calls = %d, want 1", len(runner.calls))
	}
	if runner.calls[0].name != "atlas" {
		t.Fatalf("command = %q, want atlas", runner.calls[0].name)
	}
}

func TestRunAtlasCmd_InstallsLocalAtlasWhenMissing(t *testing.T) {
	restore := stubAtlasResolution(
		func(CmdRunner) bool { return true },
		func(string) (string, error) { return "", errors.New("missing") },
		func(io.Writer, io.Writer) (string, error) { return "/tmp/tools/atlas", nil },
	)
	defer restore()

	out := &bytes.Buffer{}
	runner := &fakeRunner{}
	cli := CLI{Out: out, Err: &bytes.Buffer{}, Runner: runner}
	code := cli.runAtlasCmd("migrate", "apply")
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if len(runner.calls) != 1 {
		t.Fatalf("calls = %d, want 1", len(runner.calls))
	}
	if runner.calls[0].name != "/tmp/tools/atlas" {
		t.Fatalf("command = %q, want /tmp/tools/atlas", runner.calls[0].name)
	}
	if !strings.Contains(out.String(), "installed local pinned atlas") {
		t.Fatalf("stdout = %q, want install message", out.String())
	}
}

func TestRunAtlasCmd_FallsBackToGoRunWhenInstallFails(t *testing.T) {
	restore := stubAtlasResolution(
		func(CmdRunner) bool { return true },
		func(string) (string, error) { return "", errors.New("missing") },
		func(io.Writer, io.Writer) (string, error) { return "", errors.New("install failed") },
	)
	defer restore()

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	runner := &fakeRunner{}
	cli := CLI{Out: out, Err: errOut, Runner: runner}
	code := cli.runAtlasCmd("migrate", "apply")
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if len(runner.calls) != 1 {
		t.Fatalf("calls = %d, want 1", len(runner.calls))
	}
	if runner.calls[0].name != "go" {
		t.Fatalf("command = %q, want go", runner.calls[0].name)
	}
	if !strings.Contains(strings.Join(runner.calls[0].args, " "), "run "+atlasGoRunRef) {
		t.Fatalf("args = %v, want go run atlas module", runner.calls[0].args)
	}
	if !strings.Contains(errOut.String(), "atlas auto-install failed") {
		t.Fatalf("stderr = %q, want auto-install failure message", errOut.String())
	}
}

func stubAtlasResolution(
	isExec func(CmdRunner) bool,
	lookPath func(string) (string, error),
	install func(io.Writer, io.Writer) (string, error),
) func() {
	prevIsExec := isExecRunnerFn
	prevLookPath := atlasLookPathFn
	prevInstall := atlasInstallFn
	isExecRunnerFn = isExec
	atlasLookPathFn = lookPath
	atlasInstallFn = install
	return func() {
		isExecRunnerFn = prevIsExec
		atlasLookPathFn = prevLookPath
		atlasInstallFn = prevInstall
	}
}
