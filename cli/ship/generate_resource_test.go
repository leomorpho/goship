package ship

import (
	"bytes"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNormalizeResourceName(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantSnake  string
		wantKebab  string
		wantPascal string
		wantErr    bool
	}{
		{name: "snake", input: "blog_post", wantSnake: "blog_post", wantKebab: "blog-post", wantPascal: "BlogPost"},
		{name: "kebab", input: "blog-post", wantSnake: "blog_post", wantKebab: "blog-post", wantPascal: "BlogPost"},
		{name: "camel", input: "BlogPost", wantSnake: "blog_post", wantKebab: "blog-post", wantPascal: "BlogPost"},
		{name: "acronym", input: "APIClient", wantSnake: "api_client", wantKebab: "api-client", wantPascal: "ApiClient"},
		{name: "spaced", input: "user profile", wantSnake: "user_profile", wantKebab: "user-profile", wantPascal: "UserProfile"},
		{name: "empty", input: "   ", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeResourceName(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Snake != tt.wantSnake {
				t.Fatalf("snake = %q, want %q", got.Snake, tt.wantSnake)
			}
			if got.Kebab != tt.wantKebab {
				t.Fatalf("kebab = %q, want %q", got.Kebab, tt.wantKebab)
			}
			if got.Pascal != tt.wantPascal {
				t.Fatalf("pascal = %q, want %q", got.Pascal, tt.wantPascal)
			}
		})
	}
}

func TestGenerateResourceScaffold(t *testing.T) {
	root := t.TempDir()
	basePath := filepath.Join(root, "app", "goship")

	result, err := generateResourceScaffold(resourceGenerateOptions{
		Name:  "contact_form",
		Path:  basePath,
		Auth:  "public",
		Views: "templ",
	})
	if err != nil {
		t.Fatalf("generateResourceScaffold error: %v", err)
	}

	handlerPath := filepath.Join(basePath, "web", "routes", "contact_form.go")
	viewPath := filepath.Join(basePath, "views", "web", "pages", "contact_form.templ")

	if _, err := os.Stat(handlerPath); err != nil {
		t.Fatalf("expected handler file, stat error: %v", err)
	}
	if _, err := os.Stat(viewPath); err != nil {
		t.Fatalf("expected templ file, stat error: %v", err)
	}
	handlerContent, err := os.ReadFile(handlerPath)
	if err != nil {
		t.Fatalf("read handler file: %v", err)
	}
	handlerText := string(handlerContent)
	if !strings.Contains(handlerText, "webui.NewPage(ctx)") {
		t.Fatalf("expected templ handler to build page object, got:\n%s", handlerText)
	}
	if !strings.Contains(handlerText, "return r.ctr.RenderPage(ctx, page)") {
		t.Fatalf("expected templ handler to render page, got:\n%s", handlerText)
	}
	if !strings.Contains(handlerText, `github.com/leomorpho/goship/app/goship/views/web/pages/gen`) {
		t.Fatalf("expected templ handler to import pages/gen, got:\n%s", handlerText)
	}
	viewContent, err := os.ReadFile(viewPath)
	if err != nil {
		t.Fatalf("read templ file: %v", err)
	}
	if !strings.Contains(string(viewContent), "templ ContactFormPage(page *webui.Page)") {
		t.Fatalf("expected templ page signature with webui.Page, got:\n%s", string(viewContent))
	}
	if len(result.CreatedFiles) != 2 {
		t.Fatalf("created files = %d, want 2", len(result.CreatedFiles))
	}
	if !strings.Contains(result.RouteSnippet, `g.GET("/contact-form", contactForm.Get)`) {
		t.Fatalf("unexpected route snippet: %s", result.RouteSnippet)
	}
}

func TestGenerateResourceScaffold_NoneViews(t *testing.T) {
	root := t.TempDir()
	basePath := filepath.Join(root, "app", "goship")

	result, err := generateResourceScaffold(resourceGenerateOptions{
		Name:  "inbox",
		Path:  basePath,
		Auth:  "auth",
		Views: "none",
	})
	if err != nil {
		t.Fatalf("generateResourceScaffold error: %v", err)
	}

	if len(result.CreatedFiles) != 1 {
		t.Fatalf("created files = %d, want 1", len(result.CreatedFiles))
	}
	if !strings.Contains(result.RouteSnippet, `onboardedGroup.GET("/inbox", inbox.Get)`) {
		t.Fatalf("unexpected auth route snippet: %s", result.RouteSnippet)
	}
}

