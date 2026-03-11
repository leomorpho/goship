package ship

import (
	"bytes"
	"errors"
	"os"
	"strings"
	"testing"
)

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

func TestRunDBReset_DBURLResolutionError(t *testing.T) {
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
	code := cli.Run([]string{"db:reset"})
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

func TestRunDBReset_NonLocalRequiresForce(t *testing.T) {
	useLocalAppEnv(t)
	errOut := &bytes.Buffer{}
	runner := &fakeRunner{}
	cli := CLI{
		Out:    &bytes.Buffer{},
		Err:    errOut,
		Runner: runner,
		ResolveDBURL: func() (string, error) {
			return "postgres://user:pass@db.example.com:5432/app", nil
		},
	}

	code := cli.Run([]string{"db:reset"})
	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if len(runner.calls) != 0 {
		t.Fatalf("runner calls = %v, want no commands", runner.calls)
	}
	if !strings.Contains(errOut.String(), "without --force") {
		t.Fatalf("stderr = %q, want non-local force guard", errOut.String())
	}
}

func TestRunDBReset_NonLocalWithForce(t *testing.T) {
	useLocalAppEnv(t)
	errOut := &bytes.Buffer{}
	runner := &fakeRunner{}
	cli := CLI{
		Out:    &bytes.Buffer{},
		Err:    errOut,
		Runner: runner,
		ResolveDBURL: func() (string, error) {
			return "postgres://user:pass@db.example.com:5432/app", nil
		},
	}

	code := cli.Run([]string{"db:reset", "--force", "--yes"})
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if len(runner.calls) != 2 {
		t.Fatalf("runner calls = %v, want clean+apply", runner.calls)
	}
	if runner.calls[0].name != "goose" || runner.calls[1].name != "goose" {
		t.Fatalf("unexpected commands: %+v", runner.calls)
	}
	if errOut.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", errOut.String())
	}
}

func TestRunDBReset_WithSeed(t *testing.T) {
	useLocalAppEnv(t)
	runner := &fakeRunner{}
	cli := CLI{
		Out:    &bytes.Buffer{},
		Err:    &bytes.Buffer{},
		Runner: runner,
		ResolveDBURL: func() (string, error) {
			return testDBURL, nil
		},
	}

	code := cli.Run([]string{"db:reset", "--seed", "--yes"})
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if len(runner.calls) != 3 {
		t.Fatalf("runner calls = %v, want clean+apply+seed", runner.calls)
	}
	if runner.calls[2].name != "go" {
		t.Fatalf("third command = %q, want go seed", runner.calls[2].name)
	}
}

func TestRunDBReset_DryRun(t *testing.T) {
	useLocalAppEnv(t)
	runner := &fakeRunner{}
	out := &bytes.Buffer{}
	cli := CLI{
		Out:    out,
		Err:    &bytes.Buffer{},
		Runner: runner,
		ResolveDBURL: func() (string, error) {
			return testDBURL, nil
		},
	}
	code := cli.Run([]string{"db:reset", "--dry-run"})
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if len(runner.calls) != 0 {
		t.Fatalf("runner calls = %v, want none", runner.calls)
	}
	if !strings.Contains(out.String(), "mode: dry-run") {
		t.Fatalf("stdout = %q, want dry-run plan output", out.String())
	}
}

func TestRunDBReset_ProductionRequiresForceAndYes(t *testing.T) {
	prev := os.Getenv("APP_ENV")
	t.Setenv("APP_ENV", "production")
	t.Cleanup(func() { _ = os.Setenv("APP_ENV", prev) })

	errOut := &bytes.Buffer{}
	cli := CLI{
		Out:    &bytes.Buffer{},
		Err:    errOut,
		Runner: &fakeRunner{},
		ResolveDBURL: func() (string, error) {
			return testDBURL, nil
		},
	}
	code := cli.Run([]string{"db:reset", "--yes"})
	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if !strings.Contains(errOut.String(), "in production") {
		t.Fatalf("stderr = %q, want production guard", errOut.String())
	}
}

func TestRunDBDrop_DryRun(t *testing.T) {
	useLocalAppEnv(t)
	out := &bytes.Buffer{}
	cli := CLI{
		Out:    out,
		Err:    &bytes.Buffer{},
		Runner: &fakeRunner{},
		ResolveDBURL: func() (string, error) {
			return testDBURL, nil
		},
	}
	code := cli.Run([]string{"db:drop", "--dry-run"})
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if !strings.Contains(out.String(), "DB drop plan") {
		t.Fatalf("stdout = %q, want drop plan output", out.String())
	}
}

func TestRunDBCreate_DryRun(t *testing.T) {
	useLocalAppEnv(t)
	out := &bytes.Buffer{}
	runner := &fakeRunner{}
	cli := CLI{
		Out:    out,
		Err:    &bytes.Buffer{},
		Runner: runner,
		ResolveDBURL: func() (string, error) {
			return testDBURL, nil
		},
	}
	code := cli.Run([]string{"db:create", "--dry-run"})
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if len(runner.calls) != 0 {
		t.Fatalf("runner calls = %v, want none", runner.calls)
	}
	if !strings.Contains(out.String(), "DB create plan") {
		t.Fatalf("stdout = %q, want create plan output", out.String())
	}
}

func TestRunDBGenerate_DryRun(t *testing.T) {
	useLocalAppEnv(t)
	out := &bytes.Buffer{}
	runner := &fakeRunner{}
	cli := CLI{
		Out:    out,
		Err:    &bytes.Buffer{},
		Runner: runner,
	}
	code := cli.Run([]string{"db:generate", "--dry-run"})
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if len(runner.calls) != 0 {
		t.Fatalf("runner calls = %v, want none", runner.calls)
	}
	if !strings.Contains(out.String(), "DB generate plan") {
		t.Fatalf("stdout = %q, want generate plan output", out.String())
	}
}

func TestIsLocalDBURL(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		wantOK bool
	}{
		{name: "sqlite local", input: "sqlite://file:dev.db?_fk=1", wantOK: true},
		{name: "postgres localhost", input: "postgres://user:pass@localhost:5432/app", wantOK: true},
		{name: "postgres 127", input: "postgres://user:pass@127.0.0.1:5432/app", wantOK: true},
		{name: "mysql local service", input: "mysql://user:pass@mysql:3306/app", wantOK: true},
		{name: "remote host", input: "postgres://user:pass@db.example.com:5432/app", wantOK: false},
		{name: "invalid", input: "://bad", wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ok := isLocalDBURL(tt.input)
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}
		})
	}
}

func TestIsLocalDBURL_UsesConfiguredHosts(t *testing.T) {
	t.Setenv("SHIP_LOCAL_DB_HOSTS", "db.example.com, localhost")
	if !isLocalDBURL("postgres://user:pass@db.example.com:5432/app") {
		t.Fatal("expected configured host to be treated as local")
	}
}

func useLocalAppEnv(t *testing.T) {
	t.Helper()
	t.Setenv("APP_ENV", "local")
}
