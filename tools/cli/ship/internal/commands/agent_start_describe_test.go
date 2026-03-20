package commands

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAgentStartUsesDefaultDescribeOutput(t *testing.T) {
	root := t.TempDir()
	writeGoModule(t, root)

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
	deps := AgentDeps{
		Out:          out,
		Err:          errOut,
		FindGoModule: findGoModuleTest,
		RunGitWorktreeAdd: func(_ string, worktreePath, _ string) error {
			return os.MkdirAll(worktreePath, 0o755)
		},
	}

	code := RunAgent([]string{"start", "--task", "default describe", "--id", "T01"}, deps)
	if code != 0 {
		t.Fatalf("start failed: stderr=%s", errOut.String())
	}

	taskFile := filepath.Join(root, ".worktrees", "T01", "TASK.md")
	content, err := os.ReadFile(taskFile)
	if err != nil {
		t.Fatalf("missing TASK.md: %v", err)
	}
	task := string(content)
	if !strings.Contains(task, "\"routes\"") || !strings.Contains(task, "\"modules\"") {
		t.Fatalf("task file missing describe payload: %s", task)
	}
}
