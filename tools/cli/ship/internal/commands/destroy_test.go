package commands

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunDestroy_UsageAndValidation(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{
			name:    "missing args",
			args:    nil,
			wantErr: "usage: ship destroy resource:<name>",
		},
		{
			name:    "invalid artifact format",
			args:    []string{"resource"},
			wantErr: "invalid artifact format",
		},
		{
			name:    "unsupported kind",
			args:    []string{"island:counter"},
			wantErr: "unsupported destroy artifact kind",
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			out := &bytes.Buffer{}
			errOut := &bytes.Buffer{}
			code := RunDestroy(tt.args, DestroyDeps{Out: out, Err: errOut})
			if code == 0 {
				t.Fatalf("expected non-zero exit code, got %d", code)
			}
			if !strings.Contains(errOut.String(), tt.wantErr) {
				t.Fatalf("stderr = %q, want contains %q", errOut.String(), tt.wantErr)
			}
		})
	}
}

func TestRunDestroy_ResourceRemovesGeneratedTargetsDeterministically(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	mustMkdirAll(t, filepath.Join(root, "app", "web", "controllers"))
	mustMkdirAll(t, filepath.Join(root, "app", "views", "web", "pages"))
	mustMkdirAll(t, filepath.Join(root, "app", "web", "routenames"))

	routerPath := filepath.Join(root, "app", "router.go")
	routeNamesPath := filepath.Join(root, "app", "web", "routenames", "routenames.go")
	controllerPath := filepath.Join(root, "app", "web", "controllers", "contact.go")
	testPath := filepath.Join(root, "app", "web", "controllers", "contact_test.go")
	viewPath := filepath.Join(root, "app", "views", "web", "pages", "contact.templ")

	mustWriteFile(t, routerPath, `package app

func registerPublicRoutes() {
	// ship:routes:public:start
	// ship:generated:contact
	contact := controllers.NewContactRoute(ctr)
	g.GET("/contact", contact.Get).Name = routeNames.RouteNameContact

	// ship:routes:public:end
}
`)
	mustWriteFile(t, routeNamesPath, `package routenames

const (
	RouteNameLandingPage = "landing_page"
	RouteNameContact = "contact"
)
`)
	mustWriteFile(t, controllerPath, `package controllers

type contact struct {
	ctr ui.Controller
}

func NewContactRoute(ctr ui.Controller) *contact {
	return &contact{ctr: ctr}
}
`)
	mustWriteFile(t, testPath, `package controllers

func TestContactRoute_Get(t *testing.T) {
	// SCAFFOLD: implement Contact show — should return 200 with contact details
}
`)
	mustWriteFile(t, viewPath, `package pages

templ ContactPage(page *ui.Page) {
	<p>Scaffold page for contact. Replace with your real UI.</p>
}
`)

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	code := RunDestroy([]string{"resource:contact"}, DestroyDeps{Out: out, Err: errOut, Cwd: root})
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, errOut.String())
	}

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	wantPrefixes := []string{
		"app/router.go: removed generated route registration",
		"app/web/routenames/routenames.go: removed generated route name constant",
		"app/views/web/pages/contact.templ: deleted file",
		"app/web/controllers/contact_test.go: deleted file",
		"app/web/controllers/contact.go: deleted file",
	}
	if len(lines) != len(wantPrefixes) {
		t.Fatalf("line count = %d, want %d\n%s", len(lines), len(wantPrefixes), out.String())
	}
	for i := range wantPrefixes {
		if lines[i] != wantPrefixes[i] {
			t.Fatalf("line[%d] = %q, want %q", i, lines[i], wantPrefixes[i])
		}
	}

	for _, path := range []string{controllerPath, testPath, viewPath} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("expected %s to be deleted; stat err=%v", path, err)
		}
	}

	routerContent := mustReadFile(t, routerPath)
	if strings.Contains(routerContent, "// ship:generated:contact") {
		t.Fatalf("router should not contain generated marker after destroy:\n%s", routerContent)
	}
	if strings.Contains(routerContent, "RouteNameContact") {
		t.Fatalf("router should not contain route name after destroy:\n%s", routerContent)
	}

	routeNamesContent := mustReadFile(t, routeNamesPath)
	if strings.Contains(routeNamesContent, "RouteNameContact") {
		t.Fatalf("route names should not contain RouteNameContact after destroy:\n%s", routeNamesContent)
	}
}

func TestRunDestroy_ResourceSkipsNonManagedTargetsAndFailsWithoutMutations(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	mustMkdirAll(t, filepath.Join(root, "app", "web", "controllers"))
	mustMkdirAll(t, filepath.Join(root, "app", "views", "web", "pages"))
	mustMkdirAll(t, filepath.Join(root, "app", "web", "routenames"))

	mustWriteFile(t, filepath.Join(root, "app", "router.go"), `package app
func registerPublicRoutes() {}
`)
	mustWriteFile(t, filepath.Join(root, "app", "web", "routenames", "routenames.go"), `package routenames
const (
	RouteNameLandingPage = "landing_page"
)
`)
	mustWriteFile(t, filepath.Join(root, "app", "web", "controllers", "contact.go"), `package controllers
// handwritten controller with no generator ownership signal
`)
	mustWriteFile(t, filepath.Join(root, "app", "views", "web", "pages", "contact.templ"), `package pages
templ Arbitrary(page *ui.Page) {}
`)

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	code := RunDestroy([]string{"resource:contact"}, DestroyDeps{Out: out, Err: errOut, Cwd: root})
	if code == 0 {
		t.Fatal("expected non-zero exit code when nothing is recognized as generator-managed")
	}
	if !strings.Contains(errOut.String(), "refusing to destroy") {
		t.Fatalf("stderr = %q, want refusal message", errOut.String())
	}
	if !strings.Contains(out.String(), "skipped (no generator route marker found)") {
		t.Fatalf("stdout = %q, want skipped route marker reason", out.String())
	}
	if !strings.Contains(out.String(), "skipped (missing ownership signal: type contact struct {)") {
		t.Fatalf("stdout = %q, want skipped ownership reason for controller", out.String())
	}
}

func mustMkdirAll(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
}

func mustWriteFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func mustReadFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(b)
}
