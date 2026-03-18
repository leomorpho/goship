package generators

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestGeneratorOutputSnapshotContract(t *testing.T) {
	var out bytes.Buffer
	writeGeneratorReport(
		&out,
		"job",
		false,
		[]string{
			"app/jobs/backfill_user_stats.go",
			"app/jobs/backfill_user_stats_test.go",
		},
		[]string{
			"docs/reference/01-cli.md",
		},
		[]generatorPreview{
			{
				Title: "Registration snippet",
				Body:  `core.Jobs.Register("jobs.backfill_user_stats", appjobs.NewBackfillUserStatsJob(nil))`,
			},
		},
		[]string{
			"go test ./app/jobs -count=1",
		},
	)

	assertGeneratorSnapshot(t, "generator_report.golden", out.String())
}

func assertGeneratorSnapshot(t *testing.T, name, got string) {
	t.Helper()

	path := filepath.Join("testdata", name)
	if os.Getenv("UPDATE_GENERATOR_SNAPSHOTS") == "1" {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
		}
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatalf("write snapshot %s: %v", path, err)
		}
	}

	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read snapshot %s: %v", path, err)
	}
	if string(want) != got {
		t.Fatalf("snapshot drift for %s\n\nwant:\n%s\n\ngot:\n%s", path, string(want), got)
	}
}
