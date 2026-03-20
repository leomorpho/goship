package commands

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/leomorpho/goship/config"
)

func TestRunDBPromote(t *testing.T) {
	t.Run("prints dry-run mutation plan for sqlite sources", func(t *testing.T) {
		root := t.TempDir()
		envPath := filepath.Join(root, ".env")
		initial := strings.Join([]string{
			"PAGODA_RUNTIME_PROFILE=single-node",
			"PAGODA_PROCESSES_WEB=true",
			"PAGODA_PROCESSES_WORKER=true",
			"PAGODA_PROCESSES_SCHEDULER=true",
			"PAGODA_PROCESSES_COLOCATED=true",
			"PAGODA_ADAPTERS_DB=sqlite",
			"PAGODA_DATABASE_DRIVER=sqlite",
			"PAGODA_DB_DRIVER=sqlite",
			"PAGODA_DATABASE_DBMODE=embedded",
			"PAGODA_ADAPTERS_CACHE=otter",
			"PAGODA_CACHE_DRIVER=otter",
			"PAGODA_ADAPTERS_JOBS=backlite",
			"PAGODA_JOBS_DRIVER=backlite",
			"",
		}, "\n")
		if err := os.WriteFile(envPath, []byte(initial), 0o644); err != nil {
			t.Fatalf("write env: %v", err)
		}

		prevWD, err := os.Getwd()
		if err != nil {
			t.Fatalf("getwd: %v", err)
		}
		if err := os.Chdir(root); err != nil {
			t.Fatalf("chdir %s: %v", root, err)
		}
		t.Cleanup(func() { _ = os.Chdir(prevWD) })

		out := &bytes.Buffer{}
		errOut := &bytes.Buffer{}

		code := RunDB([]string{"promote", "--dry-run"}, DBDeps{
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
			"- mode: dry-run (no files changed)",
			"promotion_state_schema: promotion-state-machine-v1",
			"current_state: sqlite-source-ready",
			"next_state: config-mutated-awaiting-import",
			"promotion_path: sqlite-to-postgres-manual-v1",
			"compatible_targets: postgres",
			"state: config-mutated-awaiting-import (partial) allow_promote=false",
			"step: freeze writes for the source app",
			"step: switch config to Postgres and unfreeze writes",
			"set: PAGODA_RUNTIME_PROFILE=server-db",
			"set: PAGODA_ADAPTERS_DB=postgres",
			"set: PAGODA_ADAPTERS_CACHE=redis",
			"set: PAGODA_ADAPTERS_JOBS=asynq",
			"note: dry-run only; rerun without --dry-run to update .env",
		} {
			if !strings.Contains(out.String(), token) {
				t.Fatalf("stdout missing %q:\n%s", token, out.String())
			}
		}

		body, err := os.ReadFile(envPath)
		if err != nil {
			t.Fatalf("read env: %v", err)
		}
		if string(body) != initial {
			t.Fatalf("dry-run should not modify .env\nbefore:\n%s\nafter:\n%s", initial, string(body))
		}
	})

	t.Run("applies canonical promotion config mutations", func(t *testing.T) {
		root := t.TempDir()
		envPath := filepath.Join(root, ".env")
		initial := strings.Join([]string{
			"PAGODA_RUNTIME_PROFILE=single-node",
			"PAGODA_PROCESSES_WEB=true",
			"PAGODA_PROCESSES_WORKER=true",
			"PAGODA_PROCESSES_SCHEDULER=true",
			"PAGODA_PROCESSES_COLOCATED=true",
			"PAGODA_ADAPTERS_DB=sqlite",
			"PAGODA_DATABASE_DRIVER=sqlite",
			"PAGODA_DB_DRIVER=sqlite",
			"PAGODA_DATABASE_DBMODE=embedded",
			"PAGODA_ADAPTERS_CACHE=otter",
			"PAGODA_CACHE_DRIVER=otter",
			"PAGODA_ADAPTERS_JOBS=backlite",
			"PAGODA_JOBS_DRIVER=backlite",
			"",
		}, "\n")
		if err := os.WriteFile(envPath, []byte(initial), 0o644); err != nil {
			t.Fatalf("write env: %v", err)
		}

		prevWD, err := os.Getwd()
		if err != nil {
			t.Fatalf("getwd: %v", err)
		}
		if err := os.Chdir(root); err != nil {
			t.Fatalf("chdir %s: %v", root, err)
		}
		t.Cleanup(func() { _ = os.Chdir(prevWD) })

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

		body, err := os.ReadFile(envPath)
		if err != nil {
			t.Fatalf("read env: %v", err)
		}
		for _, want := range []string{
			"PAGODA_RUNTIME_PROFILE=server-db",
			"PAGODA_PROCESSES_WEB=true",
			"PAGODA_PROCESSES_WORKER=false",
			"PAGODA_PROCESSES_SCHEDULER=false",
			"PAGODA_PROCESSES_COLOCATED=false",
			"PAGODA_ADAPTERS_DB=postgres",
			"PAGODA_DATABASE_DRIVER=postgres",
			"PAGODA_DB_DRIVER=postgres",
			"PAGODA_DATABASE_DBMODE=standalone",
			"PAGODA_ADAPTERS_CACHE=redis",
			"PAGODA_CACHE_DRIVER=redis",
			"PAGODA_ADAPTERS_JOBS=asynq",
			"PAGODA_JOBS_DRIVER=asynq",
		} {
			if !strings.Contains(string(body), want) {
				t.Fatalf("env missing %q:\n%s", want, string(body))
			}
		}
		for _, token := range []string{
			"applied canonical promotion config in",
			"next: ship db:migrate",
			"next: ship db:export --json",
			"next: ship db:import --json",
			"next: ship db:verify-import --json",
		} {
			if !strings.Contains(out.String(), token) {
				t.Fatalf("stdout missing %q:\n%s", token, out.String())
			}
		}
	})

	t.Run("prints json plan for tooling", func(t *testing.T) {
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
			Database struct {
				Mode              string   `json:"mode"`
				Driver            string   `json:"driver"`
				CompatibleTargets []string `json:"compatible_targets"`
				PromotionPath     string   `json:"promotion_path"`
			} `json:"database"`
			StateMachine struct {
				SchemaVersion string `json:"schema_version"`
				CurrentState  string `json:"current_state"`
				NextState     string `json:"next_state"`
				Blockers      []struct {
					ID string `json:"id"`
				} `json:"blockers"`
			} `json:"state_machine"`
			Steps             []string `json:"steps"`
			SuggestedCommands []string `json:"suggested_commands"`
			MutationPlan      struct {
				DryRun bool              `json:"dry_run"`
				Values map[string]string `json:"values"`
			} `json:"mutation_plan"`
		}
		if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
			t.Fatalf("decode json: %v\n%s", err, out.String())
		}
		if payload.Database.PromotionPath != config.PromotionPathSQLiteToPostgresManualV1 {
			t.Fatalf("promotion_path = %q", payload.Database.PromotionPath)
		}
		if payload.StateMachine.SchemaVersion != "promotion-state-machine-v1" {
			t.Fatalf("state_machine.schema_version = %q", payload.StateMachine.SchemaVersion)
		}
		if payload.StateMachine.CurrentState != "sqlite-source-ready" {
			t.Fatalf("state_machine.current_state = %q", payload.StateMachine.CurrentState)
		}
		if payload.StateMachine.NextState != "config-mutated-awaiting-import" {
			t.Fatalf("state_machine.next_state = %q", payload.StateMachine.NextState)
		}
		if len(payload.StateMachine.Blockers) != 0 {
			t.Fatalf("expected no blockers for sqlite source: %+v", payload.StateMachine.Blockers)
		}
		if len(payload.Steps) == 0 {
			t.Fatalf("expected steps in %s", out.String())
		}
		if !payload.MutationPlan.DryRun {
			t.Fatalf("expected dry_run mutation plan in %s", out.String())
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
		for _, token := range []string{
			"ship db:migrate",
		} {
			if !containsString(payload.SuggestedCommands, token) {
				t.Fatalf("suggested commands missing %q:\n%s", token, out.String())
			}
		}
	})

	t.Run("blocks partial postgres promotion state", func(t *testing.T) {
		out := &bytes.Buffer{}
		errOut := &bytes.Buffer{}

		code := RunDB([]string{"promote", "--json"}, DBDeps{
			Out: out,
			Err: errOut,
			LoadConfig: func() (config.Config, error) {
				cfg := config.Config{}
				cfg.Database.DbMode = config.DBModeStandalone
				cfg.Database.Driver = config.DBDriverPostgres
				return cfg, nil
			},
		})
		if code != 1 {
			t.Fatalf("exit code = %d, want 1", code)
		}

		var payload struct {
			StateMachine struct {
				CurrentState string `json:"current_state"`
				Blockers     []struct {
					ID string `json:"id"`
				} `json:"blockers"`
			} `json:"state_machine"`
			Note string `json:"note"`
		}
		if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
			t.Fatalf("decode json: %v\n%s", err, out.String())
		}
		if payload.StateMachine.CurrentState != "config-mutated-awaiting-import" {
			t.Fatalf("state_machine.current_state = %q", payload.StateMachine.CurrentState)
		}
		if len(payload.StateMachine.Blockers) != 1 || payload.StateMachine.Blockers[0].ID != "promotion-state.partial-transition-blocked" {
			t.Fatalf("unexpected blockers: %+v", payload.StateMachine.Blockers)
		}
		if !strings.Contains(payload.Note, "promotion blocked") {
			t.Fatalf("note = %q", payload.Note)
		}
	})
}
