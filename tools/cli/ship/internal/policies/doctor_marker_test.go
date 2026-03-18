package policies

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunDoctorChecks_MarkerIntegrity(t *testing.T) {
	t.Run("missing container markers is an error", func(t *testing.T) {
		root := t.TempDir()
		writeDoctorFixture(t, root)
		path := filepath.Join(root, "app", "foundation", "container.go")
		if err := os.WriteFile(path, []byte("package foundation\n"), 0o644); err != nil {
			t.Fatal(err)
		}

		issues := RunDoctorChecks(root)
		issue := mustFindIssueCode(t, issues, "DX005")
		if doctorIssueSeverity(issue) != "error" {
			t.Fatalf("severity = %q, want error", doctorIssueSeverity(issue))
		}
	})

	t.Run("unpaired container marker is an error", func(t *testing.T) {
		root := t.TempDir()
		writeDoctorFixture(t, root)
		path := filepath.Join(root, "app", "foundation", "container.go")
		content := `package foundation

func NewContainer() *Container {
	c := &Container{}
	// ship:container:start
	return c
}

type Container struct{}
`
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}

		issues := RunDoctorChecks(root)
		issue := mustFindIssueCode(t, issues, "DX005")
		if doctorIssueSeverity(issue) != "error" {
			t.Fatalf("severity = %q, want error", doctorIssueSeverity(issue))
		}
	})
}

func TestRunDoctorChecks_UnpairedMarkerWillBecomeError_RedSpec(t *testing.T) {
	root := t.TempDir()
	writeDoctorFixture(t, root)
	path := filepath.Join(root, "app", "foundation", "container.go")
	content := `package foundation

func NewContainer() *Container {
	c := &Container{}
	// ship:container:start
	return c
}

type Container struct{}
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	issues := RunDoctorChecks(root)
	issue := mustFindIssueCode(t, issues, "DX005")
	if doctorIssueSeverity(issue) != "error" {
		t.Fatalf("severity = %q, want error", doctorIssueSeverity(issue))
	}
}
