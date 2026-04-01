package policies

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestDocFrameworkBoundaryFilesStayAligned(t *testing.T) {
	t.Parallel()

	extensionZones := readPolicyRepoFile(t, "docs/architecture/10-extension-zones.md")
	roadmap := readPolicyRepoFile(t, "docs/roadmap/01-framework-plan.md")
	cognitiveModel := readPolicyRepoFile(t, "docs/architecture/08-cognitive-model.md")
	cliReference := readPolicyRepoFile(t, "docs/reference/01-cli.md")

	for _, token := range []string{
		"`app/container.go`",
		"`app/router.go`",
		"`app/schedules.go`",
	} {
		if !strings.Contains(extensionZones, token) {
			t.Fatalf("docs/architecture/10-extension-zones.md missing %q", token)
		}
		if !strings.Contains(roadmap, token) {
			t.Fatalf("docs/roadmap/01-framework-plan.md missing %q", token)
		}
		if !strings.Contains(cliReference, token) {
			t.Fatalf("docs/reference/01-cli.md missing %q", token)
		}
	}

	if !strings.Contains(extensionZones, "`app/foundation/container.go`") {
		t.Fatal("docs/architecture/10-extension-zones.md missing generated-app seam `app/foundation/container.go`")
	}
	if !strings.Contains(roadmap, "`app/foundation/container.go`") {
		t.Fatal("docs/roadmap/01-framework-plan.md missing generated-app seam `app/foundation/container.go`")
	}
	if !strings.Contains(cognitiveModel, "`app/container.go` -> `app/router.go` -> `app/schedules.go`") {
		t.Fatal("docs/architecture/08-cognitive-model.md missing canonical framework seam rule with concrete paths")
	}
}

func TestExtensionZoneManifestAcceptsFrameworkRepoAndGeneratedAppBoundaries(t *testing.T) {
	t.Parallel()

	frameworkRoot := t.TempDir()
	createFrameworkRepoWorkspace(t, frameworkRoot)
	writeTestFile(t, frameworkRoot, filepath.Join("app", "container.go"), "package app\n")
	writeTestFile(t, frameworkRoot, filepath.Join("app", "router.go"), "package app\n")
	writeTestFile(t, frameworkRoot, filepath.Join("app", "schedules.go"), "package app\n")
	writeTestFile(t, frameworkRoot, filepath.Join("docs", "architecture", "10-extension-zones.md"), strings.Join([]string{
		"# Extension Zones",
		"",
		"## Extension Zones",
		"",
		"- `framework/`",
		"- `app/`",
		"",
		"## Protected Contract Zones",
		"",
		"- `app/container.go`",
		"- `app/router.go`",
		"- `app/schedules.go`",
		"- `app/foundation/container.go`",
		"- `config/modules.yaml`",
		"- `tools/agent-policy/allowed-commands.yaml`",
		"",
	}, "\n"))

	if issues := checkExtensionZoneManifest(frameworkRoot); len(issues) != 0 {
		t.Fatalf("checkExtensionZoneManifest(framework repo) issues = %#v, want none", issues)
	}

	generatedRoot := t.TempDir()
	createGeneratedAppWorkspace(t, generatedRoot)
	writeTestFile(t, generatedRoot, filepath.Join("docs", "architecture", "10-extension-zones.md"), strings.Join([]string{
		"# Extension Zones",
		"",
		"## Extension Zones",
		"",
		"- `app/`",
		"- `framework/`",
		"",
		"## Protected Contract Zones",
		"",
		"- `app/router.go`",
		"- `app/foundation/container.go`",
		"- `config/modules.yaml`",
		"- `tools/agent-policy/allowed-commands.yaml`",
		"",
	}, "\n"))

	if issues := checkExtensionZoneManifest(generatedRoot); len(issues) != 0 {
		t.Fatalf("checkExtensionZoneManifest(generated app) issues = %#v, want none", issues)
	}
}

func TestCLIDocsCoverageRejectsStaleFrameworkRootSeamTokens(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	createFrameworkRepoWorkspace(t, root)
	writeTestFile(t, root, filepath.Join("app", "container.go"), "package app\n")
	writeTestFile(t, root, filepath.Join("app", "router.go"), "package app\n")
	writeTestFile(t, root, filepath.Join("app", "schedules.go"), "package app\n")
	writeTestFile(t, root, filepath.Join("docs", "reference", "01-cli.md"), strings.Join([]string{
		"## Minimal V1 Command Set",
		"## Implementation Mapping (Current Repo)",
		"## Generator test strategy",
		"ship doctor",
		"ship verify",
		"ship agent:setup",
		"ship agent:check",
		"ship agent:status",
		"ship new <app>",
		"ship upgrade",
		"ship make:resource",
		"ship make:model",
		"ship make:controller",
		"ship make:scaffold",
		"ship make:module",
		"ship db:migrate",
		"ship test --integration",
		"extension-zone manifest",
		"`container.go`",
		"`router.go`",
		"`schedules.go`",
		"",
	}, "\n"))

	issues := checkCLIDocsCoverage(root)
	if !hasDoctorIssueContaining(issues, "missing framework repo command token: \"`app/container.go`\"") {
		t.Fatalf("checkCLIDocsCoverage() missing app/container.go issue\nissues = %#v", issues)
	}
	if !hasDoctorIssueContaining(issues, "stale framework-shell link token: \"`container.go`\"") {
		t.Fatalf("checkCLIDocsCoverage() missing stale bare container.go issue\nissues = %#v", issues)
	}
}

func readPolicyRepoFile(t *testing.T, rel string) string {
	t.Helper()

	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller(0) failed")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", "..", "..", "..", ".."))
	content, err := os.ReadFile(filepath.Join(repoRoot, rel))
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v", rel, err)
	}
	return string(content)
}
