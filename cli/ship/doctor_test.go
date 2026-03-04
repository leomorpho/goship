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
		if err := os.RemoveAll(filepath.Join(root, "apps", "goship", "jobs")); err != nil {
			t.Fatal(err)
		}
		issues := runDoctorChecks(root)
		mustContainIssueCode(t, issues, "DX001")
	})

	t.Run("forbidden legacy path present", func(t *testing.T) {
		root := t.TempDir()
		writeDoctorFixture(t, root)
		if err := os.MkdirAll(filepath.Join(root, "apps", "goship", "domains"), 0o755); err != nil {
			t.Fatal(err)
		}
		issues := runDoctorChecks(root)
		mustContainIssueCode(t, issues, "DX003")
	})

	t.Run("router marker missing", func(t *testing.T) {
		root := t.TempDir()
		writeDoctorFixture(t, root)
		router := filepath.Join(root, "apps", "goship", "router.go")
		if err := os.WriteFile(router, []byte("package goship\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		issues := runDoctorChecks(root)
		mustContainIssueCode(t, issues, "DX005")
	})

	t.Run("router marker order invalid", func(t *testing.T) {
		root := t.TempDir()
		writeDoctorFixture(t, root)
		router := filepath.Join(root, "apps", "goship", "router.go")
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
		viewmodels := filepath.Join(root, "apps", "goship", "web", "viewmodels", "user.go")
		if err := os.WriteFile(viewmodels, []byte("package types\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		issues := runDoctorChecks(root)
		mustContainIssueCode(t, issues, "DX007")
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
		path := filepath.Join(root, "apps", "goship", "web", "ui", "too_big.go")
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
		filepath.Join(root, "apps", "goship", "app"),
		filepath.Join(root, "apps", "goship", "foundation"),
		filepath.Join(root, "apps", "goship", "web", "controllers"),
		filepath.Join(root, "apps", "goship", "web", "middleware"),
		filepath.Join(root, "apps", "goship", "web", "ui"),
		filepath.Join(root, "apps", "goship", "web", "viewmodels"),
		filepath.Join(root, "apps", "goship", "web", "routenames"),
		filepath.Join(root, "apps", "goship", "jobs"),
		filepath.Join(root, "apps", "goship", "views"),
		filepath.Join(root, "apps", "goship", "db", "schema"),
		filepath.Join(root, "docs", "architecture"),
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	files := map[string]string{
		filepath.Join(root, "apps", "goship", "router.go"): `package goship

func registerPublicRoutes() {
	// ship:routes:public:start
	// ship:routes:public:end
}

func registerAuthRoutes() {
	// ship:routes:auth:start
	// ship:routes:auth:end
}
`,
		filepath.Join(root, "apps", "goship", "foundation", "container.go"):         "package foundation\n",
		filepath.Join(root, "apps", "goship", "web", "ui", "page.go"):               "package ui\n",
		filepath.Join(root, "apps", "goship", "web", "viewmodels", "page_data.go"):  "package viewmodels\n",
		filepath.Join(root, "apps", "goship", "web", "routenames", "routenames.go"): "package routenames\n",
		filepath.Join(root, "docs", "00-index.md"):                                  "# Index\n",
		filepath.Join(root, "docs", "architecture", "01-architecture.md"):           "# Architecture\n",
		filepath.Join(root, "docs", "architecture", "08-cognitive-model.md"):        "# Cognitive Model\n",
		filepath.Join(root, ".gitignore"): strings.Join([]string{
			"/web",
			"/worker",
			"/seed",
			"/ship",
			"/ship-mcp",
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
