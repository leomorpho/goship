//go:build integration

package integration

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestShipDBGenerateRunsCoreThenSortedModuleConfigs(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go toolchain not available")
	}

	repoRoot := mustRepoRootFromFile(t)
	shipBin := filepath.Join(t.TempDir(), "ship")

	buildCtx, buildCancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer buildCancel()
	build := exec.CommandContext(buildCtx, "go", "build", "-o", shipBin, "./tools/cli/ship/cmd/ship")
	build.Dir = repoRoot
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build ship binary: %v: %s", err, string(out))
	}

	workspace := filepath.Join(t.TempDir(), "workspace")
	if err := os.MkdirAll(workspace, 0o755); err != nil {
		t.Fatalf("mkdir workspace: %v", err)
	}

	fakeBin := filepath.Join(t.TempDir(), "fake-bin")
	if err := os.MkdirAll(fakeBin, 0o755); err != nil {
		t.Fatalf("mkdir fake bin: %v", err)
	}
	logPath := filepath.Join(t.TempDir(), "bobgen.log")
	writeFakeBobgenLogger(t, fakeBin)

	runShip := func(dir string, env []string, args ...string) string {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		cmd := exec.CommandContext(ctx, shipBin, args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), env...)
		out, err := cmd.CombinedOutput()
		msg := string(out)
		if err != nil {
			if isLikelyEnvironmentConstraint(msg) {
				t.Skipf("skipping db:generate modules integration (environment/toolchain constraint): %v: %s", err, msg)
			}
			var exitErr *exec.ExitError
			if errors.As(err, &exitErr) {
				t.Fatalf("ship %s failed (%d): %s", strings.Join(args, " "), exitErr.ExitCode(), msg)
			}
			t.Fatalf("ship %s failed: %v: %s", strings.Join(args, " "), err, msg)
		}
		return msg
	}

	runShip(workspace, nil, "new", "demo", "--module", "example.com/demo")
	projectRoot := filepath.Join(workspace, "demo")

	moduleConfig := "modules:\n  - zeta\n  - alpha\n"
	if err := os.WriteFile(filepath.Join(projectRoot, "config", "modules.yaml"), []byte(moduleConfig), 0o644); err != nil {
		t.Fatalf("write modules config: %v", err)
	}
	for _, module := range []string{"alpha", "zeta"} {
		dir := filepath.Join(projectRoot, "modules", module, "db")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir module db dir %s: %v", module, err)
		}
		config := "sql:\n  dialect: psql\n  pattern: \"modules/" + module + "/db/migrate/migrations/*.sql\"\noutput: \"modules/" + module + "/db/gen\"\n"
		if err := os.WriteFile(filepath.Join(dir, "bobgen.yaml"), []byte(config), 0o644); err != nil {
			t.Fatalf("write module bobgen config %s: %v", module, err)
		}
	}

	env := []string{
		"PATH=" + fakeBin + string(os.PathListSeparator) + os.Getenv("PATH"),
		"SHIP_TEST_BOBGEN_LOG=" + logPath,
	}
	runShip(projectRoot, env, "db:generate")

	logBytes, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read fake bobgen log: %v", err)
	}
	lines := nonEmptyLines(string(logBytes))
	want := []string{
		"db/bobgen.yaml",
		"modules/alpha/db/bobgen.yaml",
		"modules/zeta/db/bobgen.yaml",
	}
	if len(lines) != len(want) {
		t.Fatalf("unexpected bobgen invocation count: got=%d want=%d lines=%v", len(lines), len(want), lines)
	}
	for i := range want {
		if lines[i] != want[i] {
			t.Fatalf("unexpected invocation order at index %d: got=%q want=%q full=%v", i, lines[i], want[i], lines)
		}
	}
}

func writeFakeBobgenLogger(t *testing.T, binDir string) {
	t.Helper()
	script := `#!/usr/bin/env bash
set -euo pipefail
if [ "${1:-}" != "-c" ] || [ -z "${2:-}" ]; then
  echo "expected bobgen-sql -c <config>" >&2
  exit 2
fi
if [ -z "${SHIP_TEST_BOBGEN_LOG:-}" ]; then
  echo "SHIP_TEST_BOBGEN_LOG is required" >&2
  exit 2
fi
printf '%s\n' "$2" >> "$SHIP_TEST_BOBGEN_LOG"
`
	path := filepath.Join(binDir, "bobgen-sql")
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake bobgen logger: %v", err)
	}
}
