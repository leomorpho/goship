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
	if _, ok := payload["upgrade_readiness"]; !ok {
		t.Fatalf("runtime report missing upgrade_readiness contract section:\n%s", out.String())
	}
	if _, ok := payload["managed_hooks"]; !ok {
		t.Fatalf("runtime report missing managed_hooks contract section:\n%s", out.String())
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

	var processTopology struct {
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
	}
	if err := json.Unmarshal(payload["process_topology"], &processTopology); err != nil {
		t.Fatalf("process_topology should decode: %v\n%s", err, out.String())
	}
	if !processTopology.Web.Enabled || processTopology.Web.Source != "framework-default" {
		t.Fatalf("process_topology.web = %+v, want enabled framework-default", processTopology.Web)
	}
	if !processTopology.Worker.Enabled || processTopology.Worker.Source != "framework-default" {
		t.Fatalf("process_topology.worker = %+v, want enabled framework-default", processTopology.Worker)
	}
	if processTopology.Web.RealtimeRole != "realtime-edge" {
		t.Fatalf("process_topology.web.realtime_role = %q, want realtime-edge", processTopology.Web.RealtimeRole)
	}
	if processTopology.Worker.RealtimeRole != "realtime-worker" {
		t.Fatalf("process_topology.worker.realtime_role = %q, want realtime-worker", processTopology.Worker.RealtimeRole)
	}
	if !processTopology.Scheduler.Enabled || processTopology.Scheduler.Source != "framework-default" {
		t.Fatalf("process_topology.scheduler = %+v, want enabled framework-default", processTopology.Scheduler)
	}
	if !processTopology.CoLocated.Enabled || processTopology.CoLocated.Source != "framework-default" {
		t.Fatalf("process_topology.co_located = %+v, want enabled framework-default", processTopology.CoLocated)
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
		"extension-zone-drift":      "observe",
		"protected-contract-drift":  "recover",
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

func TestRunRuntimeReport_ManagedHooksContractShape_RedSpec(t *testing.T) {
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
					HooksMaxSkewSeconds:  120,
					HooksNonceTTLSeconds: 240,
					RuntimeReport: runtimeconfig.Report{
						Mode: runtimeconfig.ModeStandalone,
						Keys: map[string]runtimeconfig.KeyState{},
					},
				},
			}, nil
		},
	})
	if code != 0 {
		t.Fatalf("exit code = %d, stderr=%s", code, errOut.String())
	}

	var payload struct {
		ManagedHooks struct {
			SchemaVersion    string `json:"schema_version"`
			TimestampHeader  string `json:"timestamp_header"`
			NonceHeader      string `json:"nonce_header"`
			SignatureHeader  string `json:"signature_header"`
			SignaturePayload string `json:"signature_payload"`
			MaxSkewSeconds   int    `json:"max_skew_seconds"`
			NonceTTLSeconds  int    `json:"nonce_ttl_seconds"`
			RotationHeader   string `json:"rotation_header"`
		} `json:"managed_hooks"`
	}
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatalf("decode runtime report: %v\n%s", err, out.String())
	}

	if payload.ManagedHooks.SchemaVersion != "managed-hook-contract-v1" {
		t.Fatalf("managed_hooks.schema_version = %q, want managed-hook-contract-v1", payload.ManagedHooks.SchemaVersion)
	}
	if payload.ManagedHooks.TimestampHeader != "X-GoShip-Timestamp" {
		t.Fatalf("managed_hooks.timestamp_header = %q, want X-GoShip-Timestamp", payload.ManagedHooks.TimestampHeader)
	}
	if payload.ManagedHooks.NonceHeader != "X-GoShip-Nonce" {
		t.Fatalf("managed_hooks.nonce_header = %q, want X-GoShip-Nonce", payload.ManagedHooks.NonceHeader)
	}
	if payload.ManagedHooks.SignatureHeader != "X-GoShip-Signature" {
		t.Fatalf("managed_hooks.signature_header = %q, want X-GoShip-Signature", payload.ManagedHooks.SignatureHeader)
	}
	if payload.ManagedHooks.SignaturePayload != "METHOD\\nPATH_WITH_QUERY\\nTIMESTAMP\\nNONCE\\nRAW_BODY" {
		t.Fatalf("managed_hooks.signature_payload = %q, want canonical payload format", payload.ManagedHooks.SignaturePayload)
	}
	if payload.ManagedHooks.MaxSkewSeconds != 120 {
		t.Fatalf("managed_hooks.max_skew_seconds = %d, want 120", payload.ManagedHooks.MaxSkewSeconds)
	}
	if payload.ManagedHooks.NonceTTLSeconds != 240 {
		t.Fatalf("managed_hooks.nonce_ttl_seconds = %d, want 240", payload.ManagedHooks.NonceTTLSeconds)
	}
	if payload.ManagedHooks.RotationHeader != "PAGODA_MANAGED_HOOKS_PREVIOUS_SECRET" {
		t.Fatalf("managed_hooks.rotation_header = %q, want PAGODA_MANAGED_HOOKS_PREVIOUS_SECRET", payload.ManagedHooks.RotationHeader)
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
		`"framework_version"`,
		`"module_adoption"`,
		`"upgrade_readiness"`,
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
	if !strings.Contains(cliRef, "current framework version") {
		t.Fatal("cli reference should document runtime report current framework version metadata")
	}
	if !strings.Contains(scopeDoc, "module adoption metadata") {
		t.Fatal("scope analysis should describe runtime report module adoption metadata")
	}
	if !strings.Contains(scopeDoc, "current framework version") {
		t.Fatal("scope analysis should describe runtime report current framework version metadata")
	}
	if !strings.Contains(managedDoc, "per-module adoption metadata") {
		t.Fatal("managed-mode contract doc should describe the per-module adoption metadata surface")
	}
	if !strings.Contains(cliRef, "upgrade readiness") {
		t.Fatal("cli reference should document runtime report upgrade readiness metadata")
	}
}

