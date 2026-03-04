package generators

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseGenerateModelArgs(t *testing.T) {
	name, fields, force, err := ParseGenerateModelArgs([]string{"Post", "title:string", "published_at:time", "--force"})
	if err != nil {
		t.Fatalf("parseGenerateModelArgs error = %v", err)
	}
	if name != "Post" {
		t.Fatalf("name = %q, want Post", name)
	}
	if !force {
		t.Fatal("force = false, want true")
	}
	if len(fields) != 2 {
		t.Fatalf("fields len = %d, want 2", len(fields))
	}
	if fields[0].Name != "title" || fields[0].Type != "string" {
		t.Fatalf("field[0] = %+v", fields[0])
	}
}

func TestParseGenerateModelArgs_InvalidField(t *testing.T) {
	_, _, _, err := ParseGenerateModelArgs([]string{"Post", "Title:string"})
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
}

func TestRenderEntSchema(t *testing.T) {
	content, err := RenderEntSchema("Post", []ModelField{
		{Name: "title", Type: "string"},
		{Name: "published_at", Type: "time"},
		{Name: "is_live", Type: "bool"},
	})
	if err != nil {
		t.Fatalf("renderEntSchema error = %v", err)
	}
	if !strings.Contains(content, "type Post struct") {
		t.Fatalf("missing schema type declaration:\n%s", content)
	}
	if !strings.Contains(content, `field.String("title")`) {
		t.Fatalf("missing string field call:\n%s", content)
	}
	if !strings.Contains(content, `field.Time("published_at")`) {
		t.Fatalf("missing time field call:\n%s", content)
	}
}

func TestRunGenerateModel_WithFieldsWritesSchema(t *testing.T) {
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
	code := RunGenerateModel([]string{"Post", "title:string"}, GenerateModelDeps{
		Out: out,
		Err: errOut,
		RunCmd: func(name string, args ...string) int {
			return runner.RunCode(name, args...)
		},
		HasFile:      testHasFile,
		EntSchemaDir: "db/schema",
	})
	if code != 0 {
		t.Fatalf("exit code = %d, stderr=%s", code, errOut.String())
	}

	schemaPath := filepath.Join(root, "db", "schema", "post.go")
	b, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}
	if !strings.Contains(string(b), `field.String("title")`) {
		t.Fatalf("generated schema missing field:\n%s", string(b))
	}
	if len(runner.calls) != 1 {
		t.Fatalf("runner calls len = %d, want 1", len(runner.calls))
	}
	if runner.calls[0].name != "go" {
		t.Fatalf("runner call name = %q, want go", runner.calls[0].name)
	}
}

func TestRunGenerateModel_RefuseOverwriteWithoutForce(t *testing.T) {
	root := t.TempDir()
	prevWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(prevWD) })
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	schemaPath := filepath.Join(root, "db", "schema", "post.go")
	if err := os.MkdirAll(filepath.Dir(schemaPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(schemaPath, []byte("package schema\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	runner := &fakeRunner{}
	code := RunGenerateModel([]string{"Post", "title:string"}, GenerateModelDeps{
		Out: out,
		Err: errOut,
		RunCmd: func(name string, args ...string) int {
			return runner.RunCode(name, args...)
		},
		HasFile:      testHasFile,
		EntSchemaDir: "db/schema",
	})
	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if !strings.Contains(errOut.String(), "refusing to overwrite") {
		t.Fatalf("stderr = %q, want overwrite refusal", errOut.String())
	}
	if len(runner.calls) != 0 {
		t.Fatalf("runner calls = %v, want none", runner.calls)
	}
}
