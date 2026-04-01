package generators

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type MakeScheduleOptions struct {
	Name string
	Cron string
}

type ScheduleDeps struct {
	Out io.Writer
	Err io.Writer
	Cwd string
}

func RunMakeSchedule(args []string, d ScheduleDeps) int {
	opts, err := ParseMakeScheduleArgs(args)
	if err != nil {
		fmt.Fprintf(d.Err, "invalid make:schedule arguments: %v\n", err)
		return 1
	}

	cwd := d.Cwd
	if strings.TrimSpace(cwd) == "" {
		var wdErr error
		cwd, wdErr = os.Getwd()
		if wdErr != nil {
			fmt.Fprintf(d.Err, "failed to resolve working directory: %v\n", wdErr)
			return 1
		}
	}
	if looksLikeStarterScaffoldRoot(cwd) {
		fmt.Fprintln(d.Err, "make:schedule is not supported on the starter scaffold yet; no files were changed")
		return 1
	}

	schedulesPath := filepath.Join(cwd, "app", "schedules", "schedules.go")
	contentBytes, err := os.ReadFile(schedulesPath)
	if err != nil {
		fmt.Fprintf(d.Err, "failed to read schedules file %s: %v\n", schedulesPath, err)
		return 1
	}

	entry, err := renderScheduleEntry(opts)
	if err != nil {
		fmt.Fprintf(d.Err, "failed to build schedule entry: %v\n", err)
		return 1
	}
	updated, changed, err := insertBetweenMarkers(
		string(contentBytes),
		"// ship:schedules:start",
		"// ship:schedules:end",
		entry,
	)
	if err != nil {
		fmt.Fprintf(d.Err, "failed to insert schedule entry: %v\n", err)
		return 1
	}
	if changed {
		if err := os.WriteFile(schedulesPath, []byte(updated), 0o644); err != nil {
			fmt.Fprintf(d.Err, "failed to write schedules file %s: %v\n", schedulesPath, err)
			return 1
		}
		fmt.Fprintf(d.Out, "Inserted schedule %q into %s\n", opts.Name, schedulesPath)
		return 0
	}

	fmt.Fprintf(d.Out, "Schedule %q already exists in %s\n", opts.Name, schedulesPath)
	return 0
}

func ParseMakeScheduleArgs(args []string) (MakeScheduleOptions, error) {
	opts := MakeScheduleOptions{}
	if len(args) == 0 {
		return opts, errors.New(`usage: ship make:schedule <Name> --cron "<expr>"`)
	}

	opts.Name = strings.TrimSpace(args[0])
	if opts.Name == "" || strings.HasPrefix(opts.Name, "-") {
		return opts, errors.New(`usage: ship make:schedule <Name> --cron "<expr>"`)
	}

	for i := 1; i < len(args); i++ {
		switch {
		case strings.HasPrefix(args[i], "--cron="):
			opts.Cron = strings.TrimSpace(strings.TrimPrefix(args[i], "--cron="))
		case args[i] == "--cron":
			if i+1 >= len(args) {
				return opts, errors.New(`usage: ship make:schedule <Name> --cron "<expr>"`)
			}
			i++
			opts.Cron = strings.TrimSpace(args[i])
		default:
			return opts, fmt.Errorf("unknown option: %s", args[i])
		}
	}

	if strings.TrimSpace(opts.Cron) == "" {
		return opts, errors.New(`usage: ship make:schedule <Name> --cron "<expr>"`)
	}
	return opts, nil
}

func renderScheduleEntry(opts MakeScheduleOptions) (string, error) {
	tokens := splitWords(opts.Name)
	if len(tokens) == 0 {
		return "", errors.New("schedule name must contain letters or numbers")
	}
	snake := strings.Join(tokens, "_")
	cronExpr := strings.TrimSpace(opts.Cron)

	return fmt.Sprintf(`
	// ship:schedule:%s
	_, _ = s.AddFunc(%q, func() {
		_, _ = jobs.Enqueue(context.Background(), %q, nil, core.EnqueueOptions{})
	})
`, snake, cronExpr, snake), nil
}

func insertBetweenMarkers(src, start, end, snippet string) (string, bool, error) {
	startIdx := strings.Index(src, start)
	endIdx := strings.Index(src, end)
	if startIdx == -1 || endIdx == -1 {
		return "", false, fmt.Errorf("marker pair %q / %q not found", start, end)
	}
	if endIdx <= startIdx {
		return "", false, fmt.Errorf("marker %q appears after %q", end, start)
	}
	block := src[startIdx:endIdx]
	trimmed := strings.TrimSpace(snippet)
	if trimmed == "" {
		return src, false, nil
	}
	if strings.Contains(block, trimmed) {
		return src, false, nil
	}

	insert := snippet
	if !strings.HasSuffix(block, "\n") {
		insert = "\n" + insert
	}
	if !strings.HasSuffix(insert, "\n") {
		insert += "\n"
	}

	var b bytes.Buffer
	b.WriteString(src[:endIdx])
	b.WriteString(insert)
	b.WriteString(src[endIdx:])
	return b.String(), true, nil
}
