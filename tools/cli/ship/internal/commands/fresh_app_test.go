package commands

import (
	"context"
	"encoding/json"
	"net/http/cookiejar"
	"net/url"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestFreshApp(t *testing.T) {
	shipbin := buildShipBinary(t)
	appPath := scaffoldFreshAppViaShip(t, shipbin, false)

	runCmd(t, appPath, shipbin, "db:migrate")
	runGoTestAll(t, appPath)
	runCmd(t, appPath, shipbin, "doctor", "--json")
	runCmd(t, appPath, shipbin, "verify", "--profile", "fast")
}

func TestFreshAppStartupSmoke(t *testing.T) {
	shipbin := buildShipBinary(t)
	appPath := scaffoldFreshAppViaShip(t, shipbin, false)
	runCmd(t, appPath, shipbin, "db:migrate")

	port := reserveFreePort(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	webCmd := exec.CommandContext(ctx, "go", "run", "./cmd/web")
	webCmd.Dir = appPath
	webCmd.Env = append(os.Environ(), "PORT="+port)
	logPath := filepath.Join(t.TempDir(), "starter-web.log")
	logFile, err := os.Create(logPath)
	if err != nil {
		t.Fatalf("os.Create(log) error = %v", err)
	}
	defer logFile.Close()
	webCmd.Stdout = logFile
	webCmd.Stderr = logFile
	if err := webCmd.Start(); err != nil {
		t.Fatalf("webCmd.Start() error = %v", err)
	}
	defer func() {
		cancel()
		_ = webCmd.Wait()
	}()

	baseURL := "http://127.0.0.1:" + port
	waitForHTTP200(t, baseURL+"/")

	assertHTTPBodyContains(t, baseURL+"/", "Fresh App Ready")
	assertHTTPBodyEquals(t, baseURL+"/up", "alive")
	assertHTTPBodyEquals(t, baseURL+"/health", "alive")
	assertHTTPBodyEquals(t, baseURL+"/health/readiness", "ready")

	worker := exec.Command("go", "run", "./cmd/worker")
	worker.Dir = appPath
	workerOut, err := worker.CombinedOutput()
	if err != nil {
		t.Fatalf("worker boot failed: %v\n%s", err, workerOut)
	}
	if !strings.Contains(string(workerOut), "starter worker ready") {
		t.Fatalf("worker output missing readiness marker\n%s", workerOut)
	}
}

func TestFreshAppAPI(t *testing.T) {
	shipbin := buildShipBinary(t)
	appPath := scaffoldFreshAppViaShip(t, shipbin, true)

	runCmd(t, appPath, shipbin, "db:migrate")
	runGoTestAll(t, appPath)
	runCmd(t, appPath, shipbin, "doctor", "--json")
	runCmd(t, appPath, shipbin, "verify", "--profile", "fast")

	routesJSON := strings.TrimSpace(runCmd(t, appPath, shipbin, "routes", "--json"))
	if routesJSON == "[]" {
		t.Fatalf("ship routes --json returned empty inventory for API-only app")
	}
	var routes []map[string]any
	if err := json.Unmarshal([]byte(routesJSON), &routes); err != nil {
		t.Fatalf("json.Unmarshal(routes) error = %v\n%s", err, routesJSON)
	}
	if len(routes) == 0 {
		t.Fatal("expected non-empty API-only routes inventory")
	}
	if _, ok := routes[0]["operation_id"]; !ok {
		t.Fatalf("route metadata missing operation_id\n%s", routesJSON)
	}
}

func TestFreshAppAPIStartupSmoke(t *testing.T) {
	shipbin := buildShipBinary(t)
	appPath := scaffoldFreshAppViaShip(t, shipbin, true)
	runCmd(t, appPath, shipbin, "db:migrate")

	port := reserveFreePort(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	webCmd := exec.CommandContext(ctx, "go", "run", "./cmd/web")
	webCmd.Dir = appPath
	webCmd.Env = append(os.Environ(), "PORT="+port)
	logPath := filepath.Join(t.TempDir(), "starter-api-web.log")
	logFile, err := os.Create(logPath)
	if err != nil {
		t.Fatalf("os.Create(log) error = %v", err)
	}
	defer logFile.Close()
	webCmd.Stdout = logFile
	webCmd.Stderr = logFile
	if err := webCmd.Start(); err != nil {
		t.Fatalf("webCmd.Start() error = %v", err)
	}
	defer func() {
		cancel()
		_ = webCmd.Wait()
	}()

	baseURL := "http://127.0.0.1:" + port
	waitForHTTP200(t, baseURL+"/")
	assertHTTPJSONField(t, baseURL+"/", `"route":"landing_page"`)
	assertHTTPBodyEquals(t, baseURL+"/up", "alive")
	assertHTTPBodyEquals(t, baseURL+"/health", "alive")
	assertHTTPJSONField(t, baseURL+"/health/readiness", `"status":"ready"`)
	assertHTTPJSONField(t, baseURL+"/auth/login", `"route":"login"`)
}

func TestFreshAppNoInfraDefaultPath(t *testing.T) {
	shipbin := buildShipBinary(t)
	appPath := scaffoldFreshAppViaShip(t, shipbin, false)
	runCmd(t, appPath, shipbin, "db:migrate")

	port := reserveFreePort(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	webCmd := exec.CommandContext(ctx, "go", "run", "./cmd/web")
	webCmd.Dir = appPath
	webCmd.Env = append(os.Environ(), "PORT="+port)
	logPath := filepath.Join(t.TempDir(), "starter-noinfra-web.log")
	logFile, err := os.Create(logPath)
	if err != nil {
		t.Fatalf("os.Create(log) error = %v", err)
	}
	defer logFile.Close()
	webCmd.Stdout = logFile
	webCmd.Stderr = logFile
	if err := webCmd.Start(); err != nil {
		t.Fatalf("webCmd.Start() error = %v", err)
	}
	defer func() {
		cancel()
		if webCmd.Process != nil {
			_ = webCmd.Process.Kill()
		}
		_ = webCmd.Wait()
	}()

	baseURL := "http://127.0.0.1:" + port
	waitForHTTP200(t, baseURL+"/")
	assertHTTPBodyEquals(t, baseURL+"/up", "alive")
	assertHTTPBodyEquals(t, baseURL+"/health", "alive")
	assertHTTPBodyEquals(t, baseURL+"/health/readiness", "ready")

	worker := exec.Command("go", "run", "./cmd/worker")
	worker.Dir = appPath
	workerOut, err := worker.CombinedOutput()
	if err != nil {
		t.Fatalf("worker boot failed: %v\n%s", err, workerOut)
	}
	if !strings.Contains(string(workerOut), "starter worker ready") {
		t.Fatalf("worker output missing readiness marker\n%s", workerOut)
	}
}

func TestFreshAppShipTestCommands(t *testing.T) {
	shipbin := buildShipBinary(t)
	appPath := scaffoldFreshAppViaShip(t, shipbin, false)

	runCmd(t, appPath, shipbin, "test")
	runCmd(t, appPath, shipbin, "test", "--integration")
}

func TestFreshAppShipDevDefaultMode(t *testing.T) {
	shipbin := buildShipBinary(t)
	appPath := scaffoldFreshAppViaShip(t, shipbin, false)
	runCmd(t, appPath, shipbin, "db:migrate")

	port := reserveFreePort(t)
	setAppPort(t, appPath, port)
	devCmd, cancel := startShipCommand(t, appPath, shipbin, []string{"dev"})
	defer stopShipCommand(devCmd, cancel)

	baseURL := "http://127.0.0.1:" + port
	waitForHTTP200(t, baseURL+"/")
	assertHTTPBodyEquals(t, baseURL+"/up", "alive")
}

func TestFreshAppShipDevModes(t *testing.T) {
	shipbin := buildShipBinary(t)
	appPath := scaffoldFreshAppViaShip(t, shipbin, false)
	runCmd(t, appPath, shipbin, "db:migrate")

	runCmd(t, appPath, shipbin, "dev", "--worker")

	port := reserveFreePort(t)
	setAppPort(t, appPath, port)
	devCmd, cancel := startShipCommand(t, appPath, shipbin, []string{"dev", "--all"})
	defer stopShipCommand(devCmd, cancel)

	baseURL := "http://127.0.0.1:" + port
	waitForHTTP200(t, baseURL+"/")
	assertHTTPBodyEquals(t, baseURL+"/up", "alive")
}

func TestFreshAppProfileAndAdapterMutation(t *testing.T) {
	shipbin := buildShipBinary(t)
	appPath := scaffoldFreshAppViaShip(t, shipbin, false)

	runCmd(t, appPath, shipbin, "profile:set", "distributed")
	runCmd(t, appPath, shipbin, "adapter:set", "db=sqlite", "cache=otter", "jobs=backlite", "pubsub=inproc")

	envBody, err := os.ReadFile(filepath.Join(appPath, ".env"))
	if err != nil {
		t.Fatalf("os.ReadFile(.env) error = %v", err)
	}
	envText := string(envBody)
	for _, want := range []string{
		"PAGODA_RUNTIME_PROFILE=distributed",
		"PAGODA_PROCESSES_WEB=true",
		"PAGODA_PROCESSES_WORKER=true",
		"PAGODA_ADAPTERS_DB=sqlite",
		"PAGODA_ADAPTERS_CACHE=otter",
		"PAGODA_ADAPTERS_JOBS=backlite",
		"PAGODA_ADAPTERS_PUBSUB=inproc",
	} {
		if !strings.Contains(envText, want) {
			t.Fatalf(".env missing %q\n%s", want, envText)
		}
	}

	runCmd(t, appPath, shipbin, "verify", "--profile", "fast")
}

func TestFreshAppVerifyProfiles(t *testing.T) {
	shipbin := buildShipBinary(t)
	appPath := scaffoldFreshAppViaShip(t, shipbin, false)
	runCmd(t, appPath, shipbin, "db:migrate")

	runCmd(t, appPath, shipbin, "verify", "--profile", "fast")
	runCmd(t, appPath, shipbin, "verify", "--profile", "standard")
	runCmd(t, appPath, shipbin, "verify", "--profile", "strict")
}

func TestFreshAppAuthFlow(t *testing.T) {
	shipbin := buildShipBinary(t)
	appPath := scaffoldFreshAppViaShip(t, shipbin, false)
	runCmd(t, appPath, shipbin, "db:migrate")

	port := reserveFreePort(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	webCmd := exec.CommandContext(ctx, "go", "run", "./cmd/web")
	webCmd.Dir = appPath
	webCmd.Env = append(os.Environ(), "PORT="+port)
	logPath := filepath.Join(t.TempDir(), "starter-auth-web.log")
	logFile, err := os.Create(logPath)
	if err != nil {
		t.Fatalf("os.Create(log) error = %v", err)
	}
	defer logFile.Close()
	webCmd.Stdout = logFile
	webCmd.Stderr = logFile
	if err := webCmd.Start(); err != nil {
		t.Fatalf("webCmd.Start() error = %v", err)
	}
	defer func() {
		cancel()
		if webCmd.Process != nil {
			_ = webCmd.Process.Kill()
		}
		_ = webCmd.Wait()
	}()

	baseURL := "http://127.0.0.1:" + port
	waitForHTTP200(t, baseURL+"/")

	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("cookiejar.New() error = %v", err)
	}
	client := &http.Client{
		Jar: jar,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Timeout: 5 * time.Second,
	}

	form := url.Values{}
	form.Set("display_name", "Playwright User")
	form.Set("email", "starter@example.com")
	form.Set("password", "Password123!")
	form.Set("birthdate", "1990-01-01")
	form.Set("relationship_status", "single")

	resp, err := client.PostForm(baseURL+"/auth/register", form)
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("register status = %d, want 303", resp.StatusCode)
	}
	if got := resp.Header.Get("Location"); got != "/auth/profile" {
		t.Fatalf("register redirect = %q, want %q", got, "/auth/profile")
	}

	resp, err = client.Get(baseURL + "/auth/logout")
	if err != nil {
		t.Fatalf("logout failed: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("logout status = %d, want 303", resp.StatusCode)
	}
	if got := resp.Header.Get("Location"); got != "/auth/login" {
		t.Fatalf("logout redirect = %q, want %q", got, "/auth/login")
	}

	resp, err = client.Get(baseURL + "/auth/profile")
	if err != nil {
		t.Fatalf("protected route failed: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("protected route status = %d, want 303", resp.StatusCode)
	}
	if got := resp.Header.Get("Location"); got != "/auth/login?next=%2Fauth%2Fprofile" {
		t.Fatalf("protected route redirect = %q", got)
	}

	loginForm := url.Values{}
	loginForm.Set("email", "starter@example.com")
	loginForm.Set("password", "Password123!")
	loginForm.Set("next", "/auth/profile")
	resp, err = client.PostForm(baseURL+"/auth/login", loginForm)
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("login status = %d, want 303", resp.StatusCode)
	}
	if got := resp.Header.Get("Location"); got != "/auth/profile" {
		t.Fatalf("login redirect = %q, want %q", got, "/auth/profile")
	}
}

func buildShipBinary(t *testing.T) string {
	t.Helper()

	bin := filepath.Join(t.TempDir(), "ship")
	cmd := exec.Command("go", "build", "-o", bin, "./tools/cli/ship/cmd/ship")
	cmd.Dir = repoRootFromCommandsTest(t)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go build ship failed: %v\n%s", err, out)
	}
	return bin
}

func scaffoldFreshAppViaShip(t *testing.T, shipbin string, apiOnly bool) string {
	t.Helper()

	root := t.TempDir()
	args := []string{"new", "demo", "--module", "example.com/demo", "--no-i18n"}
	if apiOnly {
		args = append(args, "--api")
	}
	cmd := exec.Command(shipbin, args...)
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("ship new failed: %v\n%s", err, out)
	}
	return filepath.Join(root, "demo")
}

func runCmd(t *testing.T, dir, bin string, args ...string) string {
	t.Helper()

	cmd := exec.Command(bin, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s failed: %v\n%s", strings.Join(args, " "), err, out)
	}
	return string(out)
}

func runGoTestAll(t *testing.T, dir string) {
	t.Helper()
	cmd := exec.Command("go", "test", "./...")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go test ./... failed: %v\n%s", err, out)
	}
}

func waitForHTTP200(t *testing.T, url string) {
	t.Helper()
	client := &http.Client{Timeout: 500 * time.Millisecond}
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := client.Get(url)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return
			}
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", url)
}

