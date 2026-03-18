package commands

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/leomorpho/goship/config"
	"github.com/leomorpho/goship/framework/runtimeconfig"
)

func TestRunRuntimeReport(t *testing.T) {
	t.Run("json payload includes canonical sections", func(t *testing.T) {
		out := &bytes.Buffer{}
		errOut := &bytes.Buffer{}

		code := RunRuntimeReport([]string{"--json"}, RuntimeReportDeps{
			Out: out,
			Err: errOut,
			LoadConfig: func() (config.Config, error) {
				cfg := config.Config{
					Runtime: config.RuntimeConfig{Profile: config.RuntimeProfileSingleNode},
					Adapters: config.AdaptersConfig{
						DB:     "sqlite",
						Cache:  "otter",
						Jobs:   "backlite",
						PubSub: "inproc",
					},
					Processes: config.ProcessesConfig{
						Web:       true,
						Worker:    true,
						Scheduler: true,
						CoLocated: true,
					},
					Database: config.DatabaseConfig{
						DbMode: config.DBModeEmbedded,
						Driver: config.DBDriverSQLite,
					},
					Managed: config.ManagedConfig{
						RuntimeReport: runtimeconfig.Report{
							Mode: runtimeconfig.ModeStandalone,
							Keys: map[string]runtimeconfig.KeyState{
								"adapters.cache": {Value: "otter", Source: runtimeconfig.SourceFrameworkDefault},
							},
						},
					},
				}
				return cfg, nil
			},
		})
		if code != 0 {
			t.Fatalf("exit code = %d, stderr=%s", code, errOut.String())
		}
		if errOut.Len() != 0 {
			t.Fatalf("stderr = %q, want empty", errOut.String())
		}

		var payload map[string]any
		if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
			t.Fatalf("decode json: %v\n%s", err, out.String())
		}
		for _, key := range []string{"profile", "adapters", "processes", "web", "database", "managed"} {
			if _, ok := payload[key]; !ok {
				t.Fatalf("missing runtime report key %q in %s", key, out.String())
			}
		}
	})

	t.Run("requires json flag", func(t *testing.T) {
		out := &bytes.Buffer{}
		errOut := &bytes.Buffer{}
		code := RunRuntimeReport(nil, RuntimeReportDeps{
			Out: out,
			Err: errOut,
			LoadConfig: func() (config.Config, error) {
				return config.Config{}, nil
			},
		})
		if code != 1 {
			t.Fatalf("exit code = %d, want 1", code)
		}
		if !strings.Contains(errOut.String(), "requires --json") {
			t.Fatalf("stderr = %q, want json requirement", errOut.String())
		}
	})
}
