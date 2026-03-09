package policies

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunDoctorChecks_FileSizes(t *testing.T) {
	t.Run("go file over warning threshold is warning", func(t *testing.T) {
		root := t.TempDir()
		writeDoctorFixture(t, root)
		path := filepath.Join(root, "app", "web", "ui", "warn.go")
		writeSizedGoFile(t, path, "package ui\n", 320)

		issues := RunDoctorChecks(root)
		issue := mustFindIssueCode(t, issues, "DX010")
		if issue.Severity != "warning" {
			t.Fatalf("severity = %q, want warning", issue.Severity)
		}
	})

	t.Run("go file over hard cap is error", func(t *testing.T) {
		root := t.TempDir()
		writeDoctorFixture(t, root)
	path := filepath.Join(root, "app", "web", "ui", "error.go")
	writeSizedGoFile(t, path, "package ui\n", 1105)

		issues := RunDoctorChecks(root)
		issue := mustFindIssueCode(t, issues, "DX010")
		if doctorIssueSeverity(issue) != "error" {
			t.Fatalf("severity = %q, want error", doctorIssueSeverity(issue))
		}
	})

	t.Run("templ file over warning threshold is warning", func(t *testing.T) {
		root := t.TempDir()
		writeDoctorFixture(t, root)
		path := filepath.Join(root, "app", "views", "web", "pages", "warn.templ")
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		writeSizedTemplFile(t, path, "templ Warn() {\n", 220)

		issues := RunDoctorChecks(root)
		issue := mustFindIssueCode(t, issues, "DX010")
		if issue.Severity != "warning" {
			t.Fatalf("severity = %q, want warning", issue.Severity)
		}
	})

	t.Run("templ file over hard cap is error", func(t *testing.T) {
		root := t.TempDir()
		writeDoctorFixture(t, root)
		path := filepath.Join(root, "app", "views", "web", "pages", "error.templ")
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		writeSizedTemplFile(t, path, "templ Error() {\n", 420)

		issues := RunDoctorChecks(root)
		issue := mustFindIssueCode(t, issues, "DX010")
		if doctorIssueSeverity(issue) != "error" {
			t.Fatalf("severity = %q, want error", doctorIssueSeverity(issue))
		}
	})

	t.Run("legacy oversized templ file is grandfathered as warning", func(t *testing.T) {
		root := t.TempDir()
		writeDoctorFixture(t, root)
		path := filepath.Join(root, "app", "views", "web", "pages", "preferences.templ")
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		writeSizedTemplFile(t, path, "templ Preferences() {\n", 420)

		issues := RunDoctorChecks(root)
		issue := mustFindIssueCode(t, issues, "DX010")
		if issue.Severity != "warning" {
			t.Fatalf("severity = %q, want warning", issue.Severity)
		}
		if !strings.Contains(issue.Message, "grandfathered") {
			t.Fatalf("message = %q, want grandfathered note", issue.Message)
		}
	})

	t.Run("generated go files are excluded", func(t *testing.T) {
		root := t.TempDir()
		writeDoctorFixture(t, root)
		for _, rel := range []string{
			filepath.Join("app", "web", "ui", "skip.templ.go"),
			filepath.Join("app", "web", "ui", "skip_sql.go"),
			filepath.Join("app", "web", "ui", "bob_skip.go"),
			filepath.Join("app", "web", "ui", "skip_test.go"),
		} {
			writeSizedGoFile(t, filepath.Join(root, rel), "package ui\n", 700)
		}

		issues := RunDoctorChecks(root)
		for _, issue := range issues {
			if issue.Code == "DX010" {
				t.Fatalf("unexpected DX010 issue for excluded generated/test file: %+v", issue)
			}
		}
	})
}

func writeSizedGoFile(t *testing.T, path string, header string, lines int) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	var b strings.Builder
	b.WriteString(header)
	for i := 0; i < lines; i++ {
		b.WriteString("var _ = 1\n")
	}
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeSizedTemplFile(t *testing.T, path string, header string, lines int) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	var b strings.Builder
	b.WriteString(header)
	for i := 0; i < lines; i++ {
		b.WriteString("  <div></div>\n")
	}
	b.WriteString("}\n")
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}
}

func mustFindIssueCode(t *testing.T, issues []DoctorIssue, code string) DoctorIssue {
	t.Helper()
	for _, issue := range issues {
		if issue.Code == code {
			return issue
		}
	}
	t.Fatalf("expected issue code %s, got %+v", code, issues)
	return DoctorIssue{}
}
