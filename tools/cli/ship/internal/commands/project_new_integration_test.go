package commands

import (
	"bytes"
	"context"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"

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

func TestNewProjectIntegration_FreshAppMigratesBootsRendersAndVerifies(t *testing.T) {
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
	shipBin := buildShipBinaryForProjectNew(t)
	toolBin := scaffoldFreshAppTooling(t)
	env := append(os.Environ(), "PATH="+toolBin+string(os.PathListSeparator)+os.Getenv("PATH"))

	if output, err := runCommand(projectRoot, env, shipBin, "db:migrate"); err != nil {
		t.Fatalf("ship db:migrate failed: %v\n%s", err, output)
	}
	if _, err := os.Stat(filepath.Join(projectRoot, "tmp", "starter.db")); err != nil {
		t.Fatalf("expected migrated sqlite database: %v", err)
	}

	if output, err := runCommand(projectRoot, env, shipBin, "verify", "--profile", "fast"); err != nil {
		t.Fatalf("ship verify --profile fast failed: %v\n%s", err, output)
	}

	port := reservePort(t)
	webBin := filepath.Join(t.TempDir(), "starter-web")
	if output, err := runCommand(projectRoot, env, "go", "build", "-o", webBin, "./cmd/web"); err != nil {
		t.Fatalf("build starter web binary failed: %v\n%s", err, output)
	}
	serverCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	serverCmd := exec.CommandContext(serverCtx, webBin)
	serverCmd.Dir = projectRoot
	serverCmd.Env = append(env, "PORT="+port)
	serverLog := &bytes.Buffer{}
	serverCmd.Stdout = serverLog
	serverCmd.Stderr = serverLog
	if err := serverCmd.Start(); err != nil {
		t.Fatalf("start cmd/web: %v", err)
	}
	t.Cleanup(func() {
		cancel()
		_ = serverCmd.Wait()
	})

	baseURL := "http://127.0.0.1:" + port
	waitForStarterServer(t, baseURL+"/health/readiness", serverLog)
	assertStarterRouteContains(t, baseURL+"/", "Demo")
	assertStarterRouteContains(t, baseURL+"/auth/homeFeed", "Home Feed")
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

func buildShipBinaryForProjectNew(t *testing.T) string {
	t.Helper()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test file path")
	}
	moduleRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))

	binDir := t.TempDir()
	binPath := filepath.Join(binDir, "ship")
	cmd := exec.Command("go", "build", "-o", binPath, "./cmd/ship")
	cmd.Dir = moduleRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build ship binary: %v\n%s", err, output)
	}
	return binPath
}

func scaffoldFreshAppTooling(t *testing.T) string {
	t.Helper()

	toolDir := t.TempDir()
	writeExecutable(t, filepath.Join(toolDir, "templ"), "#!/bin/sh\nexit 0\n")
	writeExecutable(t, filepath.Join(toolDir, "goose"), `#!/bin/sh
set -eu

dir=""
driver=""
conn=""
command=""

while [ "$#" -gt 0 ]; do
  case "$1" in
    -dir)
      dir="$2"
      shift 2
      ;;
    *)
      if [ -z "$driver" ]; then
        driver="$1"
      elif [ -z "$conn" ]; then
        conn="$1"
      elif [ -z "$command" ]; then
        command="$1"
      fi
      shift
      ;;
  esac
done

if [ "$driver" != "sqlite3" ] || [ "$command" != "up" ]; then
  echo "fake goose only supports sqlite3 up" >&2
  exit 1
fi

mkdir -p "$(dirname "$conn")"
touch "$conn"
printf 'goose up %s %s\n' "$dir" "$conn"
`)
	return toolDir
}

func writeExecutable(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatalf("write executable %s: %v", path, err)
	}
}

func runCommand(dir string, env []string, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Env = env
	output, err := cmd.CombinedOutput()
	return string(output), err
}

func reservePort(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve port: %v", err)
	}
	defer ln.Close()
	_, port, err := net.SplitHostPort(ln.Addr().String())
	if err != nil {
		t.Fatalf("split host/port: %v", err)
	}
	return port
}

func waitForStarterServer(t *testing.T, url string, serverLog *bytes.Buffer) {
	t.Helper()
	client := &http.Client{Timeout: 2 * time.Second}
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := client.Get(url)
		if err == nil {
			body, readErr := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			if readErr == nil && resp.StatusCode == http.StatusOK && strings.Contains(string(body), "ready") {
				return
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("starter server did not become ready\n%s", serverLog.String())
}

func assertStarterRouteContains(t *testing.T, url string, want string) {
	t.Helper()
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body for %s: %v", url, err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET %s status=%d body=%s", url, resp.StatusCode, string(body))
	}
	if !strings.Contains(string(body), want) {
		t.Fatalf("GET %s body missing %q:\n%s", url, want, string(body))
	}
}
