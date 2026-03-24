package generators

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	cmd "github.com/leomorpho/goship/tools/cli/ship/internal/commands"
	policies "github.com/leomorpho/goship/tools/cli/ship/internal/policies"
)

func TestGeneratorsAcceptance_FreshAppMultiCommandMutation(t *testing.T) {
	root := t.TempDir()
	prevWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(prevWD) })
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	if code := cmd.RunNew([]string{"demo", "--module", "example.com/demo"}, cmd.NewDeps{
		Out:                        out,
		Err:                        errOut,
		ParseAgentPolicyBytes:      policies.ParsePolicyBytes,
		RenderAgentPolicyArtifacts: policies.RenderPolicyArtifacts,
		AgentPolicyFilePath:        policies.AgentPolicyFilePath,
	}); code != 0 {
		t.Fatalf("ship new failed with code = %d, stderr=%s", code, errOut.String())
	}

	projectRoot := filepath.Join(root, "demo")
	if err := os.Chdir(projectRoot); err != nil {
		t.Fatal(err)
	}

	out.Reset()
	errOut.Reset()
	if code := RunMakeController([]string{"AuditTrail", "--actions", "index,show", "--wire"}, ControllerDeps{
		Out:                    out,
		Err:                    errOut,
		HasFile:                testHasFile,
		EnsureRouteNamesImport: EnsureRouteNamesImport,
		WireRouteSnippet:       WireRouteSnippet,
	}); code != 0 {
		t.Fatalf("make:controller failed with code = %d, stderr=%s", code, errOut.String())
	}

	out.Reset()
	errOut.Reset()
	if code := RunGenerateResource([]string{"status_page", "--path", "app", "--views", "none", "--wire"}, out, errOut); code != 0 {
		t.Fatalf("make:resource failed with code = %d, stderr=%s", code, errOut.String())
	}

	out.Reset()
	errOut.Reset()
	if code := RunMakeJob([]string{"BackfillUserStats"}, MakeJobDeps{Out: out, Err: errOut, Cwd: projectRoot}); code != 0 {
		t.Fatalf("make:job failed with code = %d, stderr=%s", code, errOut.String())
	}

	out.Reset()
	errOut.Reset()
	if code := RunMakeModule([]string{"EmailSubscriptions"}, ModuleDeps{Out: out, Err: errOut, PathExists: testHasFile}); code != 0 {
		t.Fatalf("make:module failed with code = %d, stderr=%s", code, errOut.String())
	}

	requiredFiles := []string{
		filepath.Join(projectRoot, "app", "web", "controllers", "audit_trail.go"),
		filepath.Join(projectRoot, "app", "web", "controllers", "status_page.go"),
		filepath.Join(projectRoot, "app", "jobs", "backfill_user_stats.go"),
		filepath.Join(projectRoot, "modules", "emailsubscriptions", "go.mod"),
	}
	for _, p := range requiredFiles {
		if !testHasFile(p) {
			t.Fatalf("expected generated file %s", p)
		}
	}

	routerBytes, err := os.ReadFile(filepath.Join(projectRoot, "app", "router.go"))
	if err != nil {
		t.Fatal(err)
	}
	routerText := string(routerBytes)
	for _, token := range []string{
		"ship:generated:audit_trail",
		`g.GET("/audit-trail", auditTrail.Index)`,
		`g.GET("/audit-trail/:id", auditTrail.Show)`,
		"ship:generated:status_page",
		`g.GET("/status-page", statusPage.Get).Name = routeNames.RouteNameStatusPage`,
	} {
		if !strings.Contains(routerText, token) {
			t.Fatalf("router missing %q:\n%s", token, routerText)
		}
	}

	routeNamesBytes, err := os.ReadFile(filepath.Join(projectRoot, "app", "web", "routenames", "routenames.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(routeNamesBytes), `RouteNameStatusPage = "status_page"`) {
		t.Fatalf("route names missing generated status_page constant:\n%s", string(routeNamesBytes))
	}
}
