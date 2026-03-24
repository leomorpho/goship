package commands

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	policies "github.com/leomorpho/goship/tools/cli/ship/internal/policies"
)

func TestFreshAppSmoke_LocalScaffoldPreflightFlow(t *testing.T) {
	projectRoot := scaffoldFreshAppForSmoke(t)
	started := time.Now()

	issues := policies.FastPathGeneratedAppIssues(projectRoot)
	if len(issues) > 0 {
		t.Fatalf("fresh app scaffold preflight should pass, got:\n%s", formatVerifyDoctorIssues(issues))
	}

	for _, rel := range []string{
		filepath.Join("app", "router.go"),
		filepath.Join("app", "foundation", "container.go"),
		filepath.Join("cmd", "web", "main.go"),
		filepath.Join("cmd", "worker", "main.go"),
		filepath.Join("styles", "styles.css"),
	} {
		if _, err := os.Stat(filepath.Join(projectRoot, rel)); err != nil {
			t.Fatalf("expected fresh app smoke scaffold file %s: %v", rel, err)
		}
	}

	if elapsed := time.Since(started); elapsed > 5*time.Second {
		t.Fatalf("fresh-app local scaffold smoke exceeded time budget: %s", elapsed)
	}
}

func TestFreshAppSmoke_LocalVerifyAndDevPreflightFlow(t *testing.T) {
	projectRoot := scaffoldFreshAppForSmoke(t)
	started := time.Now()

	prevWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(prevWD) })
	if err := os.Chdir(projectRoot); err != nil {
		t.Fatal(err)
	}

	devIssues := runDevFastPathPreflight(DevDeps{
		FindGoModule:            findGoModuleTestProjectNew,
		FastPathGeneratedIssues: policies.FastPathGeneratedAppIssues,
	})
	if len(devIssues) > 0 {
		t.Fatalf("fresh app dev preflight should pass, got:\n%s", formatVerifyDoctorIssues(devIssues))
	}

	verifyOut := &bytes.Buffer{}
	verifyErr := &bytes.Buffer{}
	code := RunVerify([]string{"--profile", verifyProfileFast, "--skip-tests"}, VerifyDeps{
		Out:          verifyOut,
		Err:          verifyErr,
		FindGoModule: findGoModuleTestProjectNew,
		RelocateTempl: func(string) error {
			return nil
		},
		RunStep: func(name string, args ...string) (int, string, error) {
			return 0, "ok", nil
		},
		RunDoctor: func() (int, string, error) {
			return 0, `{"ok":true,"issues":[]}`, nil
		},
		Now: fakeTickNow(1 * time.Millisecond),
	})
	if code != 0 {
		t.Fatalf("fresh app verify smoke failed: code=%d stderr=%s", code, verifyErr.String())
	}
	if verifyErr.Len() != 0 {
		t.Fatalf("fresh app verify smoke stderr should be empty, got %q", verifyErr.String())
	}
	if !bytes.Contains(verifyOut.Bytes(), []byte("verify passed")) {
		t.Fatalf("fresh app verify smoke output = %q, want verify passed", verifyOut.String())
	}

	if elapsed := time.Since(started); elapsed > 8*time.Second {
		t.Fatalf("fresh-app local verify/dev smoke exceeded time budget: %s", elapsed)
	}
}

func scaffoldFreshAppForSmoke(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
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
	if code := RunNew([]string{"demo", "--module", "example.com/demo"}, NewDeps{
		Out:                        out,
		Err:                        errOut,
		ParseAgentPolicyBytes:      policies.ParsePolicyBytes,
		RenderAgentPolicyArtifacts: policies.RenderPolicyArtifacts,
		AgentPolicyFilePath:        policies.AgentPolicyFilePath,
	}); code != 0 {
		t.Fatalf("ship new failed in smoke scaffold: code=%d stderr=%s", code, errOut.String())
	}

	return filepath.Join(root, "demo")
}
