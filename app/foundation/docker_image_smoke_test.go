//go:build integration

package foundation

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestDockerImageBuildAndWebStartup(t *testing.T) {
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker not available in PATH")
	}

	repoRoot := mustRepoRoot(t)
	imageTag := fmt.Sprintf("goship-integration-smoke:%d", time.Now().UnixNano())
	containerName := fmt.Sprintf("goship-smoke-%d", time.Now().UnixNano())

	t.Cleanup(func() {
		_ = runCommand(context.Background(), "docker", "rm", "-f", containerName)
		_ = runCommand(context.Background(), "docker", "rmi", "-f", imageTag)
	})

	buildCtx, cancelBuild := context.WithTimeout(context.Background(), 12*time.Minute)
	defer cancelBuild()
	_, err := commandOutput(buildCtx,
		"docker", "build", "--network", "host", "-t", imageTag, "-f", filepath.Join(repoRoot, "infra", "docker", "Dockerfile"), repoRoot,
	)
	if err != nil {
		if isTransientNetworkFailure(err.Error()) {
			t.Skipf("skipping docker smoke test due to transient network failure: %v", err)
		}
		t.Fatalf("docker build failed: %v", err)
	}

	runCtx, cancelRun := context.WithTimeout(context.Background(), time.Minute)
	defer cancelRun()
	if err := runCommand(runCtx,
		"docker", "run", "-d", "--rm", "--name", containerName,
		"-e", "PAGODA_APP_ENVIRONMENT=local",
		imageTag, "web",
	); err != nil {
		t.Fatalf("docker run failed: %v", err)
	}

	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		running, err := commandOutput(context.Background(), "docker", "inspect", "-f", "{{.State.Running}}", containerName)
		if err != nil {
			break
		}
		if strings.TrimSpace(running) == "true" {
			return
		}
		time.Sleep(500 * time.Millisecond)
	}

	logs, _ := commandOutput(context.Background(), "docker", "logs", containerName)
	t.Fatalf("container did not stay running; logs:\n%s", logs)
}

func mustRepoRoot(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to resolve test file path")
	}
	start := filepath.Dir(thisFile)
	root, err := findRepoRoot(start)
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func findRepoRoot(start string) (string, error) {
	dir := filepath.Clean(start)
	for {
		if hasFile(filepath.Join(dir, "go.mod")) && hasFile(filepath.Join(dir, "infra", "docker", "Dockerfile")) {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", errors.New("failed to locate repo root (expected go.mod and infra/docker/Dockerfile)")
}

func hasFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func runCommand(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s %s: %w\n%s", name, strings.Join(args, " "), err, string(out))
	}
	return nil
}

func commandOutput(ctx context.Context, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%s %s: %w\n%s", name, strings.Join(args, " "), err, string(out))
	}
	return string(out), nil
}

func isTransientNetworkFailure(msg string) bool {
	probes := []string{
		"lookup storage.googleapis.com",
		"i/o timeout",
		"no such host",
		"tls handshake timeout",
		"connection reset by peer",
	}
	for _, p := range probes {
		if strings.Contains(strings.ToLower(msg), strings.ToLower(p)) {
			return true
		}
	}
	return false
}
