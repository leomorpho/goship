package commands

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/leomorpho/goship/config"
	"github.com/leomorpho/goship/config/runtimeconfig"
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
		for _, key := range []string{"profile", "adapters", "processes", "process_topology", "metrics", "web", "database", "managed", "framework_version", "module_adoption", "upgrade_readiness"} {
			if _, ok := payload[key]; !ok {
				t.Fatalf("missing runtime report key %q in %s", key, out.String())
			}
		}
	})

	t.Run("json payload includes module adoption metadata", func(t *testing.T) {
		root := t.TempDir()
		writeDescribeFixture(t, root)
		cliPath := filepath.Join(root, "tools", "cli", "ship", "internal", "cli", "cli.go")
		if err := os.MkdirAll(filepath.Dir(cliPath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(cliPath, []byte("package cli\nconst gooseGoRunRef = \"github.com/pressly/goose/v3/cmd/goose@v3.26.0\"\n"), 0o644); err != nil {
			t.Fatal(err)
		}

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
			FrameworkVersion string `json:"framework_version"`
			ModuleAdoption   []struct {
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
		if len(payload.ModuleAdoption) != 5 {
			t.Fatalf("module adoption len = %d, want 5", len(payload.ModuleAdoption))
		}
		if payload.FrameworkVersion != "v3.26.0" {
			t.Fatalf("framework_version = %q, want v3.26.0", payload.FrameworkVersion)
		}
		entries := map[string]struct {
			ModulePath string
			Source     string
			Installed  bool
		}{}
		for _, entry := range payload.ModuleAdoption {
			entries[entry.ID] = struct {
				ModulePath string
				Source     string
				Installed  bool
			}{
				ModulePath: entry.ModulePath,
				Source:     entry.Source,
				Installed:  entry.Installed,
			}
		}
		notifications, ok := entries["notifications"]
		if !ok {
			t.Fatal("module adoption missing notifications")
		}
		if notifications.ModulePath != "github.com/leomorpho/goship-modules/notifications" {
			t.Fatalf("module path = %q, want notifications module path", notifications.ModulePath)
		}
		if notifications.Source != "local-replace" {
			t.Fatalf("source = %q, want local-replace", notifications.Source)
		}
		if !notifications.Installed {
			t.Fatal("installed = false, want true")
		}
		for _, id := range []string{"emailsubscriptions", "jobs", "paidsubscriptions", "storage"} {
			entry, ok := entries[id]
			if !ok {
				t.Fatalf("module adoption missing %s", id)
			}
			if entry.Source != "first-party-catalog" {
				t.Fatalf("module %s source = %q, want first-party-catalog", id, entry.Source)
			}
			if entry.Installed {
				t.Fatalf("module %s installed=true, want false", id)
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

	t.Run("metrics contract reports framework-default export availability", func(t *testing.T) {
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
								"metrics.enabled": {Value: "true", Source: runtimeconfig.SourceFrameworkDefault},
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

		var payload struct {
			Metrics struct {
				SchemaVersion string `json:"schema_version"`
				Enabled       bool   `json:"enabled"`
				Exporter      string `json:"exporter"`
				Format        string `json:"format"`
				Path          string `json:"path"`
				Source        string `json:"source"`
			} `json:"metrics"`
		}
		if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
			t.Fatalf("decode runtime report: %v\n%s", err, out.String())
		}

		if payload.Metrics.SchemaVersion != "metrics-export-contract-v1" {
			t.Fatalf("metrics.schema_version = %q, want metrics-export-contract-v1", payload.Metrics.SchemaVersion)
		}
		if !payload.Metrics.Enabled {
			t.Fatalf("metrics.enabled = false, want true")
		}
		if payload.Metrics.Exporter != "prometheus" {
			t.Fatalf("metrics.exporter = %q, want prometheus", payload.Metrics.Exporter)
		}
		if payload.Metrics.Format != "prometheus-text" {
			t.Fatalf("metrics.format = %q, want prometheus-text", payload.Metrics.Format)
		}
		if payload.Metrics.Path != "/metrics" {
			t.Fatalf("metrics.path = %q, want /metrics", payload.Metrics.Path)
		}
		if payload.Metrics.Source != "framework-default" {
			t.Fatalf("metrics.source = %q, want framework-default", payload.Metrics.Source)
		}
	})
}
