package ship

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseMakeScaffoldArgs(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr string
		check   func(t *testing.T, opts scaffoldMakeOptions)
	}{
		{
			name: "full flags",
			args: []string{"Post", "title:string", "--api", "--migrate", "--dry-run", "--force", "--views=none", "--auth=auth", "--path=app/goship"},
			check: func(t *testing.T, opts scaffoldMakeOptions) {
				if opts.ModelName != "Post" || len(opts.Fields) != 1 {
					t.Fatalf("unexpected parsed scaffold opts: %+v", opts)
				}
				if !opts.API || !opts.Migrate || !opts.DryRun || !opts.Force {
					t.Fatalf("missing expected booleans in %+v", opts)
				}
			},
		},
		{
			name:    "invalid model name",
			args:    []string{"post"},
			wantErr: "invalid model name",
		},
		{
			name:    "invalid auth",
			args:    []string{"Post", "--auth=private"},
			wantErr: "invalid --auth value",
		},
		{
			name:    "invalid views",
			args:    []string{"Post", "--views=react"},
			wantErr: "invalid --views value",
		},
		{
			name:    "unknown option",
			args:    []string{"Post", "--wat"},
			wantErr: "unknown option",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts, err := parseMakeScaffoldArgs(tt.args)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error = %v, want contains %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseMakeScaffoldArgs error = %v", err)
			}
			if tt.check != nil {
				tt.check(t, opts)
			}
		})
	}
}

func TestRunMakeScaffold_DryRun(t *testing.T) {
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	cli := CLI{Out: out, Err: errOut, Runner: &fakeRunner{}}
	code := cli.Run([]string{"make:scaffold", "Post", "title:string", "--dry-run"})
	if code != 0 {
		t.Fatalf("exit code = %d, stderr=%s", code, errOut.String())
	}
	if !strings.Contains(out.String(), "Scaffold plan (dry-run):") {
		t.Fatalf("missing dry-run plan output:\n%s", out.String())
	}
	if !strings.Contains(out.String(), "ship make:model Post") {
		t.Fatalf("missing model step output:\n%s", out.String())
	}
}

func TestRunMakeScaffold_DryRunAPIOmitsResourceStep(t *testing.T) {
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	cli := CLI{Out: out, Err: errOut, Runner: &fakeRunner{}}
	code := cli.Run([]string{"make:scaffold", "Post", "title:string", "--dry-run", "--api"})
	if code != 0 {
		t.Fatalf("exit code = %d, stderr=%s", code, errOut.String())
	}
	if strings.Contains(out.String(), "make:resource") {
		t.Fatalf("resource step should be omitted in API mode:\n%s", out.String())
	}
}

func TestRunMakeScaffold_Integration(t *testing.T) {
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
	cli := CLI{
		Out:    out,
		Err:    errOut,
		Runner: &fakeRunner{},
		ResolveDBURL: func() (string, error) {
			return testDBURL, nil
		},
	}

	seedScaffoldTargets(t, root)

	code := cli.Run([]string{"make:scaffold", "Post", "title:string"})
	if code != 0 {
		t.Fatalf("exit code = %d, stderr=%s", code, errOut.String())
	}

	if !hasFile(filepath.Join(root, "app", "goship", "ent", "schema", "post.go")) {
		t.Fatalf("missing scaffolded model schema")
	}
	if !hasFile(filepath.Join(root, "app", "goship", "web", "routes", "posts.go")) {
		t.Fatalf("missing scaffolded controller file")
	}
	if !hasFile(filepath.Join(root, "app", "goship", "web", "routes", "post.go")) {
		t.Fatalf("missing scaffolded resource route file")
	}
	if !hasFile(filepath.Join(root, "app", "goship", "views", "web", "pages", "post.templ")) {
		t.Fatalf("missing scaffolded resource view")
	}
}

func TestRunMakeScaffold_IntegrationAPI_NoResourceArtifacts(t *testing.T) {
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
	cli := CLI{Out: out, Err: errOut, Runner: &fakeRunner{}}
	seedScaffoldTargets(t, root)

	code := cli.Run([]string{"make:scaffold", "Post", "title:string", "--api"})
	if code != 0 {
		t.Fatalf("exit code = %d, stderr=%s", code, errOut.String())
	}
	if hasFile(filepath.Join(root, "app", "goship", "web", "routes", "post.go")) {
		t.Fatalf("resource route file should not exist in --api mode")
	}
	if hasFile(filepath.Join(root, "app", "goship", "views", "web", "pages", "post.templ")) {
		t.Fatalf("resource view should not exist in --api mode")
	}
}

func TestRunMakeScaffold_IntegrationMigrate_CallsAtlasApply(t *testing.T) {
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
	cli := CLI{
		Out:    out,
		Err:    errOut,
		Runner: runner,
		ResolveDBURL: func() (string, error) {
			return testDBURL, nil
		},
	}
	seedScaffoldTargets(t, root)

	code := cli.Run([]string{"make:scaffold", "Post", "title:string", "--migrate"})
	if code != 0 {
		t.Fatalf("exit code = %d, stderr=%s", code, errOut.String())
	}

	var foundApply bool
	for _, call := range runner.calls {
		if call.name == "atlas" && strings.Join(call.args, " ") == "migrate apply --dir "+atlasDir+" --url "+testDBURL {
			foundApply = true
			break
		}
	}
	if !foundApply {
		t.Fatalf("expected atlas migrate apply call, calls=%v", runner.calls)
	}
}

func TestRunMakeScaffold_MigrateMissingDBURLFails(t *testing.T) {
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
	cli := CLI{
		Out:    out,
		Err:    errOut,
		Runner: &fakeRunner{},
		ResolveDBURL: func() (string, error) {
			return "", errors.New("missing db url")
		},
	}
	seedScaffoldTargets(t, root)

	code := cli.Run([]string{"make:scaffold", "Post", "title:string", "--migrate"})
	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if !strings.Contains(errOut.String(), "failed to resolve database URL") {
		t.Fatalf("stderr = %q, want db url failure", errOut.String())
	}
}

func seedScaffoldTargets(t *testing.T, root string) {
	t.Helper()
	routerPath := filepath.Join(root, "app", "goship", "router.go")
	if err := os.MkdirAll(filepath.Dir(routerPath), 0o755); err != nil {
		t.Fatal(err)
	}
	routerContent := `package goship

import (
	routeNames "github.com/leomorpho/goship/app/goship/web/routenames"
	"github.com/leomorpho/goship/app/goship/web/routes"
)

func registerPublicRoutes() {
	// ship:routes:public:start
	// ship:routes:public:end
}

func registerAuthRoutes() {
	// ship:routes:auth:start
	// ship:routes:auth:end
}
`
	if err := os.WriteFile(routerPath, []byte(routerContent), 0o644); err != nil {
		t.Fatal(err)
	}
	routeNamesPath := filepath.Join(root, "pkg", "routing", "routenames", "routenames.go")
	if err := os.MkdirAll(filepath.Dir(routeNamesPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(routeNamesPath, []byte("package routenames\n\nconst (\n)\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}
