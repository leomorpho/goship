//go:build integration

package integration

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func mustRepoRootFromFile(t *testing.T) string {
	t.Helper()

	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to resolve current test file path")
	}

	dir := filepath.Clean(filepath.Dir(thisFile))
	for {
		goMod := filepath.Join(dir, "go.mod")
		shipMain := filepath.Join(dir, "tools", "cli", "ship", "cmd", "ship", "main.go")
		if fileExists(goMod) && fileExists(shipMain) {
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	t.Fatalf("failed to locate repository root from %s", thisFile)
	return ""
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
