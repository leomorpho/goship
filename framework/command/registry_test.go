package command

import (
	"context"
	"errors"
	"strings"
	"testing"
)

type testCommand struct {
	name string
	run  func(context.Context, []string) error
}

func (c testCommand) Name() string        { return c.name }
func (c testCommand) Description() string { return "test" }
func (c testCommand) Run(ctx context.Context, args []string) error {
	if c.run != nil {
		return c.run(ctx, args)
	}
	return nil
}

func TestRegistryRegisterAndRun(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	called := false
	err := reg.Register(testCommand{
		name: "send:digest",
		run: func(_ context.Context, args []string) error {
			called = true
			if len(args) != 1 || args[0] != "--dry-run" {
				t.Fatalf("args = %v, want [--dry-run]", args)
			}
			return nil
		},
	})
	if err != nil {
		t.Fatalf("register error: %v", err)
	}

	if err := reg.Run(context.Background(), []string{"send:digest", "--dry-run"}); err != nil {
		t.Fatalf("run error: %v", err)
	}
	if !called {
		t.Fatal("expected command to run")
	}
}

func TestRegistryValidation(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	if err := reg.Register(nil); err == nil {
		t.Fatal("expected nil command error")
	}
	if err := reg.Register(testCommand{name: ""}); err == nil {
		t.Fatal("expected empty-name error")
	}
	if err := reg.Register(testCommand{name: "demo"}); err != nil {
		t.Fatalf("register demo error: %v", err)
	}
	if err := reg.Register(testCommand{name: "demo"}); err == nil {
		t.Fatal("expected duplicate-name error")
	}
}

func TestRegistryRunErrors(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	if err := reg.Run(context.Background(), nil); err == nil {
		t.Fatal("expected missing command error")
	}
	if err := reg.Run(context.Background(), []string{"missing"}); err == nil {
		t.Fatal("expected unknown command error")
	}

	sentinel := errors.New("boom")
	if err := reg.Register(testCommand{name: "explode", run: func(context.Context, []string) error { return sentinel }}); err != nil {
		t.Fatalf("register explode error: %v", err)
	}
	if err := reg.Run(context.Background(), []string{"explode"}); !errors.Is(err, sentinel) {
		t.Fatalf("run error = %v, want %v", err, sentinel)
	}
}

func TestRegistryUsageIncludesRegisteredCommands(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	_ = reg.Register(testCommand{name: "zeta"})
	_ = reg.Register(testCommand{name: "alpha"})
	usage := reg.Usage()

	if !strings.Contains(usage, "alpha") || !strings.Contains(usage, "zeta") {
		t.Fatalf("usage missing command names: %s", usage)
	}
}
