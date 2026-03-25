package agenteval

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadTaskAttempts(t *testing.T) {
	attempts, err := LoadTaskAttempts("testdata/cold_start_attempts_baseline.json")
	if err != nil {
		t.Fatalf("LoadTaskAttempts returned error: %v", err)
	}
	if len(attempts) == 0 {
		t.Fatal("expected non-empty attempts")
	}
	if attempts["discover-cli-surface"].FirstSurface == "" {
		t.Fatal("expected discover-cli-surface attempt data")
	}
}

func TestEvaluatePackWithThresholdAndWriteReport(t *testing.T) {
	pack, err := LoadColdStartTaskPack("testdata/cold_start_task_pack.json")
	if err != nil {
		t.Fatalf("LoadColdStartTaskPack returned error: %v", err)
	}
	attempts, err := LoadTaskAttempts("testdata/cold_start_attempts_baseline.json")
	if err != nil {
		t.Fatalf("LoadTaskAttempts returned error: %v", err)
	}

	report := EvaluatePackWithThreshold(pack, attempts, 0.66)
	if !report.Passed {
		t.Fatalf("report should pass at threshold 0.66: %+v", report)
	}
	if report.Summary.SuccessRate < 0.66 {
		t.Fatalf("success rate=%f want >= 0.66", report.Summary.SuccessRate)
	}

	out := filepath.Join(t.TempDir(), "agent-eval-report.json")
	if err := WriteRunReport(out, report); err != nil {
		t.Fatalf("WriteRunReport returned error: %v", err)
	}

	b, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read report file: %v", err)
	}
	var decoded RunReport
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatalf("decode report json: %v", err)
	}
	if decoded.PackName != pack.Name {
		t.Fatalf("pack name=%q want %q", decoded.PackName, pack.Name)
	}
}
