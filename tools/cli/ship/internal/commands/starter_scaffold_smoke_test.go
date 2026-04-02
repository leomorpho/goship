package commands

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/leomorpho/goship/tools/cli/ship/internal/generators"
	policies "github.com/leomorpho/goship/tools/cli/ship/internal/policies"
)

func TestStarterMakeResourceAndDestroyStayBuildable(t *testing.T) {
	appPath := scaffoldStarterApp(t)

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd() error = %v", err)
	}
	defer func() { _ = os.Chdir(wd) }()
	if err := os.Chdir(appPath); err != nil {
		t.Fatalf("os.Chdir(%q) error = %v", appPath, err)
	}

	var out bytes.Buffer
	if code := generators.RunGenerateResource([]string{"contact", "--wire"}, &out, &out); code != 0 {
		t.Fatalf("RunGenerateResource() exit code = %d\n%s", code, out.String())
	}

	buildStarterApp(t, appPath, "go build after make:resource")

	out.Reset()
	if code := RunDestroy([]string{"resource:contact"}, DestroyDeps{Out: &out, Err: &out, Cwd: appPath}); code != 0 {
		t.Fatalf("RunDestroy() exit code = %d\n%s", code, out.String())
	}

	buildStarterApp(t, appPath, "go build after destroy")
}

func TestStarterCRUDScaffoldIsUseful(t *testing.T) {
	appPath := scaffoldStarterApp(t)

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd() error = %v", err)
	}
	defer func() { _ = os.Chdir(wd) }()
	if err := os.Chdir(appPath); err != nil {
		t.Fatalf("os.Chdir(%q) error = %v", appPath, err)
	}

	var out bytes.Buffer
	if code := generators.RunGenerateResource([]string{"contact", "--wire"}, &out, &out); code != 0 {
		t.Fatalf("RunGenerateResource() exit code = %d\n%s", code, out.String())
	}

	pageBody, err := os.ReadFile(filepath.Join(appPath, "app", "views", "web", "pages", "gen", "contact.go"))
	if err != nil {
		t.Fatalf("os.ReadFile(generated page) error = %v", err)
	}
	if !strings.Contains(string(pageBody), "CRUD scaffold") {
		t.Fatalf("generated starter page should describe CRUD scaffold\n%s", pageBody)
	}
	if !strings.Contains(string(pageBody), "ship:generated:resource:contact") {
		t.Fatalf("generated starter page should carry resource ownership header\n%s", pageBody)
	}
	routerBody, err := os.ReadFile(filepath.Join(appPath, "app", "router.go"))
	if err != nil {
		t.Fatalf("os.ReadFile(router.go) error = %v", err)
	}
	if !strings.Contains(string(routerBody), `Kind: RouteKindResource`) {
		t.Fatalf("generated router should use explicit resource route kind\n%s", routerBody)
	}
	if !strings.Contains(string(routerBody), `Actions: []string{"index", "show", "create", "update", "destroy"}`) {
		t.Fatalf("generated router should include explicit CRUD actions\n%s", routerBody)
	}

	buildStarterApp(t, appPath, "go build after CRUD make:resource")
}

func TestStarterCRUDResourceGeneratorIsIdempotentByRefusal(t *testing.T) {
	appPath := scaffoldStarterApp(t)

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd() error = %v", err)
	}
	defer func() { _ = os.Chdir(wd) }()
	if err := os.Chdir(appPath); err != nil {
		t.Fatalf("os.Chdir(%q) error = %v", appPath, err)
	}

	var out bytes.Buffer
	if code := generators.RunGenerateResource([]string{"contact", "--wire"}, &out, &out); code != 0 {
		t.Fatalf("RunGenerateResource() first exit code = %d\n%s", code, out.String())
	}

	pagePath := filepath.Join(appPath, "app", "views", "web", "pages", "gen", "contact.go")
	before, err := os.ReadFile(pagePath)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v", pagePath, err)
	}

	out.Reset()
	if code := generators.RunGenerateResource([]string{"contact", "--wire"}, &out, &out); code == 0 {
		t.Fatalf("RunGenerateResource() second exit code = 0, want refusal\n%s", out.String())
	}
	if !strings.Contains(out.String(), "refusing to overwrite existing file") {
		t.Fatalf("RunGenerateResource() second output = %q, want overwrite refusal", out.String())
	}

	after, err := os.ReadFile(pagePath)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) after second run error = %v", pagePath, err)
	}
	if string(before) != string(after) {
		t.Fatalf("generated page mutated on refused second make:resource\nbefore:\n%s\nafter:\n%s", before, after)
	}
}

