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

func TestShipDBMigrateRunsCoreThenSortedModules(t *testing.T) {
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
	logPath := filepath.Join(t.TempDir(), "goose.log")
	writeFakeGoose(t, fakeBin)

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
				t.Skipf("skipping modules order integration (environment/toolchain constraint): %v: %s", err, msg)
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
		dir := filepath.Join(projectRoot, "modules", module, "db", "migrate", "migrations")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir module migration dir %s: %v", module, err)
		}
	}

	env := []string{
		"PATH=" + fakeBin + string(os.PathListSeparator) + os.Getenv("PATH"),
		"DATABASE_URL=sqlite://file:module_order.db?_fk=1",
		"SHIP_TEST_GOOSE_LOG=" + logPath,
	}
	runShip(projectRoot, env, "db:migrate")

	logBytes, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read fake goose log: %v", err)
	}
	lines := nonEmptyLines(string(logBytes))
	want := []string{
		"db/migrate/migrations",
		"modules/alpha/db/migrate/migrations",
		"modules/zeta/db/migrate/migrations",
	}
	if len(lines) != len(want) {
		t.Fatalf("unexpected goose invocation count: got=%d want=%d lines=%v", len(lines), len(want), lines)
	}
	for i := range want {
		if lines[i] != want[i] {
			t.Fatalf("unexpected invocation order at index %d: got=%q want=%q full=%v", i, lines[i], want[i], lines)
		}
	}
}

func writeFakeGoose(t *testing.T, binDir string) {
	t.Helper()
	script := `#!/usr/bin/env bash
set -euo pipefail
if [ "${1:-}" != "-dir" ] || [ -z "${2:-}" ]; then
  echo "expected goose -dir <dir> ..." >&2
  exit 2
fi
if [ -z "${SHIP_TEST_GOOSE_LOG:-}" ]; then
  echo "SHIP_TEST_GOOSE_LOG is required" >&2
  exit 2
fi
printf '%s\n' "$2" >> "$SHIP_TEST_GOOSE_LOG"
`
	path := filepath.Join(binDir, "goose")
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake goose: %v", err)
	}
}

func nonEmptyLines(v string) []string {
	parts := strings.Split(v, "\n")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}
