package policies

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func doctorDepsForTest(out, errOut *bytes.Buffer) DoctorDeps {
	return DoctorDeps{
		Out:          out,
		Err:          errOut,
		FindGoModule: findGoModuleTest,
		LookPath: func(string) (string, error) {
			return "", errors.New("not found")
		},
	}
}

func TestRunDoctor(t *testing.T) {
	t.Run("help", func(t *testing.T) {
		if code := RunDoctor([]string{"--help"}, doctorDepsForTest(&bytes.Buffer{}, &bytes.Buffer{})); code != 0 {
			t.Fatalf("exit code = %d, want 0", code)
		}
	})

	t.Run("unexpected args", func(t *testing.T) {
		out := &bytes.Buffer{}
		errOut := &bytes.Buffer{}
		code := RunDoctor([]string{"extra"}, doctorDepsForTest(out, errOut))
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
		code := RunDoctor([]string{"--json", "extra"}, doctorDepsForTest(out, errOut))
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
	if code := RunDoctor([]string{}, doctorDepsForTest(out, errOut)); code != 0 {
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
		if code := RunDoctor([]string{"--json"}, doctorDepsForTest(out, errOut)); code != 0 {
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
		if !payload.UpgradeReadiness.Ready {
			t.Fatalf("upgrade_readiness.ready = false, want true")
		}
		if len(payload.UpgradeReadiness.Blockers) != 0 {
			t.Fatalf("upgrade_readiness.blockers len = %d, want 0", len(payload.UpgradeReadiness.Blockers))
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
		if code := RunDoctor([]string{"--json"}, doctorDepsForTest(out, errOut)); code != 1 {
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

	t.Run("upgrade readiness blocker is surfaced in json and human output", func(t *testing.T) {
		root := t.TempDir()
		writeDoctorFixture(t, root)
		if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/doctor\n\ngo 1.25\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		cliPath := filepath.Join(root, "tools", "cli", "ship", "internal", "cli", "cli.go")
		if err := os.MkdirAll(filepath.Dir(cliPath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(cliPath, []byte("package ship\nconst missingUpgradeMarker = \"x\"\n"), 0o644); err != nil {
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

		jsonOut := &bytes.Buffer{}
		jsonErr := &bytes.Buffer{}
		if code := RunDoctor([]string{"--json"}, doctorDepsForTest(jsonOut, jsonErr)); code != 1 {
			t.Fatalf("doctor json exit code = %d, want 1", code)
		}
		var payload doctorJSONResult
		if err := json.Unmarshal(jsonOut.Bytes(), &payload); err != nil {
			t.Fatalf("failed to decode json: %v", err)
		}
		if payload.UpgradeReadiness.Ready {
			t.Fatalf("upgrade_readiness.ready = true, want false")
		}
		if len(payload.UpgradeReadiness.Blockers) == 0 {
			t.Fatal("upgrade_readiness.blockers len = 0, want >= 1")
		}
		if payload.UpgradeReadiness.Blockers[0].ID != "upgrade.convention_drift" {
			t.Fatalf("blocker id = %q, want upgrade.convention_drift", payload.UpgradeReadiness.Blockers[0].ID)
		}

		humanOut := &bytes.Buffer{}
		humanErr := &bytes.Buffer{}
		if code := RunDoctor([]string{}, doctorDepsForTest(humanOut, humanErr)); code != 1 {
			t.Fatalf("doctor human exit code = %d, want 1", code)
		}
		if !strings.Contains(humanErr.String(), "upgrade readiness: blocked") {
			t.Fatalf("stderr = %q, want blocked upgrade readiness section", humanErr.String())
		}
	})

	t.Run("fixture with missing config file reports DX002", func(t *testing.T) {
		root := t.TempDir()
		writeDoctorFixture(t, root)
		if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/doctor\n\ngo 1.25\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.Remove(filepath.Join(root, "config", "modules.yaml")); err != nil {
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
		if code := RunDoctor([]string{"--json"}, doctorDepsForTest(out, errOut)); code != 1 {
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
		if !doctorPayloadHasIssue(payload.Issues, "DX002", "config/modules.yaml") {
			t.Fatalf("expected DX002 issue for config/modules.yaml, got %+v", payload.Issues)
		}
	})

	t.Run("fixture with broken firebase secret reports DX022", func(t *testing.T) {
		root := t.TempDir()
		writeDoctorFixture(t, root)
		if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/doctor\n\ngo 1.25\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, ".env"), []byte("PAGODA_APP_FIREBASEBASE64ACCESSKEYS=not-base64\n"), 0o644); err != nil {
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
		if code := RunDoctor([]string{"--json"}, doctorDepsForTest(out, errOut)); code != 1 {
			t.Fatalf("doctor exit code = %d, want 1", code)
		}
		if errOut.Len() != 0 {
			t.Fatalf("stderr = %q, want empty stderr", errOut.String())
		}

		var payload doctorJSONResult
		if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
			t.Fatalf("failed to decode json: %v", err)
		}
		if !doctorPayloadHasIssue(payload.Issues, "DX022", "PAGODA_APP_FIREBASEBASE64ACCESSKEYS") {
			t.Fatalf("expected DX022 firebase secret issue, got %+v", payload.Issues)
		}
	})

	t.Run("fixture with invalid adapter reports DX022", func(t *testing.T) {
		root := t.TempDir()
		writeDoctorFixture(t, root)
		if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/doctor\n\ngo 1.25\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, ".env"), []byte("PAGODA_ADAPTERS_CACHE=bogus\n"), 0o644); err != nil {
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
		if code := RunDoctor([]string{"--json"}, doctorDepsForTest(out, errOut)); code != 1 {
			t.Fatalf("doctor exit code = %d, want 1", code)
		}
		if errOut.Len() != 0 {
			t.Fatalf("stderr = %q, want empty stderr", errOut.String())
		}

		var payload doctorJSONResult
		if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
			t.Fatalf("failed to decode json: %v", err)
		}
		if !doctorPayloadHasIssue(payload.Issues, "DX022", "unknown cache adapter") {
			t.Fatalf("expected DX022 adapter issue, got %+v", payload.Issues)
		}
	})
}

func doctorPayloadHasIssue(issues []doctorJSONIssue, issueType string, detailContains string) bool {
	for _, issue := range issues {
		if issue.Type != issueType {
			continue
		}
		if strings.Contains(issue.Detail, detailContains) {
			return true
		}
	}
	return false
}