func TestStarterCRUDDestroyFailsCleanlyWhenRepeated(t *testing.T) {
	appPath := scaffoldStarterApp(t)

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd() error = %v", err)
	}
	defer func() { _ = os.Chdir(wd) }()
	if err := os.Chdir(appPath); err != nil {
		t.Fatalf("os.Chdir(%q) error = %v", appPath, err)
	}

	var out bytes.Buffer
	if code := generators.RunGenerateResource([]string{"contact", "--wire"}, &out, &out); code != 0 {
		t.Fatalf("RunGenerateResource() exit code = %d\n%s", code, out.String())
	}

	out.Reset()
	if code := RunDestroy([]string{"resource:contact"}, DestroyDeps{Out: &out, Err: &out, Cwd: appPath}); code != 0 {
		t.Fatalf("RunDestroy() first exit code = %d\n%s", code, out.String())
	}

	out.Reset()
	if code := RunDestroy([]string{"resource:contact"}, DestroyDeps{Out: &out, Err: &out, Cwd: appPath}); code == 0 {
		t.Fatalf("RunDestroy() second exit code = 0, want clean refusal\n%s", out.String())
	}
	if !strings.Contains(out.String(), "no generator-managed targets matched") {
		t.Fatalf("RunDestroy() second output = %q, want no-match refusal", out.String())
	}

	buildStarterApp(t, appPath, "go build after repeated destroy refusal")
}

func TestStarterCRUDDestroyUsesOwnershipHeaderAfterUserEdits(t *testing.T) {
	appPath := scaffoldStarterApp(t)

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd() error = %v", err)
	}
	defer func() { _ = os.Chdir(wd) }()
	if err := os.Chdir(appPath); err != nil {
		t.Fatalf("os.Chdir(%q) error = %v", appPath, err)
	}

	var out bytes.Buffer
	if code := generators.RunGenerateResource([]string{"contact", "--wire"}, &out, &out); code != 0 {
		t.Fatalf("RunGenerateResource() exit code = %d\n%s", code, out.String())
	}

	pagePath := filepath.Join(appPath, "app", "views", "web", "pages", "gen", "contact.go")
	pageBody, err := os.ReadFile(pagePath)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v", pagePath, err)
	}
	customized := strings.Replace(string(pageBody), "Starter CRUD scaffold for contact with list/create/show/edit/delete runtime support.", "Customized starter CRUD copy.", 1)
	if err := os.WriteFile(pagePath, []byte(customized), 0o644); err != nil {
		t.Fatalf("os.WriteFile(%q) error = %v", pagePath, err)
	}

	out.Reset()
	if code := RunDestroy([]string{"resource:contact"}, DestroyDeps{Out: &out, Err: &out, Cwd: appPath}); code != 0 {
		t.Fatalf("RunDestroy() exit code = %d\n%s", code, out.String())
	}
	if _, err := os.Stat(pagePath); !os.IsNotExist(err) {
		t.Fatalf("expected generated page to be removed, stat err = %v", err)
	}

	buildStarterApp(t, appPath, "go build after ownership-header destroy")
}

