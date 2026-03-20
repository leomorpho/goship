package generators

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	scaffoldGooseDir = "db/migrate/migrations"
)

func TestParseMakeScaffoldArgs(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr string
		check   func(t *testing.T, opts ScaffoldMakeOptions)
	}{
		{
			name: "full flags",
			args: []string{"Post", "title:string", "--api", "--migrate", "--dry-run", "--force", "--views=none", "--auth=auth", "--path=app"},
			check: func(t *testing.T, opts ScaffoldMakeOptions) {
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
			opts, err := ParseMakeScaffoldArgs(tt.args)
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
	runner := &fakeRunner{}
	code := RunMakeScaffold([]string{"Post", "title:string", "--dry-run"}, makeScaffoldDeps(out, errOut, runner))
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
	runner := &fakeRunner{}
	code := RunMakeScaffold([]string{"Post", "title:string", "--dry-run", "--api"}, makeScaffoldDeps(out, errOut, runner))
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
	runner := &fakeRunner{}

	seedScaffoldTargets(t, root)

	code := RunMakeScaffold([]string{"Post", "title:string"}, makeScaffoldDeps(out, errOut, runner))
	if code != 0 {
		t.Fatalf("exit code = %d, stderr=%s", code, errOut.String())
	}

	if !testHasFile(filepath.Join(root, "db", "queries", "post.sql")) {
		t.Fatalf("missing scaffolded model query file")
	}
	if !testHasFile(filepath.Join(root, "app", "web", "controllers", "posts.go")) {
		t.Fatalf("missing scaffolded controller file")
	}
	if !testHasFile(filepath.Join(root, "app", "web", "controllers", "post.go")) {
		t.Fatalf("missing scaffolded resource route file")
	}
	if !testHasFile(filepath.Join(root, "app", "views", "web", "pages", "post.templ")) {
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
	runner := &fakeRunner{}
	seedScaffoldTargets(t, root)

	code := RunMakeScaffold([]string{"Post", "title:string", "--api"}, makeScaffoldDeps(out, errOut, runner))
	if code != 0 {
		t.Fatalf("exit code = %d, stderr=%s", code, errOut.String())
	}
	if testHasFile(filepath.Join(root, "app", "web", "controllers", "post.go")) {
		t.Fatalf("resource route file should not exist in --api mode")
	}
	if testHasFile(filepath.Join(root, "app", "views", "web", "pages", "post.templ")) {
		t.Fatalf("resource view should not exist in --api mode")
	}
}

func TestRunMakeScaffold_IntegrationMigrate_CallsDBMigrate(t *testing.T) {
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
	seedScaffoldTargets(t, root)

	code := RunMakeScaffold([]string{"Post", "title:string", "--migrate"}, makeScaffoldDeps(out, errOut, runner))
	if code != 0 {
		t.Fatalf("exit code = %d, stderr=%s", code, errOut.String())
	}

	var foundMigrate bool
	for _, call := range runner.calls {
		if call.name == "shipdb" && strings.Join(call.args, " ") == "migrate" {
			foundMigrate = true
			break
		}
	}
	if !foundMigrate {
		t.Fatalf("expected db migrate call, calls=%v", runner.calls)
	}
}

func TestRunMakeScaffold_IntegrationAuthUsesOnboardedGroupAndStableOutput(t *testing.T) {
	run := func(t *testing.T) (string, string, string) {
		t.Helper()

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

		seedScaffoldTargets(t, root)

		code := RunMakeScaffold([]string{"Post", "title:string", "--auth=auth"}, makeScaffoldDeps(out, errOut, runner))
		if code != 0 {
			t.Fatalf("exit code = %d, stderr=%s", code, errOut.String())
		}

		routerBytes, err := os.ReadFile(filepath.Join(root, "app", "router.go"))
		if err != nil {
			t.Fatalf("read router: %v", err)
		}
		routeNamesBytes, err := os.ReadFile(filepath.Join(root, "app", "web", "routenames", "routenames.go"))
		if err != nil {
			t.Fatalf("read routenames: %v", err)
		}

		return out.String(), string(routerBytes), string(routeNamesBytes)
	}

	out1, router1, routeNames1 := run(t)
	out2, router2, routeNames2 := run(t)

	if out1 != out2 {
		t.Fatalf("scaffold report changed across identical auth runs\n--- run 1 ---\n%s\n--- run 2 ---\n%s", out1, out2)
	}
	if router1 != router2 {
		t.Fatalf("router changed across identical auth runs\n--- run 1 ---\n%s\n--- run 2 ---\n%s", router1, router2)
	}
	if routeNames1 != routeNames2 {
		t.Fatalf("route names changed across identical auth runs\n--- run 1 ---\n%s\n--- run 2 ---\n%s", routeNames1, routeNames2)
	}

	for _, want := range []string{
		"ship:routes:auth:start",
		"ship:generated:posts",
		"posts := controllers.NewPostsController(nil)",
		`onboardedGroup.GET("/posts", posts.Index)`,
		`onboardedGroup.GET("/posts/:id", posts.Show)`,
		`onboardedGroup.POST("/posts", posts.Create)`,
		`onboardedGroup.PUT("/posts/:id", posts.Update)`,
		`onboardedGroup.DELETE("/posts/:id", posts.Destroy)`,
		"ship:generated:post",
		"post := controllers.NewPostRoute(ctr, nil)",
		`onboardedGroup.GET("/post", post.Get).Name = routeNames.RouteNamePost`,
	} {
		if !strings.Contains(router1, want) {
			t.Fatalf("router missing %q:\n%s", want, router1)
		}
	}
	if strings.Count(routeNames1, `RouteNamePost = "post"`) != 1 {
		t.Fatalf("route names should contain exactly one post constant:\n%s", routeNames1)
	}
}

func makeScaffoldDeps(out, errOut *bytes.Buffer, runner *fakeRunner) ScaffoldDeps {
	return ScaffoldDeps{
		Out: out,
		Err: errOut,
		RunModel: func(args []string) int {
			return RunGenerateModel(args, GenerateModelDeps{
				Out: out,
				Err: errOut,
				RunCmd: func(name string, args ...string) int {
					return runner.RunCode(name, args...)
				},
				HasFile:  testHasFile,
				QueryDir: "db/queries",
			})
		},
		RunDBMake: func(args []string) int {
			if len(args) != 1 {
				return 1
			}
			return runner.RunCode("goose", "-dir", scaffoldGooseDir, "create", args[0], "sql")
		},
		RunDBMigrate: func(args []string) int {
			return runner.RunCode("shipdb", append([]string{"migrate"}, args...)...)
		},
		RunController: func(args []string) int {
			return RunMakeController(args, ControllerDeps{
				Out:                    out,
				Err:                    errOut,
				HasFile:                testHasFile,
				EnsureRouteNamesImport: EnsureRouteNamesImport,
				WireRouteSnippet:       WireRouteSnippet,
			})
		},
		RunResource: func(args []string) int {
			return RunGenerateResource(args, out, errOut)
		},
	}
}

func seedScaffoldTargets(t *testing.T, root string) {
	t.Helper()
	routerPath := filepath.Join(root, "app", "router.go")
	if err := os.MkdirAll(filepath.Dir(routerPath), 0o755); err != nil {
		t.Fatal(err)
	}
	routerContent := `package goship

import (
	routeNames "github.com/leomorpho/goship/app/web/routenames"
	"github.com/leomorpho/goship/app/web/controllers"
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
	routeNamesPath := filepath.Join(root, "app", "web", "routenames", "routenames.go")
	if err := os.MkdirAll(filepath.Dir(routeNamesPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(routeNamesPath, []byte("package routenames\n\nconst (\n)\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}
