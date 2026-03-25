package commands

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	policies "github.com/leomorpho/goship/tools/cli/ship/internal/policies"
)

func TestRunVerify(t *testing.T) {
	t.Run("rejects unsupported runtime contract version", func(t *testing.T) {
		out := &bytes.Buffer{}
		errOut := &bytes.Buffer{}
		code := RunVerify([]string{"--runtime-contract-version", "runtime-contract-v9"}, VerifyDeps{
			Out: out,
			Err: errOut,
		})
		if code != 1 {
			t.Fatalf("exit code = %d, want 1", code)
		}
		if !strings.Contains(errOut.String(), "unsupported runtime contract version") {
			t.Fatalf("stderr = %q", errOut.String())
		}
	})

	t.Run("rejects unsupported upgrade contract version", func(t *testing.T) {
		out := &bytes.Buffer{}
		errOut := &bytes.Buffer{}
		code := RunVerify([]string{"--upgrade-contract-version", "upgrade-readiness-v9"}, VerifyDeps{
			Out: out,
			Err: errOut,
		})
		if code != 1 {
			t.Fatalf("exit code = %d, want 1", code)
		}
		if !strings.Contains(errOut.String(), "unsupported upgrade contract version") {
			t.Fatalf("stderr = %q", errOut.String())
		}
	})

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
		if !regexp.MustCompile(`templ generate \(\d+ms\)`).MatchString(out.String()) {
			t.Fatalf("stdout = %q, want templ progress timing", out.String())
		}
		if !strings.Contains(out.String(), "nilaway not installed; skipping") {
			t.Fatalf("stdout = %q, want nilaway skip message", out.String())
		}
		if !strings.Contains(out.String(), "skipped via --skip-tests") {
			t.Fatalf("stdout = %q, want skip-tests message", out.String())
		}
		if !strings.Contains(out.String(), "startup smoke checks skipped via --skip-tests") {
			t.Fatalf("stdout = %q, want startup-smoke skip message", out.String())
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
		if !strings.Contains(out.String(), "startup smoke checks skipped in fast profile") {
			t.Fatalf("stdout = %q, want fast-profile startup-smoke skip message", out.String())
		}
	})

	t.Run("standard profile runs nilaway and tests", func(t *testing.T) {
		root := t.TempDir()
		writeVerifyGoMod(t, root)
		writeVerifyGoWork(t, root)
		prevWD := chdirVerifyRoot(t, root)
		t.Cleanup(func() { _ = os.Chdir(prevWD) })

		calls := make([]string, 0)
		out := &bytes.Buffer{}
		errOut := &bytes.Buffer{}
		code := RunVerify([]string{"--profile", "standard"}, VerifyDeps{
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
		if len(calls) != 5 {
			t.Fatalf("calls len = %d, want 5", len(calls))
		}
		if calls[4] != "go test ./tools/cli/ship/internal/commands -run TestFreshAppStartupSmoke -count=1" {
			t.Fatalf("startup smoke call = %q, want startup smoke verify gate", calls[4])
		}
		for _, token := range []string{"verify passed"} {
			if !strings.Contains(out.String(), token) {
				t.Fatalf("stdout missing %q:\n%s", token, out.String())
			}
		}
		for _, token := range []string{"nilaway not installed; skipping", "skipped in fast profile"} {
			if strings.Contains(out.String(), token) {
				t.Fatalf("stdout unexpectedly contained %q:\n%s", token, out.String())
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
		if !strings.Contains(errOut.String(), "Next step: run `ship runtime:report --json`") {
			t.Fatalf("stderr = %q, want operator guidance", errOut.String())
		}
	})

	t.Run("failure output includes concise total elapsed timing", func(t *testing.T) {
		root := t.TempDir()
		writeVerifyGoMod(t, root)
		prevWD := chdirVerifyRoot(t, root)
		t.Cleanup(func() { _ = os.Chdir(prevWD) })

		out := &bytes.Buffer{}
		errOut := &bytes.Buffer{}
		calls := make([]string, 0)

		base := time.Unix(0, 0)
		timepoints := []time.Time{
			base,
			base.Add(10 * time.Millisecond),
			base.Add(25 * time.Millisecond),
			base.Add(40 * time.Millisecond),
			base.Add(70 * time.Millisecond),
			base.Add(90 * time.Millisecond),
		}
		nowCalls := 0

		code := RunVerify([]string{}, VerifyDeps{
			Out:          out,
			Err:          errOut,
			FindGoModule: findVerifyGoModule,
			RelocateTempl: func(rootPath string) error {
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
			Now: func() time.Time {
				if nowCalls >= len(timepoints) {
					return timepoints[len(timepoints)-1]
				}
				v := timepoints[nowCalls]
				nowCalls++
				return v
			},
		})
		if code != 1 {
			t.Fatalf("exit code = %d, want 1", code)
		}
		if !regexp.MustCompile(`verify failed at go build \./\.\.\. \(\d+ms\)`).MatchString(errOut.String()) {
			t.Fatalf("stderr = %q, want failed-step timing", errOut.String())
		}
		if !strings.Contains(errOut.String(), "verify failed (90ms)") {
			t.Fatalf("stderr = %q, want total elapsed timing", errOut.String())
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
		if len(payload.Steps) != 10 {
			t.Fatalf("steps len = %d, want 10", len(payload.Steps))
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
		if payload.Steps[7].Name != "startup smoke checks" {
			t.Fatalf("startup smoke step name = %q, want startup smoke checks", payload.Steps[7].Name)
		}
		if payload.Steps[8].Name != "standalone exportability gate" {
			t.Fatalf("exportability step name = %q, want standalone exportability gate", payload.Steps[8].Name)
		}
		if payload.Steps[9].Name != "orchestration contract mismatch preflight" {
			t.Fatalf("final step name = %q, want orchestration contract mismatch preflight", payload.Steps[9].Name)
		}

		var raw map[string]any
		if err := json.Unmarshal(out.Bytes(), &raw); err != nil {
			t.Fatalf("decode raw json: %v", err)
		}
		if _, ok := raw["elapsed_ms"]; !ok {
			t.Fatalf("json payload missing elapsed_ms: %s", out.String())
		}
		rawSteps, ok := raw["steps"].([]any)
		if !ok || len(rawSteps) == 0 {
			t.Fatalf("json payload missing steps array: %s", out.String())
		}
		firstStep, ok := rawSteps[0].(map[string]any)
		if !ok {
			t.Fatalf("first step is not object: %T", rawSteps[0])
		}
		if _, ok := firstStep["duration_ms"]; !ok {
			t.Fatalf("json payload missing step duration_ms: %s", out.String())
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
		canonicalDoc := filepath.Join(root, "docs", "architecture", "06-known-gaps-and-risks.md")
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
		if !strings.Contains(errOut.String(), "docs/architecture/06-known-gaps-and-risks.md:1") {
			t.Fatalf("stderr = %q, want file:line diagnostic", errOut.String())
		}
		if !strings.Contains(errOut.String(), "rewrite canonical docs to describe the current hard-cut model only") {
			t.Fatalf("stderr = %q, want replacement guidance", errOut.String())
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

	t.Run("fails when startup smoke checks fail", func(t *testing.T) {
		root := t.TempDir()
		writeVerifyGoMod(t, root)
		writeVerifyGoWork(t, root)

		prevWD := chdirVerifyRoot(t, root)
		t.Cleanup(func() { _ = os.Chdir(prevWD) })

		out := &bytes.Buffer{}
		errOut := &bytes.Buffer{}
		code := RunVerify([]string{}, VerifyDeps{
			Out:          out,
			Err:          errOut,
			FindGoModule: findVerifyGoModule,
			RelocateTempl: func(rootPath string) error {
				return nil
			},
			RunStep: func(name string, args ...string) (int, string, error) {
				if name == "go" && len(args) == 5 && args[0] == "test" && args[1] == "./tools/cli/ship/internal/commands" && args[2] == "-run" && args[3] == "TestFreshAppStartupSmoke" {
					return 1, "startup smoke failed", nil
				}
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
		if !strings.Contains(errOut.String(), "verify failed at startup smoke checks") {
			t.Fatalf("stderr = %q, want startup-smoke step failure", errOut.String())
		}
		if !strings.Contains(errOut.String(), "startup smoke failed") {
			t.Fatalf("stderr = %q, want startup-smoke subprocess output", errOut.String())
		}
	})

	t.Run("fails when managed-settings contract drifts", func(t *testing.T) {
		root := t.TempDir()
		writeVerifyGoMod(t, root)
		writeVerifyGoWork(t, root)
		managedSettingsPath := filepath.Join(root, "config", "managed_settings.go")
		if err := os.MkdirAll(filepath.Dir(managedSettingsPath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(managedSettingsPath, []byte(`package config

func managedSettingAccess() string { return "editable" }
`), 0o644); err != nil {
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
		if code != 1 {
			t.Fatalf("exit code = %d, want 1", code)
		}
		if errOut.Len() != 0 {
			t.Fatalf("stderr = %q, want empty", errOut.String())
		}

		var payload verifyJSONResult
		if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
			t.Fatalf("decode json: %v\n%s", err, out.String())
		}
		if payload.OK {
			t.Fatalf("payload.OK = true, want false")
		}
		if len(payload.Steps) != 10 {
			t.Fatalf("steps len = %d, want 10", len(payload.Steps))
		}
		last := payload.Steps[len(payload.Steps)-1]
		if last.Name != "orchestration contract mismatch preflight" {
			t.Fatalf("last step name = %q, want orchestration contract mismatch preflight", last.Name)
		}
		if last.OK {
			t.Fatalf("last step should fail, got %+v", last)
		}
		if !strings.Contains(last.Output, "config/managed_settings.go") {
			t.Fatalf("last step output = %q, want managed settings file diagnostic", last.Output)
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

	t.Run("framework repo layout failures stop verify before build", func(t *testing.T) {
		root := t.TempDir()
		writeVerifyGoMod(t, root)
		writeVerifyGoWork(t, root)
		if err := os.MkdirAll(filepath.Join(root, "tools", "cli", "ship"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Join(root, "app"), 0o755); err != nil {
			t.Fatal(err)
		}

		prevWD := chdirVerifyRoot(t, root)
		t.Cleanup(func() { _ = os.Chdir(prevWD) })

		out := &bytes.Buffer{}
		errOut := &bytes.Buffer{}
		calls := make([]string, 0)
		code := RunVerify([]string{"--skip-tests"}, VerifyDeps{
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
				return "/usr/bin/" + file, nil
			},
			RunDoctor: func() (int, string, error) {
				t.Fatal("doctor should not run after canonical repo layout failure")
				return 0, "", nil
			},
		})
		if code != 1 {
			t.Fatalf("exit code = %d, want 1", code)
		}
		if len(calls) != 0 {
			t.Fatalf("verify should fail before subprocesses, got calls %+v", calls)
		}
		if !strings.Contains(errOut.String(), "verify failed at canonical repo layout") {
			t.Fatalf("stderr = %q, want repo layout failure step", errOut.String())
		}
		if !strings.Contains(errOut.String(), "forbidden top-level path present: app") {
			t.Fatalf("stderr = %q, want forbidden app path diagnostic", errOut.String())
		}
		if !strings.Contains(errOut.String(), "Next step: run `ship doctor --json`") {
			t.Fatalf("stderr = %q, want operator guidance", errOut.String())
		}
	})

	t.Run("generated app scaffold failures stop verify before build", func(t *testing.T) {
		root := t.TempDir()
		writeVerifyGoMod(t, root)
		writeVerifyGoWork(t, root)
		if err := os.MkdirAll(filepath.Join(root, "config"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Join(root, "docs"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, "config", "modules.yaml"), []byte("modules: []\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, "docs", "00-index.md"), []byte("# Index\n"), 0o644); err != nil {
			t.Fatal(err)
		}

		prevWD := chdirVerifyRoot(t, root)
		t.Cleanup(func() { _ = os.Chdir(prevWD) })

		out := &bytes.Buffer{}
		errOut := &bytes.Buffer{}
		calls := make([]string, 0)
		code := RunVerify([]string{"--skip-tests"}, VerifyDeps{
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
				return "/usr/bin/" + file, nil
			},
			RunDoctor: func() (int, string, error) {
				t.Fatal("doctor should not run after generated app scaffold failure")
				return 0, "", nil
			},
		})
		if code != 1 {
			t.Fatalf("exit code = %d, want 1", code)
		}
		if len(calls) != 0 {
			t.Fatalf("verify should fail before subprocesses, got calls %+v", calls)
		}
		if !strings.Contains(errOut.String(), "verify failed at generated app scaffold") {
			t.Fatalf("stderr = %q, want generated scaffold failure step", errOut.String())
		}
		if !strings.Contains(errOut.String(), "missing required directory: app") {
			t.Fatalf("stderr = %q, want required-directory diagnostic", errOut.String())
		}
		if !strings.Contains(errOut.String(), "owner: ship new scaffold generator") {
			t.Fatalf("stderr = %q, want owning generator hint", errOut.String())
		}
		if !strings.Contains(errOut.String(), "Next step: run `ship doctor --json`") {
			t.Fatalf("stderr = %q, want operator guidance", errOut.String())
		}
	})

	t.Run("framework repo layout rejects legacy app shell runtime fallback paths", func(t *testing.T) {
		root := t.TempDir()
		writeVerifyGoMod(t, root)
		writeVerifyGoWork(t, root)
		if err := os.MkdirAll(filepath.Join(root, "tools", "cli", "ship"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Join(root, "app", "foundation"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Join(root, "app", "schedules"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, "app", "router.go"), []byte("package app\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, "app", "foundation", "container.go"), []byte("package foundation\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, "app", "schedules", "schedules.go"), []byte("package schedules\n"), 0o644); err != nil {
			t.Fatal(err)
		}

		prevWD := chdirVerifyRoot(t, root)
		t.Cleanup(func() { _ = os.Chdir(prevWD) })

		out := &bytes.Buffer{}
		errOut := &bytes.Buffer{}
		calls := make([]string, 0)
		code := RunVerify([]string{"--skip-tests"}, VerifyDeps{
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
				return "/usr/bin/" + file, nil
			},
			RunDoctor: func() (int, string, error) {
				t.Fatal("doctor should not run after canonical repo layout failure")
				return 0, "", nil
			},
		})
		if code != 1 {
			t.Fatalf("exit code = %d, want 1", code)
		}
		if len(calls) != 0 {
			t.Fatalf("verify should fail before subprocesses, got calls %+v", calls)
		}
		for _, token := range []string{
			"missing canonical top-level path: container.go",
			"missing canonical top-level path: router.go",
			"missing canonical top-level path: schedules.go",
		} {
			if !strings.Contains(errOut.String(), token) {
				t.Fatalf("stderr = %q, want %q", errOut.String(), token)
			}
		}
	})
}

func TestFormatVerifyDoctorIssues_IncludesOwningHintAccuracy(t *testing.T) {
	t.Parallel()

	issues := []policies.DoctorIssue{
		{
			Code:    "DX001",
			Message: "missing required directory: app/foundation",
			Fix:     "create app/foundation or regenerate the app scaffold with `ship new`",
		},
		{
			Code:    "DX030",
			Message: "canonical docs contain deprecated wording",
			Fix:     "update docs wording",
		},
	}

	formatted := formatVerifyDoctorIssues(issues)
	if !strings.Contains(formatted, "owner: ship new scaffold generator (tools/cli/ship/internal/commands/project_new.go)") {
		t.Fatalf("formatted issues missing generator owner hint:\n%s", formatted)
	}
	if !strings.Contains(formatted, "owner: doctor policy checks (tools/cli/ship/internal/policies/doctor.go)") {
		t.Fatalf("formatted issues missing doctor owner hint:\n%s", formatted)
	}
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
