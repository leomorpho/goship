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

func TestRenderModelQueryTemplate(t *testing.T) {
	content := RenderModelQueryTemplate("Post", []ModelField{
		{Name: "title", Type: "string"},
		{Name: "published_at", Type: "time"},
		{Name: "is_live", Type: "bool"},
	})
	if !strings.Contains(content, "-- Model: Post") {
		t.Fatalf("missing model declaration:\n%s", content)
	}
	if !strings.Contains(content, "-- - title:string") {
		t.Fatalf("missing title field comment:\n%s", content)
	}
	if !strings.Contains(content, "name: InsertPost") {
		t.Fatalf("missing insert query section:\n%s", content)
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
		HasFile:  testHasFile,
		QueryDir: "db/queries",
	})
	if code != 0 {
		t.Fatalf("exit code = %d, stderr=%s", code, errOut.String())
	}

	queryPath := filepath.Join(root, "db", "queries", "post.sql")
	b, err := os.ReadFile(queryPath)
	if err != nil {
		t.Fatalf("read query file: %v", err)
	}
	if !strings.Contains(string(b), "-- - title:string") {
		t.Fatalf("generated query scaffold missing field:\n%s", string(b))
	}
	if len(runner.calls) != 0 {
		t.Fatalf("runner calls len = %d, want 0", len(runner.calls))
	}
	for _, token := range []string{
		"make:model result",
		"Created:",
		"Next:",
	} {
		if !strings.Contains(out.String(), token) {
			t.Fatalf("stdout missing %q:\n%s", token, out.String())
		}
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
	queryPath := filepath.Join(root, "db", "queries", "post.sql")
	if err := os.MkdirAll(filepath.Dir(queryPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(queryPath, []byte("-- existing\n"), 0o644); err != nil {
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
		HasFile:  testHasFile,
		QueryDir: "db/queries",
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