func TestStarterMakeControllerUsesStarterCRUDContract(t *testing.T) {
	appPath := scaffoldStarterApp(t)

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd() error = %v", err)
	}
	defer func() { _ = os.Chdir(wd) }()
	if err := os.Chdir(appPath); err != nil {
		t.Fatalf("os.Chdir(%q) error = %v", appPath, err)
	}

	var out bytes.Buffer
	code := generators.RunMakeController([]string{"Contact", "--actions", "index,show", "--wire"}, generators.ControllerDeps{
		Out: &out,
		Err: &out,
		HasFile: func(path string) bool {
			_, err := os.Stat(path)
			return err == nil
		},
		EnsureRouteNamesImport: generators.EnsureRouteNamesImport,
		WireRouteSnippet:       generators.WireRouteSnippet,
	})
	if code != 0 {
		t.Fatalf("RunMakeController() exit code = %d, want success\n%s", code, out.String())
	}
	pagePath := filepath.Join(appPath, "app", "views", "web", "pages", "gen", "contact.go")
	if _, err := os.Stat(pagePath); err != nil {
		t.Fatalf("starter controller page missing: %v", err)
	}
	pageBody, err := os.ReadFile(pagePath)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v", pagePath, err)
	}
	if !strings.Contains(string(pageBody), "ship:generated:controller:contact") {
		t.Fatalf("starter controller page should carry controller ownership header\n%s", pageBody)
	}
	if _, err := os.Stat(filepath.Join(appPath, "app", "web", "controllers", "contact.go")); err == nil {
		t.Fatal("framework controller file should not be created for starter controller generation")
	}
	routerAfter, err := os.ReadFile(filepath.Join(appPath, "app", "router.go"))
	if err != nil {
		t.Fatalf("os.ReadFile(router.go) after error = %v", err)
	}
	if !strings.Contains(string(routerAfter), `Kind: RouteKindResource`) {
		t.Fatalf("starter controller route should use explicit resource kind\n%s", routerAfter)
	}
	if !strings.Contains(string(routerAfter), `Actions: []string{"index", "show"}`) {
		t.Fatalf("starter controller route should preserve requested actions\n%s", routerAfter)
	}
	buildStarterApp(t, appPath, "go build after starter make:controller")
}

func TestStarterMakeIslandCreatesCanonicalArtifacts(t *testing.T) {
	appPath := scaffoldStarterApp(t)

	var out bytes.Buffer
	code := generators.RunMakeIsland([]string{"ContactCard"}, generators.MakeIslandDeps{
		Out: &out,
		Err: &out,
		Cwd: appPath,
	})
	if code != 0 {
		t.Fatalf("RunMakeIsland() exit code = %d\n%s", code, out.String())
	}

	for _, rel := range []string{
		filepath.Join("frontend", "islands", "ContactCard.js"),
		filepath.Join("app", "views", "web", "components", "contact_card_island.templ"),
	} {
		if _, err := os.Stat(filepath.Join(appPath, rel)); err != nil {
			t.Fatalf("generated island artifact %q missing: %v", rel, err)
		}
	}
}

func TestStarterMakeModelStaysBuildable(t *testing.T) {
	appPath := scaffoldStarterApp(t)

	var out bytes.Buffer
	if code := generators.RunGenerateModel([]string{"Post", "title:string", "published:bool"}, generators.GenerateModelDeps{
		Out: &out,
		Err: &out,
		HasFile: func(path string) bool {
			_, err := os.Stat(path)
			return err == nil
		},
		QueryDir: filepath.Join(appPath, "db", "queries"),
	}); code != 0 {
		t.Fatalf("RunGenerateModel() exit code = %d\n%s", code, out.String())
	}
	queryBody, err := os.ReadFile(filepath.Join(appPath, "db", "queries", "post.sql"))
	if err != nil {
		t.Fatalf("os.ReadFile(generated model query) error = %v", err)
	}
	for _, want := range []string{
		"-- ship:generated:model:post",
		"-- Suggested migration columns:",
		"-- - title TEXT",
		"-- - published BOOLEAN",
		"-- name: CreatePost :one",
		"-- name: ListPosts :many",
		"-- name: UpdatePost :one",
		"-- name: DeletePost :exec",
	} {
		if !strings.Contains(string(queryBody), want) {
			t.Fatalf("generated model query missing %q\n%s", want, queryBody)
		}
	}

	out.Reset()
	buildStarterApp(t, appPath, "go build after make:model")
}

func TestStarterMakeLocaleWorksWhenI18nBaselineExists(t *testing.T) {
	appPath := scaffoldStarterAppWithI18n(t)

	var out bytes.Buffer
	if code := generators.RunMakeLocale([]string{"es"}, generators.LocaleDeps{Out: &out, Err: &out, Cwd: appPath}); code != 0 {
		t.Fatalf("RunMakeLocale() exit code = %d\n%s", code, out.String())
	}

	if _, err := os.Stat(filepath.Join(appPath, "locales", "es.toml")); err != nil {
		t.Fatalf("generated locale missing: %v", err)
	}
}

