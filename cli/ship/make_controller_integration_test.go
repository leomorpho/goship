package ship

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMakeControllerIntegration_GeneratesControllerAndSnippet(t *testing.T) {
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

	code := cli.Run([]string{"make:controller", "Posts", "--actions", "index,show"})
	if code != 0 {
		t.Fatalf("exit code = %d, stderr=%s", code, errOut.String())
	}

	controllerPath := filepath.Join(root, "apps", "goship", "web", "controllers", "posts.go")
	content, err := os.ReadFile(controllerPath)
	if err != nil {
		t.Fatalf("read controller: %v", err)
	}
	text := string(content)
	if !strings.Contains(text, "func (c *posts) Index") {
		t.Fatalf("missing Index action:\n%s", text)
	}
	if !strings.Contains(text, "func (c *posts) Show") {
		t.Fatalf("missing Show action:\n%s", text)
	}
	if !strings.Contains(out.String(), "Route snippet:") {
		t.Fatalf("stdout missing route snippet:\n%s", out.String())
	}
}

func TestMakeControllerIntegration_WireIntoRouter(t *testing.T) {
	root := t.TempDir()
	prevWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(prevWD) })
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}

	routerPath := filepath.Join(root, "apps", "goship", "router.go")
	if err := os.MkdirAll(filepath.Dir(routerPath), 0o755); err != nil {
		t.Fatal(err)
	}
	routerContent := `package goship

import (
	routeNames "github.com/leomorpho/goship/apps/goship/web/routenames"
	"github.com/leomorpho/goship/apps/goship/web/controllers"
)

func registerPublicRoutes() {
	// ship:routes:public:start
	// ship:routes:public:end
}

func registerAuthRoutes() {
	// ship:routes:auth:start
	// ship:routes:auth:end
}
`
	if err := os.WriteFile(routerPath, []byte(routerContent), 0o644); err != nil {
		t.Fatal(err)
	}

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	cli := CLI{Out: out, Err: errOut, Runner: &fakeRunner{}}
	code := cli.Run([]string{"make:controller", "Posts", "--actions", "index", "--wire"})
	if code != 0 {
		t.Fatalf("exit code = %d, stderr=%s", code, errOut.String())
	}

	updated, err := os.ReadFile(routerPath)
	if err != nil {
		t.Fatalf("read router: %v", err)
	}
	if !strings.Contains(string(updated), "ship:generated:posts") {
		t.Fatalf("router missing generated marker:\n%s", string(updated))
	}
	if !strings.Contains(string(updated), `g.GET("/posts", posts.Index)`) {
		t.Fatalf("router missing index route:\n%s", string(updated))
	}
}

func TestMakeControllerIntegration_WireStableAcrossMultipleRuns(t *testing.T) {
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
	if code := cli.Run([]string{"new", "demo", "--module", "example.com/demo"}); code != 0 {
		t.Fatalf("ship new failed with code = %d, stderr=%s", code, errOut.String())
	}

	projectRoot := filepath.Join(root, "demo")
	if err := os.Chdir(projectRoot); err != nil {
		t.Fatal(err)
	}

	routerPath := filepath.Join(projectRoot, "apps", "goship", "router.go")
	router, err := os.ReadFile(routerPath)
	if err != nil {
		t.Fatal(err)
	}
	routerNoRouteNames := strings.ReplaceAll(string(router), "routeNames \"example.com/demo/apps/goship/web/routenames\"\n", "")
	if err := os.WriteFile(routerPath, []byte(routerNoRouteNames), 0o644); err != nil {
		t.Fatal(err)
	}

	out.Reset()
	errOut.Reset()
	if code := cli.Run([]string{"make:controller", "Posts", "--actions", "index", "--wire"}); code != 0 {
		t.Fatalf("first make:controller failed with code = %d, stderr=%s", code, errOut.String())
	}
	out.Reset()
	errOut.Reset()
	if code := cli.Run([]string{"make:controller", "Comments", "--actions", "index", "--wire"}); code != 0 {
		t.Fatalf("second make:controller failed with code = %d, stderr=%s", code, errOut.String())
	}

	updated, err := os.ReadFile(routerPath)
	if err != nil {
		t.Fatal(err)
	}
	routerText := string(updated)
	if strings.Count(routerText, `routeNames "github.com/leomorpho/goship/apps/goship/web/routenames"`) != 1 {
		t.Fatalf("routeNames import should be inserted once, got router:\n%s", routerText)
	}
	if strings.Count(routerText, "ship:generated:posts") != 1 || strings.Count(routerText, "ship:generated:comments") != 1 {
		t.Fatalf("expected one generated block per controller, got router:\n%s", routerText)
	}
}
