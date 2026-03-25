package commands

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestDocsRouteContract_RedSpec(t *testing.T) {
	root := repoRootFromCommandsTest(t)

	routeDoc := mustReadText(t, filepath.Join(root, "docs", "architecture", "04-http-routes.md"))
	scopeDoc := mustReadText(t, filepath.Join(root, "docs", "architecture", "03-project-scope-analysis.md"))

	for _, token := range []string{
		"POST /managed/backup",
		"POST /managed/restore",
		"GET /managed/status",
		"GET /auth/realtime",
		"GET /dev/mail",
		"GET /auth/admin/managed-settings",
		"GET /auth/admin/flags",
		"GET /auth/admin/trash",
	} {
		if !strings.Contains(routeDoc, token) {
			t.Fatalf("route doc should include %q", token)
		}
	}
	if strings.Contains(routeDoc, "GET /install-app") {
		t.Fatal("route doc should not list removed /install-app route")
	}
	for _, token := range []string{
		"Notification-center routes are owned by `modules/notifications/routes`",
		"Managed settings status page at `/auth/admin/managed-settings`",
		"CI now carries a dedicated Cherie compatibility smoke baseline",
	} {
		if !strings.Contains(scopeDoc, token) {
			t.Fatalf("scope doc should include %q", token)
		}
	}
}

func TestCIContract_DefinesDocSyncAndDeadRouteGuards_RedSpec(t *testing.T) {
	root := repoRootFromCommandsTest(t)

	workflow := mustReadText(t, filepath.Join(root, ".github", "workflows", "test.yml"))
	makefile := mustReadText(t, filepath.Join(root, "Makefile"))
	devGuide := mustReadText(t, filepath.Join(root, "docs", "guides", "02-development-workflows.md"))

	if !strings.Contains(workflow, "\n  doc_sync:\n") {
		t.Fatal("test workflow should define a dedicated doc_sync job")
	}
	if !strings.Contains(workflow, "run: make test-doc-sync") {
		t.Fatal("doc_sync CI job should invoke make test-doc-sync")
	}
	if !strings.Contains(workflow, "\n  dead_route_regression:\n") {
		t.Fatal("test workflow should define a dedicated dead_route_regression job")
	}
	if !strings.Contains(workflow, "run: make test-dead-routes") {
		t.Fatal("dead_route_regression CI job should invoke make test-dead-routes")
	}
	if !strings.Contains(makefile, ".PHONY: test-doc-sync") {
		t.Fatal("Makefile should expose a canonical test-doc-sync entrypoint")
	}
	if !strings.Contains(makefile, ".PHONY: test-dead-routes") {
		t.Fatal("Makefile should expose a canonical test-dead-routes entrypoint")
	}
	if !strings.Contains(devGuide, "make test-doc-sync") || !strings.Contains(devGuide, "make test-dead-routes") {
		t.Fatal("development workflow guide should document the doc-sync and dead-route guardrails")
	}
}

func TestCIContract_DefinesDocDriftGate_RedSpec(t *testing.T) {
	root := repoRootFromCommandsTest(t)

	workflow := mustReadText(t, filepath.Join(root, ".github", "workflows", "test.yml"))
	makefile := mustReadText(t, filepath.Join(root, "Makefile"))

	if !strings.Contains(workflow, "\n  doc_drift:\n") {
		t.Fatal("test workflow should define a dedicated doc_drift job")
	}
	if !strings.Contains(workflow, "run: make test-doc-drift") {
		t.Fatal("doc_drift CI job should invoke make test-doc-drift")
	}
	if !strings.Contains(makefile, ".PHONY: test-doc-drift") {
		t.Fatal("Makefile should expose a canonical test-doc-drift entrypoint")
	}
}

