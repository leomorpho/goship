package generators

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunMakeFactory_GeneratesFactoryFile(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}

	code := RunMakeFactory([]string{"User"}, FactoryDeps{
		Out: out,
		Err: errOut,
		Cwd: root,
	})
	if code != 0 {
		t.Fatalf("exit code = %d, stderr=%s", code, errOut.String())
	}

	path := filepath.Join(root, "tests", "factories", "user_factory.go")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read generated file: %v", err)
	}
	text := string(content)
	if !strings.Contains(text, "type UserRecord struct") {
		t.Fatalf("expected UserRecord type, got:\n%s", text)
	}
	if !strings.Contains(text, `func (UserRecord) TableName() string { return "users" }`) {
		t.Fatalf("expected users table name, got:\n%s", text)
	}
	if !strings.Contains(text, "var User = factory.New") {
		t.Fatalf("expected factory variable, got:\n%s", text)
	}
}

func TestRunMakeFactory_RefusesOverwrite(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	target := filepath.Join(root, "tests", "factories", "user_factory.go")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("package factories\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	code := RunMakeFactory([]string{"User"}, FactoryDeps{
		Out: out,
		Err: errOut,
		Cwd: root,
	})
	if code == 0 {
		t.Fatalf("expected overwrite attempt to fail")
	}
	if !strings.Contains(errOut.String(), "refusing to overwrite existing factory file") {
		t.Fatalf("stderr = %q, want overwrite error", errOut.String())
	}
}

func TestRunMakeFactory_Help(t *testing.T) {
	t.Parallel()

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	code := RunMakeFactory([]string{"--help"}, FactoryDeps{
		Out: out,
		Err: errOut,
	})
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if !strings.Contains(out.String(), "usage: ship make:factory <Name>") {
		t.Fatalf("stdout = %q, want usage", out.String())
	}
	if errOut.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", errOut.String())
	}
}
