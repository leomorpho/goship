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
			RunStep: func(name string, args ...string) (int, string, error) {
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

	t.Run("fast profile skips nilaway and tests", func(t *testing.T) {
		root := t.TempDir()
		writeVerifyGoMod(t, root)
		prevWD := chdirVerifyRoot(t, root)
		t.Cleanup(func() { _ = os.Chdir(prevWD) })

		calls := make([]string, 0)
		out := &bytes.Buffer{}
		errOut := &bytes.Buffer{}
		code := RunVerify([]string{"--profile", "fast"}, VerifyDeps{
			Out:          out,
			Err:          errOut,
			FindGoModule: findVerifyGoModule,
			RelocateTempl: func(rootPath string) error {
				return nil
			},
			RunStep: func(name string, args ...string) (int, string, error) {
				calls = append(calls, name+" "+strings.Join(args, " "))
				return 0, "ok", nil
			},
			LookPath: func(file string) (string, error) {
				t.Fatalf("fast profile should not resolve %s", file)
				return "", nil
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
		if len(calls) != 2 {
			t.Fatalf("calls len = %d, want 2", len(calls))
		}
		for _, token := range []string{"skipped in fast profile", "verify passed"} {
			if !strings.Contains(out.String(), token) {
				t.Fatalf("stdout missing %q:\n%s", token, out.String())
			}
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
			RunStep: func(name string, args ...string) (int, string, error) {
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
		writeVerifyGoWork(t, root)
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
			RunStep: func(name string, args ...string) (int, string, error) {
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
		if len(payload.Steps) != 9 {
			t.Fatalf("steps len = %d, want 9", len(payload.Steps))
		}
		if payload.Steps[2].Name != "ship doctor --json" {
			t.Fatalf("doctor step name = %q, want ship doctor --json", payload.Steps[2].Name)
		}
		if payload.Steps[3].Name != "no-compatibility/deprecation invariant" {
			t.Fatalf("wording step name = %q, want no-compatibility/deprecation invariant", payload.Steps[3].Name)
		}
		if payload.Steps[4].Name != "module compatibility policy" {
			t.Fatalf("module compatibility step name = %q, want module compatibility policy", payload.Steps[4].Name)
		}
		if payload.Steps[7].Name != "standalone exportability gate" {
			t.Fatalf("step 8 name = %q, want standalone exportability gate", payload.Steps[7].Name)
		}
		if payload.Steps[8].Name != "orchestration contract mismatch preflight" {
			t.Fatalf("final step name = %q, want orchestration contract mismatch preflight", payload.Steps[8].Name)
		}
	})

	t.Run("strict profile requires nilaway", func(t *testing.T) {
		root := t.TempDir()
		writeVerifyGoMod(t, root)
		prevWD := chdirVerifyRoot(t, root)
		t.Cleanup(func() { _ = os.Chdir(prevWD) })

		out := &bytes.Buffer{}
		errOut := &bytes.Buffer{}
		code := RunVerify([]string{"--profile", "strict"}, VerifyDeps{
			Out:          out,
			Err:          errOut,
			FindGoModule: findVerifyGoModule,
			RelocateTempl: func(rootPath string) error {
				return nil
			},
			RunStep: func(name string, args ...string) (int, string, error) {
				return 0, "ok", nil
			},
			LookPath: func(file string) (string, error) {
				return "", errors.New("missing")
			},
			RunDoctor: func() (int, string, error) {
				return 0, `{"ok":true,"issues":[]}`, nil
			},
		})
		if code != 1 {
			t.Fatalf("exit code = %d, want 1", code)
		}
		if !strings.Contains(errOut.String(), "verify failed at nilaway ./...") {
			t.Fatalf("stderr = %q, want strict nilaway failure", errOut.String())
		}
		if !strings.Contains(errOut.String(), "nilaway is required in strict profile") {
			t.Fatalf("stderr = %q, want strict nilaway message", errOut.String())
		}
	})

	t.Run("fails on hard-cut wording invariant drift", func(t *testing.T) {
		root := t.TempDir()
		writeVerifyGoMod(t, root)
		writeVerifyGoWork(t, root)
		canonicalDoc := filepath.Join(root, "docs", "reference", "01-cli.md")
		if err := os.MkdirAll(filepath.Dir(canonicalDoc), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(canonicalDoc, []byte("deprecated alias\n"), 0o644); err != nil {
			t.Fatal(err)
		}

		prevWD := chdirVerifyRoot(t, root)
		t.Cleanup(func() { _ = os.Chdir(prevWD) })

		out := &bytes.Buffer{}
		errOut := &bytes.Buffer{}
		code := RunVerify([]string{"--skip-tests"}, VerifyDeps{
			Out:          out,
			Err:          errOut,
			FindGoModule: findVerifyGoModule,
			RelocateTempl: func(rootPath string) error {
				return nil
			},
			RunStep: func(name string, args ...string) (int, string, error) {
				return 0, "ok", nil
			},
			LookPath: func(file string) (string, error) {
				return "/usr/bin/" + file, nil
			},
			RunDoctor: func() (int, string, error) {
				return 0, `{"ok":true,"issues":[]}`, nil
			},
		})
		if code != 1 {
			t.Fatalf("exit code = %d, want 1", code)
		}
		if !strings.Contains(errOut.String(), "verify failed at no-compatibility/deprecation invariant") {
			t.Fatalf("stderr = %q, want wording invariant failure", errOut.String())
		}
		if !strings.Contains(errOut.String(), "docs/reference/01-cli.md:1") {
			t.Fatalf("stderr = %q, want file:line diagnostic", errOut.String())
		}
	})

	t.Run("fails on module compatibility policy drift", func(t *testing.T) {
		root := t.TempDir()
		writeVerifyGoMod(t, root)
		writeVerifyGoWork(t, root)
		moduleDir := filepath.Join(root, "modules", "notifications")
		if err := os.MkdirAll(moduleDir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(moduleDir, "go.mod"), []byte("module github.com/leomorpho/goship-modules/notifications\n\ngo 1.25\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte(`module example.com/verify

go 1.25

require github.com/leomorpho/goship-modules/notifications v1.2.3

replace github.com/leomorpho/goship-modules/notifications => ./modules/notifications
`), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, "go.work"), []byte(`go 1.25

use (
	.
	./modules/notifications
)
`), 0o644); err != nil {
			t.Fatal(err)
		}

		prevWD := chdirVerifyRoot(t, root)
		t.Cleanup(func() { _ = os.Chdir(prevWD) })

		out := &bytes.Buffer{}
		errOut := &bytes.Buffer{}
		code := RunVerify([]string{"--skip-tests"}, VerifyDeps{
			Out:          out,
			Err:          errOut,
			FindGoModule: findVerifyGoModule,
			RelocateTempl: func(rootPath string) error {
				return nil
			},
			RunStep: func(name string, args ...string) (int, string, error) {
				return 0, "ok", nil
			},
			LookPath: func(file string) (string, error) {
				return "/usr/bin/" + file, nil
			},
			RunDoctor: func() (int, string, error) {
				return 0, `{"ok":true,"issues":[]}`, nil
			},
		})
		if code != 1 {
			t.Fatalf("exit code = %d, want 1", code)
		}
		if !strings.Contains(errOut.String(), "verify failed at module compatibility policy") {
			t.Fatalf("stderr = %q, want module compatibility failure", errOut.String())
		}
		if !strings.Contains(errOut.String(), "v0.0.0") {
			t.Fatalf("stderr = %q, want local workspace version policy diagnostic", errOut.String())
		}
	})

	t.Run("includes orchestration contract mismatch preflight", func(t *testing.T) {
		root := t.TempDir()
		writeVerifyGoMod(t, root)
		writeVerifyGoWork(t, root)

		prevWD := chdirVerifyRoot(t, root)
		t.Cleanup(func() { _ = os.Chdir(prevWD) })

		out := &bytes.Buffer{}
		errOut := &bytes.Buffer{}
		code := RunVerify([]string{"--json", "--skip-tests"}, VerifyDeps{
			Out:          out,
			Err:          errOut,
			FindGoModule: findVerifyGoModule,
			RelocateTempl: func(rootPath string) error {
				return nil
			},
			RunStep: func(name string, args ...string) (int, string, error) {
				return 0, "ok", nil
			},
			LookPath: func(file string) (string, error) {
				return "/usr/bin/" + file, nil
			},
			RunDoctor: func() (int, string, error) {
				return 0, `{"ok":true,"issues":[]}`, nil
			},
		})
		if code != 0 {
			t.Fatalf("exit code = %d, stderr=%s", code, errOut.String())
		}

		var payload verifyJSONResult
		if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
			t.Fatalf("decode json: %v", err)
		}
		for _, step := range payload.Steps {
			if step.Name == "orchestration contract mismatch preflight" {
				return
			}
		}
		t.Fatalf("verify JSON missing orchestration contract mismatch preflight step:\n%s", out.String())
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
			RunStep: func(name string, args ...string) (int, string, error) {
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

	t.Run("warns when scaffold tests are still skipped", func(t *testing.T) {
		root := t.TempDir()
		writeVerifyGoMod(t, root)
		testFile := filepath.Join(root, "scaffold_test.go")
		content := `package verify

import "testing"

func TestScaffoldTodo(t *testing.T) {
	t.Skip("scaffold: implement resource behavior")
}
`
		if err := os.WriteFile(testFile, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}

		prevWD := chdirVerifyRoot(t, root)
		t.Cleanup(func() { _ = os.Chdir(prevWD) })

		out := &bytes.Buffer{}
		errOut := &bytes.Buffer{}
		code := RunVerify([]string{"--skip-tests"}, VerifyDeps{
			Out:          out,
			Err:          errOut,
			FindGoModule: findVerifyGoModule,
			RelocateTempl: func(rootPath string) error {
				return nil
			},
			RunStep: func(name string, args ...string) (int, string, error) {
				return 0, "ok", nil
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
		if !strings.Contains(out.String(), "Warning: 1 scaffolded tests are still skipped.") {
			t.Fatalf("stdout = %q, want scaffold warning", out.String())
		}
		if !strings.Contains(out.String(), "scaffold_test.go:TestScaffoldTodo") {
			t.Fatalf("stdout = %q, want file and test name", out.String())
		}
	})

	t.Run("json output includes warning severity for scaffold skips", func(t *testing.T) {
		root := t.TempDir()
		writeVerifyGoMod(t, root)
		testFile := filepath.Join(root, "scaffold_test.go")
		content := `package verify

import "testing"

func TestScaffoldTodo(t *testing.T) {
	t.Skip("scaffold: implement resource behavior")
}
`
		if err := os.WriteFile(testFile, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}

		prevWD := chdirVerifyRoot(t, root)
		t.Cleanup(func() { _ = os.Chdir(prevWD) })

		out := &bytes.Buffer{}
		errOut := &bytes.Buffer{}
		code := RunVerify([]string{"--json", "--skip-tests"}, VerifyDeps{
			Out:          out,
			Err:          errOut,
			FindGoModule: findVerifyGoModule,
			RelocateTempl: func(rootPath string) error {
				return nil
			},
			RunStep: func(name string, args ...string) (int, string, error) {
				return 0, "ok", nil
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
		foundWarning := false
		for _, step := range payload.Steps {
			if step.Name == "scaffold skip checks" && step.Severity == "warning" {
				foundWarning = true
				break
			}
		}
		if !foundWarning {
			t.Fatalf("expected scaffold warning step, got %+v", payload.Steps)
		}
	})
}

func writeVerifyGoMod(t *testing.T, root string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/verify\n\ngo 1.25\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	writeVerifyGoWork(t, root)
}

func writeVerifyGoWork(t *testing.T, root string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, "go.work"), []byte("go 1.25\n\nuse .\n"), 0o644); err != nil {
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