func assertHTTPBodyContains(t *testing.T, url, want string) {
	t.Helper()
	body := getHTTPBody(t, url)
	if !strings.Contains(body, want) {
		t.Fatalf("%s body missing %q\nbody:\n%s", url, want, body)
	}
}

func assertHTTPBodyEquals(t *testing.T, url, want string) {
	t.Helper()
	body := strings.TrimSpace(getHTTPBody(t, url))
	if body != want {
		t.Fatalf("%s body = %q, want %q", url, body, want)
	}
}

func assertHTTPJSONField(t *testing.T, url, want string) {
	t.Helper()
	body := strings.TrimSpace(getHTTPBody(t, url))
	if !strings.Contains(body, want) {
		t.Fatalf("%s body missing %q\nbody:\n%s", url, want, body)
	}
}

func getHTTPBody(t *testing.T, url string) string {
	t.Helper()
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		t.Fatalf("GET %s failed: %v", url, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("reading %s failed: %v", url, err)
	}
	return string(body)
}

func reserveFreePort(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen() error = %v", err)
	}
	defer l.Close()
	_, port, err := net.SplitHostPort(l.Addr().String())
	if err != nil {
		t.Fatalf("SplitHostPort() error = %v", err)
	}
	return port
}

func repoRootFromCommandsTest(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd() error = %v", err)
	}
	return filepath.Clean(filepath.Join(wd, "..", "..", "..", "..", ".."))
}

