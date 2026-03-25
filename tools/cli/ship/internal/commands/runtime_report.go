package commands

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/leomorpho/goship/config"
	"github.com/leomorpho/goship/framework/runtimeconfig"
	"github.com/leomorpho/goship/framework/runtimeplan"
	frameworksecurity "github.com/leomorpho/goship/framework/security"
)

type RuntimeReportDeps struct {
	Out          io.Writer
	Err          io.Writer
	LoadConfig   func() (config.Config, error)
	FindGoModule func(start string) (string, string, error)
}

type runtimeReport struct {
	ContractVersion  string                         `json:"contract_version"`
	Handshake        runtimeReportHandshake         `json:"handshake"`
	Divergence       runtimeReportDivergence        `json:"divergence"`
	Profile          string                         `json:"profile"`
	Adapters         runtimeReportAdapters          `json:"adapters"`
	Processes        runtimeReportProcesses         `json:"processes"`
	ProcessTopology  runtimeReportProcessTopology   `json:"process_topology"`
	ManagedHooks     runtimeReportManagedHooks      `json:"managed_hooks"`
	Web              runtimeplan.WebFeatures        `json:"web"`
	Database         config.DatabaseRuntimeMetadata `json:"database"`
	Managed          config.ManagedRuntimeMetadata  `json:"managed"`
	FrameworkVersion string                         `json:"framework_version"`
	ModuleAdoption   []describeModuleAdoption       `json:"module_adoption"`
	UpgradeReadiness runtimeReportUpgradeReadiness  `json:"upgrade_readiness"`
}

type runtimeReportAdapters struct {
	DB     string `json:"db"`
	Cache  string `json:"cache"`
	Jobs   string `json:"jobs"`
	PubSub string `json:"pubsub"`
}

type runtimeReportProcesses struct {
	Web       bool `json:"web"`
	Worker    bool `json:"worker"`
	Scheduler bool `json:"scheduler"`
	CoLocated bool `json:"co_located"`
}

type runtimeReportProcessTopology struct {
	Web       runtimeReportProcessTopologyEntry `json:"web"`
	Worker    runtimeReportProcessTopologyEntry `json:"worker"`
	Scheduler runtimeReportProcessTopologyEntry `json:"scheduler"`
	CoLocated runtimeReportProcessTopologyEntry `json:"co_located"`
}

type runtimeReportProcessTopologyEntry struct {
	Enabled      bool   `json:"enabled"`
	Source       string `json:"source"`
	RealtimeRole string `json:"realtime_role,omitempty"`
}

type runtimeReportManagedHooks struct {
	SchemaVersion    string `json:"schema_version"`
	TimestampHeader  string `json:"timestamp_header"`
	NonceHeader      string `json:"nonce_header"`
	SignatureHeader  string `json:"signature_header"`
	SignaturePayload string `json:"signature_payload"`
	MaxSkewSeconds   int    `json:"max_skew_seconds"`
	NonceTTLSeconds  int    `json:"nonce_ttl_seconds"`
	RotationHeader   string `json:"rotation_header"`
}

type runtimeReportHandshake struct {
	SchemaVersion string                         `json:"schema_version"`
	Profile       string                         `json:"profile"`
	Managed       config.ManagedRuntimeMetadata  `json:"managed"`
	Database      config.DatabaseRuntimeMetadata `json:"database"`
}

type runtimeReportDivergence struct {
	SchemaVersion string                            `json:"schema_version"`
	CurrentStatus string                            `json:"current_status"`
	Classes       []runtimeReportDivergenceClass    `json:"classes"`
	Escalation    runtimeReportDivergenceEscalation `json:"escalation"`
}

type runtimeReportDivergenceClass struct {
	ID         string `json:"id"`
	Meaning    string `json:"meaning"`
	Trigger    string `json:"trigger"`
	Escalation string `json:"escalation"`
}

type runtimeReportDivergenceEscalation struct {
	SchemaVersion     string `json:"schema_version"`
	RepeatedThreshold int    `json:"repeated_threshold"`
	ObserveAction     string `json:"observe_action"`
	ReviewAction      string `json:"review_action"`
	RecoveryAction    string `json:"recovery_action"`
}

