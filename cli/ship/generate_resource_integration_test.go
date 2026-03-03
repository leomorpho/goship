package ship

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
	cli := CLI{Out: out, Err: errOut, Runner: &fakeRunner{}}
	code := cli.Run([]string{"new", "demo", "--module", "example.com/demo"})
	if code != 0 {
		t.Fatalf("ship new failed with code = %d, stderr=%s", code, errOut.String())
	}

	projectRoot := filepath.Join(root, "demo")
	if err := os.Chdir(projectRoot); err != nil {
		t.Fatal(err)
	}

	routerPath := filepath.Join(projectRoot, "app", "goship", "router.go")
	routeNamesPath := filepath.Join(projectRoot, "pkg", "routing", "routenames", "routenames.go")

	out.Reset()
	errOut.Reset()
	code = cli.Run([]string{"make:resource", "contact_form", "--path", "app/goship", "--auth", "public", "--views", "templ", "--wire"})
	if code != 0 {
		t.Fatalf("exit code = %d, stderr=%s", code, errOut.String())
	}

	handlerPath := filepath.Join(projectRoot, "app", "goship", "web", "routes", "contact_form.go")
	handlerBytes, err := os.ReadFile(handlerPath)
	if err != nil {
		t.Fatalf("read handler: %v", err)
	}
	handlerExpected := `package routes

import (
	"github.com/labstack/echo/v4"
	"github.com/leomorpho/goship/app/goship/views"
	"github.com/leomorpho/goship/app/goship/views/web/layouts/gen"
	"github.com/leomorpho/goship/app/goship/views/web/pages/gen"
	"github.com/leomorpho/goship/app/goship/controller"
)

type contactForm struct {
	ctr controller.Controller
}

func NewContactFormRoute(ctr controller.Controller) *contactForm {
	return &contactForm{ctr: ctr}
}

func (r *contactForm) Get(ctx echo.Context) error {
	page := controller.NewPage(ctx)
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

	templPath := filepath.Join(projectRoot, "app", "goship", "views", "web", "pages", "contact_form.templ")
	templBytes, err := os.ReadFile(templPath)
	if err != nil {
		t.Fatalf("read templ: %v", err)
	}
	templExpected := `package pages

import "github.com/leomorpho/goship/app/goship/controller"

templ ContactFormPage(page *controller.Page) {
	<section>
		<h1>ContactForm</h1>
		<p>TODO: implement contact-form page.</p>
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