func setAppPort(t *testing.T, appPath, port string) {
	t.Helper()
	envPath := filepath.Join(appPath, ".env")
	body, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatalf("os.ReadFile(.env) error = %v", err)
	}
	updated := strings.ReplaceAll(string(body), "PORT=3000", "PORT="+port)
	if err := os.WriteFile(envPath, []byte(updated), 0o644); err != nil {
		t.Fatalf("os.WriteFile(.env) error = %v", err)
	}
}

func startShipCommand(t *testing.T, appPath, shipbin string, args []string, extraEnv ...string) (*exec.Cmd, context.CancelFunc) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, shipbin, args...)
	cmd.Dir = appPath
	cmd.Env = append(os.Environ(), extraEnv...)
	logPath := filepath.Join(t.TempDir(), "ship-command.log")
	logFile, err := os.Create(logPath)
	if err != nil {
		t.Fatalf("os.Create(log) error = %v", err)
	}
	t.Cleanup(func() { _ = logFile.Close() })
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	if err := cmd.Start(); err != nil {
		t.Fatalf("cmd.Start() error = %v", err)
	}
	return cmd, cancel
}

func stopShipCommand(cmd *exec.Cmd, cancel context.CancelFunc) {
	cancel()
	if cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
	_ = cmd.Wait()
}