func TestDocs_DBExportAndRuntimeReportContractsStayInSync_RedSpec(t *testing.T) {
	root := repoRootFromCommandsTest(t)

	cliDoc := mustReadText(t, filepath.Join(root, "docs", "reference", "01-cli.md"))
	scopeDoc := mustReadText(t, filepath.Join(root, "docs", "architecture", "03-project-scope-analysis.md"))
	risksDoc := mustReadText(t, filepath.Join(root, "docs", "architecture", "06-known-gaps-and-risks.md"))

	for _, token := range []string{
		"`ship db:export [--json]` -> reports the SQLite export manifest checksum contract from current runtime metadata; `--json` emits a structured export report with the typed backup manifest payload, suggested next commands, and planning note for agents/tooling",
		"`ship db:promote` text and `--json` outputs now surface the canonical SQLite-to-Postgres manual runbook path (`docs/guides/14-sqlite-to-postgres-promotion-runbook.md`) so humans and LLMs can continue the manual-first workflow from one source of truth.",
		"`ship runtime:report --json` -> machine-readable runtime capability report covering active profile, adapters, process plan, source-aware `process_topology` (including web/worker realtime roles when enabled), web features, DB runtime metadata, managed-key sources, current framework version, per-module adoption metadata, upgrade readiness metadata, and a versioned handshake envelope",
	} {
		if !strings.Contains(cliDoc, token) {
			t.Fatalf("CLI reference should include %q", token)
		}
	}

	for _, token := range []string{
		"`ship db:export --json` exposes a structured SQLite export report with a typed `backup-manifest-v1` payload, checksum evidence, suggested next commands, and a planning-only note for agents/tooling.",
		"`ship db:promote` now publishes the canonical manual runbook doc path (`docs/guides/14-sqlite-to-postgres-promotion-runbook.md`) in both text and JSON outputs so operators and agents can continue with one deterministic checklist.",
		"`ship runtime:report --json` emits the canonical machine-readable runtime capability payload from config/runtime-plan metadata, including active profile, adapters, process plan, web features, DB runtime metadata, managed-key sources, current framework version, per-module adoption metadata, and a versioned handshake envelope.",
	} {
		if !strings.Contains(scopeDoc, token) {
			t.Fatalf("scope doc should include %q", token)
		}
	}

	for _, token := range []string{
		"`ship db:export --json` already emits a structured export report with checksum-backed `backup-manifest-v1` evidence and follow-up command hints, but the underlying import/verification engine is still manual-first.",
		"Operators should use `docs/guides/14-sqlite-to-postgres-promotion-runbook.md` as the canonical manual-first SQLite-to-Postgres promotion runbook surfaced directly by `ship db:promote` output.",
		"`ship runtime:report --json` now exposes the effective profile, adapters, process plan, web features, DB runtime metadata, managed-key sources, current framework version, per-module adoption metadata, and a versioned handshake envelope for orchestration preflight.",
	} {
		if !strings.Contains(risksDoc, token) {
			t.Fatalf("known-gaps doc should include %q", token)
		}
	}
}

func TestDocs_StagedRolloutDecisionContractStaysInSync_RedSpec(t *testing.T) {
	root := repoRootFromCommandsTest(t)

	cliDoc := mustReadText(t, filepath.Join(root, "docs", "reference", "01-cli.md"))
	scopeDoc := mustReadText(t, filepath.Join(root, "docs", "architecture", "03-project-scope-analysis.md"))
	managedDoc := mustReadText(t, filepath.Join(root, "docs", "architecture", "09-standalone-and-managed-mode.md"))
	risksDoc := mustReadText(t, filepath.Join(root, "docs", "architecture", "06-known-gaps-and-risks.md"))
	roadmapDoc := mustReadText(t, filepath.Join(root, "docs", "roadmap", "01-framework-plan.md"))

	for _, token := range []string{
		"staged-rollout-decision-v1",
		"ship runtime:report --json",
		"policy_input_version",
	} {
		if !strings.Contains(cliDoc, token) {
			t.Fatalf("CLI reference should include %q", token)
		}
		if !strings.Contains(scopeDoc, token) {
			t.Fatalf("scope doc should include %q", token)
		}
		if !strings.Contains(managedDoc, token) {
			t.Fatalf("managed-mode doc should include %q", token)
		}
		if !strings.Contains(risksDoc, token) {
			t.Fatalf("known-gaps doc should include %q", token)
		}
		if !strings.Contains(roadmapDoc, token) {
			t.Fatalf("roadmap should include %q", token)
		}
	}
}