func TestRunRuntimeReport_ModuleAdoptionIncludesFirstPartyBaseline_RedSpec(t *testing.T) {
	root := t.TempDir()
	cliPath := filepath.Join(root, "tools", "cli", "ship", "internal", "cli", "cli.go")
	if err := os.MkdirAll(filepath.Dir(cliPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/runtime-report\n\ngo 1.25\n"), 0o644); err != nil {
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
						Mode: runtimeconfig.ModeStandalone,
					},
				},
			}, nil
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

	expected := map[string]string{
		"emailsubscriptions": "github.com/leomorpho/goship-modules/emailsubscriptions",
		"jobs":               "github.com/leomorpho/goship-modules/jobs",
		"notifications":      "github.com/leomorpho/goship-modules/notifications",
		"paidsubscriptions":  "github.com/leomorpho/goship-modules/paidsubscriptions",
		"storage":            "github.com/leomorpho/goship-modules/storage",
	}
	if len(payload.ModuleAdoption) != len(expected) {
		t.Fatalf("module adoption len = %d, want %d", len(payload.ModuleAdoption), len(expected))
	}
	for _, entry := range payload.ModuleAdoption {
		wantPath, ok := expected[entry.ID]
		if !ok {
			t.Fatalf("unexpected first-party module adoption entry id=%q", entry.ID)
		}
		if entry.ModulePath != wantPath {
			t.Fatalf("module %q path = %q, want %q", entry.ID, entry.ModulePath, wantPath)
		}
		if entry.Version != "v0.0.0" {
			t.Fatalf("module %q version = %q, want v0.0.0", entry.ID, entry.Version)
		}
		if entry.Source != "first-party-catalog" {
			t.Fatalf("module %q source = %q, want first-party-catalog", entry.ID, entry.Source)
		}
		if entry.Installed {
			t.Fatalf("module %q installed = true, want false", entry.ID)
		}
	}
}

