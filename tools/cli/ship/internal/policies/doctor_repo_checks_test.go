package policies

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLooksLikeCanonicalFrameworkRepoIgnoresGeneratedAppScaffold(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	createGeneratedAppWorkspace(t, root)

	if looksLikeCanonicalFrameworkRepo(root) {
		t.Fatalf("looksLikeCanonicalFrameworkRepo(%q) = true, want false for generated app workspace", root)
	}
}

func TestFrameworkRepoChecksOnlyApplyToFrameworkRepo(t *testing.T) {
	t.Parallel()

	generatedRoot := t.TempDir()
	createGeneratedAppWorkspace(t, generatedRoot)

	if issues := CheckCanonicalRepoTopLevelPaths(generatedRoot); len(issues) != 0 {
		t.Fatalf("CheckCanonicalRepoTopLevelPaths(generated app) issues = %#v, want none", issues)
	}
	if issues := checkFrameworkCIVerifyGate(generatedRoot); len(issues) != 0 {
		t.Fatalf("checkFrameworkCIVerifyGate(generated app) issues = %#v, want none", issues)
	}

	frameworkRoot := t.TempDir()
	createFrameworkRepoWorkspace(t, frameworkRoot)

	if !looksLikeCanonicalFrameworkRepo(frameworkRoot) {
		t.Fatalf("looksLikeCanonicalFrameworkRepo(%q) = false, want true for framework repo workspace", frameworkRoot)
	}

	topLevelIssues := CheckCanonicalRepoTopLevelPaths(frameworkRoot)
	if !hasDoctorIssueContaining(topLevelIssues, "missing canonical top-level path: modules") {
		t.Fatalf("CheckCanonicalRepoTopLevelPaths(framework repo) missing modules issue\nissues = %#v", topLevelIssues)
	}
	if !hasDoctorIssueContaining(topLevelIssues, "missing canonical top-level path: frontend") {
		t.Fatalf("CheckCanonicalRepoTopLevelPaths(framework repo) missing frontend issue\nissues = %#v", topLevelIssues)
	}

	workflowIssues := checkFrameworkCIVerifyGate(frameworkRoot)
	if !hasDoctorIssueContaining(workflowIssues, "missing CI workflow gate for strict framework verify profile") {
		t.Fatalf("checkFrameworkCIVerifyGate(framework repo) missing workflow issue\nissues = %#v", workflowIssues)
	}
}

func createGeneratedAppWorkspace(t *testing.T, root string) {
	t.Helper()

	dirs := []string{
		"app/foundation",
		"app/jobs",
		"app/views",
		"app/web/controllers",
		"app/web/middleware",
		"app/web/routenames",
		"app/web/ui",
		"app/web/viewmodels",
		"config",
		"db/migrate/migrations",
		"db/queries",
		"docs/architecture",
	}
	for _, rel := range dirs {
		if err := os.MkdirAll(filepath.Join(root, rel), 0o755); err != nil {
			t.Fatalf("os.MkdirAll(%q) error = %v", rel, err)
		}
	}

	files := map[string]string{
		".gitignore": "/web\n/worker\n/seed\n/ship\n/ship-mcp\n",
		"app/router.go": strings.Join([]string{
			"package app",
			"",
			"func Router() {",
			"\t// ship:routes:public:start",
			"\t// ship:routes:public:end",
			"\t// ship:routes:auth:start",
			"\t// ship:routes:auth:end",
			"\t// ship:routes:external:start",
			"\t// ship:routes:external:end",
			"}",
			"",
		}, "\n"),
		"app/foundation/container.go": strings.Join([]string{
			"package foundation",
			"",
			"func Build() {",
			"\t// ship:container:start",
			"\t// ship:container:end",
			"}",
			"",
		}, "\n"),
		"app/web/routenames/routenames.go":        "package routenames\n",
		"config/modules.yaml":                     "modules: []\n",
		"db/bobgen.yaml":                          "queries: db/queries\n",
		"docs/00-index.md":                        "# Docs\n",
		"docs/architecture/01-architecture.md":    "# Architecture\n",
		"docs/architecture/08-cognitive-model.md": "# Cognitive Model\n",
		"go.mod": "module example.com/demo\n\ngo 1.24.0\n",
	}
	for rel, content := range files {
		writeTestFile(t, root, rel, content)
	}
}

func createFrameworkRepoWorkspace(t *testing.T, root string) {
	t.Helper()

	dirs := []string{
		"framework/core",
		"tools/cli/ship",
	}
	for _, rel := range dirs {
		if err := os.MkdirAll(filepath.Join(root, rel), 0o755); err != nil {
			t.Fatalf("os.MkdirAll(%q) error = %v", rel, err)
		}
	}

	writeTestFile(t, root, "go.work", "go 1.25.6\n")
}

func writeTestFile(t *testing.T, root, rel, content string) {
	t.Helper()

	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("os.MkdirAll(%q) error = %v", filepath.Dir(rel), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("os.WriteFile(%q) error = %v", rel, err)
	}
}

func hasDoctorIssueContaining(issues []DoctorIssue, needle string) bool {
	for _, issue := range issues {
		if strings.Contains(issue.Message, needle) || strings.Contains(issue.File, needle) {
			return true
		}
	}
	return false
}
