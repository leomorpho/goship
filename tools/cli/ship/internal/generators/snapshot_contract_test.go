package generators

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestGeneratorOutputSnapshotContract(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		snapshot string
		kind     string
		dryRun   bool
		created  []string
		updated  []string
		previews []generatorPreview
		next     []string
	}{
		{
			name:     "job output",
			snapshot: "generator_report_job.golden",
			kind:     "job",
			created: []string{
				"app/jobs/backfill_user_stats.go",
				"app/jobs/backfill_user_stats_test.go",
			},
			updated: []string{
				"docs/reference/01-cli.md",
			},
			previews: []generatorPreview{
				{
					Title: "Registration snippet",
					Body:  `core.Jobs.Register("jobs.backfill_user_stats", appjobs.NewBackfillUserStatsJob(nil))`,
				},
			},
			next: []string{
				"go test ./app/jobs -count=1",
			},
		},
		{
			name:     "resource dry-run output",
			snapshot: "generator_report_resource_dry_run.golden",
			kind:     "resource",
			dryRun:   true,
			created: []string{
				"app/web/controllers/inbox.go",
				"app/views/web/pages/inbox.templ",
			},
			previews: []generatorPreview{
				{
					Title: "Route name constant for app/web/routenames/routenames.go",
					Body:  `RouteNameInbox = "inbox"`,
				},
				{
					Title: "Router snippet for app/router.go",
					Body:  `g.GET("/inbox", inbox.Get).Name = routeNames.RouteNameInbox`,
				},
			},
		},
		{
			name:     "model output",
			snapshot: "generator_report_model.golden",
			kind:     "model",
			created: []string{
				"db/queries/blog_post.sql",
			},
			next: []string{
				"ship db:make create_blog_posts_table",
				"edit db/migrate/migrations/*_create_blog_posts_table.sql",
				"ship db:migrate",
				"ship db:generate",
			},
		},
		{
			name:     "command output",
			snapshot: "generator_report_command.golden",
			kind:     "command",
			created: []string{
				"app/commands/backfill_user_stats.go",
			},
			updated: []string{
				"cmd/cli/main.go",
			},
		},
		{
			name:     "controller output",
			snapshot: "generator_report_controller.golden",
			kind:     "controller",
			created: []string{
				"app/web/controllers/posts.go",
			},
			previews: []generatorPreview{
				{
					Title: "Router snippet for app/router.go",
					Body: `posts := controllers.NewPosts(container)
g.GET("/posts", posts.Index).Name = routeNames.RouteNamePosts`,
				},
			},
		},
		{
			name:     "mailer output",
			snapshot: "generator_report_mailer.golden",
			kind:     "mailer",
			created: []string{
				"app/views/emails/welcome_digest.templ",
			},
			updated: []string{
				"app/web/controllers/mail_preview.go",
				"app/router.go",
				"app/web/routenames/routenames.go",
			},
		},
		{
			name:     "island output",
			snapshot: "generator_report_island.golden",
			kind:     "island",
			created: []string{
				"frontend/islands/StatusBadge.js",
				"app/views/web/components/status_badge_island.templ",
			},
			previews: []generatorPreview{
				{
					Title: "Templ usage snippet",
					Body: `@components.StatusBadgeIsland(map[string]any{
	"label": "StatusBadge",
	"initialCount": 0,
})`,
				},
			},
			next: []string{
				"ship templ generate --file app/views/web/components/status_badge_island.templ",
				"make build-js",
				"render @components.StatusBadgeIsland(...) from the page or component that should mount this island",
			},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var out bytes.Buffer
			writeGeneratorReport(&out, tc.kind, tc.dryRun, tc.created, tc.updated, tc.previews, tc.next)
			assertGeneratorSnapshot(t, tc.snapshot, out.String())
		})
	}
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
