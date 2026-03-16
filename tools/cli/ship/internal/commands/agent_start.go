package commands

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"
)

const taskDescriptionMaxBytes = 64 * 1024

func runAgentStart(args []string, d AgentDeps) int {
	fs := flag.NewFlagSet("agent:start", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	taskDesc := fs.String("task", "", "task description")
	rawID := fs.String("id", "", "task identifier (defaults to UTC timestamp)")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(d.Err, "invalid agent:start arguments: %v\n", err)
		return 1
	}
	if fs.NArg() > 0 {
		fmt.Fprintf(d.Err, "unexpected agent:start arguments: %v\n", fs.Args())
		return 1
	}

	desc := strings.TrimSpace(*taskDesc)
	if desc == "" {
		stdinDesc, err := readTaskDescFromStdin()
		if err != nil {
			fmt.Fprintf(d.Err, "failed to read task description: %v\n", err)
			return 1
		}
		desc = stdinDesc
	}
	if desc == "" {
		fmt.Fprintln(d.Err, "task description is required (use --task or pipe text into stdin)")
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

	identifier := sanitizeTaskID(*rawID)
	if identifier == "" {
		identifier = time.Now().UTC().Format("20060102T150405Z")
	}
	worktreesDir := filepath.Join(root, ".worktrees")
	if err := os.MkdirAll(worktreesDir, 0o755); err != nil {
		fmt.Fprintf(d.Err, "failed to create .worktrees directory: %v\n", err)
		return 1
	}
	worktreePath := filepath.Join(worktreesDir, identifier)
	if _, err := os.Stat(worktreePath); err == nil {
		fmt.Fprintf(d.Err, "worktree %s already exists\n", identifier)
		return 1
	}
	branch := fmt.Sprintf("agent/%s", identifier)
	gitAdd := d.RunGitWorktreeAdd
	if gitAdd == nil {
		gitAdd = defaultGitWorktreeAdd
	}
	if err := gitAdd(root, worktreePath, branch); err != nil {
		fmt.Fprintf(d.Err, "failed to create worktree: %v\n", err)
		return 1
	}

	describeJSON := d.DescribeJSON
	if describeJSON == nil {
		describeJSON = defaultDescribeJSON
	}
	describeOutput, err := describeJSON(root)
	if err != nil {
		fmt.Fprintf(d.Err, "failed to capture codebase map: %v\n", err)
		return 1
	}

	claudeFiles, err := collectClaudeFiles(root)
	if err != nil {
		fmt.Fprintf(d.Err, "failed to load CLAUDE context: %v\n", err)
		return 1
	}

	taskContent := buildTaskFileContent(identifier, desc, describeOutput, claudeFiles)
	taskPath := filepath.Join(worktreePath, "TASK.md")
	if err := os.WriteFile(taskPath, []byte(taskContent), 0o644); err != nil {
		fmt.Fprintf(d.Err, "failed to write TASK.md: %v\n", err)
		return 1
	}

	if err := ensureWorktreesIgnored(root); err != nil {
		fmt.Fprintf(d.Err, "warning: failed to add .worktrees to .gitignore: %v\n", err)
	}

	relPath := filepath.ToSlash(filepath.Join(".worktrees", identifier))
	fmt.Fprintf(d.Out, "Worktree created at %s. Branch: %s.\n", relPath, branch)
	return 0
}

func readTaskDescFromStdin() (string, error) {
	info, err := os.Stdin.Stat()
	if err != nil {
		return "", err
	}
	if info.Mode()&os.ModeCharDevice != 0 {
		return "", nil
	}
	data, err := io.ReadAll(io.LimitReader(os.Stdin, taskDescriptionMaxBytes))
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func sanitizeTaskID(raw string) string {
	trimmed := strings.TrimSpace(raw)
	var b strings.Builder
	lastDash := false
	for _, r := range trimmed {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' || r == '.':
			b.WriteRune(r)
			lastDash = false
		case unicode.IsSpace(r) || r == '/' || r == '\\' || r == ':':
			if b.Len() == 0 || lastDash {
				continue
			}
			b.WriteRune('-')
			lastDash = true
		default:
			lastDash = false
		}
	}
	return strings.Trim(b.String(), "-._")
}

func defaultGitWorktreeAdd(root, worktreePath, branch string) error {
	cmd := exec.Command("git", "worktree", "add", worktreePath, "-b", branch)
	cmd.Dir = root
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git worktree add failed: %w\n%s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func defaultDescribeJSON(root string) (string, error) {
	var out bytes.Buffer
	var errOut bytes.Buffer
	var exitCode int
	if err := withWorkingDir(root, func() error {
		exitCode = RunDescribe([]string{}, DescribeDeps{
			Out: &out,
			Err: &errOut,
			FindGoModule: func(start string) (string, string, error) {
				return root, "", nil
			},
		})
		return nil
	}); err != nil {
		return "", err
	}
	if exitCode != 0 {
		return "", fmt.Errorf("ship describe failed: %s", strings.TrimSpace(errOut.String()))
	}
	return out.String(), nil
}

type claudeFile struct {
	Path    string
	Content string
}

func collectClaudeFiles(root string) ([]claudeFile, error) {
	files := make([]claudeFile, 0)
	if err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		name := d.Name()
		if name != "CLAUDE.md" && name != "CLAUDE.md.template" {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		files = append(files, claudeFile{Path: filepath.ToSlash(rel), Content: string(content)})
		return nil
	}); err != nil {
		return nil, err
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return files, nil
}

func buildTaskFileContent(id, description, describeJSON string, claudeFiles []claudeFile) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# TASK %s\n\n", id)
	b.WriteString("## Task\n\n")
	b.WriteString(description)
	b.WriteString("\n\n")
	b.WriteString("## Codebase State (ship describe --json)\n\n")
	b.WriteString("```json\n")
	b.WriteString(strings.TrimSpace(describeJSON))
	b.WriteString("\n```\n\n")
	b.WriteString("## CLAUDE Context\n\n")
	if len(claudeFiles) == 0 {
		b.WriteString("_No CLAUDE.md documents found in this repo._\n")
		return b.String()
	}
	for _, f := range claudeFiles {
		fmt.Fprintf(&b, "### %s\n\n", f.Path)
		b.WriteString("```\n")
		b.WriteString(strings.TrimSpace(f.Content))
		b.WriteString("\n```\n\n")
	}
	return b.String()
}

func ensureWorktreesIgnored(root string) error {
	path := filepath.Join(root, ".gitignore")
	entry := ".worktrees/"
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return os.WriteFile(path, []byte(entry+"\n"), 0o644)
		}
		return err
	}
	content := string(data)
	if strings.Contains(content, entry) {
		return nil
	}
	if len(content) > 0 && content[len(content)-1] != '\n' {
		content += "\n"
	}
	return os.WriteFile(path, []byte(content+entry+"\n"), 0o644)
}
