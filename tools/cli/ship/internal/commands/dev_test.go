package commands

import (
	"bytes"
	"testing"
)

func TestRunDev_DefaultModeUsesResolver(t *testing.T) {
	t.Parallel()

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	devAllCalls := 0
	runCalls := 0

	code := RunDev([]string{}, DevDeps{
		Out: out,
		Err: errOut,
		RunCmd: func(name string, args ...string) int {
			runCalls++
			return 0
		},
		RunDevAll: func() int {
			devAllCalls++
			return 0
		},
		ResolveDefaultMode: func() (string, error) {
			return "all", nil
		},
	})
	if code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
	if devAllCalls != 1 {
		t.Fatalf("RunDevAll calls = %d, want 1", devAllCalls)
	}
	if runCalls != 0 {
		t.Fatalf("RunCmd calls = %d, want 0", runCalls)
	}
}

func TestRunDev_ExplicitModeOverridesResolver(t *testing.T) {
	t.Parallel()

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	devAllCalls := 0
	gotName := ""
	gotArgs := []string{}

	code := RunDev([]string{"web"}, DevDeps{
		Out: out,
		Err: errOut,
		RunCmd: func(name string, args ...string) int {
			gotName = name
			gotArgs = append([]string{}, args...)
			return 0
		},
		RunDevAll: func() int {
			devAllCalls++
			return 0
		},
		ResolveDefaultMode: func() (string, error) {
			return "all", nil
		},
	})
	if code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
	if devAllCalls != 0 {
		t.Fatalf("RunDevAll calls = %d, want 0", devAllCalls)
	}
	if gotName != "go" {
		t.Fatalf("RunCmd name = %q, want go", gotName)
	}
	if len(gotArgs) != 2 || gotArgs[0] != "run" || gotArgs[1] != "./cmd/web" {
		t.Fatalf("RunCmd args = %v, want [run ./cmd/web]", gotArgs)
	}
}
