//go:build integration

package integration

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestEntGenerateFromDBSchemaPath(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go toolchain not available")
	}

	shipRoot := mustRepoRootFromFile(t)
	workspaceRoot := filepath.Clean(filepath.Join(shipRoot, "..", "..", ".."))
	outDir := filepath.Join(workspaceRoot, "tmp", "ent-generate-smoke-"+strconv.FormatInt(time.Now().UnixNano(), 10))
	modCache := filepath.Join(t.TempDir(), "go-mod-integration")
	t.Cleanup(func() {
		_ = os.RemoveAll(outDir)
	})
	if err := os.MkdirAll(modCache, 0o755); err != nil {
		t.Fatalf("mkdir mod cache: %v", err)
	}
	outRel, err := filepath.Rel(workspaceRoot, outDir)
	if err != nil {
		t.Fatalf("resolve relative out dir: %v", err)
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatalf("mkdir out dir: %v", err)
	}
	// Ent expects the target to be a valid Go package root.
	if err := os.WriteFile(filepath.Join(outDir, "generate.go"), []byte("package ent\n"), 0o644); err != nil {
		t.Fatalf("write target package stub: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, "go", "run",
		"entgo.io/ent/cmd/ent",
		"generate",
		"--feature", "sql/upsert,sql/execquery",
		"--target", "./"+filepath.ToSlash(outRel),
		"./db/schema",
	)
	cmd.Dir = workspaceRoot
	cmd.Env = append(os.Environ(),
		"GOMODCACHE="+modCache,
		"GOFLAGS=-modcacherw",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := string(out)
		// CI/dev environments without network access should not fail this suite.
		if strings.Contains(msg, "proxy.golang.org") ||
			strings.Contains(strings.ToLower(msg), "no such host") ||
			strings.Contains(strings.ToLower(msg), "dial tcp") {
			t.Skipf("skipping ent generate smoke (network unavailable): %v", err)
		}
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			t.Fatalf("ent generate failed (%d): %s", exitErr.ExitCode(), msg)
		}
		t.Fatalf("ent generate failed: %v: %s", err, msg)
	}

	if _, statErr := os.Stat(filepath.Join(outDir, "ent.go")); statErr != nil {
		t.Fatalf("expected generated ent output in %s: %v", outDir, statErr)
	}
}

func mustRepoRootFromFile(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to resolve test file path")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}
