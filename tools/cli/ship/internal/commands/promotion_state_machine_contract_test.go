package commands

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/leomorpho/goship/config"
)

func TestPromotionStateMachineContract_RedSpec(t *testing.T) {
	t.Run("promote json report includes canonical state machine", func(t *testing.T) {
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
			StateMachine struct {
				SchemaVersion string   `json:"schema_version"`
				InitialState  string   `json:"initial_state"`
				TerminalState string   `json:"terminal_state"`
				UnsafeStates  []string `json:"unsafe_states"`
				States        []struct {
					Name           string   `json:"name"`
					Kind           string   `json:"kind"`
					AllowsCommands []string `json:"allows_commands"`
					BlocksCommands []string `json:"blocks_commands"`
				} `json:"states"`
			} `json:"state_machine"`
		}
		if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
			t.Fatalf("decode json: %v\n%s", err, out.String())
		}

		if payload.StateMachine.SchemaVersion != "promotion-state-machine-v1" {
			t.Fatalf("schema_version = %q, want promotion-state-machine-v1", payload.StateMachine.SchemaVersion)
		}
		if payload.StateMachine.InitialState != "source-ready" {
			t.Fatalf("initial_state = %q, want source-ready", payload.StateMachine.InitialState)
		}
		if payload.StateMachine.TerminalState != "completed" {
			t.Fatalf("terminal_state = %q, want completed", payload.StateMachine.TerminalState)
		}

		for _, want := range []string{
			"config-applied",
			"source-exported",
			"target-migrated",
			"target-imported",
			"verification-failed",
		} {
			if !containsString(payload.StateMachine.UnsafeStates, want) {
				t.Fatalf("unsafe_states missing %q:\n%s", want, out.String())
			}
		}

		requiredStates := map[string]struct {
			kind   string
			blocks []string
		}{
			"source-ready":        {kind: "safe"},
			"config-applied":      {kind: "unsafe-partial", blocks: []string{"ship db:promote"}},
			"source-exported":     {kind: "unsafe-partial", blocks: []string{"ship db:promote", "ship db:export --json"}},
			"target-migrated":     {kind: "unsafe-partial", blocks: []string{"ship db:promote", "ship db:export --json"}},
			"target-imported":     {kind: "unsafe-partial", blocks: []string{"ship db:promote", "ship db:export --json", "ship db:import --json"}},
			"verification-failed": {kind: "unsafe-partial", blocks: []string{"ship db:promote", "ship db:export --json", "ship db:import --json"}},
			"completed":           {kind: "terminal", blocks: []string{"ship db:promote", "ship db:export --json", "ship db:import --json", "ship db:verify-import --json"}},
		}

		for _, state := range payload.StateMachine.States {
			want, ok := requiredStates[state.Name]
			if !ok {
				continue
			}
			if state.Kind != want.kind {
				t.Fatalf("state %q kind = %q, want %q", state.Name, state.Kind, want.kind)
			}
			for _, cmd := range want.blocks {
				if !containsString(state.BlocksCommands, cmd) {
					t.Fatalf("state %q should block %q:\n%s", state.Name, cmd, out.String())
				}
			}
			delete(requiredStates, state.Name)
		}
		if len(requiredStates) != 0 {
			t.Fatalf("missing expected states: %#v\n%s", requiredStates, out.String())
		}
	})

	t.Run("promote text report exposes partial-transition blocking", func(t *testing.T) {
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
			"state_machine: promotion-state-machine-v1",
			"unsafe_states: config-applied, source-exported, target-migrated, target-imported, verification-failed",
			"blocked: config-applied -> ship db:promote",
			"blocked: target-imported -> ship db:import --json",
		} {
			if !strings.Contains(out.String(), token) {
				t.Fatalf("stdout missing %q:\n%s", token, out.String())
			}
		}
	})
}
