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
	writeSizedGoFile(t, warnPath, "package ui\n", 820)

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
		if code := RunDoctor([]string{}, doctorDepsForTest(out, errOut)); code != 0 {
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
		if code := RunDoctor([]string{"--json"}, doctorDepsForTest(out, errOut)); code != 0 {
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

func TestDoctorCommand_NilawayWarningsAreNonBlocking(t *testing.T) {
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
	code := RunDoctor([]string{"--json"}, DoctorDeps{
		Out:          out,
		Err:          errOut,
		FindGoModule: findGoModuleTest,
		LookPath: func(string) (string, error) {
			return "/usr/bin/nilaway", nil
		},
		RunCmd: func(dir string, name string, args ...string) (int, string, error) {
			return 1, filepath.Join(dir, "app", "router.go") + ":10:5: possible nil panic", nil
		},
	})
	if code != 0 {
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
	found := false
	for _, issue := range payload.Issues {
		if issue.Type == "DX025" {
			found = true
			if issue.Severity != "warning" {
				t.Fatalf("severity = %q, want warning", issue.Severity)
			}
			if issue.File != "app/router.go" {
				t.Fatalf("file = %q, want app/router.go", issue.File)
			}
		}
	}
	if !found {
		t.Fatalf("issues = %+v, want DX025 nilaway warning", payload.Issues)
	}
}