type runtimeReportUpgradeReadiness struct {
	Ready    bool                          `json:"ready"`
	Blockers []runtimeReportUpgradeBlocker `json:"blockers"`
}

type runtimeReportUpgradeBlocker struct {
	ID          string `json:"id"`
	Detail      string `json:"detail"`
	Remediation string `json:"remediation"`
}

func RunRuntimeReport(args []string, d RuntimeReportDeps) int {
	for _, arg := range args {
		if arg == "-h" || arg == "--help" || arg == "help" {
			PrintRuntimeReportHelp(d.Out)
			return 0
		}
	}

	fs := flag.NewFlagSet("runtime:report", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	jsonOutput := fs.Bool("json", false, "print JSON output")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(d.Err, "invalid runtime:report arguments: %v\n", err)
		return 1
	}
	if fs.NArg() > 0 {
		fmt.Fprintf(d.Err, "unexpected runtime:report arguments: %v\n", fs.Args())
		return 1
	}
	if !*jsonOutput {
		fmt.Fprintln(d.Err, "runtime:report currently requires --json")
		fmt.Fprintln(d.Err, "Run `ship runtime:report --json` to get the machine-readable runtime report payload.")
		return 1
	}
	if d.LoadConfig == nil {
		fmt.Fprintln(d.Err, "runtime:report requires config loader dependency")
		return 1
	}

	cfg, err := d.LoadConfig()
	if err != nil {
		fmt.Fprintf(d.Err, "runtime:report failed to load config: %v\n", err)
		fmt.Fprintln(d.Err, "Verify `.env` and PAGODA_* runtime variables, then retry `ship runtime:report --json`.")
		return 1
	}

	plan, err := runtimeplan.Resolve(&cfg)
	if err != nil {
		fmt.Fprintf(d.Err, "runtime:report failed to resolve runtime plan: %v\n", err)
		return 1
	}

	moduleAdoption := make([]describeModuleAdoption, 0)
	frameworkVersion := ""
	root := ""
	if d.FindGoModule != nil {
		wd, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(d.Err, "runtime:report failed to resolve working directory: %v\n", err)
			return 1
		}
		resolvedRoot, _, err := d.FindGoModule(wd)
		if err != nil {
			fmt.Fprintf(d.Err, "runtime:report failed to resolve project root (go.mod): %v\n", err)
			return 1
		}
		root = resolvedRoot
		modules, err := collectDescribeModules(root)
		if err != nil {
			fmt.Fprintf(d.Err, "runtime:report failed to collect module inventory: %v\n", err)
			return 1
		}
		moduleAdoption, err = collectDescribeModuleAdoption(root, modules)
		if err != nil {
			fmt.Fprintf(d.Err, "runtime:report failed to collect module adoption: %v\n", err)
			return 1
		}
		frameworkVersion, err = detectFrameworkVersionFromRoot(root)
		if err != nil {
			fmt.Fprintf(d.Err, "runtime:report failed to detect framework version: %v\n", err)
			return 1
		}
	}

	report := runtimeReport{
		ContractVersion: runtimeContractVersion,
		Handshake: runtimeReportHandshake{
			SchemaVersion: runtimeHandshakeSchemaVersion,
			Profile:       plan.Profile,
			Managed:       cfg.RuntimeMetadata().Managed,
			Database:      cfg.RuntimeMetadata().Database,
		},
		Divergence: buildRuntimeReportDivergence(),
		Profile:    plan.Profile,
		Adapters: runtimeReportAdapters{
			DB:     cfg.Adapters.DB,
			Cache:  cfg.Adapters.Cache,
			Jobs:   cfg.Adapters.Jobs,
			PubSub: cfg.Adapters.PubSub,
		},
		Processes: runtimeReportProcesses{
			Web:       plan.RunWeb,
			Worker:    plan.RunWorker,
			Scheduler: plan.RunScheduler,
			CoLocated: plan.CoLocated,
		},
		ManagedHooks: buildManagedHooksContract(cfg),
		Web: runtimeplan.ResolveWebFeatures(
			plan,
			stringsTrim(cfg.Adapters.Cache) != "",
			stringsTrim(cfg.Adapters.PubSub) != "",
		),
		Database:         cfg.RuntimeMetadata().Database,
		Managed:          cfg.RuntimeMetadata().Managed,
		FrameworkVersion: frameworkVersion,
		ModuleAdoption:   moduleAdoption,
		UpgradeReadiness: evaluateRuntimeUpgradeReadiness(root, cfg),
	}
	report.ProcessTopology = buildRuntimeReportProcessTopology(cfg, report.Web)

	enc := json.NewEncoder(d.Out)
	enc.SetIndent("", "  ")
	if err := enc.Encode(report); err != nil {
		fmt.Fprintf(d.Err, "failed to encode runtime report: %v\n", err)
		return 1
	}
	return 0
}

