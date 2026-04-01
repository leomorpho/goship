package commands

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	appconfig "github.com/leomorpho/goship/config"
)

type ConfigDeps struct {
	Out          io.Writer
	Err          io.Writer
	FindGoModule func(start string) (string, string, error)
}

type configValidateJSON struct {
	OK              bool               `json:"ok"`
	Variables       []appconfig.EnvVar `json:"variables"`
	MissingRequired []string           `json:"missing_required,omitempty"`
	SemanticIssues  []string           `json:"semantic_issues,omitempty"`
}

func RunConfig(args []string, d ConfigDeps) int {
	if len(args) == 0 {
		PrintConfigHelp(d.Out)
		return 0
	}

	switch args[0] {
	case "help", "-h", "--help":
		PrintConfigHelp(d.Out)
		return 0
	case "validate":
		return runConfigValidate(args[1:], d)
	default:
		fmt.Fprintf(d.Err, "unknown config command: %s\n\n", args[0])
		PrintConfigHelp(d.Err)
		return 1
	}
}

func PrintConfigHelp(w io.Writer) {
	fmt.Fprintln(w, "ship config commands:")
	fmt.Fprintln(w, "  ship config:validate [--json]  Validate known config variables and required env coverage")
}

func runConfigValidate(args []string, d ConfigDeps) int {
	for _, arg := range args {
		if arg == "help" || arg == "-h" || arg == "--help" {
			PrintConfigHelp(d.Out)
			return 0
		}
	}

	fs := flag.NewFlagSet("config:validate", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	jsonOutput := fs.Bool("json", false, "output config validation as JSON")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(d.Err, "invalid config validate arguments: %v\n", err)
		return 1
	}
	if fs.NArg() > 0 {
		fmt.Fprintf(d.Err, "unexpected config validate arguments: %v\n", fs.Args())
		return 1
	}

	root := "."
	if d.FindGoModule != nil {
		wd, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(d.Err, "failed to resolve working directory: %v\n", err)
			return 1
		}
		foundRoot, _, err := d.FindGoModule(wd)
		if err != nil {
			fmt.Fprintf(d.Err, "failed to resolve project root (go.mod): %v\n", err)
			return 1
		}
		root = foundRoot
	}

	vars, err := appconfig.EnvVars()
	if err != nil {
		fmt.Fprintf(d.Err, "failed to collect config variables: %v\n", err)
		return 1
	}
	missing, err := appconfig.MissingRequiredEnv(root)
	if err != nil {
		fmt.Fprintf(d.Err, "failed to validate required config variables: %v\n", err)
		return 1
	}
	semanticIssues := make([]string, 0)
	if cfg, cfgErr := appconfig.GetConfig(); cfgErr == nil {
		for _, issue := range appconfig.ValidateConfigSemantics(cfg) {
			semanticIssues = append(semanticIssues, issue.Error())
		}
	} else {
		semanticIssues = append(semanticIssues, fmt.Sprintf("failed to load config for semantic validation: %v", cfgErr))
	}

	if *jsonOutput {
		payload := configValidateJSON{
			OK:        len(missing) == 0 && len(semanticIssues) == 0,
			Variables: vars,
		}
		if len(missing) > 0 {
			payload.MissingRequired = envVarNames(missing)
		}
		if len(semanticIssues) > 0 {
			payload.SemanticIssues = semanticIssues
		}
		enc := json.NewEncoder(d.Out)
		enc.SetIndent("", "  ")
		if err := enc.Encode(payload); err != nil {
			fmt.Fprintf(d.Err, "failed to encode config validate output: %v\n", err)
			return 1
		}
		if payload.OK {
			return 0
		}
		return 1
	}

	tw := tabwriter.NewWriter(d.Out, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "NAME\tSTATUS\tTYPE\tDEFAULT\tALIASES")
	for _, item := range vars {
		status := "optional"
		if item.Required {
			status = "required"
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n",
			item.Name,
			status,
			item.Type,
			item.Default,
			strings.Join(item.Aliases, ","),
		)
	}
	if err := tw.Flush(); err != nil {
		fmt.Fprintf(d.Err, "failed to write config validate output: %v\n", err)
		return 1
	}

	if len(missing) == 0 && len(semanticIssues) == 0 {
		fmt.Fprintln(d.Out)
		fmt.Fprintln(d.Out, "config validation: OK")
		return 0
	}

	if len(missing) > 0 {
		fmt.Fprintln(d.Err, "missing required environment variables:")
		for _, name := range envVarNames(missing) {
			fmt.Fprintf(d.Err, "- %s\n", name)
		}
	}
	if len(semanticIssues) > 0 {
		fmt.Fprintln(d.Err, "config semantic issues:")
		for _, issue := range semanticIssues {
			fmt.Fprintf(d.Err, "- %s\n", issue)
		}
	}
	return 1
}

func envVarNames(vars []appconfig.EnvVar) []string {
	names := make([]string, 0, len(vars))
	for _, item := range vars {
		names = append(names, item.Name)
	}
	return names
}
