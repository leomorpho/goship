package commands

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/leomorpho/goship/config"
)

func TestRunDBPromote_JsonMutationPlanContract(t *testing.T) {
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}

	code := RunDB([]string{"promote", "--json", "--dry-run"}, DBDeps{
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
		MutationPlan struct {
			DryRun bool              `json:"dry_run"`
			Values map[string]string `json:"values"`
		} `json:"mutation_plan"`
	}
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatalf("decode json: %v\n%s", err, out.String())
	}

	if !payload.MutationPlan.DryRun {
		t.Fatalf("expected dry-run mutation plan in %s", out.String())
	}
	for key, want := range map[string]string{
		"PAGODA_RUNTIME_PROFILE": "server-db",
		"PAGODA_ADAPTERS_DB":     "postgres",
		"PAGODA_ADAPTERS_CACHE":  "redis",
		"PAGODA_ADAPTERS_JOBS":   "asynq",
	} {
		if got := payload.MutationPlan.Values[key]; got != want {
			t.Fatalf("mutation_plan.values[%q] = %q, want %q\n%s", key, got, want, out.String())
		}
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
