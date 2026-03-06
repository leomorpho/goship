package commands

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRewriteGooseVersion(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cli.go")
	input := `package ship

const (
	gooseGoRunRef = "github.com/pressly/goose/v3/cmd/goose@v3.26.0"
)
`
	if err := os.WriteFile(path, []byte(input), 0o644); err != nil {
		t.Fatal(err)
	}

	old, updated, changed, err := RewriteGooseVersion(path, "v3.27.0")
	if err != nil {
		t.Fatalf("rewriteGooseVersion failed: %v", err)
	}
	if !changed {
		t.Fatal("expected changed=true")
	}
	if old != "v3.26.0" {
		t.Fatalf("old=%q want %q", old, "v3.26.0")
	}
	if !strings.Contains(updated, `gooseGoRunRef = "github.com/pressly/goose/v3/cmd/goose@v3.27.0"`) {
		t.Fatalf("updated text missing target version:\n%s", updated)
	}
}

func TestRunUpgrade(t *testing.T) {
	t.Run("missing version", func(t *testing.T) {
		out := &bytes.Buffer{}
		errOut := &bytes.Buffer{}
		code := RunUpgrade([]string{}, UpgradeDeps{Out: out, Err: errOut, FindGoModule: findGoModuleTest})
		if code != 1 {
			t.Fatalf("code=%d want=1", code)
		}
		if !strings.Contains(errOut.String(), "missing required --to version") {
			t.Fatalf("stderr=%q", errOut.String())
		}
	})

	t.Run("invalid version format", func(t *testing.T) {
		out := &bytes.Buffer{}
		errOut := &bytes.Buffer{}
		code := RunUpgrade([]string{"--to", "3.27.0"}, UpgradeDeps{Out: out, Err: errOut, FindGoModule: findGoModuleTest})
		if code != 1 {
			t.Fatalf("code=%d want=1", code)
		}
		if !strings.Contains(errOut.String(), "version must start with 'v'") {
			t.Fatalf("stderr=%q", errOut.String())
		}
	})

	t.Run("unexpected positional args", func(t *testing.T) {
		out := &bytes.Buffer{}
		errOut := &bytes.Buffer{}
		code := RunUpgrade([]string{"goose", "--to", "v3.27.0"}, UpgradeDeps{Out: out, Err: errOut, FindGoModule: findGoModuleTest})
		if code != 1 {
			t.Fatalf("code=%d want=1", code)
		}
		if !strings.Contains(errOut.String(), "unexpected upgrade arguments") {
			t.Fatalf("stderr=%q", errOut.String())
		}
	})

	t.Run("goose dry run", func(t *testing.T) {
		root := t.TempDir()
		if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/demo\n\ngo 1.25\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		cliPath := filepath.Join(root, "tools", "cli", "ship", "internal", "cli", "cli.go")
		if err := os.MkdirAll(filepath.Dir(cliPath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(cliPath, []byte(`package ship
const gooseGoRunRef = "github.com/pressly/goose/v3/cmd/goose@v3.26.0"
`), 0o644); err != nil {
			t.Fatal(err)
		}

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
		code := RunUpgrade([]string{"--to", "v3.27.0", "--dry-run"}, UpgradeDeps{Out: out, Err: errOut, FindGoModule: findGoModuleTest})
		if code != 0 {
			t.Fatalf("code=%d stderr=%s", code, errOut.String())
		}
		if !strings.Contains(out.String(), "dry-run: would update goose") {
			t.Fatalf("stdout=%q", out.String())
		}

		b, err := os.ReadFile(cliPath)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(b), "@v3.26.0") {
			t.Fatalf("cli.go should be unchanged in dry-run, got:\n%s", string(b))
		}
	})

	t.Run("goose update writes file", func(t *testing.T) {
		root := t.TempDir()
		if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/demo\n\ngo 1.25\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		cliPath := filepath.Join(root, "tools", "cli", "ship", "internal", "cli", "cli.go")
		if err := os.MkdirAll(filepath.Dir(cliPath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(cliPath, []byte(`package ship
const gooseGoRunRef = "github.com/pressly/goose/v3/cmd/goose@v3.26.0"
`), 0o644); err != nil {
			t.Fatal(err)
		}

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
		code := RunUpgrade([]string{"--to", "v3.27.0"}, UpgradeDeps{Out: out, Err: errOut, FindGoModule: findGoModuleTest})
		if code != 0 {
			t.Fatalf("code=%d stderr=%s", code, errOut.String())
		}
		b, err := os.ReadFile(cliPath)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(b), "@v3.27.0") {
			t.Fatalf("expected goose version update in cli.go, got:\n%s", string(b))
		}
	})
}
