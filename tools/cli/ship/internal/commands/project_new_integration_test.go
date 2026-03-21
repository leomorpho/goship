package commands

import (
	"bytes"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	gen "github.com/leomorpho/goship/tools/cli/ship/internal/generators"
	policies "github.com/leomorpho/goship/tools/cli/ship/internal/policies"
)

func TestNewProjectIntegration_SupportsMakeModelQueryScaffold(t *testing.T) {
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
	if !strings.Contains(out.String(), "Next: cd demo && ship module:add <module> && make run") {
		t.Fatalf("stdout = %q, want post-install hint", out.String())
	}

	projectRoot := filepath.Join(root, "demo")
	gotLayout, err := snapshotGeneratedProjectLayout(projectRoot)
	if err != nil {
		t.Fatalf("snapshotGeneratedProjectLayout: %v", err)
	}
	wantLayout := canonicalGeneratedProjectLayoutSnapshot(NewProjectOptions{
		Name:    "demo",
		Module:  "example.com/demo",
		AppPath: projectRoot,
	}, defaultNewLayoutArtifactPaths())
	if !slices.Equal(gotLayout, wantLayout) {
		t.Fatalf("fresh scaffold layout mismatch\nwant:\n%s\ngot:\n%s", strings.Join(wantLayout, "\n"), strings.Join(gotLayout, "\n"))
	}

	entMigrationsKeep := filepath.Join(projectRoot, "db", "migrate", "migrations", ".gitkeep")
	if _, err := os.Stat(entMigrationsKeep); err != nil {
		t.Fatalf("expected migrations scaffold at %s: %v", entMigrationsKeep, err)
	}
	bobgenConfig := filepath.Join(projectRoot, "db", "bobgen.yaml")
	if _, err := os.Stat(bobgenConfig); err != nil {
		t.Fatalf("expected bobgen config scaffold at %s: %v", bobgenConfig, err)
	}
	routerBytes, err := os.ReadFile(filepath.Join(projectRoot, "app", "router.go"))
	if err != nil {
		t.Fatalf("read generated router: %v", err)
	}
	if !strings.Contains(string(routerBytes), "RouteNameHomeFeed") {
		t.Fatalf("expected generated router copied from starter:\n%s", string(routerBytes))
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
	if err := checkStandaloneExportability(projectRoot); err != nil {
		t.Fatalf("fresh scaffold should remain free of control-plane dependency drift: %v", err)
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
		HasFile:  testHasFile,
		QueryDir: "db/queries",
	}); code != 0 {
		t.Fatalf("ship make:model failed: code=%d stderr=%s", code, errOut.String())
	}

	generatedQuery := filepath.Join(projectRoot, "db", "queries", "post.sql")
	b, err := os.ReadFile(generatedQuery)
	if err != nil {
		t.Fatalf("expected generated model query at %s: %v", generatedQuery, err)
	}
	if !strings.Contains(string(b), "-- - title:string") {
		t.Fatalf("generated query scaffold missing expected field:\n%s", string(b))
	}

	if len(runner.calls) != 0 {
		t.Fatalf("runner call count = %d, want 0", len(runner.calls))
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
