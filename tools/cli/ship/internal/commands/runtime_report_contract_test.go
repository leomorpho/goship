package commands

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/leomorpho/goship/config"
	"github.com/leomorpho/goship/framework/runtimeconfig"
)

func TestPrintRootHelp_ListsRuntimeReport_RedSpec(t *testing.T) {
	out := captureHelp(t, PrintRootHelp)
	line := findLineByPrefix(out, "  ship runtime:report --json")
	if line == "" {
		t.Fatalf("root help missing runtime report line\n%s", out)
	}
	if !containsRuntimeReportTokens(line, "machine-readable", "runtime", "capability") {
		t.Fatalf("runtime report help line should describe machine-readable runtime capability output: %q", line)
	}
}

func TestRunRuntimeReport_JSONContract_RedSpec(t *testing.T) {
	root := repoRootForRuntimeReportTest(t)
	cliSource := mustReadRuntimeReportText(t, filepath.Join(root, "tools", "cli", "ship", "internal", "cli", "cli.go"))
	helpSource := mustReadRuntimeReportText(t, filepath.Join(root, "tools", "cli", "ship", "internal", "commands", "help.go"))

	if !strings.Contains(cliSource, `case "runtime":`) && !strings.Contains(cliSource, `case "runtime:report":`) {
		t.Fatal("cli dispatcher does not yet expose a runtime report command path")
	}
	if !strings.Contains(helpSource, "ship runtime:report --json") {
		t.Fatal("help output does not yet advertise ship runtime:report --json")
	}
}

func TestRunRuntimeReport_EmitsVersionedHandshakeEnvelope_RedSpec(t *testing.T) {
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}

	code := RunRuntimeReport([]string{"--json"}, RuntimeReportDeps{
		Out: out,
		Err: errOut,
		LoadConfig: func() (config.Config, error) {
			return config.Config{
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
						Mode:      runtimeconfig.ModeManaged,
						Authority: "control-plane",
						Keys: map[string]runtimeconfig.KeyState{
							"adapters.cache": {Value: "otter", Source: runtimeconfig.SourceFrameworkDefault},
						},
					},
				},
			}, nil
		},
	})
	if code != 0 {
		t.Fatalf("exit code = %d, stderr=%s", code, errOut.String())
	}

	var payload map[string]json.RawMessage
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatalf("decode runtime report: %v\n%s", err, out.String())
	}

	var contractVersion string
	if err := json.Unmarshal(payload["contract_version"], &contractVersion); err != nil {
		t.Fatalf("contract_version should be present and decodable: %v\n%s", err, out.String())
	}
	if contractVersion != "runtime-contract-v1" {
		t.Fatalf("contract_version = %q, want runtime-contract-v1", contractVersion)
	}

	handshakeRaw, ok := payload["handshake"]
	if !ok {
		t.Fatalf("runtime report missing handshake envelope:\n%s", out.String())
	}

	var handshake struct {
		SchemaVersion string `json:"schema_version"`
		Profile       string `json:"profile"`
		Managed       struct {
			Mode string `json:"mode"`
		} `json:"managed"`
		Database struct {
			Driver string `json:"driver"`
		} `json:"database"`
	}
	if err := json.Unmarshal(handshakeRaw, &handshake); err != nil {
		t.Fatalf("handshake should decode: %v\n%s", err, out.String())
	}
	if handshake.SchemaVersion != "runtime-handshake-v1" {
		t.Fatalf("handshake schema_version = %q, want runtime-handshake-v1", handshake.SchemaVersion)
	}
	if handshake.Profile != "single-node" {
		t.Fatalf("handshake profile = %q, want single-node", handshake.Profile)
	}
	if handshake.Managed.Mode != "managed" {
		t.Fatalf("handshake managed.mode = %q, want managed", handshake.Managed.Mode)
	}
	if handshake.Database.Driver != "sqlite" {
		t.Fatalf("handshake database.driver = %q, want sqlite", handshake.Database.Driver)
	}
}

func TestRunRuntimeReport_EmitsManagedDivergenceContract_RedSpec(t *testing.T) {
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
					Cache:  "managed-cache",
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
					Enabled:   true,
					Authority: "control-plane",
				},
			}
			cfg.Managed.RuntimeReport = runtimeconfig.BuildReport(runtimeconfig.LayerInputs{
				Defaults: map[string]string{
					"adapters.cache":  "otter",
					"adapters.db":     "sqlite",
					"adapters.jobs":   "backlite",
					"adapters.pubsub": "inproc",
				},
				EffectiveValues: map[string]string{
					"adapters.cache":  cfg.Adapters.Cache,
					"adapters.db":     cfg.Adapters.DB,
					"adapters.jobs":   cfg.Adapters.Jobs,
					"adapters.pubsub": cfg.Adapters.PubSub,
				},
				RepoSet: map[string]bool{},
				EnvSet:  map[string]bool{},
				ManagedSet: map[string]bool{
					"adapters.cache": true,
				},
				ManagedEnabled: true,
				Authority:      cfg.Managed.Authority,
			})
			cfg.Adapters.Cache = "rolled-back-cache"
			return cfg, nil
		},
	})
	if code != 0 {
		t.Fatalf("exit code = %d, stderr=%s", code, errOut.String())
	}

	var payload struct {
		Managed struct {
			Divergence struct {
				SchemaVersion string `json:"schema_version"`
				Items         []struct {
					Key             string `json:"key"`
					Classification  string `json:"classification"`
					ImmediateAction string `json:"immediate_action"`
					RepeatedAction  string `json:"repeated_action"`
					RollbackTarget  string `json:"rollback_target"`
				} `json:"items"`
			} `json:"divergence"`
		} `json:"managed"`
	}
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatalf("decode runtime report: %v\n%s", err, out.String())
	}

	if payload.Managed.Divergence.SchemaVersion != "managed-divergence-v1" {
		t.Fatalf("managed divergence schema_version = %q, want managed-divergence-v1", payload.Managed.Divergence.SchemaVersion)
	}
	if len(payload.Managed.Divergence.Items) != 1 {
		t.Fatalf("managed divergence items len = %d, want 1\n%s", len(payload.Managed.Divergence.Items), out.String())
	}

	item := payload.Managed.Divergence.Items[0]
	if item.Key != "adapters.cache" {
		t.Fatalf("managed divergence key = %q, want adapters.cache", item.Key)
	}
	if item.Classification != "drift" {
		t.Fatalf("managed divergence classification = %q, want drift", item.Classification)
	}
	if item.ImmediateAction != "rollback" {
		t.Fatalf("managed divergence immediate_action = %q, want rollback", item.ImmediateAction)
	}
	if item.RepeatedAction != "upstream-module-candidate-review" {
		t.Fatalf("managed divergence repeated_action = %q, want upstream-module-candidate-review", item.RepeatedAction)
	}
	if item.RollbackTarget != "framework-default" {
		t.Fatalf("managed divergence rollback_target = %q, want framework-default", item.RollbackTarget)
	}
}

func containsRuntimeReportTokens(text string, want ...string) bool {
	for _, token := range want {
		if !strings.Contains(text, token) {
			return false
		}
	}
	return true
}

func repoRootForRuntimeReportTest(t *testing.T) string {
	t.Helper()
	return repoRootFromCommandsTest(t)
}

func mustReadRuntimeReportText(t *testing.T, path string) string {
	t.Helper()

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(content)
}
