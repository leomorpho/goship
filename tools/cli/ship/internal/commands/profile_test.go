package commands

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestRunProfileSet_WritesCanonicalPresets(t *testing.T) {
	cases := []struct {
		name   string
		preset string
		want   []string
	}{
		{
			name:   "single-binary",
			preset: "single-binary",
			want: []string{
				"PAGODA_RUNTIME_PROFILE=single-node",
				"PAGODA_PROCESSES_WEB=true",
				"PAGODA_PROCESSES_WORKER=true",
				"PAGODA_PROCESSES_SCHEDULER=true",
				"PAGODA_PROCESSES_COLOCATED=true",
			},
		},
		{
			name:   "standard",
			preset: "standard",
			want: []string{
				"PAGODA_RUNTIME_PROFILE=server-db",
				"PAGODA_PROCESSES_WEB=true",
				"PAGODA_PROCESSES_WORKER=false",
				"PAGODA_PROCESSES_SCHEDULER=false",
				"PAGODA_PROCESSES_COLOCATED=false",
			},
		},
		{
			name:   "distributed",
			preset: "distributed",
			want: []string{
				"PAGODA_RUNTIME_PROFILE=distributed",
				"PAGODA_PROCESSES_WEB=true",
				"PAGODA_PROCESSES_WORKER=true",
				"PAGODA_PROCESSES_SCHEDULER=true",
				"PAGODA_PROCESSES_COLOCATED=false",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			envPath := filepath.Join(root, ".env")
			if err := os.WriteFile(envPath, []byte("PAGODA_RUNTIME_PROFILE=server-db\n"), 0o644); err != nil {
				t.Fatalf("write env: %v", err)
			}

			prevWD, err := os.Getwd()
			if err != nil {
				t.Fatalf("getwd: %v", err)
			}
			if err := os.Chdir(root); err != nil {
				t.Fatalf("chdir %s: %v", root, err)
			}
			t.Cleanup(func() { _ = os.Chdir(prevWD) })

			out := &bytes.Buffer{}
			errOut := &bytes.Buffer{}
			if code := RunProfile([]string{"set", tc.preset}, ProfileDeps{Out: out, Err: errOut}); code != 0 {
				t.Fatalf("exit code = %d, stderr=%s", code, errOut.String())
			}

			body, err := os.ReadFile(envPath)
			if err != nil {
				t.Fatalf("read env: %v", err)
			}
			for _, want := range tc.want {
				if !strings.Contains(string(body), want) {
					t.Fatalf("env missing %q:\n%s", want, string(body))
				}
			}
			if !strings.Contains(out.String(), "profile preset") || !strings.Contains(out.String(), tc.preset) {
				t.Fatalf("stdout missing preset summary:\n%s", out.String())
			}
		})
	}
}

func TestRunProfileSet_IsIdempotent(t *testing.T) {
	root := t.TempDir()
	envPath := filepath.Join(root, ".env")
	initial := strings.Join([]string{
		"PAGODA_RUNTIME_PROFILE=single-node",
		"PAGODA_PROCESSES_WEB=true",
		"PAGODA_PROCESSES_WORKER=true",
		"PAGODA_PROCESSES_SCHEDULER=true",
		"PAGODA_PROCESSES_COLOCATED=true",
		"",
	}, "\n")
	if err := os.WriteFile(envPath, []byte(initial), 0o644); err != nil {
		t.Fatalf("write env: %v", err)
	}

	prevWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir %s: %v", root, err)
	}
	t.Cleanup(func() { _ = os.Chdir(prevWD) })

	firstOut := &bytes.Buffer{}
	firstErr := &bytes.Buffer{}
	if code := RunProfile([]string{"set", "single-binary"}, ProfileDeps{Out: firstOut, Err: firstErr}); code != 0 {
		t.Fatalf("first run exit code = %d, stderr=%s", code, firstErr.String())
	}
	firstBody, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatalf("read env after first run: %v", err)
	}

	secondOut := &bytes.Buffer{}
	secondErr := &bytes.Buffer{}
	if code := RunProfile([]string{"set", "single-binary"}, ProfileDeps{Out: secondOut, Err: secondErr}); code != 0 {
		t.Fatalf("second run exit code = %d, stderr=%s", code, secondErr.String())
	}
	secondBody, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatalf("read env after second run: %v", err)
	}
	if string(firstBody) != string(secondBody) {
		t.Fatalf("profile:set should be idempotent\nfirst:\n%s\nsecond:\n%s", string(firstBody), string(secondBody))
	}
	if !strings.Contains(secondOut.String(), "already applied") {
		t.Fatalf("second run should report no-op\n%s", secondOut.String())
	}
}

