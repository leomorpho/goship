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
