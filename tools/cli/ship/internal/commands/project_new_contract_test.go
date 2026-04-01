package commands

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	policies "github.com/leomorpho/goship/tools/cli/ship/internal/policies"
	startertemplate "github.com/leomorpho/goship/tools/cli/ship/internal/templates/starter"
)

func TestNewStarterDocsStayAligned(t *testing.T) {
	t.Parallel()

	readme := readRepoFile(t, "README.md")
	gettingStarted := readRepoFile(t, "docs/guides/01-getting-started.md")
	starterReadmeBytes, err := startertemplate.Files.ReadFile(filepath.ToSlash(filepath.Join(starterTemplateRoot, "README.md")))
	if err != nil {
		t.Fatalf("ReadFile(starter README) error = %v", err)
	}
	starterReadme := string(starterReadmeBytes)

	for name, content := range map[string]string{
		"README.md":                                  readme,
		"docs/guides/01-getting-started.md":          gettingStarted,
		"starter/testdata/scaffold/README.md": starterReadme,
	} {
		assertContains(t, name, content, "starter")
		assertContains(t, name, content, "ship db:migrate")
		assertContains(t, name, content, "ship dev")
		assertContains(t, name, content, "ship module:add")
	}

	for name, content := range map[string]string{
		"README.md":                                  readme,
		"docs/guides/01-getting-started.md":          gettingStarted,
		"starter/testdata/scaffold/README.md": starterReadme,
	} {
		assertContainsOneOf(t, name, content,
			"not supported",
			"do not rely on `ship module:add`",
		)
	}

	assertNotContains(t, "starter/testdata/scaffold/README.md", starterReadme, "Add modules with `ship module:add`")
	assertNotContains(t, "starter/testdata/scaffold/README.md", starterReadme, "go run ./cmd/web")
}

func TestRunNewPrintsCanonicalStarterNextStep(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd() error = %v", err)
	}
	root := t.TempDir()
	if err := os.Chdir(root); err != nil {
		t.Fatalf("os.Chdir(%q) error = %v", root, err)
	}
	defer func() { _ = os.Chdir(wd) }()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := RunNew([]string{"demo", "--module", "example.com/demo", "--no-i18n"}, NewDeps{
		Out: &stdout,
		Err: &stderr,
		ParseAgentPolicyBytes: func(b []byte) (policies.AgentPolicy, error) {
			return policies.AgentPolicy{}, nil
		},
		RenderAgentPolicyArtifacts: func(policy policies.AgentPolicy) (map[string][]byte, error) {
			return map[string][]byte{}, nil
		},
		AgentPolicyFilePath: policies.AgentPolicyFilePath,
	})
	if exitCode != 0 {
		t.Fatalf("RunNew() exit code = %d\nstdout:\n%s\nstderr:\n%s", exitCode, stdout.String(), stderr.String())
	}

	want := "Next: cd demo && ship db:migrate && ship dev"
	if !strings.Contains(stdout.String(), want) {
		t.Fatalf("RunNew() output missing canonical next step %q\nstdout:\n%s", want, stdout.String())
	}
}

func readRepoFile(t *testing.T, rel string) string {
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

func assertContains(t *testing.T, name, content, needle string) {
	t.Helper()
	if !strings.Contains(content, needle) {
		t.Fatalf("%s missing %q", name, needle)
	}
}

func assertContainsOneOf(t *testing.T, name, content string, needles ...string) {
	t.Helper()
	for _, needle := range needles {
		if strings.Contains(content, needle) {
			return
		}
	}
	t.Fatalf("%s missing one of %q", name, strings.Join(needles, ", "))
}

func assertNotContains(t *testing.T, name, content, needle string) {
	t.Helper()
	if strings.Contains(content, needle) {
		t.Fatalf("%s unexpectedly contains %q", name, needle)
	}
}
