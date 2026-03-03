//go:build integration

package ship

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

func TestShipNewModelAndMigrationsFlow(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go toolchain not available")
	}

	repoRoot := mustRepoRootFromFile(t)
	shipBin := filepath.Join(t.TempDir(), "ship")

	buildCtx, buildCancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer buildCancel()

	build := exec.CommandContext(buildCtx, "go", "build", "-o", shipBin, "./cli/ship/cmd/ship")
	build.Dir = repoRoot
	buildOut, buildErr := build.CombinedOutput()
	if buildErr != nil {
		t.Fatalf("build ship binary: %v: %s", buildErr, string(buildOut))
	}

	workspace := filepath.Join(t.TempDir(), "workspace")
	if err := os.MkdirAll(workspace, 0o755); err != nil {
		t.Fatalf("mkdir workspace: %v", err)
	}

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
			if isLikelyNetworkFailure(msg) {
				t.Skipf("skipping integration flow (network unavailable while resolving tool deps): %v: %s", err, msg)
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

	runShip(projectRoot, nil, "make:model", "Post", "title:string")
	if _, err := os.Stat(filepath.Join(projectRoot, "app", "goship", "db", "schema", "post.go")); err != nil {
		t.Fatalf("expected generated schema for Post: %v", err)
	}

	runShip(projectRoot, nil, "db:make", "add_posts")
	matches, err := filepath.Glob(filepath.Join(projectRoot, "app", "goship", "db", "migrate", "migrations", "*add_posts*.sql"))
	if err != nil {
		t.Fatalf("glob migration files: %v", err)
	}
	if len(matches) == 0 {
		t.Fatalf("expected migration file containing add_posts, got none")
	}

	dbURL := "sqlite://file:ship_flow_test.db?_fk=1"
	runShip(projectRoot, []string{"DATABASE_URL=" + dbURL}, "db:migrate")
	if _, err := os.Stat(filepath.Join(projectRoot, "ship_flow_test.db")); err != nil {
		t.Fatalf("expected sqlite db file after migration: %v", err)
	}
}

func isLikelyNetworkFailure(msg string) bool {
	lower := strings.ToLower(msg)
	return strings.Contains(msg, "proxy.golang.org") ||
		strings.Contains(lower, "no such host") ||
		strings.Contains(lower, "dial tcp") ||
		strings.Contains(lower, "tls handshake timeout")
}

