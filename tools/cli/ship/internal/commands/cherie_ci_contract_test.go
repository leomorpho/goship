package commands

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestCIContract_DefinesCherieCompatibilitySmokeBaseline_RedSpec(t *testing.T) {
	root := repoRootFromCommandsTest(t)

	workflow := mustReadText(t, filepath.Join(root, ".github", "workflows", "test.yml"))
	packageJSON := mustReadText(t, filepath.Join(root, "tests", "e2e", "package.json"))
	specPath := filepath.Join(root, "tests", "e2e", "tests", "cherie_compatibility.spec.ts")
	spec := mustReadText(t, specPath)

	if !strings.Contains(workflow, "\n  cherie_compatibility_smoke:\n") {
		t.Fatal("test workflow should define a dedicated cherie_compatibility_smoke job")
	}
	if !strings.Contains(workflow, "\n  verify_strict:\n") {
		t.Fatal("test workflow should define a dedicated verify_strict job for the Cherie sync baseline")
	}
	if !strings.Contains(workflow, "go run ./tools/cli/ship/cmd/ship verify --profile strict") {
		t.Fatal("strict verify workflow should invoke ship verify --profile strict")
	}
	if !strings.Contains(workflow, "needs: verify_strict") {
		t.Fatal("Cherie compatibility workflow should depend on verify_strict so the smoke gate is a downstream required check")
	}
	if !strings.Contains(workflow, "npm run test:cherie-smoke") {
		t.Fatal("Cherie compatibility workflow should invoke npm run test:cherie-smoke")
	}
	if !strings.Contains(packageJSON, `"test:cherie-smoke": "playwright test tests/cherie_compatibility.spec.ts"`) {
		t.Fatal("tests/e2e/package.json should define a dedicated test:cherie-smoke script")
	}
	for _, token := range []string{`"/up"`, `"/user/login"`, `"/auth/realtime"`} {
		if !strings.Contains(spec, token) {
			t.Fatalf("Cherie compatibility smoke spec should cover %s", token)
		}
	}
}
