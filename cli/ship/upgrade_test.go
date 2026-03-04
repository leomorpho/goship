package ship

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRewriteTemplVersion(t *testing.T) {
	path := filepath.Join(t.TempDir(), "go.mod")
	input := `module example.com/demo

go 1.25

require (
	github.com/a-h/templ v0.3.1001
)
`
	if err := os.WriteFile(path, []byte(input), 0o644); err != nil {
		t.Fatal(err)
	}

	old, updated, changed, err := rewriteTemplVersion(path, "v0.3.1002")
	if err != nil {
		t.Fatalf("rewriteTemplVersion failed: %v", err)
	}
	if !changed {
		t.Fatal("expected changed=true")
	}
	if old != "v0.3.1001" {
		t.Fatalf("old=%q want %q", old, "v0.3.1001")
	}
	if !strings.Contains(updated, "github.com/a-h/templ v0.3.1002") {
		t.Fatalf("updated text missing target version:\n%s", updated)
	}
}

func TestRewriteAtlasVersion(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cli.go")
	input := `package ship

const (
	atlasGoRunRef = "ariga.io/atlas/cmd/atlas@v0.27.1"
)
`
	if err := os.WriteFile(path, []byte(input), 0o644); err != nil {
		t.Fatal(err)
	}

	old, updated, changed, err := rewriteAtlasVersion(path, "v0.28.0")
	if err != nil {
		t.Fatalf("rewriteAtlasVersion failed: %v", err)
	}
	if !changed {
		t.Fatal("expected changed=true")
	}
	if old != "v0.27.1" {
		t.Fatalf("old=%q want %q", old, "v0.27.1")
	}
	if !strings.Contains(updated, `atlasGoRunRef = "ariga.io/atlas/cmd/atlas@v0.28.0"`) {
		t.Fatalf("updated text missing target version:\n%s", updated)
	}
}

func TestRunUpgrade(t *testing.T) {
	t.Run("missing version", func(t *testing.T) {
		out := &bytes.Buffer{}
		errOut := &bytes.Buffer{}
		cli := CLI{Out: out, Err: errOut, Runner: &fakeRunner{}}
		code := cli.Run([]string{"upgrade", "templ"})
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
		cli := CLI{Out: out, Err: errOut, Runner: &fakeRunner{}}
		code := cli.Run([]string{"upgrade", "templ", "--to", "0.3.1002"})
		if code != 1 {
			t.Fatalf("code=%d want=1", code)
		}
		if !strings.Contains(errOut.String(), "version must start with 'v'") {
			t.Fatalf("stderr=%q", errOut.String())
		}
	})

	t.Run("templ dry run", func(t *testing.T) {
		root := t.TempDir()
		if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte(`module example.com/demo

go 1.25

require github.com/a-h/templ v0.3.1001
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
		cli := CLI{Out: out, Err: errOut, Runner: &fakeRunner{}}
		code := cli.Run([]string{"upgrade", "templ", "--to", "v0.3.1002", "--dry-run"})
		if code != 0 {
			t.Fatalf("code=%d stderr=%s", code, errOut.String())
		}
		if !strings.Contains(out.String(), "dry-run: would update templ") {
			t.Fatalf("stdout=%q", out.String())
		}

		b, err := os.ReadFile(filepath.Join(root, "go.mod"))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(b), "v0.3.1001") {
			t.Fatalf("go.mod should be unchanged in dry-run, got:\n%s", string(b))
		}
	})

	t.Run("atlas update writes file", func(t *testing.T) {
		root := t.TempDir()
		if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/demo\n\ngo 1.25\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		cliPath := filepath.Join(root, "cli", "ship", "cli.go")
		if err := os.MkdirAll(filepath.Dir(cliPath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(cliPath, []byte(`package ship
const atlasGoRunRef = "ariga.io/atlas/cmd/atlas@v0.27.1"
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
		cli := CLI{Out: out, Err: errOut, Runner: &fakeRunner{}}
		code := cli.Run([]string{"upgrade", "atlas", "--to", "v0.28.0"})
		if code != 0 {
			t.Fatalf("code=%d stderr=%s", code, errOut.String())
		}
		b, err := os.ReadFile(cliPath)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(b), "@v0.28.0") {
			t.Fatalf("expected atlas version update in cli.go, got:\n%s", string(b))
		}
	})
}
