package policies

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunDoctor(t *testing.T) {
	t.Run("help", func(t *testing.T) {
		if code := RunDoctor([]string{"--help"}, DoctorDeps{Out: &bytes.Buffer{}, Err: &bytes.Buffer{}, FindGoModule: findGoModuleTest}); code != 0 {
			t.Fatalf("exit code = %d, want 0", code)
		}
	})

	t.Run("unexpected args", func(t *testing.T) {
		out := &bytes.Buffer{}
		errOut := &bytes.Buffer{}
		code := RunDoctor([]string{"extra"}, DoctorDeps{Out: out, Err: errOut, FindGoModule: findGoModuleTest})
		if code != 1 {
			t.Fatalf("exit code = %d, want 1", code)
		}
		if !strings.Contains(errOut.String(), "unexpected doctor arguments") {
			t.Fatalf("stderr = %q, want unexpected args message", errOut.String())
		}
	})

	t.Run("json output with unexpected args", func(t *testing.T) {
		out := &bytes.Buffer{}
		errOut := &bytes.Buffer{}
		code := RunDoctor([]string{"--json", "extra"}, DoctorDeps{Out: out, Err: errOut, FindGoModule: findGoModuleTest})
		if code != 1 {
			t.Fatalf("exit code = %d, want 1", code)
		}
		if errOut.Len() != 0 {
			t.Fatalf("stderr = %q, want empty stderr for json output", errOut.String())
		}

		var payload doctorJSONResult
		if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
			t.Fatalf("failed to decode json: %v", err)
		}
		if payload.OK {
			t.Fatalf("payload.OK = true, want false")
		}
		if len(payload.Issues) != 1 {
			t.Fatalf("issues len = %d, want 1", len(payload.Issues))
		}
		if payload.Issues[0].Type != "config" {
			t.Fatalf("issue type = %q, want config", payload.Issues[0].Type)
		}
		if payload.Issues[0].Severity != "error" {
			t.Fatalf("issue severity = %q, want error", payload.Issues[0].Severity)
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
	if code := RunDoctor([]string{}, DoctorDeps{Out: out, Err: errOut, FindGoModule: findGoModuleTest}); code != 0 {
		t.Fatalf("doctor exit code = %d, stderr=%s", code, errOut.String())
	}
	if !strings.Contains(out.String(), "ship doctor: OK") {
		t.Fatalf("stdout = %q, want doctor OK output", out.String())
	}
}

func TestDoctorCommand_JSONOutput(t *testing.T) {
	t.Run("ok fixture", func(t *testing.T) {
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
		if code := RunDoctor([]string{"--json"}, DoctorDeps{Out: out, Err: errOut, FindGoModule: findGoModuleTest}); code != 0 {
			t.Fatalf("doctor exit code = %d, stderr=%s", code, errOut.String())
		}
		if errOut.Len() != 0 {
			t.Fatalf("stderr = %q, want empty stderr", errOut.String())
		}

		var payload doctorJSONResult
		if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
			t.Fatalf("failed to decode json: %v", err)
		}
		if !payload.OK {
			t.Fatalf("payload.OK = false, want true")
		}
		if len(payload.Issues) != 0 {
			t.Fatalf("issues len = %d, want 0", len(payload.Issues))
		}
	})

	t.Run("fixture with issue", func(t *testing.T) {
		root := t.TempDir()
		writeDoctorFixture(t, root)
		if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/doctor\n\ngo 1.25\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.RemoveAll(filepath.Join(root, "app", "jobs")); err != nil {
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
		if code := RunDoctor([]string{"--json"}, DoctorDeps{Out: out, Err: errOut, FindGoModule: findGoModuleTest}); code != 1 {
			t.Fatalf("doctor exit code = %d, want 1", code)
		}
		if errOut.Len() != 0 {
			t.Fatalf("stderr = %q, want empty stderr", errOut.String())
		}

		var payload doctorJSONResult
		if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
			t.Fatalf("failed to decode json: %v", err)
		}
		if payload.OK {
			t.Fatalf("payload.OK = true, want false")
		}
		if len(payload.Issues) == 0 {
			t.Fatal("issues len = 0, want at least one issue")
		}
		if payload.Issues[0].Type == "" {
			t.Fatal("issue type = empty, want code")
		}
		if payload.Issues[0].Severity != "error" {
			t.Fatalf("issue severity = %q, want error", payload.Issues[0].Severity)
		}
	})
}
