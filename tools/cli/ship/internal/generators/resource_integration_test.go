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

func TestGenerateResourceIntegration_FullGenerationExactOutput(t *testing.T) {
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
	code := cmd.RunNew([]string{"demo", "--module", "example.com/demo"}, cmd.NewDeps{
		Out:                        out,
		Err:                        errOut,
		ParseAgentPolicyBytes:      policies.ParsePolicyBytes,
		RenderAgentPolicyArtifacts: policies.RenderPolicyArtifacts,
		AgentPolicyFilePath:        policies.AgentPolicyFilePath,
	})
	if code != 0 {
		t.Fatalf("ship new failed with code = %d, stderr=%s", code, errOut.String())
	}

	projectRoot := filepath.Join(root, "demo")
	if err := os.Chdir(projectRoot); err != nil {
		t.Fatal(err)
	}

	routerPath := filepath.Join(projectRoot, "app", "router.go")
	routeNamesPath := filepath.Join(projectRoot, "app", "web", "routenames", "routenames.go")

	out.Reset()
	errOut.Reset()
	code = RunGenerateResource([]string{"contact_form", "--path", "app", "--auth", "public", "--views", "templ", "--wire"}, out, errOut)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr=%s", code, errOut.String())
	}

	handlerPath := filepath.Join(projectRoot, "app", "web", "controllers", "contact_form.go")
	handlerBytes, err := os.ReadFile(handlerPath)
	if err != nil {
		t.Fatalf("read handler: %v", err)
	}
	handlerExpected := `package controllers

import (
	"github.com/labstack/echo/v4"
	"github.com/leomorpho/goship/framework/web/templates"
	layouts "github.com/leomorpho/goship/framework/web/layouts/gen"
	pages "github.com/leomorpho/goship/framework/web/pages/gen"
	"github.com/leomorpho/goship/framework/web/ui"
)

type contactForm struct {
	ctr ui.Controller
}

func NewContactFormRoute(ctr ui.Controller) *contactForm {
	return &contactForm{ctr: ctr}
}

func (r *contactForm) Get(ctx echo.Context) error {
	page := ui.NewPage(ctx)
	page.Layout = layouts.Main
	page.Name = templates.Page("contact-form")
	page.Title = "ContactForm"
	page.Component = pages.ContactFormPage(&page)
	page.HTMX.Request.Boosted = true

	return r.ctr.RenderPage(ctx, page)
}
`
	if string(handlerBytes) != handlerExpected {
		t.Fatalf("handler mismatch\n--- got ---\n%s\n--- want ---\n%s", string(handlerBytes), handlerExpected)
	}

	templPath := filepath.Join(projectRoot, "app", "views", "web", "pages", "contact_form.templ")
	templBytes, err := os.ReadFile(templPath)
	if err != nil {
		t.Fatalf("read templ: %v", err)
	}
	templExpected := `package pages

import "github.com/leomorpho/goship/framework/web/ui"

templ ContactFormPage(page *ui.Page) {
	<section>
		<h1>ContactForm</h1>
		<p>Scaffold page for contact-form. Replace with your real UI.</p>
	</section>
}
`
	if string(templBytes) != templExpected {
		t.Fatalf("templ mismatch\n--- got ---\n%s\n--- want ---\n%s", string(templBytes), templExpected)
	}

	updatedRouterBytes, err := os.ReadFile(routerPath)
	if err != nil {
		t.Fatalf("read router: %v", err)
	}
	updatedRouter := string(updatedRouterBytes)
	if !strings.Contains(updatedRouter, "ship:generated:contact_form") {
		t.Fatalf("expected generated marker in router:\n%s", updatedRouter)
	}
	if !strings.Contains(updatedRouter, `g.GET("/contact-form", contactForm.Get).Name = routeNames.RouteNameContactForm`) {
		t.Fatalf("expected generated route mapping in router:\n%s", updatedRouter)
	}

	updatedRouteNamesBytes, err := os.ReadFile(routeNamesPath)
	if err != nil {
		t.Fatalf("read routenames: %v", err)
	}
	updatedRouteNames := string(updatedRouteNamesBytes)
	if !strings.Contains(updatedRouteNames, `RouteNameContactForm = "contact_form"`) {
		t.Fatalf("expected generated route name constant:\n%s", updatedRouteNames)
	}
}

