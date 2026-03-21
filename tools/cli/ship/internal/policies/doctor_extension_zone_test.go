package policies

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunDoctorChecks_ExtensionZoneContract_RedSpec(t *testing.T) {
	root := repoRootForPolicyContractTest(t)

	zoneDoc := mustReadPolicyContractText(t, filepath.Join(root, "docs", "architecture", "10-extension-zones.md"))
	cliDoc := mustReadPolicyContractText(t, filepath.Join(root, "docs", "reference", "01-cli.md"))
	policySource := mustReadPolicyContractText(t, filepath.Join(root, "tools", "cli", "ship", "internal", "policies", "doctor_repo_checks.go"))

	for _, token := range []string{"Extension Zones", "Protected Contract Zones", "`framework/`", "`container.go`", "`router.go`", "`schedules.go`"} {
		if !strings.Contains(zoneDoc, token) {
			t.Fatalf("extension zone doc should describe %s", token)
		}
	}
	for _, stale := range []string{"`app/`", "`app/router.go`", "`app/foundation/container.go`"} {
		if strings.Contains(zoneDoc, stale) {
			t.Fatalf("extension zone doc should not mention stale shell token %s", stale)
		}
	}
	if !strings.Contains(cliDoc, "extension-zone manifest") {
		t.Fatal("CLI reference should advertise doctor enforcement for the extension-zone manifest")
	}
	if !strings.Contains(policySource, "checkExtensionZoneManifest") {
		t.Fatal("doctor repo checks should expose a dedicated extension-zone manifest check")
	}
}

func repoRootForPolicyContractTest(t *testing.T) string {
	t.Helper()

	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "Makefile")); err == nil {
			if _, err := os.Stat(filepath.Join(dir, ".docket")); err == nil {
				return dir
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find repo root")
		}
		dir = parent
	}
}

func mustReadPolicyContractText(t *testing.T, path string) string {
	t.Helper()

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(content)
}
