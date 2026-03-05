package commands

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	gen "github.com/leomorpho/goship/tools/cli/ship/internal/generators"
	policies "github.com/leomorpho/goship/tools/cli/ship/internal/policies"
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
	if code := RunNew([]string{"demo", "--module", "example.com/demo"}, NewDeps{
		Out:                        out,
		Err:                        errOut,
		ParseAgentPolicyBytes:      policies.ParsePolicyBytes,
		RenderAgentPolicyArtifacts: policies.RenderPolicyArtifacts,
		AgentPolicyFilePath:        policies.AgentPolicyFilePath,
	}); code != 0 {
		t.Fatalf("ship new failed: code=%d stderr=%s", code, errOut.String())
	}

	projectRoot := filepath.Join(root, "demo")
	entSchemaUser := filepath.Join(projectRoot, "db", "schema", "user.go")
	if _, err := os.Stat(entSchemaUser); err != nil {
		t.Fatalf("expected ent schema scaffold at %s: %v", entSchemaUser, err)
	}
	entMigrationsKeep := filepath.Join(projectRoot, "db", "migrate", "migrations", ".gitkeep")
	if _, err := os.Stat(entMigrationsKeep); err != nil {
		t.Fatalf("expected migrations scaffold at %s: %v", entMigrationsKeep, err)
	}
	bobgenConfig := filepath.Join(projectRoot, "db", "bobgen.yaml")
	if _, err := os.Stat(bobgenConfig); err != nil {
		t.Fatalf("expected bobgen config scaffold at %s: %v", bobgenConfig, err)
	}

	if err := os.Chdir(projectRoot); err != nil {
		t.Fatal(err)
	}

	out.Reset()
	errOut.Reset()
	if code := policies.RunDoctor([]string{}, policies.DoctorDeps{
		Out:          out,
		Err:          errOut,
		FindGoModule: findGoModuleTestProjectNew,
	}); code != 0 {
		t.Fatalf("ship doctor failed on fresh scaffold: code=%d stderr=%s", code, errOut.String())
	}

	out.Reset()
	errOut.Reset()
	runner := &fakeRunner{}
	if code := gen.RunGenerateModel([]string{"Post", "title:string"}, gen.GenerateModelDeps{
		Out: out,
		Err: errOut,
		RunCmd: func(name string, args ...string) int {
			return runner.RunCode(name, args...)
		},
		HasFile:      testHasFile,
		EntSchemaDir: "db/schema",
	}); code != 0 {
		t.Fatalf("ship make:model failed: code=%d stderr=%s", code, errOut.String())
	}

	generatedSchema := filepath.Join(projectRoot, "db", "schema", "post.go")
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
	if !strings.Contains(got, "--target ./db/ent ./db/schema") {
		t.Fatalf("unexpected generate args: %v", runner.calls[0].args)
	}
}

type fakeCall struct {
	name string
	args []string
}

type fakeRunner struct {
	calls []fakeCall
	code  int
}

func (f *fakeRunner) RunCode(name string, args ...string) int {
	f.calls = append(f.calls, fakeCall{name: name, args: args})
	return f.code
}

func testHasFile(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func findGoModuleTestProjectNew(start string) (string, string, error) {
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
