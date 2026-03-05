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

func TestShipModuleFlow_MigrateAndGenerate(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go toolchain not available")
	}

	repoRoot := mustRepoRootFromFile(t)
	shipBin := filepath.Join(t.TempDir(), "ship")

	buildCtx, buildCancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer buildCancel()
	build := exec.CommandContext(buildCtx, "go", "build", "-o", shipBin, "./cmd/ship")
	build.Dir = repoRoot
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build ship binary: %v: %s", err, string(out))
	}

	workspace := filepath.Join(t.TempDir(), "workspace")
	requireNoErr(t, os.MkdirAll(workspace, 0o755))

	fakeBin := filepath.Join(t.TempDir(), "fake-bin")
	requireNoErr(t, os.MkdirAll(fakeBin, 0o755))
	gooseLog := filepath.Join(t.TempDir(), "goose.log")
	bobgenLog := filepath.Join(t.TempDir(), "bobgen.log")
	writeFakeGoose(t, fakeBin)
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
				t.Skipf("skipping module flow integration (environment/toolchain constraint): %v: %s", err, msg)
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

	moduleConfig := "modules:\n  - alpha\n"
	requireNoErr(t, os.WriteFile(filepath.Join(projectRoot, "config", "modules.yaml"), []byte(moduleConfig), 0o644))
	requireNoErr(t, os.MkdirAll(filepath.Join(projectRoot, "modules", "alpha", "db", "migrate", "migrations"), 0o755))
	requireNoErr(t, os.WriteFile(
		filepath.Join(projectRoot, "modules", "alpha", "db", "bobgen.yaml"),
		[]byte("sql:\n  dialect: psql\n  pattern: \"modules/alpha/db/migrate/migrations/*.sql\"\noutput: \"modules/alpha/db/gen\"\n"),
		0o644,
	))

	env := []string{
		"PATH=" + fakeBin + string(os.PathListSeparator) + os.Getenv("PATH"),
		"DATABASE_URL=sqlite://file:module_flow.db?_fk=1",
		"SHIP_TEST_GOOSE_LOG=" + gooseLog,
		"SHIP_TEST_BOBGEN_LOG=" + bobgenLog,
	}
	runShip(projectRoot, env, "db:migrate")
	statusOut := runShip(projectRoot, env, "db:status")
	runShip(projectRoot, env, "db:generate")
	if !strings.Contains(statusOut, "== core migrations ==") || !strings.Contains(statusOut, "== module alpha migrations ==") {
		t.Fatalf("expected status output to include core/module sections, got: %s", statusOut)
	}

	gooseLines := nonEmptyLines(readFile(t, gooseLog))
	if len(gooseLines) != 4 ||
		gooseLines[0] != "db/migrate/migrations" ||
		gooseLines[1] != "modules/alpha/db/migrate/migrations" ||
		gooseLines[2] != "db/migrate/migrations" ||
		gooseLines[3] != "modules/alpha/db/migrate/migrations" {
		t.Fatalf("unexpected goose module flow: %v", gooseLines)
	}

	bobgenLines := nonEmptyLines(readFile(t, bobgenLog))
	if len(bobgenLines) != 2 || bobgenLines[0] != "db/bobgen.yaml" || bobgenLines[1] != "modules/alpha/db/bobgen.yaml" {
		t.Fatalf("unexpected bobgen module flow: %v", bobgenLines)
	}
}

func TestShipModuleMigrateFailsWhenModuleMigrationsMissing(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go toolchain not available")
	}

	repoRoot := mustRepoRootFromFile(t)
	shipBin := filepath.Join(t.TempDir(), "ship")
	build := exec.Command("go", "build", "-o", shipBin, "./cmd/ship")
	build.Dir = repoRoot
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build ship binary: %v: %s", err, string(out))
	}

	workspace := filepath.Join(t.TempDir(), "workspace")
	requireNoErr(t, os.MkdirAll(workspace, 0o755))

	cmd := exec.Command(shipBin, "new", "demo", "--module", "example.com/demo")
	cmd.Dir = workspace
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("ship new failed: %v: %s", err, string(out))
	}

	projectRoot := filepath.Join(workspace, "demo")
	moduleConfig := "modules:\n  - missingmod\n"
	requireNoErr(t, os.WriteFile(filepath.Join(projectRoot, "config", "modules.yaml"), []byte(moduleConfig), 0o644))

	cmd = exec.Command(shipBin, "db:migrate")
	cmd.Dir = projectRoot
	cmd.Env = append(os.Environ(), "DATABASE_URL=sqlite://file:module_fail.db?_fk=1")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected db:migrate to fail for missing module migrations dir, got success: %s", string(out))
	}
	if !strings.Contains(string(out), "missing migrations directory") {
		t.Fatalf("expected missing migrations error, got: %s", string(out))
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(b)
}

func requireNoErr(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
