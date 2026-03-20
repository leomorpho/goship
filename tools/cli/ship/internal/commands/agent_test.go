package commands

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	policies "github.com/leomorpho/goship/tools/cli/ship/internal/policies"
)

func TestAgentPolicySetupAndCheck(t *testing.T) {
	root := t.TempDir()
	writeGoModule(t, root)
	policyPath := filepath.Join(root, "tools", "agent-policy", "allowed-commands.yaml")
	if err := os.MkdirAll(filepath.Dir(policyPath), 0o755); err != nil {
		t.Fatal(err)
	}
	policy := strings.Join([]string{
		"version: 1",
		"commands:",
		"  - id: go_test",
		"    description: Run tests.",
		"    prefix: [\"go\", \"test\"]",
		"  - id: ship_doctor",
		"    description: Run doctor.",
		"    prefix: [\"ship\", \"doctor\"]",
		"",
	}, "\n")
	if err := os.WriteFile(policyPath, []byte(policy), 0o644); err != nil {
		t.Fatal(err)
	}

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	if code := policies.RunPolicySetup(out, errOut, root); code != 0 {
		t.Fatalf("setup code=%d stderr=%s", code, errOut.String())
	}
	if !strings.Contains(out.String(), "agent setup complete") {
		t.Fatalf("unexpected setup output: %q", out.String())
	}

	out.Reset()
	errOut.Reset()
	if code := policies.RunPolicyCheck(out, errOut, root); code != 0 {
		t.Fatalf("check code=%d stderr=%s", code, errOut.String())
	}
	if !strings.Contains(out.String(), "in sync") {
		t.Fatalf("unexpected check output: %q", out.String())
	}
}

func TestAgentPolicyCheckDetectsDrift(t *testing.T) {
	root := t.TempDir()
	writeGoModule(t, root)
	policyPath := filepath.Join(root, "tools", "agent-policy", "allowed-commands.yaml")
	if err := os.MkdirAll(filepath.Dir(policyPath), 0o755); err != nil {
		t.Fatal(err)
	}
	policy := strings.Join([]string{
		"version: 1",
		"commands:",
		"  - id: go_test",
		"    description: Run tests.",
		"    prefix: [\"go\", \"test\"]",
		"",
	}, "\n")
	if err := os.WriteFile(policyPath, []byte(policy), 0o644); err != nil {
		t.Fatal(err)
	}

	if code := policies.RunPolicySetup(&bytes.Buffer{}, &bytes.Buffer{}, root); code != 0 {
		t.Fatalf("setup failed")
	}
	if err := os.WriteFile(filepath.Join(root, "tools", "agent-policy", "generated", "codex-prefixes.txt"), []byte("stale\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	if code := policies.RunPolicyCheck(out, errOut, root); code != 1 {
		t.Fatalf("check code=%d stderr=%s", code, errOut.String())
	}
	if !strings.Contains(errOut.String(), "out of sync") {
		t.Fatalf("unexpected check stderr: %q", errOut.String())
	}
}

func TestAgentStatus(t *testing.T) {
	root := t.TempDir()
	writeGoModule(t, root)
	writeAgentPolicyFixture(t, root)
	if code := policies.RunPolicySetup(&bytes.Buffer{}, &bytes.Buffer{}, root); code != 0 {
		t.Fatalf("setup failed")
	}

	prevWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(prevWD) })
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}

	t.Run("in-sync when config contains all prefixes", func(t *testing.T) {
		cfg := filepath.Join(root, "codex-local.txt")
		content := "go test\nship doctor\n"
		if err := os.WriteFile(cfg, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}

		out := &bytes.Buffer{}
		errOut := &bytes.Buffer{}
		code := RunAgent([]string{"status", "--codex-file", cfg}, AgentDeps{Out: out, Err: errOut, FindGoModule: findGoModuleTest})
		if code != 0 {
			t.Fatalf("status code=%d stderr=%s", code, errOut.String())
		}
		if !strings.Contains(out.String(), "- codex: in-sync") {
			t.Fatalf("unexpected status output: %s", out.String())
		}
	})

	t.Run("drifted when config has subset", func(t *testing.T) {
		cfg := filepath.Join(root, "codex-drifted.txt")
		content := "go test\n"
		if err := os.WriteFile(cfg, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}

		out := &bytes.Buffer{}
		errOut := &bytes.Buffer{}
		code := RunAgent([]string{"status", "--codex-file", cfg}, AgentDeps{Out: out, Err: errOut, FindGoModule: findGoModuleTest})
		if code != 0 {
			t.Fatalf("status code=%d stderr=%s", code, errOut.String())
		}
		if !strings.Contains(out.String(), "- codex: drifted") {
			t.Fatalf("unexpected status output: %s", out.String())
		}
	})
}

