package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLocalRuntimeContract_DocsAndMakefileStayAligned_RedSpec(t *testing.T) {
	root := repoRootFromCommandsTest(t)

	makefile := mustReadText(t, filepath.Join(root, "Makefile"))
	aiGuide := mustReadText(t, filepath.Join(root, "docs", "guides", "01-ai-agent-guide.md"))
	devGuide := mustReadText(t, filepath.Join(root, "docs", "guides", "02-development-workflows.md"))
	compose := mustReadText(t, filepath.Join(root, "infra", "docker", "docker-compose.yml"))
	envExample := mustReadText(t, filepath.Join(root, ".env.example"))

	if strings.Contains(aiGuide, "make dev (default local dev: infra + web)") {
		t.Fatal("AI agent guide still describes make dev as infra + web instead of the canonical app-on loop")
	}
	if !strings.Contains(devGuide, "canonical app-on loop") {
		t.Fatal("development workflows guide should describe make dev as the canonical app-on loop")
	}
	if !strings.Contains(devGuide, "Docker Compose currently provisions:\n\n- Redis (`goship_cache`)\n- Mailpit (`goship_mailpit`)") {
		t.Fatal("development workflows guide should pin the compose-backed local accessories explicitly")
	}
	if strings.Contains(devGuide, "use `make dev` if you need the full infrastructure stack") {
		t.Fatal("development workflows guide should not describe make dev as the full infrastructure stack path")
	}
	if !strings.Contains(devGuide, "`make dev-full` if you need the full multiprocess infrastructure stack") {
		t.Fatal("development workflows guide should send full infrastructure users to make dev-full")
	}
	if !strings.Contains(makefile, "dev: ## Start the canonical app-on dev loop (single-node web loop; distributed uses full mode)") {
		t.Fatal("Makefile should advertise make dev as the canonical app-on dev loop")
	}
	if !strings.Contains(compose, "cache:") || !strings.Contains(compose, "mailpit:") {
		t.Fatal("compose file should still define cache and mailpit services for the local accessory workflow")
	}
	if strings.Contains(makefile, "db: ## Connect to the primary database") {
		t.Fatal("Makefile still advertises a compose-backed primary database shell in the default local contract")
	}
	if strings.Contains(envExample, "# Database driver: sqlite or postgres.\nPAGODA_DB_DRIVER=\n# SQLite database path when using PAGODA_DB_DRIVER=sqlite.\nPAGODA_DB_PATH=") {
		t.Fatal(".env.example should not blank out the canonical single-node SQLite defaults later in the file")
	}
}

func repoRootFromCommandsTest(t *testing.T) string {
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

func mustReadText(t *testing.T, path string) string {
	t.Helper()

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(content)
}
