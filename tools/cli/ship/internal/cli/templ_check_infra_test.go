package ship

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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

func TestRunTest_UsesProjectPackageLists(t *testing.T) {
	root := t.TempDir()
	prevWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(prevWD) })
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}

	if err := os.MkdirAll(filepath.Join(root, "tools", "scripts", "test"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "tools", "scripts", "test", "unit-packages.txt"), []byte("./framework/a\n#c\n./framework/b\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "tools", "scripts", "test", "compile-packages.txt"), []byte("./app/x\n./app/web/controllers\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	runner := &fakeRunner{}
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	cli := CLI{Out: out, Err: errOut, Runner: runner}

	code := cli.Run([]string{"test"})
	if code != 0 {
		t.Fatalf("test exit code = %d, stderr=%s", code, errOut.String())
	}

	want := []fakeCall{
		{name: "go", args: []string{"test", "./framework/a"}},
		{name: "go", args: []string{"test", "./framework/b"}},
		{name: "go", args: []string{"test", "-run", "^$", "./app/x"}},
		{name: "go", args: []string{"test", "-run", "^$", "./app/web/controllers"}},
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

func TestRunTest_FallbackToGoTestAll(t *testing.T) {
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
	code := cli.Run([]string{"test"})
	if code != 0 {
		t.Fatalf("test exit code = %d, stderr=%s", code, errOut.String())
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
