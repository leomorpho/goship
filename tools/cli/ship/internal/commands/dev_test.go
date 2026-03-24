package commands

import (
	"bytes"
	"strings"
	"testing"

	policies "github.com/leomorpho/goship/tools/cli/ship/internal/policies"
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

func TestRunDev_ExplicitFlagOverridesResolver(t *testing.T) {
	t.Parallel()

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	devAllCalls := 0
	gotName := ""
	gotArgs := []string{}

	code := RunDev([]string{"--web"}, DevDeps{
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
	if gotName != "air" {
		t.Fatalf("RunCmd name = %q, want air", gotName)
	}
	if len(gotArgs) != 2 || gotArgs[0] != "-c" || gotArgs[1] != ".air.toml" {
		t.Fatalf("RunCmd args = %v, want [-c .air.toml]", gotArgs)
	}
}

func TestRunDev_DefaultModeFallbacksStayOnWebLoop(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		resolve  func() (string, error)
		wantName string
		wantArgs []string
		wantAll  int
	}{
		{
			name: "resolver returns web",
			resolve: func() (string, error) {
				return "web", nil
			},
			wantName: "air",
			wantArgs: []string{"-c", ".air.toml"},
		},
		{
			name: "resolver returns unsupported mode",
			resolve: func() (string, error) {
				return "sidecar", nil
			},
			wantName: "air",
			wantArgs: []string{"-c", ".air.toml"},
		},
		{
			name: "resolver errors",
			resolve: func() (string, error) {
				return "", assertiveError("boom")
			},
			wantName: "air",
			wantArgs: []string{"-c", ".air.toml"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := &bytes.Buffer{}
			errOut := &bytes.Buffer{}
			devAllCalls := 0
			gotName := ""
			gotArgs := []string{}

			code := RunDev([]string{}, DevDeps{
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
				ResolveDefaultMode: tt.resolve,
			})
			if code != 0 {
				t.Fatalf("code = %d, want 0", code)
			}
			if devAllCalls != tt.wantAll {
				t.Fatalf("RunDevAll calls = %d, want %d", devAllCalls, tt.wantAll)
			}
			if gotName != tt.wantName {
				t.Fatalf("RunCmd name = %q, want %q", gotName, tt.wantName)
			}
			if len(gotArgs) != len(tt.wantArgs) {
				t.Fatalf("RunCmd args = %v, want %v", gotArgs, tt.wantArgs)
			}
			for i := range gotArgs {
				if gotArgs[i] != tt.wantArgs[i] {
					t.Fatalf("RunCmd args = %v, want %v", gotArgs, tt.wantArgs)
				}
			}
		})
	}
}

func TestRunDev_FailsFastWhenGeneratedAppScaffoldBroken(t *testing.T) {
	t.Parallel()

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	runCalls := 0

	code := RunDev([]string{}, DevDeps{
		Out: out,
		Err: errOut,
		RunCmd: func(name string, args ...string) int {
			runCalls++
			return 0
		},
		FindGoModule: func(start string) (string, string, error) {
			return "/tmp/example", "example.com/test", nil
		},
		FastPathGeneratedIssues: func(root string) []policies.DoctorIssue {
			return []policies.DoctorIssue{{
				Code:    "DX001",
				Message: "missing required directory: app/foundation",
				Fix:     "create app/foundation or regenerate the app scaffold with `ship new`",
			}}
		},
	})
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if runCalls != 0 {
		t.Fatalf("RunCmd calls = %d, want 0", runCalls)
	}
	stderr := errOut.String()
	if !strings.Contains(stderr, "dev preflight failed: generated app scaffold is broken") {
		t.Fatalf("stderr = %q, want preflight failure", stderr)
	}
	if !strings.Contains(stderr, "[DX001] missing required directory: app/foundation") {
		t.Fatalf("stderr = %q, want root-cause issue", stderr)
	}
	if !strings.Contains(stderr, "Next step: run `ship doctor --json`") {
		t.Fatalf("stderr = %q, want corrective next step", stderr)
	}
}

type assertiveError string

func (e assertiveError) Error() string {
	return string(e)
}
