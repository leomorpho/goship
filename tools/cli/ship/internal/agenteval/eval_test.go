package agenteval

import "testing"

func TestLoadColdStartTaskPack(t *testing.T) {
	pack, err := LoadColdStartTaskPack("testdata/cold_start_task_pack.json")
	if err != nil {
		t.Fatalf("LoadColdStartTaskPack returned error: %v", err)
	}
	if len(pack.Tasks) < 3 {
		t.Fatalf("task count=%d want >=3", len(pack.Tasks))
	}
	if pack.Name == "" {
		t.Fatal("pack name should not be empty")
	}
}

func TestEvaluateTaskAttempt_Scoring(t *testing.T) {
	task := TaskSpec{
		ID:                "T1",
		RequiredSurfaces:  []string{"docs/00-index.md", "docs/reference/01-cli.md"},
		RequiredTools:     []string{"rg", "go test"},
		DisallowedSurfaces: []string{"docs/roadmap/03-atomic-tasks.md"},
	}

	success := EvaluateTaskAttempt(task, TaskAttempt{
		FirstSurface:  "docs/00-index.md",
		UsedSurfaces:  []string{"docs/00-index.md", "docs/reference/01-cli.md"},
		UsedTools:     []string{"rg", "go test"},
		Completed:     true,
		AvoidedDeadEnds: true,
	})
	if success.Score != 100 || !success.Passed {
		t.Fatalf("success scoring = %+v, want score=100 passed=true", success)
	}

	failure := EvaluateTaskAttempt(task, TaskAttempt{
		FirstSurface:  "README.md",
		UsedSurfaces:  []string{"README.md", "docs/roadmap/03-atomic-tasks.md"},
		UsedTools:     []string{"go test"},
		Completed:     false,
		AvoidedDeadEnds: false,
	})
	if failure.Score >= success.Score {
		t.Fatalf("failure score=%d, want less than success %d", failure.Score, success.Score)
	}
	if failure.Passed {
		t.Fatalf("failure should not pass: %+v", failure)
	}
	if len(failure.MissingRequiredTools) == 0 {
		t.Fatalf("expected missing required tool diagnostics: %+v", failure)
	}
	if len(failure.HitDisallowedSurfaces) == 0 {
		t.Fatalf("expected disallowed surface diagnostics: %+v", failure)
	}
}

func TestEvaluatePackSummary(t *testing.T) {
	pack := TaskPack{
		Name: "cold-start",
		Tasks: []TaskSpec{
			{ID: "A", RequiredSurfaces: []string{"docs/00-index.md"}},
			{ID: "B", RequiredSurfaces: []string{"docs/reference/02-mcp.md"}},
		},
	}
	attempts := map[string]TaskAttempt{
		"A": {FirstSurface: "docs/00-index.md", Completed: true, AvoidedDeadEnds: true},
		"B": {FirstSurface: "README.md", Completed: false, AvoidedDeadEnds: false},
	}

	summary := EvaluatePack(pack, attempts)
	if summary.TotalTasks != 2 {
		t.Fatalf("TotalTasks=%d want 2", summary.TotalTasks)
	}
	if summary.PassedTasks != 1 {
		t.Fatalf("PassedTasks=%d want 1", summary.PassedTasks)
	}
	if summary.SuccessRate <= 0 || summary.SuccessRate >= 1 {
		t.Fatalf("SuccessRate=%f want between 0 and 1", summary.SuccessRate)
	}
	if len(summary.Results) != 2 {
		t.Fatalf("results len=%d want 2", len(summary.Results))
	}
}
