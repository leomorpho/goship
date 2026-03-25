package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVerifyContract_DefinesStandaloneExportabilityGate_RedSpec(t *testing.T) {
	root := repoRootFromCommandsTest(t)

	verifySource := mustReadText(t, filepath.Join(root, "tools", "cli", "ship", "internal", "commands", "verify.go"))
	verifyTests := mustReadText(t, filepath.Join(root, "tools", "cli", "ship", "internal", "commands", "project_new_integration_test.go"))
	cliDoc := mustReadText(t, filepath.Join(root, "docs", "reference", "01-cli.md"))

	if !strings.Contains(verifySource, "standalone exportability gate") {
		t.Fatal("verify should expose a named standalone exportability gate step")
	}
	if !strings.Contains(verifyTests, "control-plane dependency") {
		t.Fatal("project new integration tests should define a standalone exportability check against control-plane dependency drift")
	}
	if !strings.Contains(verifyTests, "PAGODA_MANAGED_MODE") {
		t.Fatal("project new integration tests should prove managed env-var startup without requiring control-plane source")
	}
	if !strings.Contains(verifyTests, "without managed env vars") {
		t.Fatal("project new integration tests should prove standalone startup behavior without managed env vars")
	}
	if !strings.Contains(cliDoc, "run-anywhere verification gate") {
		t.Fatal("CLI reference should describe the run-anywhere verification gate")
	}
}

func TestStarterScaffold_RemainsControlPlaneIndependent_RedSpec(t *testing.T) {
	root := repoRootFromCommandsTest(t)
	scaffoldRoot := filepath.Join(root, "tools", "cli", "ship", "internal", "templates", "starter", "testdata", "scaffold")

	var found bool
	err := filepath.WalkDir(scaffoldRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		text := mustReadText(t, path)
		if strings.Contains(strings.ToLower(text), "control-plane dependency") {
			found = true
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk scaffold: %v", err)
	}
	if found {
		t.Fatal("starter scaffold should not encode control-plane dependency wording")
	}
}

func TestCheckStandaloneExportability_RejectsControlPlaneImports_RedSpec(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "framework", "bridge")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	source := `package bridge

import cp "github.com/leomorpho/goship/tools/private/control-plane/sdk"

func _() { _ = cp.Client{} }
`
	if err := os.WriteFile(filepath.Join(target, "bad.go"), []byte(source), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	err := checkStandaloneExportability(root)
	if err == nil {
		t.Fatal("expected control-plane import coupling to fail standalone exportability")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "control-plane") {
		t.Fatalf("expected control-plane coupling error, got %v", err)
	}
}

func TestCheckStandaloneExportability_RejectsPrivateControlPlanePathStrings_RedSpec(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "framework", "contracts")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	source := `package contracts

const privateRuntimeContractPath = "../../tools/private/control-plane/docs/runtime.md"
`
	if err := os.WriteFile(filepath.Join(target, "bad.go"), []byte(source), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	err := checkStandaloneExportability(root)
	if err == nil {
		t.Fatal("expected private control-plane path assumption to fail standalone exportability")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "control-plane") {
		t.Fatalf("expected control-plane coupling error, got %v", err)
	}
}
