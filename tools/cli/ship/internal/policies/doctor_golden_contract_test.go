package policies

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestDoctorGoldenContract_RedSpec(t *testing.T) {
	packageDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}

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

	humanOut := &bytes.Buffer{}
	if code := RunDoctor([]string{}, doctorDepsForTest(humanOut, &bytes.Buffer{})); code != 0 {
		t.Fatalf("doctor human exit code = %d", code)
	}
	normalizedHuman := string(bytes.ReplaceAll(humanOut.Bytes(), []byte(root), []byte("<repo-root>")))
	assertDoctorGoldenSnapshot(t, packageDir, "doctor_human.golden", normalizedHuman)

	jsonOut := &bytes.Buffer{}
	if code := RunDoctor([]string{"--json"}, doctorDepsForTest(jsonOut, &bytes.Buffer{})); code != 0 {
		t.Fatalf("doctor json exit code = %d", code)
	}
	assertDoctorJSONGolden(t, packageDir, "doctor_json.golden", jsonOut.Bytes())
}

func assertDoctorGoldenSnapshot(t *testing.T, packageDir, name, got string) {
	t.Helper()

	path := filepath.Join(packageDir, "testdata", name)
	if os.Getenv("UPDATE_CLI_GOLDENS") == "1" {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
		}
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatalf("write snapshot %s: %v", path, err)
		}
	}

	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read snapshot %s: %v", path, err)
	}
	if string(want) != got {
		t.Fatalf("doctor golden drift for %s", path)
	}
}

func assertDoctorJSONGolden(t *testing.T, packageDir, name string, payload []byte) {
	t.Helper()

	var pretty bytes.Buffer
	if err := json.Indent(&pretty, payload, "", "  "); err != nil {
		t.Fatalf("indent json: %v", err)
	}
	pretty.WriteByte('\n')
	assertDoctorGoldenSnapshot(t, packageDir, name, pretty.String())
}
