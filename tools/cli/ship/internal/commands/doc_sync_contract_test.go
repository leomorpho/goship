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