func buildManagedHooksContract(cfg config.Config) runtimeReportManagedHooks {
	maxSkewSeconds := cfg.Managed.HooksMaxSkewSeconds
	if maxSkewSeconds <= 0 {
		maxSkewSeconds = 300
	}
	nonceTTLSeconds := cfg.Managed.HooksNonceTTLSeconds
	if nonceTTLSeconds <= 0 {
		nonceTTLSeconds = maxSkewSeconds
	}

	return runtimeReportManagedHooks{
		SchemaVersion:    "managed-hook-contract-v1",
		TimestampHeader:  frameworksecurity.HeaderManagedTimestamp,
		NonceHeader:      frameworksecurity.HeaderManagedNonce,
		SignatureHeader:  frameworksecurity.HeaderManagedSignature,
		SignaturePayload: "METHOD\\nPATH_WITH_QUERY\\nTIMESTAMP\\nNONCE\\nRAW_BODY",
		MaxSkewSeconds:   maxSkewSeconds,
		NonceTTLSeconds:  nonceTTLSeconds,
		RotationHeader:   "PAGODA_MANAGED_HOOKS_PREVIOUS_SECRET",
	}
}

func evaluateRuntimeUpgradeReadiness(root string, cfg config.Config) runtimeReportUpgradeReadiness {
	blockers := make([]runtimeReportUpgradeBlocker, 0)

	if strings.TrimSpace(root) != "" {
		cliPath := filepath.Join(root, "tools", "cli", "ship", "internal", "cli", "cli.go")
		body, err := os.ReadFile(cliPath)
		if err != nil {
			blockers = append(blockers, runtimeReportUpgradeBlocker{
				ID:          "upgrade.convention_drift",
				Detail:      "missing tools/cli/ship/internal/cli/cli.go for upgrade rewrite contract",
				Remediation: "restore canonical CLI path and rerun ship runtime:report --json",
			})
		} else {
			text := string(body)
			if !strings.Contains(text, `gooseGoRunRef = "github.com/pressly/goose/v3/cmd/goose@`) &&
				!strings.Contains(text, `gooseGoRunRef = "github.com/pressly/goose/cmd/goose@`) {
				blockers = append(blockers, runtimeReportUpgradeBlocker{
					ID:          "upgrade.convention_drift",
					Detail:      "gooseGoRunRef constant not found in canonical CLI path",
					Remediation: "run `ship verify --profile strict` and align upgrade scaffolding markers",
				})
			}
		}
	}

	if cfg.Managed.RuntimeReport.IsManagedMode() {
		if strings.TrimSpace(cfg.Managed.RuntimeReport.Authority) == "" {
			blockers = append(blockers, runtimeReportUpgradeBlocker{
				ID:          "upgrade.managed_authority_missing",
				Detail:      "managed runtime report authority is required for managed upgrade orchestration",
				Remediation: "set PAGODA_MANAGED_AUTHORITY and rerun ship runtime:report --json",
			})
		}
		verifier := frameworksecurity.NewManagedHookVerifier(
			cfg.Managed.HooksSecret,
			time.Duration(cfg.Managed.HooksMaxSkewSeconds)*time.Second,
			time.Duration(cfg.Managed.HooksNonceTTLSeconds)*time.Second,
		).WithPreviousSecret(cfg.Managed.HooksPreviousSecret)
		if ready, reason := verifier.UpgradeReadiness(); !ready {
			blockers = append(blockers, runtimeReportUpgradeBlocker{
				ID:          "upgrade.managed_hooks_secret_missing",
				Detail:      reason,
				Remediation: "set PAGODA_MANAGED_HOOKS_SECRET and rerun ship runtime:report --json",
			})
		}
	}

	return runtimeReportUpgradeReadiness{
		Ready:    len(blockers) == 0,
		Blockers: blockers,
	}
}