func TestGenerateResourceIntegration_WireStableAcrossMultipleRuns(t *testing.T) {
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

	routerPath := filepath.Join(projectRoot, "app", "router.go")
	router, err := os.ReadFile(routerPath)
	if err != nil {
		t.Fatal(err)
	}
	routerNoRouteNames := strings.ReplaceAll(string(router), "routeNames \"example.com/demo/app/web/routenames\"\n", "")
	if err := os.WriteFile(routerPath, []byte(routerNoRouteNames), 0o644); err != nil {
		t.Fatal(err)
	}

	out.Reset()
	errOut.Reset()
	if code := RunGenerateResource([]string{"inbox", "--path", "app", "--views", "none", "--wire"}, out, errOut); code != 0 {
		t.Fatalf("first make:resource failed with code = %d, stderr=%s", code, errOut.String())
	}
	out.Reset()
	errOut.Reset()
	if code := RunGenerateResource([]string{"alerts", "--path", "app", "--views", "none", "--wire"}, out, errOut); code != 0 {
		t.Fatalf("second make:resource failed with code = %d, stderr=%s", code, errOut.String())
	}

	updatedRouter, err := os.ReadFile(routerPath)
	if err != nil {
		t.Fatal(err)
	}
	routerText := string(updatedRouter)
	if strings.Count(routerText, `routeNames "github.com/leomorpho/goship/framework/web/routenames"`) != 1 {
		t.Fatalf("routeNames import should be inserted once, got router:\n%s", routerText)
	}
	if strings.Count(routerText, "ship:generated:inbox") != 1 || strings.Count(routerText, "ship:generated:alerts") != 1 {
		t.Fatalf("expected one generated block per resource, got router:\n%s", routerText)
	}
}

func TestGenerateResourceIntegration_DuplicateRunDoesNotMutateWiring(t *testing.T) {
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

	routerPath := filepath.Join(projectRoot, "app", "router.go")
	routeNamesPath := filepath.Join(projectRoot, "app", "web", "routenames", "routenames.go")

	out.Reset()
	errOut.Reset()
	if code := RunGenerateResource([]string{"inbox", "--path", "app", "--views", "none", "--wire"}, out, errOut); code != 0 {
		t.Fatalf("first make:resource failed with code = %d, stderr=%s", code, errOut.String())
	}
	routerBefore, err := os.ReadFile(routerPath)
	if err != nil {
		t.Fatal(err)
	}
	routeNamesBefore, err := os.ReadFile(routeNamesPath)
	if err != nil {
		t.Fatal(err)
	}

	out.Reset()
	errOut.Reset()
	if code := RunGenerateResource([]string{"inbox", "--path", "app", "--views", "none", "--wire"}, out, errOut); code == 0 {
		t.Fatalf("expected duplicate make:resource to fail")
	}
	if !strings.Contains(errOut.String(), "refusing to overwrite existing file") {
		t.Fatalf("expected overwrite refusal error, stderr=%s", errOut.String())
	}

	routerAfter, err := os.ReadFile(routerPath)
	if err != nil {
		t.Fatal(err)
	}
	routeNamesAfter, err := os.ReadFile(routeNamesPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(routerBefore) != string(routerAfter) {
		t.Fatalf("router changed during failed duplicate generation\nbefore:\n%s\nafter:\n%s", string(routerBefore), string(routerAfter))
	}
	if string(routeNamesBefore) != string(routeNamesAfter) {
		t.Fatalf("route names changed during failed duplicate generation\nbefore:\n%s\nafter:\n%s", string(routeNamesBefore), string(routeNamesAfter))
	}
}
