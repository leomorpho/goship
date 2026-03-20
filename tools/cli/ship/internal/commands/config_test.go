package commands

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestRunConfigValidateJSON(t *testing.T) {
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}

	code := RunConfig([]string{"validate", "--json"}, ConfigDeps{
		Out: out,
		Err: errOut,
		FindGoModule: func(start string) (string, string, error) {
			return start, "github.com/leomorpho/goship", nil
		},
	})
	if code != 0 {
		t.Fatalf("exit code = %d, want 0, stderr=%s", code, errOut.String())
	}

	var payload struct {
		OK        bool `json:"ok"`
		Variables []struct {
			Name string `json:"name"`
		} `json:"variables"`
	}
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode json: %v", err)
	}
	if !payload.OK {
		t.Fatalf("ok = false, want true")
	}
	if len(payload.Variables) == 0 {
		t.Fatal("variables = 0, want at least one config variable")
	}
}

func TestRunConfigValidateText(t *testing.T) {
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}

	code := RunConfig([]string{"validate"}, ConfigDeps{
		Out: out,
		Err: errOut,
		FindGoModule: func(start string) (string, string, error) {
			return start, "github.com/leomorpho/goship", nil
		},
	})
	if code != 0 {
		t.Fatalf("exit code = %d, want 0, stderr=%s", code, errOut.String())
	}
	if !strings.Contains(out.String(), "NAME") {
		t.Fatalf("stdout = %q, want header", out.String())
	}
	if !strings.Contains(out.String(), "config validation: OK") {
		t.Fatalf("stdout = %q, want success footer", out.String())
	}
}
