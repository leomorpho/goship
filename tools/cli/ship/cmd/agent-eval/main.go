package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/leomorpho/goship/tools/cli/ship/v2/internal/agenteval"
)

func main() {
	packPath := flag.String("pack", "tools/cli/ship/internal/agenteval/testdata/cold_start_task_pack.json", "path to task pack JSON")
	attemptsPath := flag.String("attempts", "tools/cli/ship/internal/agenteval/testdata/cold_start_attempts_baseline.json", "path to attempts JSON")
	outPath := flag.String("out", "artifacts/agent-eval-report.json", "path to write JSON report")
	threshold := flag.Float64("threshold", 0.66, "minimum success-rate threshold (0-1)")
	flag.Parse()

	pack, err := agenteval.LoadColdStartTaskPack(*packPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load task pack: %v\n", err)
		os.Exit(1)
	}
	attempts, err := agenteval.LoadTaskAttempts(*attemptsPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load attempts: %v\n", err)
		os.Exit(1)
	}

	report := agenteval.EvaluatePackWithThreshold(pack, attempts, *threshold)
	if err := agenteval.WriteRunReport(*outPath, report); err != nil {
		fmt.Fprintf(os.Stderr, "write report: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("agent eval pack=%s passed=%t success_rate=%.2f threshold=%.2f out=%s\n", report.PackName, report.Passed, report.Summary.SuccessRate, report.Threshold, *outPath)
	if !report.Passed {
		os.Exit(1)
	}
}