func TestRunRuntimeReport_UpgradedAppsRetainManagedContractShape_RedSpec(t *testing.T) {
	cases := []struct {
		name       string
		fixtureRel string
	}{
		{
			name:       "legacy fixture upgraded to canonical pin",
			fixtureRel: "testdata/upgrade_codemods/goose_legacy_before.go",
		},
		{
			name:       "canonical fixture upgraded to target pin",
			fixtureRel: "testdata/upgrade_codemods/goose_v3_before.go",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/demo\n\ngo 1.25\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			cliPath := filepath.Join(root, "tools", "cli", "ship", "internal", "cli", "cli.go")
			if err := os.MkdirAll(filepath.Dir(cliPath), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(cliPath, fixtureText(t, tc.fixtureRel), 0o644); err != nil {
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

			upgradeOut := &bytes.Buffer{}
			upgradeErr := &bytes.Buffer{}
			upgradeCode := RunUpgrade([]string{"apply", "--to", "v3.27.0"}, UpgradeDeps{
				Out:          upgradeOut,
				Err:          upgradeErr,
				FindGoModule: findGoModuleTest,
			})
			if upgradeCode != 0 {
				t.Fatalf("upgrade apply exit code=%d stderr=%s", upgradeCode, upgradeErr.String())
			}

			out := &bytes.Buffer{}
			errOut := &bytes.Buffer{}
			code := RunRuntimeReport([]string{"--json"}, RuntimeReportDeps{
				Out:          out,
				Err:          errOut,
				FindGoModule: findGoModuleTest,
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
				t.Fatalf("runtime report exit code=%d stderr=%s", code, errOut.String())
			}

			var payload struct {
				Managed struct {
					Mode            string `json:"mode"`
					Authority       string `json:"authority"`
					RegistryVersion string `json:"registry_version"`
					SchemaVersion   string `json:"schema_version"`
					Keys            map[string]struct {
						Source         string `json:"source"`
						RollbackTarget string `json:"rollback_target"`
					} `json:"keys"`
				} `json:"managed"`
				Handshake struct {
					SchemaVersion string `json:"schema_version"`
				} `json:"handshake"`
			}
			if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
				t.Fatalf("decode runtime report payload: %v\n%s", err, out.String())
			}
			if payload.Handshake.SchemaVersion != "runtime-handshake-v1" {
				t.Fatalf("handshake.schema_version=%q want runtime-handshake-v1", payload.Handshake.SchemaVersion)
			}
			if payload.Managed.Mode != "managed" {
				t.Fatalf("managed.mode=%q want managed", payload.Managed.Mode)
			}
			if payload.Managed.Authority != "control-plane" {
				t.Fatalf("managed.authority=%q want control-plane", payload.Managed.Authority)
			}
			if payload.Managed.RegistryVersion != "managed-key-registry-v1" {
				t.Fatalf("managed.registry_version=%q want managed-key-registry-v1", payload.Managed.RegistryVersion)
			}
			if payload.Managed.SchemaVersion != "managed-key-schema-v1" {
				t.Fatalf("managed.schema_version=%q want managed-key-schema-v1", payload.Managed.SchemaVersion)
			}
			cache, ok := payload.Managed.Keys["adapters.cache"]
			if !ok {
				t.Fatalf("managed.keys missing adapters.cache entry: %+v", payload.Managed.Keys)
			}
			if cache.Source != "managed-override" {
				t.Fatalf("managed.keys[adapters.cache].source=%q want managed-override", cache.Source)
			}
			if cache.RollbackTarget != "framework-default" {
				t.Fatalf("managed.keys[adapters.cache].rollback_target=%q want framework-default", cache.RollbackTarget)
			}
		})
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

func TestRunRuntimeReport_ManagedUpgradeReadinessBlocksWithoutHookSecret(t *testing.T) {
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
					Enabled:   true,
					Authority: "control-plane",
					RuntimeReport: runtimeconfig.Report{
						Mode:      runtimeconfig.ModeManaged,
						Authority: "control-plane",
					},
				},
			}, nil
		},
	})
	if code != 0 {
		t.Fatalf("exit code = %d, stderr=%s", code, errOut.String())
	}

	var payload struct {
		UpgradeReadiness struct {
			Ready    bool `json:"ready"`
			Blockers []struct {
				ID string `json:"id"`
			} `json:"blockers"`
		} `json:"upgrade_readiness"`
	}
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatalf("decode runtime report: %v\n%s", err, out.String())
	}
	if payload.UpgradeReadiness.Ready {
		t.Fatalf("upgrade_readiness.ready=true, want false\n%s", out.String())
	}
	found := false
	for _, blocker := range payload.UpgradeReadiness.Blockers {
		if blocker.ID == "upgrade.managed_hooks_secret_missing" {
			found = true
		}
	}
	if !found {
		t.Fatalf("upgrade_readiness.blockers missing managed hooks blocker:\n%s", out.String())
	}
}

func TestRunRuntimeReport_BackupMetadataAndRestoreEvidenceContracts_RedSpec(t *testing.T) {
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
			}, nil
		},
	})
	if code != 0 {
		t.Fatalf("exit code = %d, stderr=%s", code, errOut.String())
	}

	var payload struct {
		Backup struct {
			ManifestVersion string `json:"manifest_version"`
			RestoreEvidence struct {
				Status                  string   `json:"status"`
				AcceptedManifestVersion string   `json:"accepted_manifest_version"`
				PostRestoreChecks       []string `json:"post_restore_checks"`
				RecordLinks             []string `json:"record_links"`
			} `json:"restore_evidence"`
		} `json:"backup"`
	}
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatalf("decode runtime report: %v\n%s", err, out.String())
	}
	if payload.Backup.ManifestVersion != "backup-manifest-v1" {
		t.Fatalf("backup.manifest_version=%q want backup-manifest-v1", payload.Backup.ManifestVersion)
	}
	if payload.Backup.RestoreEvidence.Status != "accepted" {
		t.Fatalf("backup.restore_evidence.status=%q want accepted", payload.Backup.RestoreEvidence.Status)
	}
	if payload.Backup.RestoreEvidence.AcceptedManifestVersion != "backup-manifest-v1" {
		t.Fatalf(
			"backup.restore_evidence.accepted_manifest_version=%q want backup-manifest-v1",
			payload.Backup.RestoreEvidence.AcceptedManifestVersion,
		)
	}
	if len(payload.Backup.RestoreEvidence.PostRestoreChecks) == 0 {
		t.Fatalf("backup.restore_evidence.post_restore_checks should be non-empty\n%s", out.String())
	}
	if len(payload.Backup.RestoreEvidence.RecordLinks) != 3 {
		t.Fatalf("backup.restore_evidence.record_links len=%d want 3", len(payload.Backup.RestoreEvidence.RecordLinks))
	}
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
