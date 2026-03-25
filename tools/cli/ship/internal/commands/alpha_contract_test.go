package commands

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestAlphaContract_FrozenCommandAndRouteSurface_RedSpec(t *testing.T) {
	root := repoRootFromCommandsTest(t)
	packageDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}

	var help bytes.Buffer
	PrintRootHelp(&help)
	assertAlphaSnapshot(t, packageDir, "v0_alpha_root_help.golden", help.String())

	prevWD := packageDir
	t.Cleanup(func() { _ = os.Chdir(prevWD) })
	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir %s: %v", root, err)
	}

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	if code := RunRoutes([]string{"--json"}, RoutesDeps{
		Out: out,
		Err: errOut,
		FindGoModule: func(string) (string, string, error) {
			return root, filepath.Join(root, "go.mod"), nil
		},
	}); code != 0 {
		t.Fatalf("routes exit code = %d, stderr=%s", code, errOut.String())
	}

	var pretty bytes.Buffer
	if err := json.Indent(&pretty, out.Bytes(), "", "  "); err != nil {
		t.Fatalf("indent routes json: %v", err)
	}
	pretty.WriteByte('\n')
	assertAlphaSnapshot(t, packageDir, "v0_alpha_routes.golden", pretty.String())
}

func TestCIContract_DefinesAlphaFreezeGate_RedSpec(t *testing.T) {
	root := repoRootFromCommandsTest(t)

	workflow := mustReadText(t, filepath.Join(root, ".github", "workflows", "test.yml"))
	makefile := mustReadText(t, filepath.Join(root, "Makefile"))
	releaseDoc := mustReadText(t, filepath.Join(root, "docs", "releases", "01-v0.1.0-alpha.md"))

	if !bytes.Contains([]byte(workflow), []byte("\n  alpha_contract:\n")) {
		t.Fatal("test workflow should define a dedicated alpha_contract job")
	}
	if !bytes.Contains([]byte(workflow), []byte("\n  startup_smoke:\n")) {
		t.Fatal("test workflow should define a dedicated startup_smoke job")
	}
	if !bytes.Contains([]byte(workflow), []byte("run: make test-alpha-contracts")) {
		t.Fatal("alpha contract CI job should invoke make test-alpha-contracts")
	}
	if !bytes.Contains([]byte(workflow), []byte("go test ./tools/cli/ship/internal/commands -run TestFreshAppStartupSmoke -count=1")) {
		t.Fatal("startup smoke CI job should invoke the fresh-app startup smoke test")
	}
	if !bytes.Contains([]byte(makefile), []byte(".PHONY: test-alpha-contracts")) {
		t.Fatal("Makefile should expose a canonical test-alpha-contracts entrypoint for CI")
	}
	if !bytes.Contains([]byte(releaseDoc), []byte("UPDATE_ALPHA_CONTRACTS=1 make test-alpha-contracts")) {
		t.Fatal("release doc should describe the explicit alpha-contract refresh path")
	}
	if !bytes.Contains([]byte(releaseDoc), []byte("approved review")) {
		t.Fatal("release doc should describe the approved review policy for frozen-surface changes")
	}
}

func assertAlphaSnapshot(t *testing.T, packageDir, name, got string) {
	t.Helper()

	path := filepath.Join(packageDir, "testdata", name)
	if os.Getenv("UPDATE_ALPHA_CONTRACTS") == "1" {
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
		t.Fatalf("alpha contract drift for %s", path)
	}
}
