package commands

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/leomorpho/goship/config"
)

func TestRunDBPromote_OrchestrationHints_RedSpec(t *testing.T) {
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}

	code := RunDB([]string{"promote", "--json"}, DBDeps{
		Out: out,
		Err: errOut,
		LoadConfig: func() (config.Config, error) {
			cfg := config.Config{}
			cfg.Database.DbMode = config.DBModeEmbedded
			cfg.Database.Driver = config.DBDriverSQLite
			return cfg, nil
		},
	})
	if code != 0 {
		t.Fatalf("exit code = %d, stderr=%s", code, errOut.String())
	}

	var payload struct {
		SuggestedCommands []string `json:"suggested_commands"`
	}
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatalf("decode json: %v\n%s", err, out.String())
	}

	want := []string{
		"ship profile:set standard",
		"ship adapter:set db=postgres cache=redis jobs=asynq",
	}
	for _, token := range want {
		if !containsString(payload.SuggestedCommands, token) {
			t.Fatalf("suggested commands missing %q:\n%s", token, out.String())
		}
	}

	if !strings.Contains(out.String(), "ship profile:set standard") || !strings.Contains(out.String(), "ship adapter:set db=postgres cache=redis jobs=asynq") {
		t.Fatalf("stdout missing orchestration hints:\n%s", out.String())
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
