package ship

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunDoctorChecks(t *testing.T) {
	t.Run("valid fixture has no issues", func(t *testing.T) {
		root := t.TempDir()
		writeDoctorFixture(t, root)
		issues := runDoctorChecks(root)
		if len(issues) != 0 {
			t.Fatalf("expected no issues, got %+v", issues)
		}
	})

	t.Run("missing required directory", func(t *testing.T) {
		root := t.TempDir()
		writeDoctorFixture(t, root)
		if err := os.RemoveAll(filepath.Join(root, "apps", "site", "jobs")); err != nil {
			t.Fatal(err)
		}
		issues := runDoctorChecks(root)
		mustContainIssueCode(t, issues, "DX001")
	})

	t.Run("forbidden legacy path present", func(t *testing.T) {
		root := t.TempDir()
		writeDoctorFixture(t, root)
		if err := os.MkdirAll(filepath.Join(root, "apps", "site", "domains"), 0o755); err != nil {
			t.Fatal(err)
		}
		issues := runDoctorChecks(root)
		mustContainIssueCode(t, issues, "DX003")
	})

	t.Run("router marker missing", func(t *testing.T) {
		root := t.TempDir()
		writeDoctorFixture(t, root)
		router := filepath.Join(root, "apps", "site", "router.go")
		if err := os.WriteFile(router, []byte("package goship\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		issues := runDoctorChecks(root)
		mustContainIssueCode(t, issues, "DX005")
	})

	t.Run("router marker order invalid", func(t *testing.T) {
		root := t.TempDir()
		writeDoctorFixture(t, root)
		router := filepath.Join(root, "apps", "site", "router.go")
		content := `package goship

func registerPublicRoutes() {
	// ship:routes:public:end
	// ship:routes:public:start
}

func registerAuthRoutes() {
	// ship:routes:auth:start
	// ship:routes:auth:end
}
`
		if err := os.WriteFile(router, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		issues := runDoctorChecks(root)
		mustContainIssueCode(t, issues, "DX011")
	})

	t.Run("package naming mismatch", func(t *testing.T) {
		root := t.TempDir()
		writeDoctorFixture(t, root)
		viewmodels := filepath.Join(root, "apps", "site", "web", "viewmodels", "user.go")
		if err := os.WriteFile(viewmodels, []byte("package types\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		issues := runDoctorChecks(root)
		mustContainIssueCode(t, issues, "DX007")
	})

	t.Run("unexpected top-level directory", func(t *testing.T) {
		root := t.TempDir()
		writeDoctorFixture(t, root)
		if err := os.MkdirAll(filepath.Join(root, "scratch"), 0o755); err != nil {
			t.Fatal(err)
		}
		issues := runDoctorChecks(root)
		mustContainIssueCode(t, issues, "DX013")
	})

	t.Run("root binary artifact present", func(t *testing.T) {
		root := t.TempDir()
		writeDoctorFixture(t, root)
		if err := os.WriteFile(filepath.Join(root, "web"), []byte("binary"), 0o644); err != nil {
			t.Fatal(err)
		}
		issues := runDoctorChecks(root)
		mustContainIssueCode(t, issues, "DX008")
	})

	t.Run("gitignore missing root artifact entries", func(t *testing.T) {
		root := t.TempDir()
		writeDoctorFixture(t, root)
		if err := os.WriteFile(filepath.Join(root, ".gitignore"), []byte("/web\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		issues := runDoctorChecks(root)
		mustContainIssueCode(t, issues, "DX009")
	})

	t.Run("new oversized go file trips line budget", func(t *testing.T) {
		root := t.TempDir()
		writeDoctorFixture(t, root)
		path := filepath.Join(root, "apps", "site", "web", "ui", "too_big.go")
		var b strings.Builder
		b.WriteString("package ui\n")
		for i := 0; i < 520; i++ {
			b.WriteString("var _ = 1\n")
		}
		if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
			t.Fatal(err)
		}
		issues := runDoctorChecks(root)
		mustContainIssueCode(t, issues, "DX010")
	})

	t.Run("cli docs missing required command token", func(t *testing.T) {
		root := t.TempDir()
		writeDoctorFixture(t, root)
		cliDoc := filepath.Join(root, "docs", "reference", "01-cli.md")
		if err := os.MkdirAll(filepath.Dir(cliDoc), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(cliDoc, []byte("ship doctor\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		issues := runDoctorChecks(root)
		mustContainIssueCode(t, issues, "DX012")
	})

	t.Run("cli docs missing required section", func(t *testing.T) {
		root := t.TempDir()
		writeDoctorFixture(t, root)
		cliDoc := filepath.Join(root, "docs", "reference", "01-cli.md")
		content := strings.Join([]string{
			"ship doctor",
			"ship agent:setup",
			"ship agent:check",
			"ship agent:status",
			"ship new <app>",
			"ship upgrade",
			"ship make:resource",
			"ship make:model",
			"ship make:controller",
			"ship make:scaffold",
			"ship make:module",
			"ship db:migrate",
			"ship test --integration",
			"",
		}, "\n")
		if err := os.WriteFile(cliDoc, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		issues := runDoctorChecks(root)
		mustContainIssueCode(t, issues, "DX012")
	})

	t.Run("go.work references missing module", func(t *testing.T) {
		root := t.TempDir()
		writeDoctorFixture(t, root)
		if err := os.WriteFile(filepath.Join(root, "go.work"), []byte("go 1.25\n\nuse (\n\t.\n\t./missing-module\n)\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		issues := runDoctorChecks(root)
		mustContainIssueCode(t, issues, "DX014")
	})

	t.Run("dockerignore missing required exclusion", func(t *testing.T) {
		root := t.TempDir()
		writeDoctorFixture(t, root)
		if err := os.WriteFile(filepath.Join(root, ".dockerignore"), []byte(".git\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		issues := runDoctorChecks(root)
		mustContainIssueCode(t, issues, "DX015")
	})

	t.Run("dockerfile local replace copy ordering", func(t *testing.T) {
		root := t.TempDir()
		writeDoctorFixture(t, root)
		badDockerfile := `FROM golang:1.25.6 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
`
		if err := os.WriteFile(filepath.Join(root, "infra", "docker", "Dockerfile"), []byte(badDockerfile), 0o644); err != nil {
			t.Fatal(err)
		}
		issues := runDoctorChecks(root)
		mustContainIssueCode(t, issues, "DX016")
	})

	t.Run("agent policy artifact drift", func(t *testing.T) {
		root := t.TempDir()
		writeDoctorFixture(t, root)
		if err := os.WriteFile(filepath.Join(root, "tools", "agent-policy", "generated", "codex-prefixes.txt"), []byte("stale\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		issues := runDoctorChecks(root)
		mustContainIssueCode(t, issues, "DX017")
	})
}

func TestRunDoctor(t *testing.T) {
	t.Run("help", func(t *testing.T) {
		cli := CLI{Out: &bytes.Buffer{}, Err: &bytes.Buffer{}}
		if code := cli.runDoctor([]string{"--help"}); code != 0 {
			t.Fatalf("exit code = %d, want 0", code)
		}
	})

	t.Run("unexpected args", func(t *testing.T) {
		out := &bytes.Buffer{}
		errOut := &bytes.Buffer{}
		cli := CLI{Out: out, Err: errOut}
		code := cli.runDoctor([]string{"extra"})
		if code != 1 {
			t.Fatalf("exit code = %d, want 1", code)
		}
		if !strings.Contains(errOut.String(), "unexpected doctor arguments") {
			t.Fatalf("stderr = %q, want unexpected args message", errOut.String())
		}
	})
}

func TestDoctorCommand_IntegrationFixture(t *testing.T) {
	root := t.TempDir()
	writeDoctorFixture(t, root)
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/doctor\n\ngo 1.25\n"), 0o644); err != nil {
		t.Fatal(err)
	}

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
	cli := CLI{Out: out, Err: errOut}
	if code := cli.Run([]string{"doctor"}); code != 0 {
		t.Fatalf("doctor exit code = %d, stderr=%s", code, errOut.String())
	}
	if !strings.Contains(out.String(), "ship doctor: OK") {
		t.Fatalf("stdout = %q, want doctor OK output", out.String())
	}
}

func writeDoctorFixture(t *testing.T, root string) {
	t.Helper()

	dirs := []string{
		filepath.Join(root, "apps", "site", "app"),
		filepath.Join(root, "apps", "site", "foundation"),
		filepath.Join(root, "apps", "site", "web", "controllers"),
		filepath.Join(root, "apps", "site", "web", "middleware"),
		filepath.Join(root, "apps", "site", "web", "ui"),
		filepath.Join(root, "apps", "site", "web", "viewmodels"),
		filepath.Join(root, "apps", "site", "web", "routenames"),
		filepath.Join(root, "apps", "site", "jobs"),
		filepath.Join(root, "apps", "site", "views"),
		filepath.Join(root, "apps", "db", "schema"),
		filepath.Join(root, "apps", "db", "migrate", "migrations"),
		filepath.Join(root, "config"),
		filepath.Join(root, "docs", "architecture"),
		filepath.Join(root, "docs", "reference"),
		filepath.Join(root, "infra", "docker"),
		filepath.Join(root, "modules", "local"),
		filepath.Join(root, "apps"),
		filepath.Join(root, "tools", "agent-policy", "generated"),
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	files := map[string]string{
		filepath.Join(root, "apps", "site", "router.go"): `package goship

func registerPublicRoutes() {
	// ship:routes:public:start
	// ship:routes:public:end
}

func registerAuthRoutes() {
	// ship:routes:auth:start
	// ship:routes:auth:end
}
`,
		filepath.Join(root, "apps", "site", "foundation", "container.go"):         "package foundation\n",
		filepath.Join(root, "apps", "site", "web", "ui", "page.go"):               "package ui\n",
		filepath.Join(root, "apps", "site", "web", "viewmodels", "page_data.go"):  "package viewmodels\n",
		filepath.Join(root, "apps", "site", "web", "routenames", "routenames.go"): "package routenames\n",
		filepath.Join(root, "config", "modules.yaml"):                             "modules: []\n",
		filepath.Join(root, "docs", "00-index.md"):                                "# Index\n",
		filepath.Join(root, "docs", "architecture", "01-architecture.md"):         "# Architecture\n",
		filepath.Join(root, "docs", "architecture", "08-cognitive-model.md"):      "# Cognitive Model\n",
		filepath.Join(root, "docs", "reference", "01-cli.md"): strings.Join([]string{
			"## Minimal V1 Command Set",
			"## Implementation Mapping (Current Repo)",
			"## Generator test strategy",
			"ship doctor",
			"ship agent:setup",
			"ship agent:check",
			"ship agent:status",
			"ship new <app>",
			"ship upgrade",
			"ship make:resource",
			"ship make:model",
			"ship make:controller",
			"ship make:scaffold",
			"ship make:module",
			"ship db:migrate",
			"ship test --integration",
			"",
		}, "\n"),
		filepath.Join(root, ".gitignore"): strings.Join([]string{
			"/web",
			"/worker",
			"/seed",
			"/ship",
			"/ship-mcp",
			"",
		}, "\n"),
		filepath.Join(root, ".dockerignore"): strings.Join([]string{
			".git",
			"node_modules",
			"frontend/node_modules",
			"tmp",
			"tools/scripts/venv",
			"",
		}, "\n"),
		filepath.Join(root, "go.mod"): strings.Join([]string{
			"module example.com/root",
			"",
			"go 1.25",
			"",
			"replace example.com/local => ./modules/local",
			"",
		}, "\n"),
		filepath.Join(root, "apps", "go.mod"): strings.Join([]string{
			"module example.com/apps",
			"",
			"go 1.25",
			"",
			"replace example.com/root => ..",
			"",
		}, "\n"),
		filepath.Join(root, "go.work"): strings.Join([]string{
			"go 1.25",
			"",
			"use (",
			"\t.",
			"\t./apps",
			")",
			"",
		}, "\n"),
		filepath.Join(root, "infra", "docker", "Dockerfile"): strings.Join([]string{
			"FROM golang:1.25.6 AS builder",
			"WORKDIR /app",
			"COPY . .",
			"WORKDIR /app/apps",
			"RUN go mod download",
			"",
		}, "\n"),
		filepath.Join(root, "tools", "agent-policy", "allowed-commands.yaml"): strings.Join([]string{
			"version: 1",
			"commands:",
			"  - id: go_test",
			"    description: Run Go tests.",
			"    prefix: [\"go\", \"test\"]",
			"",
		}, "\n"),
		filepath.Join(root, "tools", "agent-policy", "generated", "allowed-prefixes.json"): strings.Join([]string{
			"{",
			"  \"version\": 1,",
			"  \"prefixes\": [",
			"    [",
			"      \"go\",",
			"      \"test\"",
			"    ]",
			"  ]",
			"}",
			"",
		}, "\n"),
		filepath.Join(root, "tools", "agent-policy", "generated", "codex-prefixes.txt"):  "go test\n",
		filepath.Join(root, "tools", "agent-policy", "generated", "claude-prefixes.txt"): "go test\n",
		filepath.Join(root, "tools", "agent-policy", "generated", "gemini-prefixes.txt"): "go test\n",
		filepath.Join(root, "tools", "agent-policy", "generated", "INSTALL.md"): strings.Join([]string{
			"# Agent Command Allowlist",
			"",
			"Source of truth: `tools/agent-policy/allowed-commands.yaml`",
			"",
			"Generated files in this directory are for local tool import.",
			"",
			"## Commands",
			"",
			"- `go test` - Run Go tests.",
			"",
			"## Setup",
			"",
			"1. Run `ship agent:setup` to sync generated artifacts.",
			"2. Import `codex-prefixes.txt`, `claude-prefixes.txt`, and `gemini-prefixes.txt` into each local tool's command-permission settings.",
			"3. Run `ship agent:check` in CI/pre-commit to enforce parity.",
			"",
		}, "\n"),
	}
	for path, content := range files {
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func mustContainIssueCode(t *testing.T, issues []doctorIssue, code string) {
	t.Helper()
	for _, issue := range issues {
		if issue.Code == code {
			return
		}
	}
	t.Fatalf("expected issue code %s, got %+v", code, issues)
}
