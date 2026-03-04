package ship

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
)

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
