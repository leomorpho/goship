package commands

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunAPI(t *testing.T) {
	root := t.TempDir()
	writeAPISpecFixture(t, root)

	prevWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(prevWD) })
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}

	t.Run("spec json output", func(t *testing.T) {
		out := &bytes.Buffer{}
		errOut := &bytes.Buffer{}
		if code := RunAPI([]string{"spec"}, APIDeps{Out: out, Err: errOut, FindGoModule: findAPISpecGoModule}); code != 0 {
			t.Fatalf("exit code = %d, stderr=%s", code, errOut.String())
		}
		if errOut.Len() != 0 {
			t.Fatalf("stderr = %q, want empty", errOut.String())
		}

		var payload map[string]any
		if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
			t.Fatalf("decode json: %v", err)
		}
		if payload["openapi"] != "3.0.0" {
			t.Fatalf("openapi = %v, want 3.0.0", payload["openapi"])
		}

		paths, _ := payload["paths"].(map[string]any)
		if _, ok := paths["/profile/{id}"]; !ok {
			t.Fatalf("paths = %#v, want /profile/{id}", paths)
		}
		if _, ok := paths["/login"]; !ok {
			t.Fatalf("paths = %#v, want /login", paths)
		}

		components, _ := payload["components"].(map[string]any)
		schemas, _ := components["schemas"].(map[string]any)
		loginSchema, _ := schemas["LoginRequest"].(map[string]any)
		required, _ := loginSchema["required"].([]any)
		if len(required) == 0 {
			t.Fatalf("LoginRequest.required = %#v, want non-empty", required)
		}
	})

	t.Run("spec out file", func(t *testing.T) {
		out := &bytes.Buffer{}
		errOut := &bytes.Buffer{}
		target := filepath.Join(root, "openapi.json")
		if code := RunAPI([]string{"spec", "--out", target}, APIDeps{Out: out, Err: errOut, FindGoModule: findAPISpecGoModule}); code != 0 {
			t.Fatalf("exit code = %d, stderr=%s", code, errOut.String())
		}
		if errOut.Len() != 0 {
			t.Fatalf("stderr = %q, want empty", errOut.String())
		}
		content, err := os.ReadFile(target)
		if err != nil {
			t.Fatalf("read output: %v", err)
		}
		if !strings.Contains(string(content), "\"openapi\": \"3.0.0\"") {
			t.Fatalf("file content = %q, want openapi marker", string(content))
		}
	})

	t.Run("help", func(t *testing.T) {
		out := &bytes.Buffer{}
		if code := RunAPI([]string{"--help"}, APIDeps{Out: out, Err: &bytes.Buffer{}, FindGoModule: findAPISpecGoModule}); code != 0 {
			t.Fatalf("exit code = %d, want 0", code)
		}
		if !strings.Contains(out.String(), "ship api commands:") {
			t.Fatalf("stdout = %q, want api help", out.String())
		}
	})
}

func writeAPISpecFixture(t *testing.T, root string) {
	t.Helper()

	dirs := []string{
		filepath.Join(root, "app", "contracts"),
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	files := map[string]string{
		filepath.Join(root, "go.mod"): "module example.com/api-spec\n\ngo 1.25\n",
		filepath.Join(root, "app", "contracts", "sample.go"): `package contracts

import "time"

// Route: GET /profile/:id
type ProfilePage struct {
	ID        int
	Name      string
	CreatedAt time.Time
}

// Route: POST /login
type LoginRequest struct {
	Email    string ` + "`validate:\"required,email\"`" + `
	Password string ` + "`validate:\"required\"`" + `
}
`,
	}

	for path, content := range files {
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func findAPISpecGoModule(start string) (string, string, error) {
	dir := start
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, "", nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", "", os.ErrNotExist
		}
		dir = parent
	}
}
