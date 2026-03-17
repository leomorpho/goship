package generators

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunMakeCommand_GeneratesCommandAndWiresRegistry(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	mainPath := filepath.Join(root, "cmd", "cli", "main.go")
	if err := os.MkdirAll(filepath.Dir(mainPath), 0o755); err != nil {
		t.Fatal(err)
	}

	mainContent := `package main

func main() {
	// ship:commands:start
	// ship:commands:end
}
`
	if err := os.WriteFile(mainPath, []byte(mainContent), 0o644); err != nil {
		t.Fatal(err)
	}

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	code := RunMakeCommand([]string{"BackfillUserStats"}, MakeCommandDeps{
		Out: out,
		Err: errOut,
		Cwd: root,
	})
	if code != 0 {
		t.Fatalf("exit code = %d, stderr=%s", code, errOut.String())
	}

	commandFile := filepath.Join(root, "app", "commands", "backfill_user_stats.go")
	content, err := os.ReadFile(commandFile)
	if err != nil {
		t.Fatalf("read command file: %v", err)
	}
	text := string(content)
	if !strings.Contains(text, "type BackfillUserStatsCommand struct") {
		t.Fatalf("expected generated command type, got:\n%s", text)
	}
	if !strings.Contains(text, `func (c *BackfillUserStatsCommand) Name() string { return "backfill:user:stats" }`) {
		t.Fatalf("expected generated command name, got:\n%s", text)
	}

	mainUpdated, err := os.ReadFile(mainPath)
	if err != nil {
		t.Fatalf("read main.go: %v", err)
	}
	mainText := string(mainUpdated)
	if !strings.Contains(mainText, "registry.Register(&appcommands.BackfillUserStatsCommand{Container: container})") {
		t.Fatalf("expected registry wiring snippet, got:\n%s", mainText)
	}
}

func TestRunMakeCommand_IsIdempotent(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	mainPath := filepath.Join(root, "cmd", "cli", "main.go")
	if err := os.MkdirAll(filepath.Dir(mainPath), 0o755); err != nil {
		t.Fatal(err)
	}
	mainContent := `package main

func main() {
	// ship:commands:start
	// ship:commands:end
}
`
	if err := os.WriteFile(mainPath, []byte(mainContent), 0o644); err != nil {
		t.Fatal(err)
	}

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	deps := MakeCommandDeps{Out: out, Err: errOut, Cwd: root}
	if code := RunMakeCommand([]string{"BackfillUserStats"}, deps); code != 0 {
		t.Fatalf("first run failed: code=%d stderr=%s", code, errOut.String())
	}

	out.Reset()
	errOut.Reset()
	if code := RunMakeCommand([]string{"BackfillUserStats"}, deps); code == 0 {
		t.Fatalf("expected duplicate run to fail")
	}

	mainUpdated, err := os.ReadFile(mainPath)
	if err != nil {
		t.Fatalf("read main.go: %v", err)
	}
	mainText := string(mainUpdated)
	if strings.Count(mainText, "BackfillUserStatsCommand") != 1 {
		t.Fatalf("expected one registry entry, got:\n%s", mainText)
	}
}
