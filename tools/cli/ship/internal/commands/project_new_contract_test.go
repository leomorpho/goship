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

func TestGettingStartedStarterDocsStayAligned(t *testing.T) {
	t.Parallel()

	readme := readRepoFile(t, "README.md")
	gettingStarted := readRepoFile(t, "docs/guides/01-getting-started.md")
	starterReadmeBytes, err := startertemplate.Files.ReadFile(filepath.ToSlash(filepath.Join(starterTemplateRoot, "README.md")))
	if err != nil {
		t.Fatalf("ReadFile(starter README) error = %v", err)
	}
	starterReadme := string(starterReadmeBytes)

	for name, content := range map[string]string{
		"README.md":                           readme,
		"docs/guides/01-getting-started.md":   gettingStarted,
		"starter/testdata/scaffold/README.md": starterReadme,
	} {
		assertContains(t, name, content, "starter")
		assertContains(t, name, content, "ship db:migrate")
		assertContains(t, name, content, "ship dev")
		assertContains(t, name, content, "ship module:add")
	}

	for name, content := range map[string]string{
		"README.md":                           readme,
		"docs/guides/01-getting-started.md":   gettingStarted,
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

func TestNewOutputPrintsCanonicalStarterNextStep(t *testing.T) {
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

func TestGettingStartedGuideStaysOnStarterHappyPath(t *testing.T) {
	t.Parallel()

	gettingStarted := readRepoFile(t, "docs/guides/01-getting-started.md")
	assertContains(t, "docs/guides/01-getting-started.md", gettingStarted, "ship db:migrate")
	assertContains(t, "docs/guides/01-getting-started.md", gettingStarted, "ship dev")
	assertContains(t, "docs/guides/01-getting-started.md", gettingStarted, "ship verify --profile fast")
	assertNotContains(t, "docs/guides/01-getting-started.md", gettingStarted, "ship dev --all")
	assertNotContains(t, "docs/guides/01-getting-started.md", gettingStarted, "ship verify --profile strict")
}

func TestAuthRouteContractSplitIsExplicit(t *testing.T) {
	t.Parallel()

	httpRoutes := readRepoFile(t, "docs/architecture/04-http-routes.md")
	apiGuide := readRepoFile(t, "docs/guides/08-building-an-api.md")
	goldenBrowser := readRepoFile(t, "tests/e2e/tests/auth_golden_flow.spec.ts")
	starterRouterBytes, err := startertemplate.Files.ReadFile(filepath.ToSlash(filepath.Join(starterTemplateRoot, "app", "router.go")))
	if err != nil {
		t.Fatalf("ReadFile(starter router) error = %v", err)
	}
	starterRouter := string(starterRouterBytes)

	assertContains(t, "docs/architecture/04-http-routes.md", httpRoutes, "framework repo app surface")
	assertContains(t, "docs/architecture/04-http-routes.md", httpRoutes, "/user/login")
	assertContains(t, "docs/architecture/04-http-routes.md", httpRoutes, "/user/register")
	assertContains(t, "docs/architecture/04-http-routes.md", httpRoutes, "/auth/logout")

	assertContains(t, "docs/guides/08-building-an-api.md", apiGuide, "/auth/login")
	assertContains(t, "docs/guides/08-building-an-api.md", apiGuide, "/auth/register")
	assertContains(t, "docs/guides/08-building-an-api.md", apiGuide, "/auth/logout")

	assertContains(t, "starter/testdata/scaffold/app/router.go", starterRouter, "/auth/login")
	assertContains(t, "starter/testdata/scaffold/app/router.go", starterRouter, "/auth/register")
	assertContains(t, "starter/testdata/scaffold/app/router.go", starterRouter, "/auth/session")
	assertContains(t, "starter/testdata/scaffold/app/router.go", starterRouter, "/auth/settings")
	assertContains(t, "starter/testdata/scaffold/app/router.go", starterRouter, "/auth/password/reset")
	assertContains(t, "starter/testdata/scaffold/app/router.go", starterRouter, "/auth/delete-account")
	assertNotContains(t, "starter/testdata/scaffold/app/router.go", starterRouter, "/user/login")
	assertNotContains(t, "starter/testdata/scaffold/app/router.go", starterRouter, "/user/register")

	assertContains(t, "tests/e2e/tests/auth_golden_flow.spec.ts", goldenBrowser, "/user/register")
	assertContains(t, "tests/e2e/tests/auth_golden_flow.spec.ts", goldenBrowser, "/auth/logout")
}

func TestStarterAuthDocsMatchExpandedContract(t *testing.T) {
	t.Parallel()

	gettingStarted := readRepoFile(t, "docs/guides/01-getting-started.md")
	starterReadmeBytes, err := startertemplate.Files.ReadFile(filepath.ToSlash(filepath.Join(starterTemplateRoot, "README.md")))
	if err != nil {
		t.Fatalf("ReadFile(starter README) error = %v", err)
	}
	starterReadme := string(starterReadmeBytes)

	assertContains(t, "docs/guides/01-getting-started.md", gettingStarted, "landing/auth/account/admin/home/profile")
	assertContains(t, "starter/testdata/scaffold/README.md", starterReadme, "account routes")
	assertContains(t, "docs/guides/01-getting-started.md", gettingStarted, "admin")
	assertContains(t, "starter/testdata/scaffold/README.md", starterReadme, "admin dashboard route")
	assertContains(t, "starter/testdata/scaffold/README.md", starterReadme, "app/policies/admin_dashboard.go")
}

func TestFrameworkBrowserSuitesDeclareTheirAuthSurface(t *testing.T) {
	t.Parallel()

	for _, rel := range []string{
		"tests/e2e/tests/auth_golden_flow.spec.ts",
		"tests/e2e/tests/cherie_compatibility.spec.ts",
		"tests/e2e/tests/goship.spec.ts",
		"tests/e2e/tests/smoke.spec.ts",
		"tests/e2e/tests/admin_scaffold.spec.ts",
	} {
		content := readRepoFile(t, rel)
		assertContains(t, rel, content, "framework repo")
	}

	workflowsGuide := readRepoFile(t, "docs/guides/02-development-workflows.md")
	assertContains(t, "docs/guides/02-development-workflows.md", workflowsGuide, "framework repo app surface")

	scopeAnalysis := readRepoFile(t, "docs/architecture/03-project-scope-analysis.md")
	assertContains(t, "docs/architecture/03-project-scope-analysis.md", scopeAnalysis, "framework repo app surface")

	knownGaps := readRepoFile(t, "docs/architecture/06-known-gaps-and-risks.md")
	assertContains(t, "docs/architecture/06-known-gaps-and-risks.md", knownGaps, "framework repo")
}

func TestPolicyGeneratorDocsMatchStarterSupport(t *testing.T) {
	t.Parallel()

	cliRef := readRepoFile(t, "docs/reference/01-cli.md")
	assertContains(t, "docs/reference/01-cli.md", cliRef, "ship make:policy <Name>")
	assertContains(t, "docs/reference/01-cli.md", cliRef, "starter-safe")
}

func TestControllerGeneratorDocsMatchStarterSupport(t *testing.T) {
	t.Parallel()

	cliRef := readRepoFile(t, "docs/reference/01-cli.md")
	assertContains(t, "docs/reference/01-cli.md", cliRef, "starter-safe now")
	assertContains(t, "docs/reference/01-cli.md", cliRef, "starter CRUD/runtime route backend")
}

func TestGeneratorSurfaceBoundaryIsExplicit(t *testing.T) {
	t.Parallel()

	cliRef := readRepoFile(t, "docs/reference/01-cli.md")
	assertContains(t, "docs/reference/01-cli.md", cliRef, "starter-safe today")
	assertContains(t, "docs/reference/01-cli.md", cliRef, "`make:resource`, `make:model`, `make:controller`, `make:policy`, `make:mailer`, `make:island`")
	assertContains(t, "docs/reference/01-cli.md", cliRef, "starter-safe when a locale baseline already exists")
	assertContains(t, "docs/reference/01-cli.md", cliRef, "framework-workspace-only for now")
	assertContains(t, "docs/reference/01-cli.md", cliRef, "`make:factory`, `make:job`, `make:schedule`, `make:command`, `make:scaffold`")
	assertContains(t, "docs/reference/01-cli.md", cliRef, "framework authoring only")
	assertContains(t, "docs/reference/01-cli.md", cliRef, "`make:module`")
}

func TestSupportedBatterySetIsExplicit(t *testing.T) {
	t.Parallel()

	readme := readRepoFile(t, "README.md")
	cliRef := readRepoFile(t, "docs/reference/01-cli.md")
	moduleWorkflow := readRepoFile(t, "docs/guides/12-add-module-workflow.md")
	moduleSurface := readRepoFile(t, "docs/architecture/11-module-surface-reset.md")

	for _, content := range []string{cliRef, moduleWorkflow} {
		for _, battery := range []string{"jobs", "storage", "emailsubscriptions"} {
			if !strings.Contains(content, battery) {
				t.Fatalf("supported battery %q missing from contract surface", battery)
			}
		}
	}
	for _, unsupported := range []string{"notifications", "paidsubscriptions"} {
		assertContains(t, "docs/architecture/11-module-surface-reset.md", moduleSurface, unsupported)
	}

	assertContains(t, "README.md", readme, "supported first-party batteries")
	assertContains(t, "README.md", readme, "jobs")
	assertContains(t, "README.md", readme, "storage")
	assertContains(t, "README.md", readme, "emailsubscriptions")
	assertNotContains(t, "README.md", readme, "installable modules for auth, profile, notifications, jobs, storage, billing, i18n, and more")
}

func TestFirstPartyRuntimeUsesSharedCompositionHelper(t *testing.T) {
	t.Parallel()

	helper := readRepoFile(t, "framework/bootstrap/first_party_runtime.go")
	assertContains(t, "framework/bootstrap/first_party_runtime.go", helper, "BuildFirstPartyServices")

	for _, rel := range []string{
		"cmd/web/main.go",
		"cmd/worker/main.go",
		"framework/testutil/http.go",
	} {
		body := readRepoFile(t, rel)
		assertContains(t, rel, body, "frameworkbootstrap.BuildFirstPartyServices")
		assertNotContains(t, rel, body, "paidsubscriptions.BuildDefaultCatalog")
		assertNotContains(t, rel, body, "notifications.New(")
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
