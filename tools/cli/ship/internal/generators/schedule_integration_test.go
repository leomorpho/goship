package generators

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunMakeSchedule_InsertsSnippetBetweenMarkers(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	schedulesPath := filepath.Join(root, "app", "schedules", "schedules.go")
	if err := os.MkdirAll(filepath.Dir(schedulesPath), 0o755); err != nil {
		t.Fatal(err)
	}

	base := `package schedules

import (
	"context"

	"github.com/robfig/cron/v3"
	"github.com/leomorpho/goship/framework/core"
)

func Register(s *cron.Cron, jobs core.Jobs) {
	// ship:schedules:start
	// ship:schedules:end
}
`
	if err := os.WriteFile(schedulesPath, []byte(base), 0o644); err != nil {
		t.Fatal(err)
	}

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	code := RunMakeSchedule([]string{"DailyReport", "--cron", "0 9 * * *"}, ScheduleDeps{
		Out: out,
		Err: errOut,
		Cwd: root,
	})
	if code != 0 {
		t.Fatalf("exit code = %d, stderr=%s", code, errOut.String())
	}

	content, err := os.ReadFile(schedulesPath)
	if err != nil {
		t.Fatal(err)
	}
	text := string(content)
	if !strings.Contains(text, `s.AddFunc("0 9 * * *", func()`) {
		t.Fatalf("expected inserted cron expression, got:\n%s", text)
	}
	if !strings.Contains(text, `jobs.Enqueue(context.Background(), "daily_report", nil, core.EnqueueOptions{})`) {
		t.Fatalf("expected inserted enqueue call, got:\n%s", text)
	}
}

func TestRunMakeSchedule_IsIdempotent(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	schedulesPath := filepath.Join(root, "app", "schedules", "schedules.go")
	if err := os.MkdirAll(filepath.Dir(schedulesPath), 0o755); err != nil {
		t.Fatal(err)
	}

	base := `package schedules

import (
	"context"

	"github.com/robfig/cron/v3"
	"github.com/leomorpho/goship/framework/core"
)

func Register(s *cron.Cron, jobs core.Jobs) {
	// ship:schedules:start
	// ship:schedules:end
}
`
	if err := os.WriteFile(schedulesPath, []byte(base), 0o644); err != nil {
		t.Fatal(err)
	}

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	deps := ScheduleDeps{Out: out, Err: errOut, Cwd: root}
	if code := RunMakeSchedule([]string{"DailyReport", "--cron", "0 9 * * *"}, deps); code != 0 {
		t.Fatalf("first run failed with code=%d stderr=%s", code, errOut.String())
	}

	out.Reset()
	errOut.Reset()
	if code := RunMakeSchedule([]string{"DailyReport", "--cron", "0 9 * * *"}, deps); code != 0 {
		t.Fatalf("second run failed with code=%d stderr=%s", code, errOut.String())
	}

	content, err := os.ReadFile(schedulesPath)
	if err != nil {
		t.Fatal(err)
	}
	text := string(content)
	if strings.Count(text, `jobs.Enqueue(context.Background(), "daily_report", nil, core.EnqueueOptions{})`) != 1 {
		t.Fatalf("expected one inserted schedule entry, got:\n%s", text)
	}
}
