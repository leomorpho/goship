package commands

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/leomorpho/goship/config"
	"github.com/leomorpho/goship/framework/runtimeplan"
)

type RuntimeReportDeps struct {
	Out          io.Writer
	Err          io.Writer
	LoadConfig   func() (config.Config, error)
	FindGoModule func(start string) (string, string, error)
}

type runtimeReport struct {
	ContractVersion string                         `json:"contract_version"`
	Handshake       runtimeReportHandshake         `json:"handshake"`
	Divergence      runtimeReportDivergence        `json:"divergence"`
	Profile         string                         `json:"profile"`
	Adapters        runtimeReportAdapters          `json:"adapters"`
	Processes       runtimeReportProcesses         `json:"processes"`
	Web             runtimeplan.WebFeatures        `json:"web"`
	Database        config.DatabaseRuntimeMetadata `json:"database"`
	Managed         config.ManagedRuntimeMetadata  `json:"managed"`
	ModuleAdoption  []describeModuleAdoption       `json:"module_adoption"`
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
		return 1
	}
	if d.LoadConfig == nil {
		fmt.Fprintln(d.Err, "runtime:report requires config loader dependency")
		return 1
	}

	cfg, err := d.LoadConfig()
	if err != nil {
		fmt.Fprintf(d.Err, "runtime:report failed to load config: %v\n", err)
		return 1
	}

	plan, err := runtimeplan.Resolve(&cfg)
	if err != nil {
		fmt.Fprintf(d.Err, "runtime:report failed to resolve runtime plan: %v\n", err)
		return 1
	}

	moduleAdoption := make([]describeModuleAdoption, 0)
	if d.FindGoModule != nil {
		wd, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(d.Err, "runtime:report failed to resolve working directory: %v\n", err)
			return 1
		}
		root, _, err := d.FindGoModule(wd)
		if err != nil {
			fmt.Fprintf(d.Err, "runtime:report failed to resolve project root (go.mod): %v\n", err)
			return 1
		}
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
		Profile: plan.Profile,
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
		Web: runtimeplan.ResolveWebFeatures(
			plan,
			stringsTrim(cfg.Adapters.Cache) != "",
			stringsTrim(cfg.Adapters.PubSub) != "",
		),
		Database:       cfg.RuntimeMetadata().Database,
		Managed:        cfg.RuntimeMetadata().Managed,
		ModuleAdoption: moduleAdoption,
	}

	enc := json.NewEncoder(d.Out)
	enc.SetIndent("", "  ")
	if err := enc.Encode(report); err != nil {
		fmt.Fprintf(d.Err, "failed to encode runtime report: %v\n", err)
		return 1
	}
	return 0
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
