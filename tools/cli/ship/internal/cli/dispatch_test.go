package ship

import (
	"bytes"
	"errors"
	"os"
	"strings"
	"testing"
)

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
			name:     "routes help",
			args:     []string{"routes", "--help"},
			wantCode: 0,
			wantOut:  "ship routes commands:",
		},
		{
			name:     "config help",
			args:     []string{"config", "help"},
			wantCode: 0,
			wantOut:  "ship config commands:",
		},
		{
			name:     "config validate json",
			args:     []string{"config:validate", "--json"},
			wantCode: 0,
			wantOut:  "\"variables\":",
		},
		{
			name:     "agent help",
			args:     []string{"agent", "--help"},
			wantCode: 0,
			wantOut:  "ship agent commands:",
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
			wantCalls: []fakeCall{{name: "goose", args: []string{"-dir", gooseDir, "postgres", testDBURL, "up"}}},
		},
		{
			name:      "db status",
			args:      []string{"db:status"},
			wantCode:  0,
			wantCalls: []fakeCall{{name: "goose", args: []string{"-dir", gooseDir, "postgres", testDBURL, "status"}}},
		},
		{
			name:      "db console",
			args:      []string{"db:console"},
			wantCode:  0,
			wantCalls: []fakeCall{{name: "psql", args: []string{testDBURL}}},
		},
		{
			name:     "db reset requires yes",
			args:     []string{"db:reset"},
			wantCode: 1,
			wantErr:  "without --yes",
		},
		{
			name:     "db reset local yes",
			args:     []string{"db:reset", "--yes"},
			wantCode: 0,
			wantCalls: []fakeCall{
				{name: "goose", args: []string{"-dir", gooseDir, "postgres", testDBURL, "reset"}},
				{name: "goose", args: []string{"-dir", gooseDir, "postgres", testDBURL, "up"}},
			},
		},
		{
			name:      "db drop local yes",
			args:      []string{"db:drop", "--yes"},
			wantCode:  0,
			wantCalls: []fakeCall{{name: "goose", args: []string{"-dir", gooseDir, "postgres", testDBURL, "reset"}}},
		},
		{
			name:      "db create",
			args:      []string{"db:create"},
			wantCode:  0,
			wantCalls: []fakeCall{{name: "goose", args: []string{"-dir", gooseDir, "postgres", testDBURL, "status"}}},
		},
		{
			name:      "db generate",
			args:      []string{"db:generate"},
			wantCode:  0,
			wantCalls: []fakeCall{{name: "bobgen-sql", args: []string{"-c", "db/bobgen.yaml"}}},
		},
		{
			name:      "db make",
			args:      []string{"db:make", "add_posts"},
			wantCode:  0,
			wantCalls: []fakeCall{{name: "goose", args: []string{"-dir", gooseDir, "create", "add_posts", "sql"}}},
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
				name: "goose",
				args: []string{"-dir", gooseDir, "postgres", testDBURL, "down"},
			}},
		},
		{
			name:     "db rollback explicit amount",
			args:     []string{"db:rollback", "3"},
			wantCode: 0,
			wantCalls: []fakeCall{{
				name: "goose",
				args: []string{"-dir", gooseDir, "postgres", testDBURL, "down-to", "3"},
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
			name:     "db reset extra arg",
			args:     []string{"db:reset", "extra"},
			wantCode: 1,
			wantErr:  "usage: ship db:reset [--seed] [--force] [--yes] [--dry-run]",
		},
		{
			name:     "db drop extra arg",
			args:     []string{"db:drop", "extra"},
			wantCode: 1,
			wantErr:  "usage: ship db:drop [--force] [--yes] [--dry-run]",
		},
		{
			name:     "db create extra arg",
			args:     []string{"db:create", "extra"},
			wantCode: 1,
			wantErr:  "usage: ship db:create [--dry-run]",
		},
		{
			name:     "db generate extra arg",
			args:     []string{"db:generate", "extra"},
			wantCode: 1,
			wantErr:  "usage: ship db:generate [--config <path>] [--dry-run]",
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
			args:      []string{"templ", "generate", "--file", "app/views/web/pages/home.templ"},
			wantCode:  0,
			wantCalls: []fakeCall{{name: "templ", args: []string{"generate", "-f", "app/views/web/pages/home.templ"}}},
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
			name:      "make model",
			args:      []string{"make:model", "Post"},
			wantCode:  0,
			wantCalls: nil,
		},
		{
			name:      "make model with fields",
			args:      []string{"make:model", "Post", "title:string"},
			wantCode:  0,
			wantCalls: nil,
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
			t.Setenv("APP_ENV", "local")

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
			cli.ResolveDBDriver = func() (string, error) {
				return "postgres", nil
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
