package generators

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateModelIntegration_WithFieldsWritesSchemaAndNextSteps(t *testing.T) {
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
	code := RunGenerateModel([]string{"BlogPost", "title:string", "published_at:time", "is_live:bool"}, GenerateModelDeps{
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

	schemaPath := filepath.Join(root, "db", "schema", "blog_post.go")
	content, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}
	text := string(content)
	if !strings.Contains(text, "type BlogPost struct") {
		t.Fatalf("missing type declaration:\n%s", text)
	}
	if !strings.Contains(text, `field.String("title")`) {
		t.Fatalf("missing title field:\n%s", text)
	}
	if !strings.Contains(text, `field.Time("published_at")`) {
		t.Fatalf("missing published_at field:\n%s", text)
	}
	if !strings.Contains(text, `field.Bool("is_live")`) {
		t.Fatalf("missing is_live field:\n%s", text)
	}

	if len(runner.calls) != 1 {
		t.Fatalf("runner call count = %d, want 1", len(runner.calls))
	}
	if runner.calls[0].name != "go" || !strings.Contains(strings.Join(runner.calls[0].args, " "), "ent generate") {
		t.Fatalf("unexpected runner call: %s %v", runner.calls[0].name, runner.calls[0].args)
	}
	if !strings.Contains(out.String(), "Next:") || !strings.Contains(out.String(), "ship db:make add_blog_posts") {
		t.Fatalf("stdout missing next-step guidance:\n%s", out.String())
	}
}

func TestGenerateModelIntegration_ForceOverwritesSchema(t *testing.T) {
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
	if err := os.WriteFile(schemaPath, []byte("package schema\n\n// old\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	runner := &fakeRunner{}
	code := RunGenerateModel([]string{"Post", "title:string", "--force"}, GenerateModelDeps{
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

	content, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}
	if strings.Contains(string(content), "// old") {
		t.Fatalf("schema file was not overwritten:\n%s", string(content))
	}
}
