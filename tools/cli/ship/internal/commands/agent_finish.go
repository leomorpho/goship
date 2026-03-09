package commands

import (
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func runAgentFinish(args []string, d AgentDeps) int {
	fs := flag.NewFlagSet("agent:finish", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	idFlag := fs.String("id", "", "task identifier created by agent:start")
	message := fs.String("message", "", "conventional commit message")
	prFlag := fs.Bool("pr", false, "create a PR via gh")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(d.Err, "invalid agent:finish arguments: %v\n", err)
		return 1
	}
	if fs.NArg() > 0 {
		fmt.Fprintf(d.Err, "unexpected agent:finish arguments: %v\n", fs.Args())
		return 1
	}

	if strings.TrimSpace(*idFlag) == "" {
		fmt.Fprintln(d.Err, "--id is required")
		return 1
	}
	sanitizedID := sanitizeTaskID(*idFlag)
	if sanitizedID == "" {
		fmt.Fprintln(d.Err, "invalid --id value")
		return 1
	}
	if strings.TrimSpace(*message) == "" {
		fmt.Fprintln(d.Err, "--message is required")
		return 1
	}

	wd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(d.Err, "failed to resolve working directory: %v\n", err)
		return 1
	}
	root, _, err := d.FindGoModule(wd)
	if err != nil {
		fmt.Fprintf(d.Err, "failed to resolve project root (go.mod): %v\n", err)
		return 1
	}

	worktreePath := filepath.Join(root, ".worktrees", sanitizedID)
	if _, err := os.Stat(worktreePath); os.IsNotExist(err) {
		fmt.Fprintf(d.Err, "worktree %s does not exist\n", sanitizedID)
		return 1
	} else if err != nil {
		fmt.Fprintf(d.Err, "failed to inspect worktree: %v\n", err)
		return 1
	}

	runVerify := d.RunVerify
	if runVerify == nil {
		runVerify = func(path string) error {
			return defaultRunVerify(path, d.Out, d.Err)
		}
	}
	if err := runVerify(worktreePath); err != nil {
		fmt.Fprintf(d.Err, "ship verify failed: %v\n", err)
		return 1
	}

	runGit := d.RunGit
	if runGit == nil {
		runGit = defaultRunGit
	}
	if err := runGit(worktreePath, "add", "-A"); err != nil {
		fmt.Fprintf(d.Err, "git add failed: %v\n", err)
		return 1
	}
	if err := runGit(worktreePath, "commit", "-m", strings.TrimSpace(*message)); err != nil {
		fmt.Fprintf(d.Err, "git commit failed: %v\n", err)
		return 1
	}

	if *prFlag {
		description := readTaskDescriptionFromFile(filepath.Join(worktreePath, "TASK.md"))
		runGh := d.RunGh
		if runGh == nil {
			runGh = defaultRunGh
		}
		body := "Agent task: " + description
		if err := runGh(root, "pr", "create", "--title", strings.TrimSpace(*message), "--body", body); err != nil {
			fmt.Fprintf(d.Err, "gh pr create failed: %v\n", err)
			return 1
		}
	}

	if err := runGit(root, "worktree", "remove", worktreePath); err != nil {
		fmt.Fprintf(d.Err, "failed to remove worktree: %v\n", err)
		return 1
	}

	fmt.Fprintf(d.Out, "Worktree %s finalized and removed.\n", sanitizedID)
	return 0
}

func defaultRunVerify(path string, out, errOut io.Writer) error {
	var exitCode int
	err := withWorkingDir(path, func() error {
		exitCode = RunVerify([]string{}, VerifyDeps{
			Out: out,
			Err: errOut,
			FindGoModule: func(start string) (string, string, error) {
				return path, "", nil
			},
		})
		return nil
	})
	if err != nil {
		return err
	}
	if exitCode != 0 {
		return fmt.Errorf("ship verify exited with %d", exitCode)
	}
	return nil
}

func defaultRunGit(dir string, args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git %s failed: %w\n%s", strings.Join(args, " "), err, strings.TrimSpace(string(output)))
	}
	return nil
}

func defaultRunGh(root string, args ...string) error {
	cmd := exec.Command("gh", args...)
	cmd.Dir = root
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("gh %s failed: %w\n%s", strings.Join(args, " "), err, strings.TrimSpace(string(output)))
	}
	return nil
}

func readTaskDescriptionFromFile(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	content := string(data)
	start := strings.Index(content, "## Task")
	if start == -1 {
		return ""
	}
	rest := strings.TrimSpace(content[start+len("## Task"):])
	end := strings.Index(rest, "\n## ")
	if end > -1 {
		rest = rest[:end]
	}
	return strings.TrimSpace(rest)
}