func writeAgentPolicyFixture(t *testing.T, root string) {
	t.Helper()
	policyPath := filepath.Join(root, "tools", "agent-policy", "allowed-commands.yaml")
	if err := os.MkdirAll(filepath.Dir(policyPath), 0o755); err != nil {
		t.Fatal(err)
	}
	policy := strings.Join([]string{
		"version: 1",
		"commands:",
		"  - id: go_test",
		"    description: Run tests.",
		"    prefix: [\"go\", \"test\"]",
		"  - id: ship_doctor",
		"    description: Run doctor.",
		"    prefix: [\"ship\", \"doctor\"]",
		"",
	}, "\n")
	if err := os.WriteFile(policyPath, []byte(policy), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeGoModule(t *testing.T, root string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/test\n\ngo 1.25\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func findGoModuleTest(start string) (string, string, error) {
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

func canonicalPath(p string) string {
	if c, err := filepath.EvalSymlinks(p); err == nil {
		return c
	}
	return filepath.Clean(p)
}

func TestAgentStartRequiresDescription(t *testing.T) {
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
	code := RunAgent([]string{"start"}, AgentDeps{
		Out:          out,
		Err:          errOut,
		FindGoModule: findGoModuleTest,
	})
	if code == 0 {
		t.Fatalf("expected failure when no task description provided")
	}
	if !strings.Contains(errOut.String(), "task description is required") {
		t.Fatalf("unexpected error output: %q", errOut.String())
	}
}

func TestAgentStartCreatesWorktree(t *testing.T) {
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

	gitignore := filepath.Join(root, ".gitignore")
	if err := os.WriteFile(gitignore, []byte("node_modules/\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	var createdPath, createdBranch string
	deps := AgentDeps{
		Out:          out,
		Err:          errOut,
		FindGoModule: findGoModuleTest,
		RunGitWorktreeAdd: func(_ string, worktreePath, branch string) error {
			createdPath = worktreePath
			createdBranch = branch
			return os.MkdirAll(worktreePath, 0o755)
		},
		DescribeJSON: func(string) (string, error) {
			return "{\"ok\":true}", nil
		},
	}

	taskID := "my task"
	code := RunAgent([]string{"start", "--task", "Build context", "--id", taskID}, deps)
	if code != 0 {
		t.Fatalf("start failed: stderr=%s", errOut.String())
	}
	if !strings.Contains(out.String(), "Worktree created at") {
		t.Fatalf("unexpected output: %q", out.String())
	}

	expectedID := sanitizeTaskID(taskID)
	if expectedID == "" {
		t.Fatal("sanitizeTaskID returned empty value")
	}
	expectedWorktree := filepath.Join(root, ".worktrees", expectedID)
	if canonical, err := filepath.EvalSymlinks(expectedWorktree); err == nil {
		expectedWorktree = canonical
	}
	if canonical, err := filepath.EvalSymlinks(createdPath); err == nil {
		createdPath = canonical
	}
	if createdPath != expectedWorktree {
		t.Fatalf("worktree path = %s, want %s", createdPath, expectedWorktree)
	}
	if createdBranch != "agent/"+expectedID {
		t.Fatalf("branch = %s, want agent/%s", createdBranch, expectedID)
	}

	taskFile := filepath.Join(expectedWorktree, "TASK.md")
	content, err := os.ReadFile(taskFile)
	if err != nil {
		t.Fatalf("missing TASK.md: %v", err)
	}
	if !strings.Contains(string(content), "Build context") {
		t.Fatalf("task file missing description: %s", string(content))
	}
	if !strings.Contains(string(content), "{\"ok\":true}") {
		t.Fatalf("task file missing describe output: %s", string(content))
	}

	gitignoreContent, err := os.ReadFile(gitignore)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(gitignoreContent), ".worktrees/") {
		t.Fatalf(".gitignore missing entry: %s", string(gitignoreContent))
	}
}

func TestAgentFinishRequiresMessage(t *testing.T) {
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
	code := RunAgent([]string{"finish", "--id", "task"}, AgentDeps{
		Out:          out,
		Err:          errOut,
		FindGoModule: findGoModuleTest,
	})
	if code == 0 {
		t.Fatalf("expected failure without --message")
	}
	if !strings.Contains(errOut.String(), "--message is required") {
		t.Fatalf("unexpected error output: %q", errOut.String())
	}
}

func TestAgentFinishRunsGitCommands(t *testing.T) {
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

	worktreeID := "my task"
	worktreePath := filepath.Join(root, ".worktrees", sanitizeTaskID(worktreeID))
	if err := os.MkdirAll(worktreePath, 0o755); err != nil {
		t.Fatal(err)
	}
	taskContent := "# TASK\n\n## Task\n\nFinish work\n\n"
	if err := os.WriteFile(filepath.Join(worktreePath, "TASK.md"), []byte(taskContent), 0o644); err != nil {
		t.Fatal(err)
	}

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	var gitCalls [][]string
	var ghArgs []string
	verifyCalled := false
	deps := AgentDeps{
		Out:          out,
		Err:          errOut,
		FindGoModule: findGoModuleTest,
		RunVerify: func(path string) error {
			if canonicalPath(path) != canonicalPath(worktreePath) {
				t.Fatalf("verify path = %s, want %s", path, worktreePath)
			}
			verifyCalled = true
			return nil
		},
		RunGit: func(dir string, args ...string) error {
			gitCalls = append(gitCalls, append([]string{dir}, args...))
			return nil
		},
		RunGh: func(rootDir string, args ...string) error {
			if canonicalPath(rootDir) != canonicalPath(worktreePath) {
				t.Fatalf("gh run from %s, want %s", rootDir, worktreePath)
			}
			ghArgs = args
			return nil
		},
	}

	message := "feat(auth): finish"
	code := RunAgent([]string{"finish", "--id", worktreeID, "--message", message, "--pr"}, deps)
	if code != 0 {
		t.Fatalf("agent finish failed: %s", errOut.String())
	}
	if !verifyCalled {
		t.Fatalf("verify not invoked")
	}
	if len(gitCalls) != 4 {
		t.Fatalf("git calls = %d, want 4", len(gitCalls))
	}
	if gitCalls[0][1] != "add" || gitCalls[0][2] != "-A" {
		t.Fatalf("unexpected git add call: %v", gitCalls[0])
	}
	if gitCalls[1][1] != "commit" || gitCalls[1][2] != "-m" {
		t.Fatalf("unexpected git commit call: %v", gitCalls[1])
	}
	if gitCalls[2][1] != "push" || gitCalls[2][2] != "-u" || gitCalls[2][3] != "origin" {
		t.Fatalf("unexpected git push call: %v", gitCalls[2])
	}
	if gitCalls[3][1] != "worktree" || gitCalls[3][2] != "remove" {
		t.Fatalf("unexpected git remove call: %v", gitCalls[3])
	}
	if len(ghArgs) == 0 || ghArgs[0] != "pr" {
		t.Fatalf("unexpected gh args: %v", ghArgs)
	}
	if !containsArg(ghArgs, "--head") || !containsArg(ghArgs, "agent/"+sanitizeTaskID(worktreeID)) {
		t.Fatalf("gh args missing --head branch: %v", ghArgs)
	}
	if !strings.Contains(out.String(), "finalized and removed") {
		t.Fatalf("unexpected output: %q", out.String())
	}
}

func TestAgentFinishRejectsNonConventionalMessage(t *testing.T) {
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

	worktreeID := "bad msg"
	worktreePath := filepath.Join(root, ".worktrees", sanitizeTaskID(worktreeID))
	if err := os.MkdirAll(worktreePath, 0o755); err != nil {
		t.Fatal(err)
	}

	verifyCalled := false
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	code := RunAgent([]string{"finish", "--id", worktreeID, "--message", "bad message"}, AgentDeps{
		Out:          out,
		Err:          errOut,
		FindGoModule: findGoModuleTest,
		RunVerify: func(string) error {
			verifyCalled = true
			return nil
		},
	})
	if code == 0 {
		t.Fatalf("expected failure for non-conventional message")
	}
	if verifyCalled {
		t.Fatalf("verify should not run when message is invalid")
	}
	if !strings.Contains(errOut.String(), "invalid --message") {
		t.Fatalf("unexpected error output: %q", errOut.String())
	}
}

func containsArg(args []string, want string) bool {
	for _, arg := range args {
		if arg == want {
			return true
		}
	}
	return false
}