func TestGenerateResourceScaffold_Validation(t *testing.T) {
	root := t.TempDir()
	basePath := filepath.Join(root, "app", "goship")

	tests := []resourceGenerateOptions{
		{Name: "x", Path: basePath, Auth: "private", Views: "templ"},
		{Name: "x", Path: basePath, Auth: "public", Views: "jsx"},
		{Name: "", Path: basePath, Auth: "public", Views: "none"},
	}

	for _, tt := range tests {
		if _, err := generateResourceScaffold(tt); err == nil {
			t.Fatalf("expected error for opts: %+v", tt)
		}
	}
}

func TestGenerateResourceScaffold_RefuseOverwrite(t *testing.T) {
	root := t.TempDir()
	basePath := filepath.Join(root, "app", "goship")

	_, err := generateResourceScaffold(resourceGenerateOptions{
		Name:  "contact",
		Path:  basePath,
		Auth:  "public",
		Views: "none",
	})
	if err != nil {
		t.Fatalf("first generation should succeed: %v", err)
	}
	_, err = generateResourceScaffold(resourceGenerateOptions{
		Name:  "contact",
		Path:  basePath,
		Auth:  "public",
		Views: "none",
	})
	if err == nil {
		t.Fatalf("expected overwrite protection error")
	}
}

func TestWireRouteSnippet(t *testing.T) {
	base := `package goship

func registerPublicRoutes() {
	// ship:routes:public:start
	// ship:routes:public:end
}

func registerAuthRoutes() {
	// ship:routes:auth:start
	// ship:routes:auth:end
}
`
	routerPath := filepath.Join(t.TempDir(), "router.go")
	if err := os.WriteFile(routerPath, []byte(base), 0o644); err != nil {
		t.Fatal(err)
	}

	publicSnippet := "\t// ship:generated:contact\n\tcontact := controllers.NewContactRoute(ctr)\n\tg.GET(\"/contact\", contact.Get).Name = routeNames.RouteNameContact\n"
	if err := wireRouteSnippet(routerPath, "public", publicSnippet, false); err != nil {
		t.Fatalf("wire public failed: %v", err)
	}
	// Idempotent second call.
	if err := wireRouteSnippet(routerPath, "public", publicSnippet, false); err != nil {
		t.Fatalf("wire public second call failed: %v", err)
	}

	authSnippet := "\t// ship:generated:inbox\n\tinbox := controllers.NewInboxRoute(ctr)\n\tonboardedGroup.GET(\"/inbox\", inbox.Get).Name = routeNames.RouteNameInbox\n"
	if err := wireRouteSnippet(routerPath, "auth", authSnippet, false); err != nil {
		t.Fatalf("wire auth failed: %v", err)
	}

	contentBytes, err := os.ReadFile(routerPath)
	if err != nil {
		t.Fatal(err)
	}
	content := string(contentBytes)
	if strings.Count(content, "ship:generated:contact") != 1 {
		t.Fatalf("expected one contact insertion, got content:\n%s", content)
	}
	if !strings.Contains(content, "ship:generated:inbox") {
		t.Fatalf("expected auth insertion, got content:\n%s", content)
	}
}

