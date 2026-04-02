package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

var starterResetTokenPattern = regexp.MustCompile(`data-reset-token>([^<]+)<`)

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

	baseURL, client, cleanup := startFreshAppWebWithClient(t, appPath)
	defer cleanup()

	resp := registerStarterUser(t, client, baseURL, "starter@example.com", "Password123!")
	_ = resp.Body.Close()
	if got := resp.Header.Get("Location"); got != "/auth/profile" {
		t.Fatalf("register redirect = %q, want %q", got, "/auth/profile")
	}

	var err error
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

func TestFreshAppAuthAccountLifecycle(t *testing.T) {
	shipbin := buildShipBinary(t)
	appPath := scaffoldFreshAppViaShip(t, shipbin, false)
	runCmd(t, appPath, shipbin, "db:migrate")

	baseURL, client, cleanup := startFreshAppWebWithClient(t, appPath)
	defer cleanup()

	resp := registerStarterUser(t, client, baseURL, "starter@example.com", "Password123!")
	_ = resp.Body.Close()

	assertHTTPStatusContainsForClient(t, client, baseURL+"/auth/session", http.StatusOK, "starter@example.com")
	assertHTTPStatusContainsForClient(t, client, baseURL+"/auth/settings", http.StatusOK, "Account settings")

	settingsForm := url.Values{}
	settingsForm.Set("display_name", "Updated Starter User")
	resp, err := client.PostForm(baseURL+"/auth/settings", settingsForm)
	if err != nil {
		t.Fatalf("settings update failed: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("settings update status = %d, want 303", resp.StatusCode)
	}
	if got := resp.Header.Get("Location"); got != "/auth/settings" {
		t.Fatalf("settings update redirect = %q, want %q", got, "/auth/settings")
	}

	assertHTTPStatusContainsForClient(t, client, baseURL+"/auth/settings", http.StatusOK, "Updated Starter User")
}

func TestFreshAppPasswordResetFlow(t *testing.T) {
	shipbin := buildShipBinary(t)
	appPath := scaffoldFreshAppViaShip(t, shipbin, false)
	runCmd(t, appPath, shipbin, "db:migrate")

	baseURL, client, cleanup := startFreshAppWebWithClient(t, appPath)
	defer cleanup()

	resp := registerStarterUser(t, client, baseURL, "starter@example.com", "Password123!")
	_ = resp.Body.Close()
	logoutStarterUser(t, client, baseURL)

	resetRequest := url.Values{}
	resetRequest.Set("email", "starter@example.com")
	resp, err := client.PostForm(baseURL+"/auth/password/reset", resetRequest)
	if err != nil {
		t.Fatalf("password reset request failed: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("password reset request status = %d, want 303", resp.StatusCode)
	}

	body := getHTTPBodyForClient(t, client, baseURL+"/auth/password/reset/confirm?email=starter@example.com")
	resetToken := extractResetToken(t, body)
	if resetToken == "" {
		t.Fatal("expected deterministic starter reset token payload")
	}

	resetConfirm := url.Values{}
	resetConfirm.Set("email", "starter@example.com")
	resetConfirm.Set("token", resetToken)
	resetConfirm.Set("password", "NewPassword123!")
	resp, err = client.PostForm(baseURL+"/auth/password/reset/confirm", resetConfirm)
	if err != nil {
		t.Fatalf("password reset confirm failed: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("password reset confirm status = %d, want 303", resp.StatusCode)
	}

	assertLoginFails(t, client, baseURL, "starter@example.com", "Password123!")
	assertLoginSucceeds(t, client, baseURL, "starter@example.com", "NewPassword123!", "/auth/profile")
}

func TestFreshAppDeleteAccountFlow(t *testing.T) {
	shipbin := buildShipBinary(t)
	appPath := scaffoldFreshAppViaShip(t, shipbin, false)
	runCmd(t, appPath, shipbin, "db:migrate")

	baseURL, client, cleanup := startFreshAppWebWithClient(t, appPath)
	defer cleanup()

	resp := registerStarterUser(t, client, baseURL, "starter@example.com", "Password123!")
	_ = resp.Body.Close()
	assertHTTPStatusContainsForClient(t, client, baseURL+"/auth/delete-account", http.StatusOK, "Delete account")

	deleteForm := url.Values{}
	deleteForm.Set("email", "starter@example.com")
	resp, err := client.PostForm(baseURL+"/auth/delete-account", deleteForm)
	if err != nil {
		t.Fatalf("delete account failed: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("delete account status = %d, want 303", resp.StatusCode)
	}
	if got := resp.Header.Get("Location"); got != "/auth/login" {
		t.Fatalf("delete account redirect = %q, want %q", got, "/auth/login")
	}

	resp, err = client.Get(baseURL + "/auth/profile")
	if err != nil {
		t.Fatalf("protected route after delete failed: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("protected route after delete status = %d, want 303", resp.StatusCode)
	}
	if got := resp.Header.Get("Location"); got != "/auth/login?next=%2Fauth%2Fprofile" {
		t.Fatalf("protected route after delete redirect = %q", got)
	}

	assertLoginFails(t, client, baseURL, "starter@example.com", "Password123!")
}

func TestFreshAppAuthValidationFailures(t *testing.T) {
	shipbin := buildShipBinary(t)
	appPath := scaffoldFreshAppViaShip(t, shipbin, false)
	runCmd(t, appPath, shipbin, "db:migrate")

	baseURL, client, cleanup := startFreshAppWebWithClient(t, appPath)
	defer cleanup()

	assertValidationFailure(t, client, baseURL+"/auth/register", url.Values{
		"display_name":        {""},
		"email":               {"starter@example.com"},
		"password":            {"Password123!"},
		"birthdate":           {"1990-01-01"},
		"relationship_status": {"single"},
	}, "display_name")

	assertValidationFailure(t, client, baseURL+"/auth/register", url.Values{
		"display_name":        {"Starter User"},
		"email":               {""},
		"password":            {"Password123!"},
		"birthdate":           {"1990-01-01"},
		"relationship_status": {"single"},
	}, "email")

	assertValidationFailure(t, client, baseURL+"/auth/register", url.Values{
		"display_name":        {"Starter User"},
		"email":               {"starter@example.com"},
		"password":            {""},
		"birthdate":           {"1990-01-01"},
		"relationship_status": {"single"},
	}, "password")

	assertValidationFailure(t, client, baseURL+"/auth/login", url.Values{
		"email":    {""},
		"password": {""},
		"next":     {"/auth/profile"},
	}, "email", "password")

	resp := registerStarterUser(t, client, baseURL, "starter@example.com", "Password123!")
	_ = resp.Body.Close()
	assertValidationFailure(t, client, baseURL+"/auth/settings", url.Values{
		"display_name": {""},
	}, "display_name")
}

func TestFreshAppPasswordResetValidationFailures(t *testing.T) {
	shipbin := buildShipBinary(t)
	appPath := scaffoldFreshAppViaShip(t, shipbin, false)
	runCmd(t, appPath, shipbin, "db:migrate")

	baseURL, client, cleanup := startFreshAppWebWithClient(t, appPath)
	defer cleanup()

	assertValidationFailure(t, client, baseURL+"/auth/password/reset", url.Values{
		"email": {""},
	}, "email")

	assertValidationFailure(t, client, baseURL+"/auth/password/reset/confirm", url.Values{
		"email":    {""},
		"token":    {""},
		"password": {""},
	}, "email", "token", "password")
}

func TestFreshAppDeleteAccountValidationFailures(t *testing.T) {
	shipbin := buildShipBinary(t)
	appPath := scaffoldFreshAppViaShip(t, shipbin, false)
	runCmd(t, appPath, shipbin, "db:migrate")

	baseURL, client, cleanup := startFreshAppWebWithClient(t, appPath)
	defer cleanup()

	resp := registerStarterUser(t, client, baseURL, "starter@example.com", "Password123!")
	_ = resp.Body.Close()

	assertValidationFailure(t, client, baseURL+"/auth/delete-account", url.Values{
		"email": {""},
	}, "email")

	assertValidationFailure(t, client, baseURL+"/auth/delete-account", url.Values{
		"email": {"other@example.com"},
	}, "email")

	assertHTTPStatusContainsForClient(t, client, baseURL+"/auth/session", http.StatusOK, "starter@example.com")
}

func TestFreshAppInlineValidationUX(t *testing.T) {
	shipbin := buildShipBinary(t)
	appPath := scaffoldFreshAppViaShip(t, shipbin, false)
	runCmd(t, appPath, shipbin, "db:migrate")

	baseURL, client, cleanup := startFreshAppWebWithClient(t, appPath)
	defer cleanup()

	body := postFormHTML(t, client, baseURL+"/auth/register", url.Values{
		"display_name":        {""},
		"email":               {"starter@example.com"},
		"password":            {""},
		"birthdate":           {"1990-01-01"},
		"relationship_status": {"single"},
	})
	assertContainsAll(t, body,
		`data-validation-for="display_name"`,
		`display name is required`,
		`data-validation-for="password"`,
		`password is required`,
		`value="starter@example.com"`,
	)
	if strings.Contains(body, `value="Password123!"`) {
		t.Fatalf("password field should not be echoed back\nbody:\n%s", body)
	}
}

func TestFreshAppAuthRouteInventoryIncludesAccountLifecycle(t *testing.T) {
	shipbin := buildShipBinary(t)
	appPath := scaffoldFreshAppViaShip(t, shipbin, false)

	routesJSON := strings.TrimSpace(runCmd(t, appPath, shipbin, "routes", "--json"))
	for _, want := range []string{
		`"path":"/auth/session"`,
		`"path":"/auth/settings"`,
		`"path":"/auth/admin"`,
		`"path":"/auth/password/reset"`,
		`"path":"/auth/password/reset/confirm"`,
		`"path":"/auth/delete-account"`,
	} {
		if !strings.Contains(routesJSON, want) {
			t.Fatalf("routes inventory missing %s\n%s", want, routesJSON)
		}
	}
}

func TestFreshAppCRUDScaffoldFlow(t *testing.T) {
	shipbin := buildShipBinary(t)
	appPath := scaffoldFreshAppViaShip(t, shipbin, false)
	runCmd(t, appPath, shipbin, "db:migrate")
	runCmd(t, appPath, shipbin, "make:resource", "contact", "--wire")

	baseURL, client, cleanup := startFreshAppWebWithClient(t, appPath)
	defer cleanup()

	assertHTTPStatusContainsForClient(t, client, baseURL+"/contact", http.StatusOK, "Create contact")

	body := postFormHTML(t, client, baseURL+"/contact", url.Values{
		"name": {""},
	})
	assertContainsAll(t, body, `data-validation-for="name"`, `name is required`)

	resp, err := client.PostForm(baseURL+"/contact", url.Values{
		"name": {"Alice"},
	})
	if err != nil {
		t.Fatalf("create contact failed: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("create contact status = %d, want 303", resp.StatusCode)
	}
	if got := resp.Header.Get("Location"); got != "/contact?id=1" {
		t.Fatalf("create contact redirect = %q, want %q", got, "/contact?id=1")
	}

	assertHTTPStatusContainsForClient(t, client, baseURL+"/contact?id=1", http.StatusOK, "Alice")
	assertHTTPStatusContainsForClient(t, client, baseURL+"/contact", http.StatusOK, "Alice")

	resp, err = client.PostForm(baseURL+"/contact?id=1", url.Values{
		"_method": {"PUT"},
		"name":    {"Alice Updated"},
	})
	if err != nil {
		t.Fatalf("update contact failed: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("update contact status = %d, want 303", resp.StatusCode)
	}
	if got := resp.Header.Get("Location"); got != "/contact?id=1" {
		t.Fatalf("update contact redirect = %q, want %q", got, "/contact?id=1")
	}

	assertHTTPStatusContainsForClient(t, client, baseURL+"/contact?id=1", http.StatusOK, "Alice Updated")

	resp, err = client.PostForm(baseURL+"/contact?id=1", url.Values{
		"_method": {"DELETE"},
	})
	if err != nil {
		t.Fatalf("delete contact failed: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("delete contact status = %d, want 303", resp.StatusCode)
	}
	if got := resp.Header.Get("Location"); got != "/contact" {
		t.Fatalf("delete contact redirect = %q, want %q", got, "/contact")
	}

	body = getHTTPBodyForClient(t, client, baseURL+"/contact")
	if strings.Contains(body, "Alice Updated") {
		t.Fatalf("deleted contact still present in index\nbody:\n%s", body)
	}
}

func TestFreshAppStarterControllerHonorsActions(t *testing.T) {
	shipbin := buildShipBinary(t)
	appPath := scaffoldFreshAppViaShip(t, shipbin, false)
	runCmd(t, appPath, shipbin, "db:migrate")
	runCmd(t, appPath, shipbin, "make:controller", "Contact", "--actions", "index,show,create", "--fields", "name:string,email:email", "--wire")

	baseURL, client, cleanup := startFreshAppWebWithClient(t, appPath)
	defer cleanup()

	assertHTTPStatusContainsForClient(t, client, baseURL+"/contact", http.StatusOK, "Create contact")
	assertHTTPStatusContainsForClient(t, client, baseURL+"/contact", http.StatusOK, `name="email"`)

	resp, err := client.PostForm(baseURL+"/contact", url.Values{
		"name":  {"Alice"},
		"email": {"alice@example.com"},
	})
	if err != nil {
		t.Fatalf("create contact failed: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("create contact status = %d, want 303", resp.StatusCode)
	}

	assertHTTPStatusContainsForClient(t, client, baseURL+"/contact?id=1", http.StatusOK, "Alice")
	assertHTTPStatusContainsForClient(t, client, baseURL+"/contact?id=1", http.StatusOK, "alice@example.com")

	req, err := http.NewRequest(http.MethodPost, baseURL+"/contact?id=1", strings.NewReader(url.Values{
		"_method": {"PUT"},
		"name":    {"Alice Updated"},
		"email":   {"alice.updated@example.com"},
	}.Encode()))
	if err != nil {
		t.Fatalf("http.NewRequest(update) error = %v", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("update contact failed: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("update contact status = %d, want 405", resp.StatusCode)
	}

	req, err = http.NewRequest(http.MethodPost, baseURL+"/contact?id=1", strings.NewReader(url.Values{
		"_method": {"DELETE"},
	}.Encode()))
	if err != nil {
		t.Fatalf("http.NewRequest(delete) error = %v", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("delete contact failed: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("delete contact status = %d, want 405", resp.StatusCode)
	}
}

func TestFreshAppCRUDScaffoldPersistsAcrossRestart(t *testing.T) {
	shipbin := buildShipBinary(t)
	appPath := scaffoldFreshAppViaShip(t, shipbin, false)
	runCmd(t, appPath, shipbin, "db:migrate")
	runCmd(t, appPath, shipbin, "make:resource", "contact", "--wire")

	baseURL, client, cleanup := startFreshAppWebWithClient(t, appPath)
	resp, err := client.PostForm(baseURL+"/contact", url.Values{
		"name": {"Alice"},
	})
	if err != nil {
		t.Fatalf("create contact failed: %v", err)
	}
	_ = resp.Body.Close()
	cleanup()

	baseURL, client, cleanup = startFreshAppWebWithClient(t, appPath)
	defer cleanup()

	assertHTTPStatusContainsForClient(t, client, baseURL+"/contact", http.StatusOK, "Alice")
	assertHTTPStatusContainsForClient(t, client, baseURL+"/contact?id=1", http.StatusOK, "Alice")
}

func TestFreshAppMultipleStarterCRUDResourcesStayIsolated(t *testing.T) {
	shipbin := buildShipBinary(t)
	appPath := scaffoldFreshAppViaShip(t, shipbin, false)
	runCmd(t, appPath, shipbin, "db:migrate")
	runCmd(t, appPath, shipbin, "make:resource", "contact", "--wire")
	runCmd(t, appPath, shipbin, "make:resource", "lead", "--wire")

	baseURL, client, cleanup := startFreshAppWebWithClient(t, appPath)
	defer cleanup()

	resp, err := client.PostForm(baseURL+"/contact", url.Values{
		"name": {"Alice"},
	})
	if err != nil {
		t.Fatalf("create contact failed: %v", err)
	}
	_ = resp.Body.Close()
	resp, err = client.PostForm(baseURL+"/lead", url.Values{
		"name": {"Bob"},
	})
	if err != nil {
		t.Fatalf("create lead failed: %v", err)
	}
	_ = resp.Body.Close()

	contactBody := getHTTPBodyForClient(t, client, baseURL+"/contact")
	if !strings.Contains(contactBody, "Alice") {
		t.Fatalf("contact index missing Alice\nbody:\n%s", contactBody)
	}
	if strings.Contains(contactBody, "Bob") {
		t.Fatalf("contact index should not contain lead data\nbody:\n%s", contactBody)
	}

	leadBody := getHTTPBodyForClient(t, client, baseURL+"/lead")
	if !strings.Contains(leadBody, "Bob") {
		t.Fatalf("lead index missing Bob\nbody:\n%s", leadBody)
	}
	if strings.Contains(leadBody, "Alice") {
		t.Fatalf("lead index should not contain contact data\nbody:\n%s", leadBody)
	}
}

func TestFreshAppSupportedBatteryCombinationStaysBuildable(t *testing.T) {
	shipbin := buildShipBinary(t)
	appPath := scaffoldFreshAppViaShip(t, shipbin, false)

	for _, battery := range []string{"jobs", "storage", "emailsubscriptions"} {
		runCmd(t, appPath, shipbin, "module:add", battery)
	}

	runGoTestAll(t, appPath)
	runCmd(t, appPath, shipbin, "verify", "--profile", "fast")
}

func TestFreshAppGeneratedResourceFieldsDriveRuntime(t *testing.T) {
	shipbin := buildShipBinary(t)
	appPath := scaffoldFreshAppViaShip(t, shipbin, false)
	runCmd(t, appPath, shipbin, "db:migrate")
	runCmd(t, appPath, shipbin, "make:resource", "contact", "--wire", "--fields", "name:string,email:email")

	baseURL, client, cleanup := startFreshAppWebWithClient(t, appPath)
	defer cleanup()

	indexBody := getHTTPBodyForClient(t, client, baseURL+"/contact")
	assertContainsAll(t, indexBody,
		`name="name"`,
		`name="email"`,
		`type="email"`,
	)

	resp, err := client.PostForm(baseURL+"/contact", url.Values{
		"name":  {"Alice"},
		"email": {"alice@example.com"},
	})
	if err != nil {
		t.Fatalf("create contact failed: %v", err)
	}
	_ = resp.Body.Close()

	showBody := getHTTPBodyForClient(t, client, baseURL+"/contact?id=1")
	assertContainsAll(t, showBody, "Alice", "alice@example.com")
}

func TestFreshAppAdminDashboardRequiresAdmin(t *testing.T) {
	shipbin := buildShipBinary(t)
	appPath := scaffoldFreshAppViaShip(t, shipbin, false)
	runCmd(t, appPath, shipbin, "db:migrate")
	runCmd(t, appPath, shipbin, "make:resource", "contact", "--wire")

	baseURL, client, cleanup := startFreshAppWebWithClient(t, appPath)
	defer cleanup()

	resp := registerStarterUser(t, client, baseURL, "admin@example.com", "Password123!")
	_ = resp.Body.Close()
	assertHTTPStatusContainsForClient(t, client, baseURL+"/auth/admin", http.StatusOK, "Admin dashboard")
	assertHTTPStatusContainsForClient(t, client, baseURL+"/auth/admin", http.StatusOK, "contact")

	logoutStarterUser(t, client, baseURL)
	resp = registerStarterUser(t, client, baseURL, "member@example.com", "Password123!")
	_ = resp.Body.Close()
	resp2, err := client.Get(baseURL + "/auth/admin")
	if err != nil {
		t.Fatalf("admin page request failed: %v", err)
	}
	_ = resp2.Body.Close()
	if resp2.StatusCode != http.StatusForbidden {
		t.Fatalf("non-admin admin page status = %d, want 403", resp2.StatusCode)
	}
}

func TestFreshAppAdminDashboardShowsResourceDrilldown(t *testing.T) {
	shipbin := buildShipBinary(t)
	appPath := scaffoldFreshAppViaShip(t, shipbin, false)
	runCmd(t, appPath, shipbin, "db:migrate")
	runCmd(t, appPath, shipbin, "make:resource", "contact", "--wire", "--fields", "name:string,email:email")
	runCmd(t, appPath, shipbin, "make:resource", "lead", "--wire", "--fields", "name:string")

	baseURL, client, cleanup := startFreshAppWebWithClient(t, appPath)
	defer cleanup()

	resp := registerStarterUser(t, client, baseURL, "admin@example.com", "Password123!")
	_ = resp.Body.Close()
	_, err := client.PostForm(baseURL+"/contact", url.Values{
		"name":  {"Alice"},
		"email": {"alice@example.com"},
	})
	if err != nil {
		t.Fatalf("create contact failed: %v", err)
	}
	_, err = client.PostForm(baseURL+"/lead", url.Values{
		"name": {"Bob"},
	})
	if err != nil {
		t.Fatalf("create lead failed: %v", err)
	}

	adminBody := getHTTPBodyForClient(t, client, baseURL+"/auth/admin")
	assertContainsAll(t, adminBody, `?resource=contact`, `?resource=lead`, `data-admin-count="contact">1<`, `data-admin-count="lead">1<`)

	contactBody := getHTTPBodyForClient(t, client, baseURL+"/auth/admin?resource=contact")
	assertContainsAll(t, contactBody, "Admin resource: contact", "Alice", "alice@example.com")
	if strings.Contains(contactBody, "Bob") {
		t.Fatalf("contact admin drilldown should not include lead data\nbody:\n%s", contactBody)
	}
}

func TestFreshAppAdminDashboardCanManageGeneratedResource(t *testing.T) {
	shipbin := buildShipBinary(t)
	appPath := scaffoldFreshAppViaShip(t, shipbin, false)
	runCmd(t, appPath, shipbin, "db:migrate")
	runCmd(t, appPath, shipbin, "make:resource", "contact", "--wire", "--fields", "name:string,email:email")

	baseURL, client, cleanup := startFreshAppWebWithClient(t, appPath)
	defer cleanup()

	resp := registerStarterUser(t, client, baseURL, "admin@example.com", "Password123!")
	_ = resp.Body.Close()

	resp, err := client.PostForm(baseURL+"/auth/admin?resource=contact", url.Values{
		"name":  {"Alice"},
		"email": {"alice@example.com"},
	})
	if err != nil {
		t.Fatalf("admin create failed: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("admin create status = %d, want 303", resp.StatusCode)
	}

	body := getHTTPBodyForClient(t, client, baseURL+"/auth/admin?resource=contact")
	assertContainsAll(t, body, "Alice", "alice@example.com")

	updateReq, err := http.NewRequest(http.MethodPost, baseURL+"/auth/admin?resource=contact&id=1", strings.NewReader(url.Values{
		"_method": {"PUT"},
		"name":    {"Alice Updated"},
		"email":   {"alice.updated@example.com"},
	}.Encode()))
	if err != nil {
		t.Fatalf("http.NewRequest(update) error = %v", err)
	}
	updateReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err = client.Do(updateReq)
	if err != nil {
		t.Fatalf("admin update failed: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("admin update status = %d, want 303", resp.StatusCode)
	}

	body = getHTTPBodyForClient(t, client, baseURL+"/auth/admin?resource=contact")
	assertContainsAll(t, body, "Alice Updated", "alice.updated@example.com")

	deleteReq, err := http.NewRequest(http.MethodPost, baseURL+"/auth/admin?resource=contact&id=1", strings.NewReader(url.Values{
		"_method": {"DELETE"},
	}.Encode()))
	if err != nil {
		t.Fatalf("http.NewRequest(delete) error = %v", err)
	}
	deleteReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err = client.Do(deleteReq)
	if err != nil {
		t.Fatalf("admin delete failed: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("admin delete status = %d, want 303", resp.StatusCode)
	}

	body = getHTTPBodyForClient(t, client, baseURL+"/auth/admin?resource=contact")
	if strings.Contains(body, "Alice Updated") {
		t.Fatalf("admin delete should remove updated row\nbody:\n%s", body)
	}
}

func TestFreshAppStorageModuleEnablesProfileUpload(t *testing.T) {
	shipbin := buildShipBinary(t)
	appPath := scaffoldFreshAppViaShip(t, shipbin, false)
	runCmd(t, appPath, shipbin, "db:migrate")
	runCmd(t, appPath, shipbin, "module:add", "storage")

	baseURL, client, cleanup := startFreshAppWebWithClient(t, appPath)
	defer cleanup()

	resp := registerStarterUser(t, client, baseURL, "admin@example.com", "Password123!")
	_ = resp.Body.Close()

	assertHTTPStatusContainsForClient(t, client, baseURL+"/auth/profile", http.StatusOK, `name="storage_upload"`)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("storage_upload", "avatar.txt")
	if err != nil {
		t.Fatalf("CreateFormFile() error = %v", err)
	}
	if _, err := part.Write([]byte("hello storage")); err != nil {
		t.Fatalf("part.Write() error = %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("writer.Close() error = %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, baseURL+"/auth/profile", &body)
	if err != nil {
		t.Fatalf("http.NewRequest(upload) error = %v", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("profile upload failed: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("profile upload status = %d, want 303", resp.StatusCode)
	}

	assertHTTPStatusContainsForClient(t, client, baseURL+"/auth/profile", http.StatusOK, "avatar.txt")
}

func TestFreshAppMailerPreviewFlow(t *testing.T) {
	shipbin := buildShipBinary(t)
	appPath := scaffoldFreshAppViaShip(t, shipbin, false)
	runCmd(t, appPath, shipbin, "db:migrate")
	runCmd(t, appPath, shipbin, "make:mailer", "Welcome")

	baseURL, client, cleanup := startFreshAppWebWithClient(t, appPath)
	defer cleanup()

	assertHTTPStatusContainsForClient(t, client, baseURL+"/dev/mail", http.StatusOK, "/dev/mail/welcome")
	assertHTTPStatusContainsForClient(t, client, baseURL+"/dev/mail/welcome", http.StatusOK, "Welcome Email")
}

func startFreshAppWebWithClient(t *testing.T, appPath string) (string, *http.Client, func()) {
	t.Helper()

	port := reserveFreePort(t)
	ctx, cancel := context.WithCancel(context.Background())
	webCmd := exec.CommandContext(ctx, "go", "run", "./cmd/web")
	webCmd.Dir = appPath
	webCmd.Env = append(os.Environ(), "PORT="+port)
	logPath := filepath.Join(t.TempDir(), "starter-auth-account-web.log")
	logFile, err := os.Create(logPath)
	if err != nil {
		t.Fatalf("os.Create(log) error = %v", err)
	}
	webCmd.Stdout = logFile
	webCmd.Stderr = logFile
	if err := webCmd.Start(); err != nil {
		_ = logFile.Close()
		t.Fatalf("webCmd.Start() error = %v", err)
	}

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

	cleanup := func() {
		cancel()
		if webCmd.Process != nil {
			_ = webCmd.Process.Kill()
		}
		_ = webCmd.Wait()
		_ = logFile.Close()
	}
	return baseURL, client, cleanup
}

func registerStarterUser(t *testing.T, client *http.Client, baseURL, email, password string) *http.Response {
	t.Helper()
	form := url.Values{}
	form.Set("display_name", "Starter User")
	form.Set("email", email)
	form.Set("password", password)
	form.Set("birthdate", "1990-01-01")
	form.Set("relationship_status", "single")

	resp, err := client.PostForm(baseURL+"/auth/register", form)
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}
	if resp.StatusCode != http.StatusSeeOther {
		_ = resp.Body.Close()
		t.Fatalf("register status = %d, want 303", resp.StatusCode)
	}
	return resp
}

func logoutStarterUser(t *testing.T, client *http.Client, baseURL string) {
	t.Helper()
	resp, err := client.Get(baseURL + "/auth/logout")
	if err != nil {
		t.Fatalf("logout failed: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("logout status = %d, want 303", resp.StatusCode)
	}
}

func assertLoginFails(t *testing.T, client *http.Client, baseURL, email, password string) {
	t.Helper()
	loginForm := url.Values{}
	loginForm.Set("email", email)
	loginForm.Set("password", password)
	loginForm.Set("next", "/auth/profile")
	resp, err := client.PostForm(baseURL+"/auth/login", loginForm)
	if err != nil {
		t.Fatalf("login request failed: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("login status = %d, want 401", resp.StatusCode)
	}
}

func assertLoginSucceeds(t *testing.T, client *http.Client, baseURL, email, password, next string) {
	t.Helper()
	loginForm := url.Values{}
	loginForm.Set("email", email)
	loginForm.Set("password", password)
	loginForm.Set("next", next)
	resp, err := client.PostForm(baseURL+"/auth/login", loginForm)
	if err != nil {
		t.Fatalf("login request failed: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("login status = %d, want 303", resp.StatusCode)
	}
	if got := resp.Header.Get("Location"); got != next {
		t.Fatalf("login redirect = %q, want %q", got, next)
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

func assertHTTPStatusContainsForClient(t *testing.T, client *http.Client, url string, wantStatus int, want string) {
	t.Helper()
	resp, err := client.Get(url)
	if err != nil {
		t.Fatalf("GET %s failed: %v", url, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("reading %s failed: %v", url, err)
	}
	if resp.StatusCode != wantStatus {
		t.Fatalf("%s status = %d, want %d\nbody:\n%s", url, resp.StatusCode, wantStatus, string(body))
	}
	if !strings.Contains(string(body), want) {
		t.Fatalf("%s body missing %q\nbody:\n%s", url, want, string(body))
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

func getHTTPBodyForClient(t *testing.T, client *http.Client, url string) string {
	t.Helper()
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

func extractResetToken(t *testing.T, body string) string {
	t.Helper()
	match := starterResetTokenPattern.FindStringSubmatch(body)
	if len(match) != 2 {
		t.Fatalf("reset token marker not found\nbody:\n%s", body)
	}
	return strings.TrimSpace(match[1])
}

func assertValidationFailure(t *testing.T, client *http.Client, endpoint string, form url.Values, fields ...string) {
	t.Helper()
	resp, err := client.PostForm(endpoint, form)
	if err != nil {
		t.Fatalf("POST %s failed: %v", endpoint, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("reading %s failed: %v", endpoint, err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("%s status = %d, want 400\nbody:\n%s", endpoint, resp.StatusCode, string(body))
	}
	if !strings.Contains(string(body), "validation_error") {
		t.Fatalf("%s body missing validation_error marker\nbody:\n%s", endpoint, string(body))
	}
	for _, field := range fields {
		if !strings.Contains(string(body), field) {
			t.Fatalf("%s body missing field %q\nbody:\n%s", endpoint, field, string(body))
		}
	}
}

func postFormHTML(t *testing.T, client *http.Client, endpoint string, form url.Values) string {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		t.Fatalf("http.NewRequest(%s) failed: %v", endpoint, err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "text/html")
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("POST %s failed: %v", endpoint, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("reading %s failed: %v", endpoint, err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("%s status = %d, want 400\nbody:\n%s", endpoint, resp.StatusCode, string(body))
	}
	return string(body)
}

func assertContainsAll(t *testing.T, body string, needles ...string) {
	t.Helper()
	for _, needle := range needles {
		if !strings.Contains(body, needle) {
			t.Fatalf("body missing %q\nbody:\n%s", needle, body)
		}
	}
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