func buildRuntimeReportProcessTopology(cfg config.Config, web runtimeplan.WebFeatures) runtimeReportProcessTopology {
	topology := runtimeconfig.BuildProcessTopology(cfg.Managed.RuntimeReport, runtimeconfig.ProcessDefaults{
		Web:       cfg.Processes.Web,
		Worker:    cfg.Processes.Worker,
		Scheduler: cfg.Processes.Scheduler,
		CoLocated: cfg.Processes.CoLocated,
	})
	payload := runtimeReportProcessTopology{
		Web: runtimeReportProcessTopologyEntry{
			Enabled: topology.Web.Enabled,
			Source:  string(topology.Web.Source),
		},
		Worker: runtimeReportProcessTopologyEntry{
			Enabled: topology.Worker.Enabled,
			Source:  string(topology.Worker.Source),
		},
		Scheduler: runtimeReportProcessTopologyEntry{
			Enabled: topology.Scheduler.Enabled,
			Source:  string(topology.Scheduler.Source),
		},
		CoLocated: runtimeReportProcessTopologyEntry{
			Enabled: topology.CoLocated.Enabled,
			Source:  string(topology.CoLocated.Source),
		},
	}

	if web.EnableRealtime {
		if payload.Web.Enabled {
			payload.Web.RealtimeRole = "realtime-edge"
		}
		if payload.Worker.Enabled {
			payload.Worker.RealtimeRole = "realtime-worker"
		}
	}

	return payload
}

func buildRuntimeReportDivergence() runtimeReportDivergence {
	return runtimeReportDivergence{
		SchemaVersion: "divergence-classification-v1",
		CurrentStatus: "baseline",
		Classes: []runtimeReportDivergenceClass{
			{
				ID:         "extension-zone-drift",
				Meaning:    "App-owned divergence that stays inside extension zones and preserves protected seams.",
				Trigger:    "Local changes stay within app, module, or UI extension zones.",
				Escalation: "observe",
			},
			{
				ID:         "protected-contract-drift",
				Meaning:    "A documented protected contract changed or drifted from the canonical runtime surface.",
				Trigger:    "Protected seams or operator-facing contract docs drift from runtime behavior.",
				Escalation: "recover",
			},
			{
				ID:         "repeated-local-divergence",
				Meaning:    "The same local patch or workaround keeps recurring and should be evaluated for upstreaming.",
				Trigger:    "Three or more repeated divergence events land against the same capability without upstreaming.",
				Escalation: "upstream-review",
			},
		},
		Escalation: runtimeReportDivergenceEscalation{
			SchemaVersion:     "divergence-escalation-v1",
			RepeatedThreshold: 3,
			ObserveAction:     "Track the drift and keep it inside extension zones.",
			ReviewAction:      "Open a framework/module review when the same divergence repeats.",
			RecoveryAction:    "Block or recover protected-contract drift before deploy, upgrade, or promotion proceeds.",
		},
	}
}

func PrintRuntimeReportHelp(w io.Writer) {
	fmt.Fprintln(w, "ship runtime:report commands:")
	fmt.Fprintln(w, "  ship runtime:report --json  Print machine-readable runtime capability report")
}

func stringsTrim(v string) string {
	return strings.TrimSpace(v)
}
