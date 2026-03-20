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
		Recovery struct {
			LinkageSchemaVersion string `json:"linkage_schema_version"`
		} `json:"recovery"`
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
	if handshake.Recovery.LinkageSchemaVersion != "incident-recovery-linkage-v1" {
		t.Fatalf("handshake recovery.linkage_schema_version = %q, want incident-recovery-linkage-v1", handshake.Recovery.LinkageSchemaVersion)
	}

	var recovery struct {
		LinkageSchemaVersion         string   `json:"linkage_schema_version"`
		RequiredRestoreLinkageFields []string `json:"required_restore_linkage_fields"`
		OptionalRestoreLinkageFields []string `json:"optional_restore_linkage_fields"`
	}
	if err := json.Unmarshal(payload["recovery"], &recovery); err != nil {
		t.Fatalf("recovery payload should decode: %v\n%s", err, out.String())
	}
	if recovery.LinkageSchemaVersion != "incident-recovery-linkage-v1" {
		t.Fatalf("recovery.linkage_schema_version = %q, want incident-recovery-linkage-v1", recovery.LinkageSchemaVersion)
	}
	if got, want := strings.Join(recovery.RequiredRestoreLinkageFields, ","), "incident_id,recovery_id"; got != want {
		t.Fatalf("recovery.required_restore_linkage_fields = %q, want %q", got, want)
	}
	if got, want := strings.Join(recovery.OptionalRestoreLinkageFields, ","), "deploy_id"; got != want {
		t.Fatalf("recovery.optional_restore_linkage_fields = %q, want %q", got, want)
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