func TestStarterFrameworkWorkspaceGeneratorsFailWithoutMutatingStarter(t *testing.T) {
	appPath := scaffoldStarterApp(t)

	cases := []struct {
		name      string
		run       func(*bytes.Buffer) int
		checkPath string
	}{
		{
			name: "make:factory",
			run: func(out *bytes.Buffer) int {
				return generators.RunMakeFactory([]string{"Post"}, generators.FactoryDeps{Out: out, Err: out, Cwd: appPath})
			},
			checkPath: filepath.Join(appPath, "tests", "factories", "post_factory.go"),
		},
		{
			name: "make:job",
			run: func(out *bytes.Buffer) int {
				return generators.RunMakeJob([]string{"BackfillUserStats"}, generators.MakeJobDeps{Out: out, Err: out, Cwd: appPath})
			},
			checkPath: filepath.Join(appPath, "app", "jobs", "backfill_user_stats.go"),
		},
		{
			name: "make:command",
			run: func(out *bytes.Buffer) int {
				return generators.RunMakeCommand([]string{"BackfillUserStats"}, generators.MakeCommandDeps{Out: out, Err: out, Cwd: appPath})
			},
			checkPath: filepath.Join(appPath, "app", "commands", "backfill_user_stats.go"),
		},
		{
			name: "make:mailer",
			run: func(out *bytes.Buffer) int {
				return generators.RunMakeMailer([]string{"Welcome"}, generators.MakeMailerDeps{Out: out, Err: out, Cwd: appPath})
			},
			checkPath: filepath.Join(appPath, "app", "views", "emails", "welcome.templ"),
		},
		{
			name: "make:schedule",
			run: func(out *bytes.Buffer) int {
				return generators.RunMakeSchedule([]string{"BackfillUserStats", "--cron", "0 0 * * *"}, generators.ScheduleDeps{Out: out, Err: out, Cwd: appPath})
			},
			checkPath: filepath.Join(appPath, "app", "schedules", "schedules.go"),
		},
		{
			name: "make:scaffold",
			run: func(out *bytes.Buffer) int {
				wd, err := os.Getwd()
				if err != nil {
					t.Fatalf("os.Getwd() error = %v", err)
				}
				if err := os.Chdir(appPath); err != nil {
					t.Fatalf("os.Chdir(%q) error = %v", appPath, err)
				}
				defer func() { _ = os.Chdir(wd) }()
				return generators.RunMakeScaffold([]string{"Post", "title:string"}, generators.ScaffoldDeps{
					Out:           out,
					Err:           out,
					RunModel:      func(args []string) int { return 0 },
					RunDBMake:     func(args []string) int { return 0 },
					RunDBMigrate:  func(args []string) int { return 0 },
					RunController: func(args []string) int { return 0 },
					RunResource:   func(args []string) int { return 0 },
				})
			},
			checkPath: filepath.Join(appPath, "app", "web", "controllers", "posts.go"),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var out bytes.Buffer
			code := tc.run(&out)
			if code == 0 {
				t.Fatalf("%s exit code = 0, want starter rejection\n%s", tc.name, out.String())
			}
			if !strings.Contains(out.String(), "starter scaffold") {
				t.Fatalf("%s output = %q, want starter scaffold rejection", tc.name, out.String())
			}
			if _, err := os.Stat(tc.checkPath); err == nil {
				t.Fatalf("%s created %q despite starter rejection", tc.name, tc.checkPath)
			}
		})
	}
}

func scaffoldStarterApp(t *testing.T) string {
	return scaffoldStarterAppWithI18nFlag(t, false)
}

func scaffoldStarterAppWithI18n(t *testing.T) string {
	return scaffoldStarterAppWithI18nFlag(t, true)
}

func scaffoldStarterAppWithI18nFlag(t *testing.T, i18nEnabled bool) string {
	root := t.TempDir()
	appPath := filepath.Join(root, "demo")

	opts := NewProjectOptions{
		Name:        "demo",
		Module:      "example.com/demo",
		AppPath:     appPath,
		UIProvider:  newUIProviderFranken,
		I18nEnabled: i18nEnabled,
	}
	deps := NewDeps{
		ParseAgentPolicyBytes: func(b []byte) (policies.AgentPolicy, error) {
			return policies.AgentPolicy{}, nil
		},
		RenderAgentPolicyArtifacts: func(policy policies.AgentPolicy) (map[string][]byte, error) {
			return map[string][]byte{}, nil
		},
		AgentPolicyFilePath: policies.AgentPolicyFilePath,
	}

	if err := ScaffoldNewProject(opts, deps); err != nil {
		t.Fatalf("ScaffoldNewProject() error = %v", err)
	}
	return appPath
}

