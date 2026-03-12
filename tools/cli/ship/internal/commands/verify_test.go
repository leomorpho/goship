package commands

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunVerify(t *testing.T) {
	t.Run("success with skipped nilaway and tests", func(t *testing.T) {
		root := t.TempDir()
		writeVerifyGoMod(t, root)
		prevWD := chdirVerifyRoot(t, root)
		t.Cleanup(func() { _ = os.Chdir(prevWD) })

		calls := make([]string, 0)
		deps := VerifyDeps{
			Out:          &bytes.Buffer{},
			Err:          &bytes.Buffer{},
			FindGoModule: findVerifyGoModule,
			RelocateTempl: func(rootPath string) error {
				if rootPath != "." {
					t.Fatalf("rootPath = %q, want .", rootPath)
				}
				return nil
			},
			RunStep: func(env []string, name string, args ...string) (int, string, error) {
				calls = append(calls, name+" "+strings.Join(args, " "))
				return 0, "ok", nil
			},
			LookPath: func(file string) (string, error) {
				if file == "templ" {
					return "/usr/bin/templ", nil
				}
				return "", errors.New("missing")
			},
			RunDoctor: func() (int, string, error) {
				return 0, `{"ok":true,"issues":[]}`, nil
			},
		}
		out := deps.Out.(*bytes.Buffer)
		if code := RunVerify([]string{"--skip-tests"}, deps); code != 0 {
			t.Fatalf("exit code = %d, want 0", code)
		}
		if len(calls) != 2 {
			t.Fatalf("calls len = %d, want 2", len(calls))
		}
		if !strings.Contains(out.String(), "verify passed") {
			t.Fatalf("stdout = %q, want success message", out.String())
		}
		if !strings.Contains(out.String(), "nilaway not installed; skipping") {
			t.Fatalf("stdout = %q, want nilaway skip message", out.String())
		}
		if !strings.Contains(out.String(), "skipped via --skip-tests") {
			t.Fatalf("stdout = %q, want skip-tests message", out.String())
		}
	})

	t.Run("fails fast on subprocess failure", func(t *testing.T) {
		root := t.TempDir()
		writeVerifyGoMod(t, root)
		prevWD := chdirVerifyRoot(t, root)
		t.Cleanup(func() { _ = os.Chdir(prevWD) })

		calls := make([]string, 0)
		out := &bytes.Buffer{}
		errOut := &bytes.Buffer{}
		code := RunVerify([]string{}, VerifyDeps{
			Out:          out,
			Err:          errOut,
			FindGoModule: findVerifyGoModule,
			RelocateTempl: func(rootPath string) error {
				if rootPath != "." {
					t.Fatalf("rootPath = %q, want .", rootPath)
				}
				return nil
			},
			RunStep: func(env []string, name string, args ...string) (int, string, error) {
				calls = append(calls, name+" "+strings.Join(args, " "))
				if len(calls) == 2 {
					return 1, "compile failed", nil
				}
				return 0, "ok", nil
			},
			LookPath: func(file string) (string, error) {
				return "/usr/bin/" + file, nil
			},
			RunDoctor: func() (int, string, error) {
				t.Fatal("doctor should not run after build failure")
				return 0, "", nil
			},
		})
		if code != 1 {
			t.Fatalf("exit code = %d, want 1", code)
		}
		if len(calls) != 2 {
			t.Fatalf("calls len = %d, want 2", len(calls))
		}
		if !strings.Contains(errOut.String(), "verify failed at go build ./...") {
			t.Fatalf("stderr = %q, want build failure step", errOut.String())
		}
		if !strings.Contains(errOut.String(), "compile failed") {
			t.Fatalf("stderr = %q, want subprocess output", errOut.String())
		}
	})

	t.Run("json output returns structured steps", func(t *testing.T) {
		root := t.TempDir()
		writeVerifyGoMod(t, root)
		prevWD := chdirVerifyRoot(t, root)
		t.Cleanup(func() { _ = os.Chdir(prevWD) })

		out := &bytes.Buffer{}
		errOut := &bytes.Buffer{}
		code := RunVerify([]string{"--json"}, VerifyDeps{
			Out:          out,
			Err:          errOut,
			FindGoModule: findVerifyGoModule,
			RelocateTempl: func(rootPath string) error {
				if rootPath != "." {
					t.Fatalf("rootPath = %q, want .", rootPath)
				}
				return nil
			},
			RunStep: func(env []string, name string, args ...string) (int, string, error) {
				return 0, name + " ok", nil
			},
			LookPath: func(file string) (string, error) {
				return "/usr/bin/" + file, nil
			},
			RunDoctor: func() (int, string, error) {
				return 0, `{"ok":true,"issues":[]}`, nil
			},
		})
		if code != 0 {
			t.Fatalf("exit code = %d, want 0", code)
		}
		if errOut.Len() != 0 {
			t.Fatalf("stderr = %q, want empty", errOut.String())
		}

		var payload verifyJSONResult
		if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
			t.Fatalf("decode json: %v", err)
		}
		if !payload.OK {
			t.Fatalf("payload.OK = false, want true")
		}
		if len(payload.Steps) != 5 {
			t.Fatalf("steps len = %d, want 5", len(payload.Steps))
		}
		if payload.Steps[2].Name != "ship doctor --json" {
			t.Fatalf("doctor step name = %q, want ship doctor --json", payload.Steps[2].Name)
		}
	})

	t.Run("fails when templ relocation fails", func(t *testing.T) {
		root := t.TempDir()
		writeVerifyGoMod(t, root)
		prevWD := chdirVerifyRoot(t, root)
		t.Cleanup(func() { _ = os.Chdir(prevWD) })

		out := &bytes.Buffer{}
		errOut := &bytes.Buffer{}
		code := RunVerify([]string{}, VerifyDeps{
			Out:          out,
			Err:          errOut,
			FindGoModule: findVerifyGoModule,
			RelocateTempl: func(rootPath string) error {
				return errors.New("relocate failed")
			},
			RunStep: func(env []string, name string, args ...string) (int, string, error) {
				return 0, "ok", nil
			},
			LookPath: func(file string) (string, error) {
				return "/usr/bin/" + file, nil
			},
			RunDoctor: func() (int, string, error) {
				t.Fatal("doctor should not run after templ relocation failure")
				return 0, "", nil
			},
		})
		if code != 1 {
			t.Fatalf("exit code = %d, want 1", code)
		}
		if !strings.Contains(errOut.String(), "verify failed at templ generate") {
			t.Fatalf("stderr = %q, want templ generate failure step", errOut.String())
		}
		if !strings.Contains(errOut.String(), "relocate failed") {
			t.Fatalf("stderr = %q, want relocation failure output", errOut.String())
		}
	})
}

func writeVerifyGoMod(t *testing.T, root string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/verify\n\ngo 1.25\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func chdirVerifyRoot(t *testing.T, root string) string {
	t.Helper()
	prevWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	return prevWD
}

func findVerifyGoModule(start string) (string, string, error) {
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