func TestDocs_UpgradeReadinessContractStaysInSync_RedSpec(t *testing.T) {
	root := repoRootFromCommandsTest(t)

	cliDoc := mustReadText(t, filepath.Join(root, "docs", "reference", "01-cli.md"))

	for _, token := range []string{
		"`ship upgrade --json`",
		"upgrade-readiness-v1",
		"schema_version",
		"blocker_classification",
		"target_version",
		"rollback_target",
		"canary",
		"verification",
		"plan",
		"safe_steps",
		"result",
		"blockers",
		"manual_follow_ups",
		"remediation_hints",
		"planned_changes",
	} {
		if !strings.Contains(cliDoc, token) {
			t.Fatalf("CLI reference should include %q", token)
		}
	}
}

func TestDocs_FrameworkFirstRuntimeSeamsStayCanonical_RedSpec(t *testing.T) {
	root := repoRootFromCommandsTest(t)

	architectureDoc := mustReadText(t, filepath.Join(root, "docs", "architecture", "01-architecture.md"))
	scopeDoc := mustReadText(t, filepath.Join(root, "docs", "architecture", "03-project-scope-analysis.md"))
	cognitiveDoc := mustReadText(t, filepath.Join(root, "docs", "architecture", "08-cognitive-model.md"))
	agentGuide := mustReadText(t, filepath.Join(root, "docs", "guides", "01-ai-agent-guide.md"))
	indexDoc := mustReadText(t, filepath.Join(root, "docs", "00-index.md"))

	for _, token := range []string{
		"`container.go`",
		"`router.go`",
		"`schedules.go`",
	} {
		if !strings.Contains(architectureDoc, token) {
			t.Fatalf("architecture doc should include %q", token)
		}
		if !strings.Contains(cognitiveDoc, token) {
			t.Fatalf("cognitive model doc should include %q", token)
		}
		if !strings.Contains(agentGuide, token) {
			t.Fatalf("agent guide should include %q", token)
		}
		if !strings.Contains(indexDoc, token) {
			t.Fatalf("docs index should include %q", token)
		}
	}

	for _, token := range []string{
		"framework-first",
		"runtime seam",
	} {
		if !strings.Contains(architectureDoc, token) {
			t.Fatalf("architecture doc should include %q", token)
		}
		if !strings.Contains(scopeDoc, token) {
			t.Fatalf("scope doc should include %q", token)
		}
		if !strings.Contains(agentGuide, token) {
			t.Fatalf("agent guide should include %q", token)
		}
	}

	for _, token := range []string{
		"app/foundation",
		"app/router.go",
		"app/web/controllers",
		"app/views",
	} {
		if strings.Contains(architectureDoc, token) {
			t.Fatalf("architecture doc should not include deleted path %q", token)
		}
		if strings.Contains(cognitiveDoc, token) {
			t.Fatalf("cognitive model doc should not include deleted path %q", token)
		}
		if strings.Contains(agentGuide, token) {
			t.Fatalf("agent guide should not include deleted path %q", token)
		}
	}
}

