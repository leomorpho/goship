package commands

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"text/tabwriter"
)

type RoutesDeps struct {
	Out          io.Writer
	Err          io.Writer
	FindGoModule func(start string) (string, string, error)
}

type routeRow struct {
	Method  string `json:"method"`
	Path    string `json:"path"`
	Auth    string `json:"auth"`
	Handler string `json:"handler"`
	File    string `json:"file,omitempty"`
}

func RunRoutes(args []string, d RoutesDeps) int {
	for _, arg := range args {
		if arg == "-h" || arg == "--help" || arg == "help" {
			PrintRoutesHelp(d.Out)
			return 0
		}
	}

	fs := flag.NewFlagSet("routes", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	jsonOutput := fs.Bool("json", false, "print JSON output")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(d.Err, "invalid routes arguments: %v\n", err)
		return 1
	}
	if fs.NArg() > 0 {
		fmt.Fprintf(d.Err, "unexpected routes arguments: %v\n", fs.Args())
		return 1
	}
	if d.FindGoModule == nil {
		fmt.Fprintln(d.Err, "routes requires FindGoModule dependency")
		return 1
	}

	wd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(d.Err, "failed to resolve working directory: %v\n", err)
		return 1
	}
	root, _, err := d.FindGoModule(wd)
	if err != nil {
		fmt.Fprintf(d.Err, "failed to resolve project root (go.mod): %v\n", err)
		return 1
	}

	var routes []describeRoute
	if err := withWorkingDir(root, func() error {
		var collectErr error
		routes, collectErr = collectDescribeRoutes(root)
		return collectErr
	}); err != nil {
		fmt.Fprintf(d.Err, "routes failed: %v\n", err)
		return 1
	}

	rows := make([]routeRow, 0, len(routes))
	for _, route := range routes {
		rows = append(rows, routeRow{
			Method:  route.Method,
			Path:    route.Path,
			Auth:    routeAuthLabel(route.Auth),
			Handler: route.Handler,
			File:    route.File,
		})
	}

	if *jsonOutput {
		enc := json.NewEncoder(d.Out)
		if err := enc.Encode(rows); err != nil {
			fmt.Fprintf(d.Err, "failed to encode routes output: %v\n", err)
			return 1
		}
		return 0
	}

	printRoutesTable(d.Out, rows)
	return 0
}

func PrintRoutesHelp(w io.Writer) {
	fmt.Fprintln(w, "ship routes commands:")
	fmt.Fprintln(w, "  ship routes")
	fmt.Fprintln(w, "  ship routes --json")
}

func routeAuthLabel(auth bool) string {
	if auth {
		return "auth"
	}
	return "public"
}

func printRoutesTable(w io.Writer, rows []routeRow) {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "METHOD\tPATH\tAUTH\tHANDLER")
	for _, row := range rows {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", row.Method, row.Path, row.Auth, row.Handler)
	}
	_ = tw.Flush()
}
