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
							"adapters.cache": {
								Value:          "otter",
								Source:         runtimeconfig.SourceManagedOverride,
								RollbackTarget: runtimeconfig.SourceFrameworkDefault,
							},
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

	var managed struct {
		Mode            string `json:"mode"`
		Authority       string `json:"authority"`
		RegistryVersion string `json:"registry_version"`
		SchemaVersion   string `json:"schema_version"`
		Keys            map[string]struct {
			Value          string `json:"value"`
			Source         string `json:"source"`
			RollbackTarget string `json:"rollback_target"`
		} `json:"keys"`
	}
	if err := json.Unmarshal(payload["managed"], &managed); err != nil {
		t.Fatalf("managed contract should decode: %v\n%s", err, out.String())
	}
	if managed.Mode != "managed" {
		t.Fatalf("managed mode = %q, want managed", managed.Mode)
	}
	if managed.Authority != "control-plane" {
		t.Fatalf("managed authority = %q, want control-plane", managed.Authority)
	}
	if managed.RegistryVersion != "managed-key-registry-v1" {
		t.Fatalf("managed registry_version = %q, want managed-key-registry-v1", managed.RegistryVersion)
	}
	if managed.SchemaVersion != "managed-key-schema-v1" {
		t.Fatalf("managed schema_version = %q, want managed-key-schema-v1", managed.SchemaVersion)
	}
	cacheKey, ok := managed.Keys["adapters.cache"]
	if !ok {
		t.Fatalf("managed keys missing adapters.cache entry:\n%s", out.String())
	}
	if cacheKey.Source != "managed-override" {
		t.Fatalf("managed adapters.cache source = %q, want managed-override", cacheKey.Source)
	}
	if cacheKey.RollbackTarget != "framework-default" {
		t.Fatalf("managed adapters.cache rollback_target = %q, want framework-default", cacheKey.RollbackTarget)
	}

	var divergence struct {
		SchemaVersion string `json:"schema_version"`
		CurrentStatus string `json:"current_status"`
		Classes       []struct {
			ID         string `json:"id"`
			Escalation string `json:"escalation"`
		} `json:"classes"`
		Escalation struct {
			SchemaVersion     string `json:"schema_version"`
			RepeatedThreshold int    `json:"repeated_threshold"`
		} `json:"escalation"`
	}
	if err := json.Unmarshal(payload["divergence"], &divergence); err != nil {
		t.Fatalf("divergence contract should decode: %v\n%s", err, out.String())
	}
	if divergence.SchemaVersion != "divergence-classification-v1" {
		t.Fatalf("divergence schema_version = %q, want divergence-classification-v1", divergence.SchemaVersion)
	}
	if divergence.CurrentStatus != "baseline" {
		t.Fatalf("divergence current_status = %q, want baseline", divergence.CurrentStatus)
	}
	if divergence.Escalation.SchemaVersion != "divergence-escalation-v1" {
		t.Fatalf("divergence escalation schema_version = %q, want divergence-escalation-v1", divergence.Escalation.SchemaVersion)
	}
	if divergence.Escalation.RepeatedThreshold != 3 {
		t.Fatalf("divergence repeated_threshold = %d, want 3", divergence.Escalation.RepeatedThreshold)
	}
	wantClasses := map[string]string{
		"extension-zone-drift":    "observe",
		"protected-contract-drift": "recover",
		"repeated-local-divergence": "upstream-review",
	}
	if len(divergence.Classes) != len(wantClasses) {
		t.Fatalf("divergence classes = %d, want %d", len(divergence.Classes), len(wantClasses))
	}
	for _, class := range divergence.Classes {
		wantEscalation, ok := wantClasses[class.ID]
		if !ok {
			t.Fatalf("unexpected divergence class %q", class.ID)
		}
		if class.Escalation != wantEscalation {
			t.Fatalf("divergence class %q escalation = %q, want %q", class.ID, class.Escalation, wantEscalation)
		}
	}
}

func TestRunRuntimeReport_ExposesModuleAdoptionMetadata_RedSpec(t *testing.T) {
	root := repoRootForRuntimeReportTest(t)

	runtimeSource := mustReadRuntimeReportText(t, filepath.Join(root, "tools", "cli", "ship", "internal", "commands", "runtime_report.go"))
	runtimeTests := mustReadRuntimeReportText(t, filepath.Join(root, "tools", "cli", "ship", "internal", "commands", "runtime_report_test.go"))
	cliRef := mustReadRuntimeReportText(t, filepath.Join(root, "docs", "reference", "01-cli.md"))
	scopeDoc := mustReadRuntimeReportText(t, filepath.Join(root, "docs", "architecture", "03-project-scope-analysis.md"))
	managedDoc := mustReadRuntimeReportText(t, filepath.Join(root, "docs", "architecture", "09-standalone-and-managed-mode.md"))

	for _, token := range []string{
		`"module_adoption"`,
		"collectDescribeModuleAdoption",
	} {
		if !strings.Contains(runtimeSource, token) {
			t.Fatalf("runtime report source should expose module adoption token %q", token)
		}
	}
	if !strings.Contains(runtimeTests, "module adoption") {
		t.Fatal("runtime report tests should lock the module adoption payload")
	}
	if !strings.Contains(cliRef, "module adoption metadata") {
		t.Fatal("cli reference should document runtime report module adoption metadata")
	}
	if !strings.Contains(scopeDoc, "module adoption metadata") {
		t.Fatal("scope analysis should describe runtime report module adoption metadata")
	}
	if !strings.Contains(managedDoc, "per-module adoption metadata") {
		t.Fatal("managed-mode contract doc should describe the per-module adoption metadata surface")
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
