package generators

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateModelIntegration_WithFieldsWritesQueryScaffoldAndNextSteps(t *testing.T) {
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
		HasFile:  testHasFile,
		QueryDir: "db/queries",
	})
	if code != 0 {
		t.Fatalf("exit code = %d, stderr=%s", code, errOut.String())
	}

	queryPath := filepath.Join(root, "db", "queries", "blog_post.sql")
	content, err := os.ReadFile(queryPath)
	if err != nil {
		t.Fatalf("read query file: %v", err)
	}
	text := string(content)
	if !strings.Contains(text, "-- Model: BlogPost") {
		t.Fatalf("missing model declaration:\n%s", text)
	}
	if !strings.Contains(text, "-- - title:string") {
		t.Fatalf("missing title field comment:\n%s", text)
	}
	if !strings.Contains(text, "-- - published_at:time") {
		t.Fatalf("missing published_at field comment:\n%s", text)
	}
	if !strings.Contains(text, "-- - is_live:bool") {
		t.Fatalf("missing is_live field comment:\n%s", text)
	}

	if len(runner.calls) != 0 {
		t.Fatalf("runner call count = %d, want 0", len(runner.calls))
	}
	if !strings.Contains(out.String(), "Next:") || !strings.Contains(out.String(), "ship db:make create_blog_posts_table") {
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

	queryPath := filepath.Join(root, "db", "queries", "post.sql")
	if err := os.MkdirAll(filepath.Dir(queryPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(queryPath, []byte("-- old\n"), 0o644); err != nil {
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
		HasFile:  testHasFile,
		QueryDir: "db/queries",
	})
	if code != 0 {
		t.Fatalf("exit code = %d, stderr=%s", code, errOut.String())
	}

	content, err := os.ReadFile(queryPath)
	if err != nil {
		t.Fatalf("read query file: %v", err)
	}
	if strings.Contains(string(content), "// old") {
		t.Fatalf("query file was not overwritten:\n%s", string(content))
	}
}