func TestRunProfileSet_RejectsUnknownPreset(t *testing.T) {
	errOut := &bytes.Buffer{}
	code := RunProfile([]string{"set", "bogus"}, ProfileDeps{Out: &bytes.Buffer{}, Err: errOut})
	if code == 0 {
		t.Fatal("expected non-zero exit code")
	}
	if !strings.Contains(errOut.String(), "unknown profile preset") {
		t.Fatalf("stderr missing unknown preset error:\n%s", errOut.String())
	}
}

func TestRunProfileSet_AcceptsCanonicalAliases(t *testing.T) {
	root := t.TempDir()
	envPath := filepath.Join(root, ".env")
	if err := os.WriteFile(envPath, []byte("PAGODA_RUNTIME_PROFILE=distributed\n"), 0o644); err != nil {
		t.Fatalf("write env: %v", err)
	}

	prevWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir %s: %v", root, err)
	}
	t.Cleanup(func() { _ = os.Chdir(prevWD) })

	for _, alias := range []string{"single-node", "server-db", "single", "local"} {
		out := &bytes.Buffer{}
		errOut := &bytes.Buffer{}
		if code := RunProfile([]string{"set", alias}, ProfileDeps{Out: out, Err: errOut}); code != 0 {
			t.Fatalf("alias %q exit code=%d stderr=%s", alias, code, errOut.String())
		}
	}
}

func TestRunProfileSet_MissingEnvFile_ProvidesActionableGuidance(t *testing.T) {
	root := t.TempDir()
	prevWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir %s: %v", root, err)
	}
	t.Cleanup(func() { _ = os.Chdir(prevWD) })

	errOut := &bytes.Buffer{}
	code := RunProfile([]string{"set", "standard"}, ProfileDeps{Out: &bytes.Buffer{}, Err: errOut})
	if code == 0 {
		t.Fatal("expected non-zero exit code")
	}
	if !strings.Contains(errOut.String(), "profile:set requires a .env file") {
		t.Fatalf("stderr missing .env error:\n%s", errOut.String())
	}
	if !strings.Contains(errOut.String(), "Next step: create a .env file") {
		t.Fatalf("stderr missing remediation guidance:\n%s", errOut.String())
	}
}

func TestRunProfileSet_WriteFailure_ProvidesActionableGuidance(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("chmod-based write-permission failure is not portable on windows")
	}

	root := t.TempDir()
	envPath := filepath.Join(root, ".env")
	if err := os.WriteFile(envPath, []byte("PAGODA_RUNTIME_PROFILE=server-db\n"), 0o444); err != nil {
		t.Fatalf("write env: %v", err)
	}

	prevWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir %s: %v", root, err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(envPath, 0o644)
		_ = os.Chdir(prevWD)
	})

	errOut := &bytes.Buffer{}
	code := RunProfile([]string{"set", "distributed"}, ProfileDeps{Out: &bytes.Buffer{}, Err: errOut})
	if code == 0 {
		t.Fatal("expected non-zero exit code")
	}
	if !strings.Contains(errOut.String(), "failed to write") {
		t.Fatalf("stderr missing write failure:\n%s", errOut.String())
	}
	if !strings.Contains(errOut.String(), "Next step: ensure .env is writable") {
		t.Fatalf("stderr missing write remediation guidance:\n%s", errOut.String())
	}
}
