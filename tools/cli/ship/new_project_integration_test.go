package ship

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewProjectIntegration_IncludesEntAndSupportsMakeModel(t *testing.T) {
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
	runner := &fakeRunner{}
	cli := CLI{Out: out, Err: errOut, Runner: runner}

	if code := cli.Run([]string{"new", "demo", "--module", "example.com/demo"}); code != 0 {
		t.Fatalf("ship new failed: code=%d stderr=%s", code, errOut.String())
	}

	projectRoot := filepath.Join(root, "demo")
	entSchemaUser := filepath.Join(projectRoot, "apps", "db", "schema", "user.go")
	if _, err := os.Stat(entSchemaUser); err != nil {
		t.Fatalf("expected ent schema scaffold at %s: %v", entSchemaUser, err)
	}
	entMigrationsKeep := filepath.Join(projectRoot, "apps", "db", "migrate", "migrations", ".gitkeep")
	if _, err := os.Stat(entMigrationsKeep); err != nil {
		t.Fatalf("expected migrations scaffold at %s: %v", entMigrationsKeep, err)
	}

	if err := os.Chdir(projectRoot); err != nil {
		t.Fatal(err)
	}

	out.Reset()
	errOut.Reset()
	if code := cli.Run([]string{"doctor"}); code != 0 {
		t.Fatalf("ship doctor failed on fresh scaffold: code=%d stderr=%s", code, errOut.String())
	}

	out.Reset()
	errOut.Reset()
	runner.calls = nil

	if code := cli.Run([]string{"make:model", "Post", "title:string"}); code != 0 {
		t.Fatalf("ship make:model failed: code=%d stderr=%s", code, errOut.String())
	}

	generatedSchema := filepath.Join(projectRoot, "apps", "db", "schema", "post.go")
	b, err := os.ReadFile(generatedSchema)
	if err != nil {
		t.Fatalf("expected generated model schema at %s: %v", generatedSchema, err)
	}
	if !strings.Contains(string(b), `field.String("title")`) {
		t.Fatalf("generated schema missing expected field:\n%s", string(b))
	}

	if len(runner.calls) != 1 {
		t.Fatalf("runner call count = %d, want 1", len(runner.calls))
	}
	got := strings.Join(runner.calls[0].args, " ")
	if !strings.Contains(got, "--target ./apps/db/ent ./apps/db/schema") {
		t.Fatalf("unexpected generate args: %v", runner.calls[0].args)
	}
}