func TestWireRouteSnippet_MarkerErrors(t *testing.T) {
	routerPath := filepath.Join(t.TempDir(), "router.go")
	if err := os.WriteFile(routerPath, []byte("package goship\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	err := wireRouteSnippet(routerPath, "public", "\tfoo := bar\n", false)
	if err == nil {
		t.Fatalf("expected marker error")
	}

	err = wireRouteSnippet(routerPath, "internal", "\tfoo := bar\n", false)
	if err == nil {
		t.Fatalf("expected auth group validation error")
	}
}

func TestWireRouteNameConstant(t *testing.T) {
	path := filepath.Join(t.TempDir(), "routenames.go")
	base := `package routenames

const (
	RouteNameLandingPage = "landing_page"
)
`
	if err := os.WriteFile(path, []byte(base), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := wireRouteNameConstant(path, "RouteNameInbox", "inbox", false); err != nil {
		t.Fatalf("wireRouteNameConstant failed: %v", err)
	}
	// idempotent
	if err := wireRouteNameConstant(path, "RouteNameInbox", "inbox", false); err != nil {
		t.Fatalf("wireRouteNameConstant second call failed: %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(content), "RouteNameInbox") != 1 {
		t.Fatalf("expected single RouteNameInbox insertion, got:\n%s", string(content))
	}
}

func TestEnsureRouteNamesImport(t *testing.T) {
	path := filepath.Join(t.TempDir(), "router.go")
	base := `package goship

import (
	"fmt"
)
`
	if err := os.WriteFile(path, []byte(base), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := ensureRouteNamesImport(path, false); err != nil {
		t.Fatalf("ensureRouteNamesImport failed: %v", err)
	}
	if err := ensureRouteNamesImport(path, false); err != nil {
		t.Fatalf("ensureRouteNamesImport second call failed: %v", err)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(content), `routeNames "github.com/leomorpho/goship/app/goship/web/routenames"`) != 1 {
		t.Fatalf("expected single routeNames import insertion, got:\n%s", string(content))
	}
}

func TestRunGenerateResourceDryRun(t *testing.T) {
	root := t.TempDir()
	prevWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(prevWD) })
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}

	routerPath := filepath.Join(root, "app", "goship", "router.go")
	if err := os.MkdirAll(filepath.Dir(routerPath), 0o755); err != nil {
		t.Fatal(err)
	}
	routerContent := `package goship

import (
	routeNames "github.com/leomorpho/goship/app/goship/web/routenames"
)

func registerPublicRoutes() {
	// ship:routes:public:start
	// ship:routes:public:end
}
`
	if err := os.WriteFile(routerPath, []byte(routerContent), 0o644); err != nil {
		t.Fatal(err)
	}

	routeNamesPath := filepath.Join(root, "pkg", "routing", "routenames", "routenames.go")
	if err := os.MkdirAll(filepath.Dir(routeNamesPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(routeNamesPath, []byte("package routenames\n\nconst (\n)\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	cli := CLI{Out: out, Err: errOut, Runner: &fakeRunner{}}
	code := cli.Run([]string{"make:resource", "inbox", "--path", "app/goship", "--wire", "--dry-run", "--views", "none"})
	if code != 0 {
		t.Fatalf("exit code = %d, stderr=%s", code, errOut.String())
	}
	if !strings.Contains(out.String(), "Dry-run mode: no files were written.") {
		t.Fatalf("expected dry-run message, stdout=%s", out.String())
	}

	handlerPath := filepath.Join(root, "app", "goship", "web", "routes", "inbox.go")
	if _, err := os.Stat(handlerPath); !os.IsNotExist(err) {
		t.Fatalf("expected no handler file to be written in dry-run mode")
	}
}

func TestRunGenerateResourceWireWritesExpected(t *testing.T) {
	root := t.TempDir()
	prevWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(prevWD) })
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}

	routerPath := filepath.Join(root, "app", "goship", "router.go")
	if err := os.MkdirAll(filepath.Dir(routerPath), 0o755); err != nil {
		t.Fatal(err)
	}
	routerContent := `package goship

import (
	"fmt"
	routeNames "github.com/leomorpho/goship/app/goship/web/routenames"
	"github.com/leomorpho/goship/app/goship/web/controllers"
)

func registerPublicRoutes() {
	// ship:routes:public:start
	// ship:routes:public:end
}
`
	if err := os.WriteFile(routerPath, []byte(routerContent), 0o644); err != nil {
		t.Fatal(err)
	}

	routeNamesPath := filepath.Join(root, "pkg", "routing", "routenames", "routenames.go")
	if err := os.MkdirAll(filepath.Dir(routeNamesPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(routeNamesPath, []byte("package routenames\n\nconst (\n)\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	cli := CLI{Out: out, Err: errOut, Runner: &fakeRunner{}}
	code := cli.Run([]string{"make:resource", "inbox", "--path", "app/goship", "--wire", "--views", "none"})
	if code != 0 {
		t.Fatalf("exit code = %d, stderr=%s", code, errOut.String())
	}

	handlerPath := filepath.Join(root, "app", "goship", "web", "routes", "inbox.go")
	if _, err := os.Stat(handlerPath); err != nil {
		t.Fatalf("expected handler file: %v", err)
	}

	routerBytes, err := os.ReadFile(routerPath)
	if err != nil {
		t.Fatal(err)
	}
	routerContentOut := string(routerBytes)
	if !strings.Contains(routerContentOut, "ship:generated:inbox") {
		t.Fatalf("expected generated marker in router, got:\n%s", routerContentOut)
	}

	routeNamesBytes, err := os.ReadFile(routeNamesPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(routeNamesBytes), `RouteNameInbox = "inbox"`) {
		t.Fatalf("expected routename constant insertion, got:\n%s", string(routeNamesBytes))
	}

	// Syntax smoke-check generated handler file.
	fset := token.NewFileSet()
	if _, err := parser.ParseFile(fset, handlerPath, nil, parser.AllErrors); err != nil {
		t.Fatalf("generated handler is not valid Go syntax: %v", err)
	}
}
