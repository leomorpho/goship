package commands

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestCIContract_DefinesDedicatedIsolationAndPortabilitySuites_RedSpec(t *testing.T) {
	root := repoRootFromCommandsTest(t)

	workflow := mustReadText(t, filepath.Join(root, ".github", "workflows", "test.yml"))
	makefile := mustReadText(t, filepath.Join(root, "Makefile"))
	moduleGate := mustReadText(t, filepath.Join(root, "tools", "scripts", "check-module-isolation.sh"))
	devGuide := mustReadText(t, filepath.Join(root, "docs", "guides", "02-development-workflows.md"))

	if !strings.Contains(workflow, "\n  module_isolation:\n") {
		t.Fatal("test workflow should define a dedicated module_isolation job")
	}
	if !strings.Contains(workflow, "run: make test-module-isolation") {
		t.Fatal("module isolation CI job should invoke make test-module-isolation")
	}
	if !strings.Contains(makefile, ".PHONY: test-module-isolation") {
		t.Fatal("Makefile should expose a canonical test-module-isolation entrypoint for CI")
	}
	if !strings.Contains(moduleGate, "module=") {
		t.Fatal("module isolation gate should report offending module context")
	}
	if !strings.Contains(moduleGate, "stale allowlist entry") {
		t.Fatal("module isolation gate should reject stale allowlist entries")
	}
	if !strings.Contains(devGuide, "make test-module-isolation") {
		t.Fatal("development workflow guide should document the module isolation gate")
	}
	if !strings.Contains(workflow, "\n  sql_portability:\n") {
		t.Fatal("test workflow should define a dedicated sql_portability job")
	}
	if !strings.Contains(workflow, "run: make test-sql-portability") {
		t.Fatal("sql portability CI job should invoke make test-sql-portability")
	}
	if !strings.Contains(makefile, ".PHONY: test-sql-portability") {
		t.Fatal("Makefile should expose a canonical test-sql-portability entrypoint for CI")
	}
}

func TestCIContract_DefinesGeneratorSnapshotAndIdempotencyGate_RedSpec(t *testing.T) {
	root := repoRootFromCommandsTest(t)

	workflow := mustReadText(t, filepath.Join(root, ".github", "workflows", "test.yml"))
	makefile := mustReadText(t, filepath.Join(root, "Makefile"))
	devGuide := mustReadText(t, filepath.Join(root, "docs", "guides", "02-development-workflows.md"))
	gateScript := mustReadText(t, filepath.Join(root, "tools", "scripts", "check-generator-contracts.sh"))

	if !strings.Contains(workflow, "\n  generator_contracts:\n") {
		t.Fatal("test workflow should define a dedicated generator_contracts job")
	}
	if !strings.Contains(workflow, "run: make test-generator-contracts") {
		t.Fatal("generator contract CI job should invoke make test-generator-contracts")
	}
	if !strings.Contains(makefile, ".PHONY: test-generator-contracts") {
		t.Fatal("Makefile should expose a canonical test-generator-contracts entrypoint for CI")
	}
	if !strings.Contains(makefile, ".PHONY: test-generator-idempotency") {
		t.Fatal("Makefile should expose a canonical test-generator-idempotency entrypoint")
	}
	if !strings.Contains(gateScript, "bash tools/scripts/check-generator-snapshots.sh") {
		t.Fatal("generator contract gate should invoke the snapshot runner explicitly")
	}
	if !strings.Contains(gateScript, "bash tools/scripts/check-generator-idempotency.sh") {
		t.Fatal("generator contract gate should invoke the idempotency runner explicitly")
	}
	if !strings.Contains(devGuide, "make test-generator-idempotency") {
		t.Fatal("development workflow guide should document the standalone idempotency runner")
	}
	if !strings.Contains(devGuide, "UPDATE_GENERATOR_SNAPSHOTS=1 make test-generator-contracts") {
		t.Fatal("development workflow guide should document the explicit snapshot refresh path")
	}
}

func TestCIContract_DefinesUIChangeVerificationPolicyAndBrowserEvidence_RedSpec(t *testing.T) {
	root := repoRootFromCommandsTest(t)

	workflow := mustReadText(t, filepath.Join(root, ".github", "workflows", "test.yml"))
	devGuide := mustReadText(t, filepath.Join(root, "docs", "guides", "02-development-workflows.md"))
	risksDoc := mustReadText(t, filepath.Join(root, "docs", "architecture", "06-known-gaps-and-risks.md"))

	for _, token := range []string{
		"\n  e2e_smoke:\n",
		"\n  cherie_compatibility_smoke:\n",
		"name: playwright-smoke-report",
		"name: playwright-cherie-smoke-report",
	} {
		if !strings.Contains(workflow, token) {
			t.Fatalf("test workflow should include %q for browser evidence coverage", token)
		}
	}

	for _, token := range []string{
		"UI-impacting changes should add or update Playwright coverage for the affected flow",
		"browser evidence should be attached or referenced in ticket or PR notes",
		"tests/e2e/playwright-report",
		"tests/e2e/test-results",
	} {
		if !strings.Contains(devGuide, token) {
			t.Fatalf("development workflow guide should include %q", token)
		}
	}

	if !strings.Contains(risksDoc, "browser evidence") {
		t.Fatal("known-gaps doc should explain how the Playwright smoke lanes provide browser evidence")
	}
}
