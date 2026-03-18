package commands

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/leomorpho/goship/config"
)

func TestRunDBPromote(t *testing.T) {
	t.Run("prints manual promotion plan for sqlite sources", func(t *testing.T) {
		out := &bytes.Buffer{}
		errOut := &bytes.Buffer{}

		code := RunDB([]string{"promote"}, DBDeps{
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
		for _, token := range []string{
			"DB promote plan:",
			"promotion_path: sqlite-to-postgres-manual-v1",
			"compatible_targets: postgres",
			"step: freeze writes for the source app",
			"step: switch config to Postgres and unfreeze writes",
			"planning only; db:promote does not mutate files or run migrations yet",
		} {
			if !strings.Contains(out.String(), token) {
				t.Fatalf("stdout missing %q:\n%s", token, out.String())
			}
		}
	})

	t.Run("prints json plan for tooling", func(t *testing.T) {
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
			Database struct {
				Mode              string   `json:"mode"`
				Driver            string   `json:"driver"`
				CompatibleTargets []string `json:"compatible_targets"`
				PromotionPath     string   `json:"promotion_path"`
			} `json:"database"`
			Steps []string `json:"steps"`
		}
		if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
			t.Fatalf("decode json: %v\n%s", err, out.String())
		}
		if payload.Database.PromotionPath != config.PromotionPathSQLiteToPostgresManualV1 {
			t.Fatalf("promotion_path = %q", payload.Database.PromotionPath)
		}
		if len(payload.Steps) == 0 {
			t.Fatalf("expected steps in %s", out.String())
		}
	})
}
