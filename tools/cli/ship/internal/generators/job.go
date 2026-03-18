package generators

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type MakeJobOptions struct {
	Name string
}

type MakeJobDeps struct {
	Out io.Writer
	Err io.Writer
	Cwd string
}

func RunMakeJob(args []string, d MakeJobDeps) int {
	opts, err := ParseMakeJobArgs(args)
	if err != nil {
		fmt.Fprintf(d.Err, "invalid make:job arguments: %v\n", err)
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

	tokens := splitWords(opts.Name)
	if len(tokens) == 0 {
		fmt.Fprintln(d.Err, "invalid make:job arguments: usage: ship make:job <Name>")
		return 1
	}

	pascal := toPascalFromParts(tokens)
	snake := strings.Join(tokens, "_")
	typeName := "job." + snake

	jobPath := filepath.Join(cwd, "app", "jobs", snake+".go")
	testPath := filepath.Join(cwd, "app", "jobs", snake+"_test.go")
	if _, err := os.Stat(jobPath); err == nil {
		fmt.Fprintf(d.Err, "refusing to overwrite existing job file: %s\n", jobPath)
		return 1
	}
	if _, err := os.Stat(testPath); err == nil {
		fmt.Fprintf(d.Err, "refusing to overwrite existing job test file: %s\n", testPath)
		return 1
	}

	if err := os.MkdirAll(filepath.Dir(jobPath), 0o755); err != nil {
		fmt.Fprintf(d.Err, "failed to create jobs directory: %v\n", err)
		return 1
	}
	if err := os.WriteFile(jobPath, []byte(renderJobFile(pascal, typeName)), 0o644); err != nil {
		fmt.Fprintf(d.Err, "failed to write job file: %v\n", err)
		return 1
	}
	if err := os.WriteFile(testPath, []byte(renderJobTestFile(pascal, typeName)), 0o644); err != nil {
		fmt.Fprintf(d.Err, "failed to write job test file: %v\n", err)
		return 1
	}

	fmt.Fprintf(d.Out, "Generated job: %s\n", jobPath)
	fmt.Fprintf(d.Out, "Generated job test: %s\n", testPath)
	fmt.Fprintf(d.Out, "Next step: wire %s with Register%s(c.CoreJobs, Handle%s) where your runtime registers app jobs.\n", typeName, pascal, pascal)
	return 0
}

func ParseMakeJobArgs(args []string) (MakeJobOptions, error) {
	opts := MakeJobOptions{}
	if len(args) == 0 {
		return opts, errors.New("usage: ship make:job <Name>")
	}
	opts.Name = strings.TrimSpace(args[0])
	if opts.Name == "" || strings.HasPrefix(opts.Name, "-") {
		return opts, errors.New("usage: ship make:job <Name>")
	}
	for i := 1; i < len(args); i++ {
		if strings.HasPrefix(args[i], "-") {
			return opts, fmt.Errorf("unknown option: %s", args[i])
		}
		return opts, fmt.Errorf("unexpected argument: %s", args[i])
	}
	return opts, nil
}

func renderJobFile(pascal, typeName string) string {
	return fmt.Sprintf(`package tasks

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/leomorpho/goship/framework/core"
)

const Type%s = %q

type %sPayload struct {
	// TODO: add typed payload fields.
}

func Register%s(jobs core.Jobs, handler core.JobHandler) error {
	if jobs == nil {
		return errors.New("jobs is required")
	}
	if handler == nil {
		return errors.New("handler is required")
	}
	return jobs.Register(Type%s, handler)
}

func Handle%s(_ context.Context, payload []byte) error {
	if len(payload) == 0 {
		return nil
	}

	var p %sPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		return fmt.Errorf("decode %%s payload: %%w", Type%s, err)
	}

	// TODO: implement job behavior.
	return nil
}
`, pascal, typeName, pascal, pascal, pascal, pascal, pascal, pascal)
}

func renderJobTestFile(pascal, typeName string) string {
	return fmt.Sprintf(`package tasks

import (
	"context"
	"strings"
	"testing"

	"github.com/leomorpho/goship/framework/core"
)

type fake%[1]sJobs struct {
	registeredName    string
	registeredHandler core.JobHandler
}

func (f *fake%[1]sJobs) Register(name string, handler core.JobHandler) error {
	f.registeredName = name
	f.registeredHandler = handler
	return nil
}

func (*fake%[1]sJobs) Enqueue(context.Context, string, []byte, core.EnqueueOptions) (string, error) {
	return "", nil
}

func (*fake%[1]sJobs) StartWorker(context.Context) error    { return nil }
func (*fake%[1]sJobs) StartScheduler(context.Context) error { return nil }
func (*fake%[1]sJobs) Stop(context.Context) error           { return nil }
func (*fake%[1]sJobs) Capabilities() core.JobCapabilities  { return core.JobCapabilities{} }

func TestRegister%[1]s(t *testing.T) {
	t.Parallel()

	jobs := &fake%[1]sJobs{}
	if err := Register%[1]s(jobs, Handle%[1]s); err != nil {
		t.Fatalf("unexpected error: %%v", err)
	}
	if jobs.registeredName != Type%[1]s {
		t.Fatalf("registeredName = %%q, want %%q", jobs.registeredName, Type%[1]s)
	}
	if jobs.registeredHandler == nil {
		t.Fatal("expected handler registration")
	}
}

func TestHandle%[1]s_InvalidPayload(t *testing.T) {
	t.Parallel()

	err := Handle%[1]s(context.Background(), []byte("{"))
	if err == nil {
		t.Fatal("expected invalid payload error")
	}
	if !strings.Contains(err.Error(), Type%[1]s) {
		t.Fatalf("error = %%q, want contains %%q", err.Error(), Type%[1]s)
	}
}
`, pascal)
}
