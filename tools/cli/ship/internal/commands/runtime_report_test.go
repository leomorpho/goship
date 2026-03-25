package commands

import (
	"bytes"
	"encoding/json"
	"os"
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
		for _, key := range []string{"profile", "adapters", "processes", "process_topology", "web", "database", "managed", "module_adoption", "upgrade_readiness"} {
			if _, ok := payload[key]; !ok {
				t.Fatalf("missing runtime report key %q in %s", key, out.String())
			}
		}
	})

	t.Run("json payload includes module adoption metadata", func(t *testing.T) {
		root := t.TempDir()
		writeDescribeFixture(t, root)

		prevWD, err := os.Getwd()
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Chdir(prevWD) })
		if err := os.Chdir(root); err != nil {
			t.Fatal(err)
		}

		out := &bytes.Buffer{}
		errOut := &bytes.Buffer{}

		code := RunRuntimeReport([]string{"--json"}, RuntimeReportDeps{
			Out:          out,
			Err:          errOut,
			FindGoModule: findDescribeGoModule,
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
						},
					},
				}
				return cfg, nil
			},
		})
		if code != 0 {
			t.Fatalf("exit code = %d, stderr=%s", code, errOut.String())
		}

		var payload struct {
			ModuleAdoption []struct {
				ID         string `json:"id"`
				ModulePath string `json:"module_path"`
				Version    string `json:"version"`
				Source     string `json:"source"`
				Installed  bool   `json:"installed"`
			} `json:"module_adoption"`
		}
		if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
			t.Fatalf("decode runtime report: %v\n%s", err, out.String())
		}
		if len(payload.ModuleAdoption) != 1 {
			t.Fatalf("module adoption len = %d, want 1", len(payload.ModuleAdoption))
		}
		adoption := payload.ModuleAdoption[0]
		if adoption.ID != "notifications" {
			t.Fatalf("module adoption id = %q, want notifications", adoption.ID)
		}
		if adoption.ModulePath != "github.com/leomorpho/goship-modules/notifications" {
			t.Fatalf("module path = %q, want notifications module path", adoption.ModulePath)
		}
		if adoption.Version != "v0.0.0" {
			t.Fatalf("version = %q, want v0.0.0", adoption.Version)
		}
		if adoption.Source != "local-replace" {
			t.Fatalf("source = %q, want local-replace", adoption.Source)
		}
		if !adoption.Installed {
			t.Fatal("installed = false, want true")
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
		if !strings.Contains(errOut.String(), "Run `ship runtime:report --json`") {
			t.Fatalf("stderr = %q, want operator guidance", errOut.String())
		}
	})

	t.Run("load config failures include operator guidance", func(t *testing.T) {
		out := &bytes.Buffer{}
		errOut := &bytes.Buffer{}
		code := RunRuntimeReport([]string{"--json"}, RuntimeReportDeps{
			Out: out,
			Err: errOut,
			LoadConfig: func() (config.Config, error) {
				return config.Config{}, os.ErrNotExist
			},
		})
		if code != 1 {
			t.Fatalf("exit code = %d, want 1", code)
		}
		if !strings.Contains(errOut.String(), "runtime:report failed to load config") {
			t.Fatalf("stderr = %q, want load-config failure", errOut.String())
		}
		if !strings.Contains(errOut.String(), "Verify `.env` and PAGODA_* runtime variables") {
			t.Fatalf("stderr = %q, want operator guidance", errOut.String())
		}
	})

	t.Run("process topology reports framework defaults and realtime roles", func(t *testing.T) {
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
						RuntimeReport: runtimeconfig.BuildReport(runtimeconfig.LayerInputs{
							Defaults: map[string]string{
								"processes.web":       "true",
								"processes.worker":    "true",
								"processes.scheduler": "true",
								"processes.colocated": "true",
							},
							EffectiveValues: map[string]string{
								"processes.web":       "true",
								"processes.worker":    "true",
								"processes.scheduler": "true",
								"processes.colocated": "true",
							},
							ManagedEnabled: false,
						}),
					},
				}
				return cfg, nil
			},
		})
		if code != 0 {
			t.Fatalf("exit code = %d, stderr=%s", code, errOut.String())
		}

		var payload struct {
			ProcessTopology struct {
				Web struct {
					Enabled      bool   `json:"enabled"`
					Source       string `json:"source"`
					RealtimeRole string `json:"realtime_role"`
				} `json:"web"`
				Worker struct {
					Enabled      bool   `json:"enabled"`
					Source       string `json:"source"`
					RealtimeRole string `json:"realtime_role"`
				} `json:"worker"`
				Scheduler struct {
					Enabled bool   `json:"enabled"`
					Source  string `json:"source"`
				} `json:"scheduler"`
				CoLocated struct {
					Enabled bool   `json:"enabled"`
					Source  string `json:"source"`
				} `json:"co_located"`
			} `json:"process_topology"`
		}
		if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
			t.Fatalf("decode runtime report: %v\n%s", err, out.String())
		}

		if !payload.ProcessTopology.Web.Enabled || payload.ProcessTopology.Web.Source != "framework-default" {
			t.Fatalf("web topology = %+v, want enabled framework-default", payload.ProcessTopology.Web)
		}
		if !payload.ProcessTopology.Worker.Enabled || payload.ProcessTopology.Worker.Source != "framework-default" {
			t.Fatalf("worker topology = %+v, want enabled framework-default", payload.ProcessTopology.Worker)
		}
		if payload.ProcessTopology.Web.RealtimeRole != "realtime-edge" {
			t.Fatalf("web realtime role = %q, want realtime-edge", payload.ProcessTopology.Web.RealtimeRole)
		}
		if payload.ProcessTopology.Worker.RealtimeRole != "realtime-worker" {
			t.Fatalf("worker realtime role = %q, want realtime-worker", payload.ProcessTopology.Worker.RealtimeRole)
		}
		if !payload.ProcessTopology.Scheduler.Enabled || payload.ProcessTopology.Scheduler.Source != "framework-default" {
			t.Fatalf("scheduler topology = %+v, want enabled framework-default", payload.ProcessTopology.Scheduler)
		}
		if !payload.ProcessTopology.CoLocated.Enabled || payload.ProcessTopology.CoLocated.Source != "framework-default" {
			t.Fatalf("co-located topology = %+v, want enabled framework-default", payload.ProcessTopology.CoLocated)
		}
	})
}