func buildStarterApp(t *testing.T, appPath, context string) {
	t.Helper()

	cmd := exec.Command("go", "build", "./...")
	cmd.Dir = appPath
	buildOut, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s failed: %v\n%s", context, err, buildOut)
	}
}

func TestStarterJobsModuleRoundTripStaysBuildable(t *testing.T) {
	appPath := scaffoldStarterApp(t)

	containerPath := filepath.Join(appPath, "app", "foundation", "container.go")
	manifestPath := filepath.Join(appPath, "config", "modules.yaml")

	beforeContainer, err := os.ReadFile(containerPath)
	if err != nil {
		t.Fatalf("os.ReadFile(container.go) error = %v", err)
	}
	beforeManifest, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("os.ReadFile(modules.yaml) error = %v", err)
	}

	if err := applyModuleAdd(appPath, moduleCatalog["jobs"], false, &bytes.Buffer{}); err != nil {
		t.Fatalf("applyModuleAdd(jobs) error = %v", err)
	}

	afterAddContainer, err := os.ReadFile(containerPath)
	if err != nil {
		t.Fatalf("os.ReadFile(container.go) after add error = %v", err)
	}
	if !strings.Contains(string(afterAddContainer), `c.EnabledModules = append(c.EnabledModules, "jobs")`) {
		t.Fatalf("container.go missing jobs starter snippet\n%s", afterAddContainer)
	}
	afterAddManifest, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("os.ReadFile(modules.yaml) after add error = %v", err)
	}
	if !strings.Contains(string(afterAddManifest), "- jobs") {
		t.Fatalf("modules.yaml missing jobs entry\n%s", afterAddManifest)
	}
	buildStarterApp(t, appPath, "go build after module:add jobs")

	if err := applyModuleRemove(appPath, moduleCatalog["jobs"], false, &bytes.Buffer{}); err != nil {
		t.Fatalf("applyModuleRemove(jobs) error = %v", err)
	}
	afterRemoveContainer, err := os.ReadFile(containerPath)
	if err != nil {
		t.Fatalf("os.ReadFile(container.go) after remove error = %v", err)
	}
	if string(beforeContainer) != string(afterRemoveContainer) {
		t.Fatalf("container.go mismatch after remove\nbefore:\n%s\nafter:\n%s", beforeContainer, afterRemoveContainer)
	}
	afterRemoveManifest, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("os.ReadFile(modules.yaml) after remove error = %v", err)
	}
	if string(beforeManifest) != string(afterRemoveManifest) {
		t.Fatalf("modules.yaml mismatch after remove\nbefore:\n%s\nafter:\n%s", beforeManifest, afterRemoveManifest)
	}
	buildStarterApp(t, appPath, "go build after module:remove jobs")
}

func TestStarterUnsupportedModuleAddFailsWithoutMutatingGoMod(t *testing.T) {
	appPath := scaffoldStarterApp(t)

	before, err := os.ReadFile(filepath.Join(appPath, "go.mod"))
	if err != nil {
		t.Fatalf("os.ReadFile(go.mod) error = %v", err)
	}

	err = applyModuleAdd(appPath, moduleCatalog["notifications"], false, &bytes.Buffer{})
	if err == nil {
		t.Fatal("applyModuleAdd() error = nil, want starter scaffold rejection")
	}
	if !strings.Contains(err.Error(), "starter scaffold") {
		t.Fatalf("applyModuleAdd() error = %v, want starter scaffold rejection", err)
	}

	after, err := os.ReadFile(filepath.Join(appPath, "go.mod"))
	if err != nil {
		t.Fatalf("os.ReadFile(go.mod) after error = %v", err)
	}
	if string(before) != string(after) {
		t.Fatalf("go.mod mutated on failed module:add\nbefore:\n%s\nafter:\n%s", before, after)
	}
}
