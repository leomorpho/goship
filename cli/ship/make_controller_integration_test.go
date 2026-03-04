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
