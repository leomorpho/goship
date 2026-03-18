package generators

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunMakeJob_GeneratesCoreJobsFirstScaffold(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	code := RunMakeJob([]string{"BackfillUserStats"}, MakeJobDeps{
		Out: out,
		Err: errOut,
		Cwd: root,
	})
	if code != 0 {
		t.Fatalf("exit code = %d, stderr=%s", code, errOut.String())
	}

	jobFile := filepath.Join(root, "app", "jobs", "backfill_user_stats.go")
	content, err := os.ReadFile(jobFile)
	if err != nil {
		t.Fatalf("read job file: %v", err)
	}
	text := string(content)
	for _, required := range []string{
		"const TypeBackfillUserStats = \"job.backfill_user_stats\"",
		"func RegisterBackfillUserStats(jobs core.Jobs, handler core.JobHandler) error",
		"func HandleBackfillUserStats(_ context.Context, payload []byte) error",
	} {
		if !strings.Contains(text, required) {
			t.Fatalf("generated job file missing %q\n%s", required, text)
		}
	}

	testFile := filepath.Join(root, "app", "jobs", "backfill_user_stats_test.go")
	testContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("read job test file: %v", err)
	}
	testText := string(testContent)
	for _, required := range []string{
		"TestRegisterBackfillUserStats",
		"TestHandleBackfillUserStats_InvalidPayload",
	} {
		if !strings.Contains(testText, required) {
			t.Fatalf("generated job test file missing %q\n%s", required, testText)
		}
	}
}

func TestRunMakeJob_IsIdempotent(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	deps := MakeJobDeps{Out: out, Err: errOut, Cwd: root}
	if code := RunMakeJob([]string{"BackfillUserStats"}, deps); code != 0 {
		t.Fatalf("first run failed: code=%d stderr=%s", code, errOut.String())
	}

	out.Reset()
	errOut.Reset()
	if code := RunMakeJob([]string{"BackfillUserStats"}, deps); code == 0 {
		t.Fatal("expected duplicate run to fail")
	}
}
