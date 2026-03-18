package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLocalRuntimeContract_DocsAndMakefileStayAligned_RedSpec(t *testing.T) {
	t.Skip("red spec: enable once local-runtime guidance is aligned across Makefile help and contributor docs")

	root := repoRootFromCommandsTest(t)

	makefile := mustReadText(t, filepath.Join(root, "Makefile"))
	aiGuide := mustReadText(t, filepath.Join(root, "docs", "guides", "01-ai-agent-guide.md"))
	devGuide := mustReadText(t, filepath.Join(root, "docs", "guides", "02-development-workflows.md"))
	compose := mustReadText(t, filepath.Join(root, "infra", "docker", "docker-compose.yml"))

	if strings.Contains(aiGuide, "make dev (default local dev: infra + web)") {
		t.Fatal("AI agent guide still describes make dev as infra + web instead of the canonical app-on loop")
	}
	if !strings.Contains(devGuide, "canonical app-on loop") {
		t.Fatal("development workflows guide should describe make dev as the canonical app-on loop")
	}
	if !strings.Contains(devGuide, "Docker Compose currently provisions:\n\n- Redis (`goship_cache`)\n- Mailpit (`goship_mailpit`)") {
		t.Fatal("development workflows guide should pin the compose-backed local accessories explicitly")
	}
	if !strings.Contains(compose, "cache:") || !strings.Contains(compose, "mailpit:") {
		t.Fatal("compose file should still define cache and mailpit services for the local accessory workflow")
	}
	if strings.Contains(makefile, "db: ## Connect to the primary database") {
		t.Fatal("Makefile still advertises a compose-backed primary database shell in the default local contract")
	}
}

func repoRootFromCommandsTest(t *testing.T) string {
	t.Helper()

	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find repo root")
		}
		dir = parent
	}
}

func mustReadText(t *testing.T, path string) string {
	t.Helper()

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(content)
}
