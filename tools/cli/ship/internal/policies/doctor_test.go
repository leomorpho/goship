package policies

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunDoctorChecks(t *testing.T) {
	t.Run("canonical framework repo skips starter-app required path checks", func(t *testing.T) {
		root := t.TempDir()
		writeCanonicalFrameworkDoctorFixture(t, root)

		issues := RunDoctorChecks(root)
		for _, issue := range issues {
			if issue.Code != "DX001" && issue.Code != "DX002" && issue.Code != "DX013" {
				continue
			}
			if strings.Contains(issue.Message, "app") || strings.Contains(issue.Message, "unexpected top-level directory: static") || strings.Contains(issue.Message, "unexpected top-level directory: styles") || strings.Contains(issue.Message, "unexpected top-level directory: testdata") {
				t.Fatalf("unexpected framework-repo issue: %+v", issue)
			}
		}
	})

	t.Run("valid fixture has no issues", func(t *testing.T) {
		root := t.TempDir()
		writeDoctorFixture(t, root)
		issues := RunDoctorChecks(root)
		if len(issues) != 0 {
			t.Fatalf("expected no issues, got %+v", issues)
		}
	})

	t.Run("missing required directory", func(t *testing.T) {
		root := t.TempDir()
		writeDoctorFixture(t, root)
		if err := os.RemoveAll(filepath.Join(root, "app", "jobs")); err != nil {
			t.Fatal(err)
		}
		issues := RunDoctorChecks(root)
		mustContainIssueCode(t, issues, "DX001")
	})

	t.Run("generated app fast-path returns root causes before secondary drift", func(t *testing.T) {
		root := t.TempDir()
		writeDoctorFixture(t, root)
		if err := os.RemoveAll(filepath.Join(root, "app", "jobs")); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Join(root, "scratch"), 0o755); err != nil {
			t.Fatal(err)
		}

		issues := FastPathGeneratedAppIssues(root)
		if len(issues) == 0 {
			t.Fatalf("expected fast-path issues")
		}
		if issues[0].Code != "DX001" {
			t.Fatalf("issues[0].Code = %q, want DX001", issues[0].Code)
		}
		for _, issue := range issues {
			if issue.Code == "DX013" {
				t.Fatalf("unexpected secondary issue in fast path: %+v", issue)
			}
		}
	})

	t.Run("generated app fast-path root causes map to scaffold generator owner hint", func(t *testing.T) {
		root := t.TempDir()
		writeDoctorFixture(t, root)
		if err := os.RemoveAll(filepath.Join(root, "app", "jobs")); err != nil {
			t.Fatal(err)
		}

		issues := FastPathGeneratedAppIssues(root)
		if len(issues) == 0 {
			t.Fatalf("expected fast-path issues")
		}
		if got, want := IssueOwnerHint(issues[0].Code), "ship new scaffold generator (tools/cli/ship/internal/commands/project_new.go)"; got != want {
			t.Fatalf("IssueOwnerHint(%q) = %q, want %q", issues[0].Code, got, want)
		}
	})

	t.Run("policy drift issue maps to doctor policy owner hint", func(t *testing.T) {
		root := t.TempDir()
		writeDoctorFixture(t, root)
		if err := os.MkdirAll(filepath.Join(root, "scratch"), 0o755); err != nil {
			t.Fatal(err)
		}

		issues := RunDoctorChecks(root)
		mustContainIssueCode(t, issues, "DX013")
		if got, want := IssueOwnerHint("DX013"), "doctor policy checks (tools/cli/ship/internal/policies/doctor.go)"; got != want {
			t.Fatalf("IssueOwnerHint(DX013) = %q, want %q", got, want)
		}
	})

	t.Run("forbidden legacy path present", func(t *testing.T) {
		root := t.TempDir()
		writeDoctorFixture(t, root)
		if err := os.MkdirAll(filepath.Join(root, "app", "domains"), 0o755); err != nil {
			t.Fatal(err)
		}
		issues := RunDoctorChecks(root)
		mustContainIssueCode(t, issues, "DX003")
	})

	t.Run("router marker missing", func(t *testing.T) {
		root := t.TempDir()
		writeDoctorFixture(t, root)
		router := filepath.Join(root, "app", "router.go")
		if err := os.WriteFile(router, []byte("package goship\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		issues := RunDoctorChecks(root)
		mustContainIssueCode(t, issues, "DX005")
	})

	t.Run("router marker order invalid", func(t *testing.T) {
		root := t.TempDir()
		writeDoctorFixture(t, root)
		router := filepath.Join(root, "app", "router.go")
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
		issues := RunDoctorChecks(root)
		mustContainIssueCode(t, issues, "DX011")
	})

	t.Run("external route marker missing", func(t *testing.T) {
		root := t.TempDir()
		writeDoctorFixture(t, root)
		router := filepath.Join(root, "app", "router.go")
		content := `package goship

func registerPublicRoutes() {
	// ship:routes:public:start
	// ship:routes:public:end
}

func registerAuthRoutes() {
	// ship:routes:auth:start
	// ship:routes:auth:end
}
`
		if err := os.WriteFile(router, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		issues := RunDoctorChecks(root)
		mustContainIssueCode(t, issues, "DX005")
	})

	t.Run("package naming mismatch", func(t *testing.T) {
		root := t.TempDir()
		writeDoctorFixture(t, root)
		viewmodels := filepath.Join(root, "app", "web", "viewmodels", "user.go")
		if err := os.WriteFile(viewmodels, []byte("package types\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		issues := RunDoctorChecks(root)
		mustContainIssueCode(t, issues, "DX007")
	})

	t.Run("unexpected top-level directory", func(t *testing.T) {
		root := t.TempDir()
		writeDoctorFixture(t, root)
		if err := os.MkdirAll(filepath.Join(root, "scratch"), 0o755); err != nil {
			t.Fatal(err)
		}
		issues := RunDoctorChecks(root)
		mustContainIssueCode(t, issues, "DX013")
	})

	t.Run("intentional githooks directory is allowed", func(t *testing.T) {
		root := t.TempDir()
		writeDoctorFixture(t, root)
		if err := os.MkdirAll(filepath.Join(root, ".githooks"), 0o755); err != nil {
			t.Fatal(err)
		}
		issues := RunDoctorChecks(root)
		mustNotContainIssueCode(t, issues, "DX013")
	})

	t.Run("locales directory is allowed", func(t *testing.T) {
		root := t.TempDir()
		writeDoctorFixture(t, root)
		if err := os.MkdirAll(filepath.Join(root, "locales"), 0o755); err != nil {
			t.Fatal(err)
		}
		issues := RunDoctorChecks(root)
		mustNotContainIssueCode(t, issues, "DX013")
	})

	t.Run("tmp directory is allowed for dev build artifacts", func(t *testing.T) {
		root := t.TempDir()
		writeDoctorFixture(t, root)
		if err := os.MkdirAll(filepath.Join(root, "tmp"), 0o755); err != nil {
			t.Fatal(err)
		}
		issues := RunDoctorChecks(root)
		mustNotContainIssueCode(t, issues, "DX013")
	})

	t.Run("missing renders comment triggers warning", func(t *testing.T) {
		root := t.TempDir()
		writeDoctorFixture(t, root)
		targetDir := filepath.Join(root, "app", "views", "web", "components")
		if err := os.MkdirAll(targetDir, 0o755); err != nil {
			t.Fatal(err)
		}
		target := filepath.Join(targetDir, "missing.templ")
		if err := os.WriteFile(target, []byte("templ MissingComponent() {}\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		issues := RunDoctorChecks(root)
		mustContainIssueCode(t, issues, "DX023")
	})

	t.Run("i18n strict mode warn surfaces non-blocking DX029 findings", func(t *testing.T) {
		root := t.TempDir()
		writeDoctorFixture(t, root)
		if err := os.WriteFile(filepath.Join(root, ".env"), []byte("PAGODA_I18N_STRICT_MODE=warn\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		controllerPath := filepath.Join(root, "app", "web", "controllers", "sample.go")
		controllerBody := "package controllers\nfunc demo() string {\n\treturn \"Welcome users\"\n}\n"
		if err := os.WriteFile(controllerPath, []byte(controllerBody), 0o644); err != nil {
			t.Fatal(err)
		}

		issues := RunDoctorChecks(root)
		mustContainIssueCode(t, issues, "DX029")
		for _, issue := range issues {
			if issue.Code == "DX029" && issue.Severity != "warning" {
				t.Fatalf("expected DX029 warning severity in warn mode, got %+v", issue)
			}
		}
	})

	t.Run("i18n strict mode error surfaces blocking DX029 findings", func(t *testing.T) {
		root := t.TempDir()
		writeDoctorFixture(t, root)
		if err := os.WriteFile(filepath.Join(root, ".env"), []byte("PAGODA_I18N_STRICT_MODE=error\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		controllerPath := filepath.Join(root, "app", "web", "controllers", "sample.go")
		controllerBody := "package controllers\nfunc demo() string {\n\treturn \"Welcome users\"\n}\n"
		if err := os.WriteFile(controllerPath, []byte(controllerBody), 0o644); err != nil {
			t.Fatal(err)
		}

		issues := RunDoctorChecks(root)
		mustContainIssueCode(t, issues, "DX029")
		for _, issue := range issues {
			if issue.Code == "DX029" && issue.Severity != "error" {
				t.Fatalf("expected DX029 error severity in error mode, got %+v", issue)
			}
		}
	})

	t.Run("i18n strict mode allowlist suppresses findings", func(t *testing.T) {
		root := t.TempDir()
		writeDoctorFixture(t, root)
		if err := os.WriteFile(filepath.Join(root, ".env"), []byte("PAGODA_I18N_STRICT_MODE=error\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		controllerPath := filepath.Join(root, "app", "web", "controllers", "sample.go")
		controllerBody := "package controllers\nfunc demo() string {\n\treturn \"Welcome users\"\n}\n"
		if err := os.WriteFile(controllerPath, []byte(controllerBody), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, ".i18n-allowlist"), []byte("app/web/controllers/sample.go:3\n"), 0o644); err != nil {
			t.Fatal(err)
		}

		issues := RunDoctorChecks(root)
		mustNotContainIssueCode(t, issues, "DX029")
	})

	t.Run("i18n strict mode stable allowlist survives line shifts", func(t *testing.T) {
		root := t.TempDir()
		writeDoctorFixture(t, root)
		if err := os.WriteFile(filepath.Join(root, ".env"), []byte("PAGODA_I18N_STRICT_MODE=error\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		controllerPath := filepath.Join(root, "app", "web", "controllers", "sample.go")
		controllerBody := "package controllers\nfunc demo() string {\n\treturn \"Welcome users\"\n}\n"
		if err := os.WriteFile(controllerPath, []byte(controllerBody), 0o644); err != nil {
			t.Fatal(err)
		}

		initialIssues := RunDoctorChecks(root)
		var stableID string
		for _, issue := range initialIssues {
			if issue.Code != "DX029" {
				continue
			}
			marker := "stable "
			start := strings.Index(issue.Message, marker)
			if start < 0 {
				continue
			}
			start += len(marker)
			end := strings.Index(issue.Message[start:], ")")
			if end < 0 {
				continue
			}
			stableID = issue.Message[start : start+end]
			break
		}
		if stableID == "" {
			t.Fatalf("expected stable i18n issue ID in DX029 message, issues=%+v", initialIssues)
		}
		if err := os.WriteFile(filepath.Join(root, ".i18n-allowlist"), []byte(stableID+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}

		shiftedControllerBody := "package controllers\n\nfunc demo() string {\n\treturn \"Welcome users\"\n}\n"
		if err := os.WriteFile(controllerPath, []byte(shiftedControllerBody), 0o644); err != nil {
			t.Fatal(err)
		}

		issues := RunDoctorChecks(root)
		mustNotContainIssueCode(t, issues, "DX029")
	})

	t.Run("i18n strict mode warn surfaces plural/select completeness findings", func(t *testing.T) {
		root := t.TempDir()
		writeDoctorFixture(t, root)
		if err := os.WriteFile(filepath.Join(root, ".env"), []byte("PAGODA_I18N_STRICT_MODE=warn\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Join(root, "locales"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, "locales", "en.toml"), []byte(`"cart.items.one" = "{{.Count}} item"`+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		controllerPath := filepath.Join(root, "app", "web", "controllers", "sample.go")
		controllerBody := "package controllers\nfunc demo() string {\n\t_ = container.I18n.TC(ctx, \"cart.items\", 2)\n\treturn \"\"\n}\n"
		if err := os.WriteFile(controllerPath, []byte(controllerBody), 0o644); err != nil {
			t.Fatal(err)
		}

		issues := RunDoctorChecks(root)
		found := false
		for _, issue := range issues {
			if issue.Code == "DX029" && strings.Contains(issue.Message, "plural_missing_other") {
				found = true
				if issue.Severity != "warning" {
					t.Fatalf("expected warning severity for completeness finding, got %+v", issue)
				}
			}
		}
		if !found {
			t.Fatalf("expected DX029 plural completeness finding, issues=%+v", issues)
		}
	})

	t.Run("i18n strict mode error blocks on completeness findings", func(t *testing.T) {
		root := t.TempDir()
		writeDoctorFixture(t, root)
		if err := os.WriteFile(filepath.Join(root, ".env"), []byte("PAGODA_I18N_STRICT_MODE=error\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Join(root, "locales"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, "locales", "en.toml"), []byte(`"profile.role.admin" = "Administrator"`+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		controllerPath := filepath.Join(root, "app", "web", "controllers", "sample.go")
		controllerBody := "package controllers\nfunc demo() string {\n\t_ = container.I18n.TS(ctx, \"profile.role\", \"admin\")\n\treturn \"\"\n}\n"
		if err := os.WriteFile(controllerPath, []byte(controllerBody), 0o644); err != nil {
			t.Fatal(err)
		}

		issues := RunDoctorChecks(root)
		found := false
		for _, issue := range issues {
			if issue.Code == "DX029" && strings.Contains(issue.Message, "select_missing_other") {
				found = true
				if issue.Severity != "error" {
					t.Fatalf("expected error severity for completeness finding, got %+v", issue)
				}
			}
		}
		if !found {
			t.Fatalf("expected DX029 select completeness finding, issues=%+v", issues)
		}
	})

	t.Run("renders comment suppresses warning", func(t *testing.T) {
		root := t.TempDir()
		writeDoctorFixture(t, root)
		targetDir := filepath.Join(root, "app", "views", "web", "components")
		if err := os.MkdirAll(targetDir, 0o755); err != nil {
			t.Fatal(err)
		}
		target := filepath.Join(targetDir, "describe.templ")
		content := "// Renders: sample component\ntempl DescribeComponent() {}\n"
		if err := os.WriteFile(target, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		issues := RunDoctorChecks(root)
		mustNotContainIssueCode(t, issues, "DX023")
	})

	t.Run("module renders warning ignored when module is disabled", func(t *testing.T) {
		root := t.TempDir()
		writeDoctorFixture(t, root)
		targetDir := filepath.Join(root, "modules", "local", "views", "web", "components")
		if err := os.MkdirAll(targetDir, 0o755); err != nil {
			t.Fatal(err)
		}
		target := filepath.Join(targetDir, "missing.templ")
		if err := os.WriteFile(target, []byte("templ MissingModuleComponent() {}\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		issues := RunDoctorChecks(root)
		mustNotContainIssueCode(t, issues, "DX023")
	})

	t.Run("module renders warning reported when module is enabled", func(t *testing.T) {
		root := t.TempDir()
		writeDoctorFixture(t, root)
		if err := os.WriteFile(filepath.Join(root, "config", "modules.yaml"), []byte("modules:\n  - local\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		targetDir := filepath.Join(root, "modules", "local", "views", "web", "components")
		if err := os.MkdirAll(targetDir, 0o755); err != nil {
			t.Fatal(err)
		}
		target := filepath.Join(targetDir, "missing.templ")
		if err := os.WriteFile(target, []byte("templ MissingModuleComponent() {}\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		issues := RunDoctorChecks(root)
		mustContainIssueCode(t, issues, "DX023")
	})

	t.Run("missing data-component triggers warning", func(t *testing.T) {
		root := t.TempDir()
		writeDoctorFixture(t, root)
		targetDir := filepath.Join(root, "app", "views", "web", "components")
		if err := os.MkdirAll(targetDir, 0o755); err != nil {
			t.Fatal(err)
		}
		target := filepath.Join(targetDir, "missing_attr.templ")
		if err := os.WriteFile(target, []byte("templ MissingAttr() {\n<div></div>\n}\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		issues := RunDoctorChecks(root)
		mustContainIssueCode(t, issues, "DX024")
	})

	t.Run("wrong data-component value warns", func(t *testing.T) {
		root := t.TempDir()
		writeDoctorFixture(t, root)
		targetDir := filepath.Join(root, "app", "views", "web", "components")
		if err := os.MkdirAll(targetDir, 0o755); err != nil {
			t.Fatal(err)
		}
		target := filepath.Join(targetDir, "wrong_attr.templ")
		content := "templ WrongAttr() {\n<div data-component=\"wrong-name\"></div>\n}\n"
		if err := os.WriteFile(target, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		issues := RunDoctorChecks(root)
		mustContainIssueCode(t, issues, "DX024")
	})

	t.Run("layout template is excluded from data-component check", func(t *testing.T) {
		root := t.TempDir()
		writeDoctorFixture(t, root)
		targetDir := filepath.Join(root, "app", "views", "web", "components")
		if err := os.MkdirAll(targetDir, 0o755); err != nil {
			t.Fatal(err)
		}
		target := filepath.Join(targetDir, "example_layout.templ")
		content := "templ ExampleLayout() {\n<div></div>\n}\n"
		if err := os.WriteFile(target, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		issues := RunDoctorChecks(root)
		mustNotContainIssueCode(t, issues, "DX024")
	})

	t.Run("module component data-component warning only when module is enabled", func(t *testing.T) {
		root := t.TempDir()
		writeDoctorFixture(t, root)
		targetDir := filepath.Join(root, "modules", "local", "views", "web", "components")
		if err := os.MkdirAll(targetDir, 0o755); err != nil {
			t.Fatal(err)
		}
		target := filepath.Join(targetDir, "missing_attr.templ")
		if err := os.WriteFile(target, []byte("templ MissingAttr() {\n<div></div>\n}\n"), 0o644); err != nil {
			t.Fatal(err)
		}

		issues := RunDoctorChecks(root)
		mustNotContainIssueCode(t, issues, "DX024")

		if err := os.WriteFile(filepath.Join(root, "config", "modules.yaml"), []byte("modules:\n  - local\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		issues = RunDoctorChecks(root)
		mustContainIssueCode(t, issues, "DX024")
	})

	t.Run("missing data-component on web layout triggers warning", func(t *testing.T) {
		root := t.TempDir()
		writeDoctorFixture(t, root)
		targetDir := filepath.Join(root, "app", "views", "web", "layouts")
		if err := os.MkdirAll(targetDir, 0o755); err != nil {
			t.Fatal(err)
		}
		target := filepath.Join(targetDir, "missing.templ")
		if err := os.WriteFile(target, []byte("templ MissingLayout() {\n<div></div>\n}\n"), 0o644); err != nil {
			t.Fatal(err)
		}

		issues := RunDoctorChecks(root)
		mustContainIssueCode(t, issues, "DX024")
	})

	t.Run("root binary artifact present", func(t *testing.T) {
		root := t.TempDir()
		writeDoctorFixture(t, root)
		if err := os.WriteFile(filepath.Join(root, "web"), []byte("binary"), 0o644); err != nil {
			t.Fatal(err)
		}
		issues := RunDoctorChecks(root)
		mustContainIssueCode(t, issues, "DX008")
	})

	t.Run("gitignore missing root artifact entries", func(t *testing.T) {
		root := t.TempDir()
		writeDoctorFixture(t, root)
		if err := os.WriteFile(filepath.Join(root, ".gitignore"), []byte("/web\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		issues := RunDoctorChecks(root)
		mustContainIssueCode(t, issues, "DX009")
	})

	t.Run("new large go file triggers file size issue", func(t *testing.T) {
		root := t.TempDir()
		writeDoctorFixture(t, root)
		path := filepath.Join(root, "app", "web", "ui", "too_big.go")
		var b strings.Builder
		b.WriteString("package ui\n")
		for i := 0; i < 820; i++ {
			b.WriteString("var _ = 1\n")
		}
		if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
			t.Fatal(err)
		}
		issues := RunDoctorChecks(root)
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
		issues := RunDoctorChecks(root)
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
		issues := RunDoctorChecks(root)
		mustContainIssueCode(t, issues, "DX012")
	})

	t.Run("canonical framework repo cli docs reject stale internal shell links", func(t *testing.T) {
		root := t.TempDir()
		writeCanonicalRepoFixture(t, root)
		cliDoc := filepath.Join(root, "docs", "reference", "01-cli.md")
		if err := os.MkdirAll(filepath.Dir(cliDoc), 0o755); err != nil {
			t.Fatal(err)
		}
		content := strings.Join([]string{
			"## Minimal V1 Command Set",
			"## Implementation Mapping (Current Repo)",
			"## Generator test strategy",
			"ship doctor",
			"ship verify",
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
			"extension-zone manifest",
			"`container.go`",
			"`router.go`",
			"`schedules.go`",
			"`app/foundation/container.go`",
			"",
		}, "\n")
		if err := os.WriteFile(cliDoc, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}

		issues := checkCLIDocsCoverage(root)
		if !containsDoctorIssueMessage(issues, "cli docs contain stale framework-shell link token") {
			t.Fatalf("expected stale framework-shell link issue, got %+v", issues)
		}
	})

	t.Run("canonical docs reject transition-era wording red spec", func(t *testing.T) {
		root := t.TempDir()
		writeDoctorFixture(t, root)

		for rel, content := range map[string]string{
			filepath.Join("docs", "architecture", "01-architecture.md"):                "compatibility window\n",
			filepath.Join("docs", "architecture", "02-structure-and-boundaries.md"):    "deprecated alias\n",
			filepath.Join("docs", "architecture", "03-project-scope-analysis.md"):      "active transitional state\n",
			filepath.Join("docs", "architecture", "07-core-interfaces.md"):             "deprecation period\n",
			filepath.Join("docs", "architecture", "09-standalone-and-managed-mode.md"): "legacy compatibility path\n",
			filepath.Join("docs", "reference", "01-cli.md"):                            "legacy compatibility path\n",
			filepath.Join("docs", "roadmap", "01-framework-plan.md"):                   "transition-era fallback\n",
		} {
			path := filepath.Join(root, rel)
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
				t.Fatal(err)
			}
		}

		issues := RunDoctorChecks(root)
		mustContainIssueCode(t, issues, "DX030")
		if !containsDoctorIssueMessage(issues, "docs/architecture/01-architecture.md:1") {
			t.Fatalf("expected DX030 to cover canonical architecture docs, got %+v", issues)
		}
		if !containsDoctorIssueMessage(issues, "docs/architecture/03-project-scope-analysis.md:1") {
			t.Fatalf("expected DX030 to include file:line diagnostics, got %+v", issues)
		}
	})

	t.Run("noncanonical historical references can be allowlisted", func(t *testing.T) {
		root := t.TempDir()
		writeDoctorFixture(t, root)

		historyDoc := filepath.Join(root, "docs", "guides", "99-history.md")
		if err := os.MkdirAll(filepath.Dir(historyDoc), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(historyDoc, []byte("This guide keeps a transition-era note.\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		allowlistPath := filepath.Join(root, "docs", "policies", "02-transition-wording-allowlist.txt")
		if err := os.MkdirAll(filepath.Dir(allowlistPath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(allowlistPath, []byte("docs/guides/99-history.md|transition-era\n"), 0o644); err != nil {
			t.Fatal(err)
		}

		issues := RunDoctorChecks(root)
		if containsDoctorIssueMessage(issues, "docs/guides/99-history.md:1") {
			t.Fatalf("allowlisted historical reference should not produce DX030, got %+v", issues)
		}
	})

	t.Run("canonical architecture docs ignore allowlist entries", func(t *testing.T) {
		root := t.TempDir()
		writeDoctorFixture(t, root)

		docPath := filepath.Join(root, "docs", "architecture", "06-known-gaps-and-risks.md")
		if err := os.MkdirAll(filepath.Dir(docPath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(docPath, []byte("This architecture note mentions transition-era history.\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		allowlistPath := filepath.Join(root, "docs", "policies", "02-transition-wording-allowlist.txt")
		if err := os.MkdirAll(filepath.Dir(allowlistPath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(allowlistPath, []byte("docs/architecture/06-known-gaps-and-risks.md|transition-era\n"), 0o644); err != nil {
			t.Fatal(err)
		}

		issues := RunDoctorChecks(root)
		if !containsDoctorIssueMessage(issues, "docs/architecture/06-known-gaps-and-risks.md:1") {
			t.Fatalf("canonical architecture docs should ignore allowlist entries, got %+v", issues)
		}
	})

	t.Run("go.work references missing module", func(t *testing.T) {
		root := t.TempDir()
		writeDoctorFixture(t, root)
		if err := os.WriteFile(filepath.Join(root, "go.work"), []byte("go 1.25\n\nuse (\n\t.\n\t./missing-module\n)\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		issues := RunDoctorChecks(root)
		mustContainIssueCode(t, issues, "DX014")
	})

	t.Run("dockerignore missing required exclusion", func(t *testing.T) {
		root := t.TempDir()
		writeDoctorFixture(t, root)
		if err := os.WriteFile(filepath.Join(root, ".dockerignore"), []byte(".git\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		issues := RunDoctorChecks(root)
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
		issues := RunDoctorChecks(root)
		mustContainIssueCode(t, issues, "DX016")
	})

	t.Run("agent policy artifact drift", func(t *testing.T) {
		root := t.TempDir()
		writeDoctorFixture(t, root)
		if err := os.WriteFile(filepath.Join(root, "tools", "agent-policy", "generated", "codex-prefixes.txt"), []byte("stale\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		issues := RunDoctorChecks(root)
		mustContainIssueCode(t, issues, "DX017")
	})

	t.Run("invalid modules manifest format", func(t *testing.T) {
		root := t.TempDir()
		writeDoctorFixture(t, root)
		if err := os.WriteFile(filepath.Join(root, "config", "modules.yaml"), []byte("modules:\n  - bad/name\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		issues := RunDoctorChecks(root)
		mustContainIssueCode(t, issues, "DX018")
	})

	t.Run("enabled module missing db artifacts", func(t *testing.T) {
		root := t.TempDir()
		writeDoctorFixture(t, root)
		if err := os.WriteFile(filepath.Join(root, "config", "modules.yaml"), []byte("modules:\n  - local\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		issues := RunDoctorChecks(root)
		mustContainIssueCode(t, issues, "DX019")
	})

	t.Run("enabled module has db artifacts", func(t *testing.T) {
		root := t.TempDir()
		writeDoctorFixture(t, root)
		if err := os.WriteFile(filepath.Join(root, "config", "modules.yaml"), []byte("modules:\n  - local\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Join(root, "modules", "local", "db", "migrate", "migrations"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, "modules", "local", "db", "bobgen.yaml"), []byte("dialect: sqlite\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		issues := RunDoctorChecks(root)
		for _, issue := range issues {
			if issue.Code == "DX019" {
				t.Fatalf("unexpected DX019 issue: %+v", issue)
			}
		}
	})

}

func findGoModuleTest(start string) (string, string, error) {
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

func writeCanonicalFrameworkDoctorFixture(t *testing.T, root string) {
	t.Helper()

	writeCanonicalRepoFixture(t, root)
	dirs := []string{
		filepath.Join(root, "db", "queries"),
		filepath.Join(root, "db", "migrate", "migrations"),
		filepath.Join(root, "docs", "architecture"),
		filepath.Join(root, "docs", "reference"),
		filepath.Join(root, "tools", "agent-policy", "generated"),
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	files := map[string]string{
		filepath.Join(root, "db", "bobgen.yaml"):                             "packages: []\n",
		filepath.Join(root, "config", "modules.yaml"):                        "modules: []\n",
		filepath.Join(root, "docs", "00-index.md"):                           "# Index\n",
		filepath.Join(root, "docs", "architecture", "01-architecture.md"):    "# Architecture\n",
		filepath.Join(root, "docs", "architecture", "08-cognitive-model.md"): "# Cognitive Model\n",
		filepath.Join(root, "docs", "architecture", "10-extension-zones.md"): strings.Join([]string{
			"## Extension Zones",
			"- `app/`",
			"",
			"## Protected Contract Zones",
			"- `framework/`",
			"- `app/router.go`",
			"- `app/foundation/container.go`",
			"- `config/modules.yaml`",
			"- `tools/agent-policy/allowed-commands.yaml`",
			"",
		}, "\n"),
		filepath.Join(root, "docs", "reference", "01-cli.md"): strings.Join([]string{
			"## Minimal V1 Command Set",
			"## Implementation Mapping (Current Repo)",
			"## Generator test strategy",
			"ship doctor",
			"ship verify",
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
			".local",
			"tools/scripts/venv",
			"",
		}, "\n"),
	}
	for path, content := range files {
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func writeDoctorFixture(t *testing.T, root string) {
	t.Helper()

	dirs := []string{
		filepath.Join(root, "app", "app"),
		filepath.Join(root, "app", "foundation"),
		filepath.Join(root, "app", "web", "controllers"),
		filepath.Join(root, "app", "web", "middleware"),
		filepath.Join(root, "app", "web", "ui"),
		filepath.Join(root, "app", "web", "viewmodels"),
		filepath.Join(root, "app", "web", "routenames"),
		filepath.Join(root, "app", "jobs"),
		filepath.Join(root, "app", "views"),
		filepath.Join(root, "db", "queries"),
		filepath.Join(root, "db", "migrate", "migrations"),
		filepath.Join(root, "config"),
		filepath.Join(root, "docs", "architecture"),
		filepath.Join(root, "docs", "reference"),
		filepath.Join(root, "infra", "docker"),
		filepath.Join(root, "modules", "local"),
		filepath.Join(root, "app"),
		filepath.Join(root, "tools", "agent-policy", "generated"),
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	files := map[string]string{
		filepath.Join(root, "app", "router.go"): `package goship

func registerPublicRoutes() {
	// ship:routes:public:start
	// ship:routes:public:end
}

func registerAuthRoutes() {
	// ship:routes:auth:start
	// ship:routes:auth:end
}

func registerExternalRoutes() {
	// ship:routes:external:start
	// ship:routes:external:end
}
`,
		filepath.Join(root, "app", "foundation", "container.go"): `package foundation

func NewContainer() *Container {
	c := &Container{}
	// ship:container:start
	// ship:container:end
	return c
}

type Container struct{}
`,
		filepath.Join(root, "app", "web", "ui", "page.go"):                "package ui\n",
		filepath.Join(root, "app", "web", "viewmodels", "page_data.go"):   "package viewmodels\n",
		filepath.Join(root, "app", "web", "routenames", "routenames.go"):  "package routenames\n",
		filepath.Join(root, "db", "bobgen.yaml"):                          "packages: []\n",
		filepath.Join(root, "config", "modules.yaml"):                     "modules: []\n",
		filepath.Join(root, "docs", "00-index.md"):                        "# Index\n",
		filepath.Join(root, "docs", "architecture", "01-architecture.md"): "# Architecture\n",
		filepath.Join(root, "docs", "architecture", "10-extension-zones.md"): strings.Join([]string{
			"## Extension Zones",
			"- `app/`",
			"",
			"## Protected Contract Zones",
			"- `framework/`",
			"- `app/router.go`",
			"- `app/foundation/container.go`",
			"- `config/modules.yaml`",
			"- `tools/agent-policy/allowed-commands.yaml`",
			"",
		}, "\n"),
		filepath.Join(root, "docs", "architecture", "08-cognitive-model.md"): "# Cognitive Model\n",
		filepath.Join(root, "docs", "reference", "01-cli.md"): strings.Join([]string{
			"## Minimal V1 Command Set",
			"## Implementation Mapping (Current Repo)",
			"## Generator test strategy",
			"ship doctor",
			"ship verify",
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
			"extension-zone manifest",
			"`container.go`",
			"`router.go`",
			"`schedules.go`",
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
			".local",
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
		filepath.Join(root, "app", "go.mod"): strings.Join([]string{
			"module example.com/app",
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
			"\t./app",
			")",
			"",
		}, "\n"),
		filepath.Join(root, "infra", "docker", "Dockerfile"): strings.Join([]string{
			"FROM golang:1.25.6 AS builder",
			"WORKDIR /app",
			"COPY . .",
			"WORKDIR /app/app",
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

func mustContainIssueCode(t *testing.T, issues []DoctorIssue, code string) {
	t.Helper()
	for _, issue := range issues {
		if issue.Code == code {
			return
		}
	}
	t.Fatalf("expected issue code %s, got %+v", code, issues)
}

func containsDoctorIssueMessage(issues []DoctorIssue, needle string) bool {
	for _, issue := range issues {
		if strings.Contains(issue.Message, needle) || strings.Contains(issue.File, needle) {
			return true
		}
	}
	return false
}

func mustNotContainIssueCode(t *testing.T, issues []DoctorIssue, code string) {
	t.Helper()
	for _, issue := range issues {
		if issue.Code == code {
			t.Fatalf("did not expect issue code %s, got %+v", code, issues)
		}
	}
}
