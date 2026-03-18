package commands

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestAdminScaffoldContract_RedSpec(t *testing.T) {
	root := repoRootFromCommandsTest(t)

	makefile := mustReadText(t, filepath.Join(root, "Makefile"))
	packageJSON := mustReadText(t, filepath.Join(root, "tests", "e2e", "package.json"))
	devGuide := mustReadText(t, filepath.Join(root, "docs", "guides", "02-development-workflows.md"))
	extensionZones := mustReadText(t, filepath.Join(root, "docs", "architecture", "10-extension-zones.md"))
	scopeDoc := mustReadText(t, filepath.Join(root, "docs", "architecture", "03-project-scope-analysis.md"))

	for _, token := range []string{
		".PHONY: e2e-admin-smoke",
		"e2e-admin-smoke:",
	} {
		if !strings.Contains(makefile, token) {
			t.Fatalf("Makefile should include %q", token)
		}
	}

	if !strings.Contains(packageJSON, `"test:admin-smoke": "playwright test tests/admin_scaffold.spec.ts"`) {
		t.Fatal("e2e package should define a dedicated admin smoke lane")
	}

	if !strings.Contains(devGuide, "make e2e-admin-smoke") {
		t.Fatal("development workflow guide should document the admin smoke lane")
	}

	if !strings.Contains(extensionZones, "`modules/admin/`") {
		t.Fatal("extension zones doc should call out modules/admin as the admin boundary")
	}
	if !strings.Contains(scopeDoc, "Playwright baseline smoke coverage") {
		t.Fatal("project scope analysis should mention Playwright baseline smoke coverage for admin")
	}
}