func TestDocs_BlessedExternalFrontendContractStaysInSync_RedSpec(t *testing.T) {
	root := repoRootFromCommandsTest(t)

	apiGuide := mustReadText(t, filepath.Join(root, "docs", "guides", "08-building-an-api.md"))
	scopeDoc := mustReadText(t, filepath.Join(root, "docs", "architecture", "03-project-scope-analysis.md"))
	managedDoc := mustReadText(t, filepath.Join(root, "docs", "architecture", "09-standalone-and-managed-mode.md"))
	roadmapDoc := mustReadText(t, filepath.Join(root, "docs", "roadmap", "01-framework-plan.md"))
	cliDoc := mustReadText(t, filepath.Join(root, "docs", "reference", "01-cli.md"))

	for _, token := range []string{
		"api-only-same-origin-sveltekit-v1",
		"same-origin auth/session",
		"cookie/CSRF",
		"SvelteKit-first",
	} {
		if !strings.Contains(apiGuide, token) {
			t.Fatalf("API guide should include %q", token)
		}
		if !strings.Contains(scopeDoc, token) {
			t.Fatalf("scope doc should include %q", token)
		}
		if !strings.Contains(managedDoc, token) {
			t.Fatalf("managed-mode doc should include %q", token)
		}
		if !strings.Contains(roadmapDoc, token) {
			t.Fatalf("roadmap should include %q", token)
		}
		if !strings.Contains(cliDoc, token) {
			t.Fatalf("CLI doc should include %q", token)
		}
	}

	for _, token := range []string{
		"ship new demo --module example.com/demo --api-only",
		"Local Development Topology (SvelteKit + GoShip API)",
		"Deployment Topology (Same-Origin Requirement)",
		"/auth/login",
		"X-CSRF-Token",
	} {
		if !strings.Contains(apiGuide, token) {
			t.Fatalf("API guide should include concrete integration token %q", token)
		}
	}
}

func TestDocs_SvelteKitReferenceAppContract_RedSpec(t *testing.T) {
	root := repoRootFromCommandsTest(t)

	refReadmePath := filepath.Join(root, "examples", "sveltekit-api-only", "README.md")
	refContractPath := filepath.Join(root, "examples", "sveltekit-api-only", "src", "lib", "server", "goship-contract.ts")
	apiGuidePath := filepath.Join(root, "docs", "guides", "08-building-an-api.md")

	refReadme := mustReadText(t, refReadmePath)
	refContract := mustReadText(t, refContractPath)
	apiGuide := mustReadText(t, apiGuidePath)

	for _, token := range []string{
		"api-only-same-origin-sveltekit-v1",
		"ship new demo --module example.com/demo --api-only",
		"same-origin auth/session",
		"X-CSRF-Token",
		"/auth/login",
	} {
		if !strings.Contains(refReadme, token) {
			t.Fatalf("reference app README should include %q", token)
		}
	}

	for _, token := range []string{
		"export type GoshipResponseEnvelope",
		"export type GoshipAPIError",
		"export async function goshipFetch",
		"X-CSRF-Token",
	} {
		if !strings.Contains(refContract, token) {
			t.Fatalf("reference app TypeScript contract should include %q", token)
		}
	}

	if !strings.Contains(apiGuide, "examples/sveltekit-api-only/README.md") {
		t.Fatal("API guide should link to the canonical sveltekit-api-only reference app")
	}
}

func TestDocs_ReadmeLandingNarrativeStaysAligned_RedSpec(t *testing.T) {
	root := repoRootFromCommandsTest(t)

	readme := mustReadText(t, filepath.Join(root, "README.md"))
	indexDoc := mustReadText(t, filepath.Join(root, "docs", "00-index.md"))
	managedDoc := mustReadText(t, filepath.Join(root, "docs", "architecture", "09-standalone-and-managed-mode.md"))

	for _, token := range []string{
		"Default path",
		"single-binary",
		"SQLite-first",
		"managed",
		"api-only-same-origin-sveltekit-v1",
		"SvelteKit-first",
	} {
		if !strings.Contains(readme, token) {
			t.Fatalf("README should include landing narrative token %q", token)
		}
	}

	for _, token := range []string{
		"README",
		"repo landing page",
	} {
		if !strings.Contains(indexDoc, token) {
			t.Fatalf("docs index should describe README entrypoint token %q", token)
		}
	}

	for _, token := range []string{
		"api-only-same-origin-sveltekit-v1",
		"SvelteKit-first",
	} {
		if !strings.Contains(managedDoc, token) {
			t.Fatalf("managed-mode contract should include %q", token)
		}
	}

	for _, token := range []string{
		"Note that any JS framework could be used.",
		"I have come to regret this",
		"TODO: the test file can be found",
	} {
		if strings.Contains(readme, token) {
			t.Fatalf("README should not include stale landing copy token %q", token)
		}
	}
}
