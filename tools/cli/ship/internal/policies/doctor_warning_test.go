package policies

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDoctorCommand_WarningsOnly(t *testing.T) {
	root := t.TempDir()
	writeDoctorFixture(t, root)
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/doctor\n\ngo 1.25\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	warnPath := filepath.Join(root, "app", "web", "ui", "warn.go")
	writeSizedGoFile(t, warnPath, "package ui\n", 320)

	prevWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(prevWD) })
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}

	t.Run("human output keeps exit zero and prints warning", func(t *testing.T) {
		out := &bytes.Buffer{}
		errOut := &bytes.Buffer{}
		if code := RunDoctor([]string{}, DoctorDeps{Out: out, Err: errOut, FindGoModule: findGoModuleTest}); code != 0 {
			t.Fatalf("doctor exit code = %d, want 0", code)
		}
		if errOut.Len() != 0 {
			t.Fatalf("stderr = %q, want empty stderr", errOut.String())
		}
		if !strings.Contains(out.String(), "warning") {
			t.Fatalf("stdout = %q, want warning output", out.String())
		}
	})

	t.Run("json output includes warning issue and ok=true", func(t *testing.T) {
		out := &bytes.Buffer{}
		errOut := &bytes.Buffer{}
		if code := RunDoctor([]string{"--json"}, DoctorDeps{Out: out, Err: errOut, FindGoModule: findGoModuleTest}); code != 0 {
			t.Fatalf("doctor exit code = %d, want 0", code)
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
		if len(payload.Issues) == 0 {
			t.Fatal("issues len = 0, want warning issue")
		}
		if payload.Issues[0].Severity != "warning" {
			t.Fatalf("severity = %q, want warning", payload.Issues[0].Severity)
		}
	})
}
