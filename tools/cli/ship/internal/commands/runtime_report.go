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
		ContractVersion: "runtime-contract-v1",
		Handshake: runtimeReportHandshake{
			SchemaVersion: "runtime-handshake-v1",
			Profile:       plan.Profile,
			Managed:       cfg.RuntimeMetadata().Managed,
			Database:      cfg.RuntimeMetadata().Database,
		},
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

func PrintRuntimeReportHelp(w io.Writer) {
	fmt.Fprintln(w, "ship runtime:report commands:")
	fmt.Fprintln(w, "  ship runtime:report --json  Print machine-readable runtime capability report")
}

func stringsTrim(v string) string {
	return strings.TrimSpace(v)
}
