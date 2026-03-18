package commands

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

func captureHelp(t *testing.T, fn func(io.Writer)) string {
	t.Helper()
	var b bytes.Buffer
	fn(&b)
	return b.String()
}

func findLineByPrefix(out, prefix string) string {
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(line, prefix) {
			return line
		}
	}
	return ""
}

func TestPrintRootHelp_DirectCommandsDiscoverableNoAliases(t *testing.T) {
	out := captureHelp(t, PrintRootHelp)

	want := map[string]string{
		"  ship new <app> [flags]":                  "Create a new app scaffold",
		"  ship dev [web|worker|all] [flags]":       "Run local runtime processes",
		"  ship check":                              "Run fast project checks",
		"  ship test [--integration]":               "Run tests (unit by default)",
		"  ship verify [--skip-tests] [--json]":     "Run full verification workflow",
		"  ship doctor [--json]":                    "Run repository policy checks",
		"  ship config:validate [--json]":           "Validate config contract",
		"  ship routes [--json]":                    "Show route inventory",
		"  ship describe [--pretty]":                "Show runtime/module inventory",
		"  ship run:command <name> [-- <args...>]":  "Run app-defined CLI command",
		"  ship module:add <name> [--dry-run]":      "Enable a module",
		"  ship module:remove <name> [--dry-run]":   "Disable a module",
		"  ship upgrade --to <version> [--dry-run]": "Upgrade pinned CLI tooling",
	}

	for prefix, desc := range want {
		line := findLineByPrefix(out, prefix)
		if line == "" {
			t.Fatalf("root help missing command line prefix: %q\n%s", prefix, out)
		}
		if !strings.Contains(line, desc) {
			t.Fatalf("root help line %q missing description %q", line, desc)
		}
	}

	if strings.Contains(out, "shipdev") {
		t.Fatalf("root help should not include alias shipdev: %q", out)
	}
}

func TestPrintRootHelp_CommandGroupsDiscoverable(t *testing.T) {
	out := captureHelp(t, PrintRootHelp)

	want := map[string]string{
		"  ship config --help": "Config command help",
		"  ship i18n --help":   "i18n command help",
		"  ship agent --help":  "Agent workflow command help",
		"  ship db --help":     "Database command help",
		"  ship make --help":   "Generator command help",
		"  ship infra --help":  "Local infrastructure command help",
		"  ship templ --help":  "Templ command help",
	}

	for prefix, desc := range want {
		line := findLineByPrefix(out, prefix)
		if line == "" {
			t.Fatalf("root help missing command-group line prefix: %q\n%s", prefix, out)
		}
		if !strings.Contains(line, desc) {
			t.Fatalf("root help line %q missing description %q", line, desc)
		}
	}
}

func TestPrintDBHelp_SubcommandsIncludeDescriptions(t *testing.T) {
	out := captureHelp(t, PrintDBHelp)

	want := map[string]string{
		"  ship db:create [--dry-run]":                           "Validate DB connectivity",
		"  ship db:generate [--config <path>] [--dry-run]":       "Generate DB access code",
		"  ship db:make <migration_name>":                        "Create a new SQL migration file",
		"  ship db:migrate":                                      "Apply pending migrations",
		"  ship db:status":                                       "Show migration status",
		"  ship db:console":                                      "Open database shell client",
		"  ship db:reset [--seed] [--force] [--yes] [--dry-run]": "Reset and re-apply migrations",
		"  ship db:drop [--force] [--yes] [--dry-run]":           "Revert all migrations",
		"  ship db:rollback [amount]":                            "Roll back one or more migration steps",
		"  ship db:seed":                                         "Run database seed command",
	}

	for prefix, desc := range want {
		line := findLineByPrefix(out, prefix)
		if line == "" {
			t.Fatalf("db help missing line prefix: %q\n%s", prefix, out)
		}
		if !strings.Contains(line, desc) {
			t.Fatalf("db help line %q missing description %q", line, desc)
		}
	}
}

func TestPrintMakeHelp_SubcommandsIncludeDescriptions(t *testing.T) {
	out := captureHelp(t, PrintMakeHelp)

	want := map[string]string{
		"  ship make:scaffold <Name>":                   "Generate model + migration + controller/resource wiring",
		"  ship make:controller <Name|NameController>":  "Generate a controller with optional route wiring",
		"  ship make:resource <name>":                   "Generate a route handler and optional page template",
		"  ship make:model <Name> [fields...]":          "Generate a DB query/model scaffold",
		"  ship make:factory <Name>":                    "Generate a test data factory",
		"  ship make:locale <code>":                     "Generate locale file from baseline keys",
		"  ship make:event <TypeName> [--force]":        "Generate a domain event type",
		"  ship make:schedule <Name> --cron \"<expr>\"": "Insert a scheduled job entry",
		"  ship make:command <Name>":                    "Generate an app CLI command",
		"  ship make:module <Name>":                     "Generate a standalone module scaffold",
	}

	for prefix, desc := range want {
		line := findLineByPrefix(out, prefix)
		if line == "" {
			t.Fatalf("make help missing line prefix: %q\n%s", prefix, out)
		}
		if !strings.Contains(line, desc) {
			t.Fatalf("make help line %q missing description %q", line, desc)
		}
	}
}

func TestPrintI18nHelp_SubcommandsIncludeDescriptions(t *testing.T) {
	out := captureHelp(t, PrintI18nHelp)

	want := map[string]string{
		"  ship i18n:init [--force]": "Scaffold baseline locale files",
		"  ship i18n:scan [--format json] [--paths <path1,path2,...>] [--limit <n>]": "Scan code for hardcoded user-facing strings",
		"  ship i18n:instrument [--apply] [--paths <path1,path2,...>] [--limit <n>]": "Build/apply safe rewrites for high-confidence findings",
		"  ship i18n:migrate [--force]":                                              "Migrate legacy locale formats to canonical TOML",
		"  ship i18n:normalize":                                                      "Canonicalize locale file ordering",
		"  ship i18n:compile":                                                        "Generate typed i18n key artifacts",
		"  ship i18n:ci":                                                             "Run strict i18n CI profile checks",
		"  ship i18n:missing":                                                        "Report missing/empty translations",
		"  ship i18n:unused":                                                         "Report unused locale keys",
	}

	for prefix, desc := range want {
		line := findLineByPrefix(out, prefix)
		if line == "" {
			t.Fatalf("i18n help missing line prefix: %q\n%s", prefix, out)
		}
		if !strings.Contains(line, desc) {
			t.Fatalf("i18n help line %q missing description %q", line, desc)
		}
	}
}

func TestPrintAgentHelp_SubcommandsIncludeDescriptions(t *testing.T) {
	out := captureHelp(t, printAgentHelp)

	want := map[string]string{
		"  ship agent:setup":                                           "Generate local agent allowlist artifacts from policy",
		"  ship agent:setup --check":                                   "Validate generated allowlist artifacts are in sync",
		"  ship agent:start --task \"Add feature\" [--id ID]":          "Create a scoped git worktree for an agent task",
		"  ship agent:finish --id TASK --message \"feat(...)\" [--pr]": "Verify, commit, optionally open PR, and clean up worktree",
		"  ship agent:check":                                           "Run policy artifact drift checks",
		"  ship agent:status [--codex-file <path>] [--claude-file <path>] [--gemini-file <path>]": "Inspect local Codex/Claude/Gemini policy sync state",
	}

	for prefix, desc := range want {
		line := findLineByPrefix(out, prefix)
		if line == "" {
			t.Fatalf("agent help missing line prefix: %q\n%s", prefix, out)
		}
		if !strings.Contains(line, desc) {
			t.Fatalf("agent help line %q missing description %q", line, desc)
		}
	}
}

func TestPrintAdditionalScopedHelp_IncludeDescriptions(t *testing.T) {
	cases := []struct {
		name string
		out  string
		want map[string]string
	}{
		{
			name: "config",
			out:  captureHelp(t, PrintConfigHelp),
			want: map[string]string{
				"  ship config:validate [--json]": "Validate known config variables",
			},
		},
		{
			name: "dev",
			out:  captureHelp(t, PrintDevHelp),
			want: map[string]string{
				"  ship dev":          "Run auto dev mode",
				"  ship dev web":      "Run web-only dev mode",
				"  ship dev worker":   "Run worker-only dev mode",
				"  ship dev all":      "Run full dev mode",
				"  ship dev --web":    "Flag form of web-only mode",
				"  ship dev --worker": "Flag form of worker-only mode",
				"  ship dev --all":    "Flag form of full mode",
			},
		},
		{
			name: "infra",
			out:  captureHelp(t, PrintInfraHelp),
			want: map[string]string{
				"  ship infra:up":   "Start local infrastructure dependencies",
				"  ship infra:down": "Stop local infrastructure dependencies",
			},
		},
		{
			name: "routes",
			out:  captureHelp(t, PrintRoutesHelp),
			want: map[string]string{
				"  ship routes":        "Print route inventory table",
				"  ship routes --json": "Print route inventory as JSON",
			},
		},
		{
			name: "describe",
			out:  captureHelp(t, PrintDescribeHelp),
			want: map[string]string{
				"  ship describe":          "Print project inventory as JSON",
				"  ship describe --pretty": "Print project inventory as pretty JSON",
			},
		},
		{
			name: "verify",
			out:  captureHelp(t, PrintVerifyHelp),
			want: map[string]string{
				"  ship verify":              "Run full verification workflow",
				"  ship verify --skip-tests": "Skip final test step",
				"  ship verify --json":       "Output verification result as JSON",
			},
		},
		{
			name: "test",
			out:  captureHelp(t, PrintTestHelp),
			want: map[string]string{
				"  ship test":               "Run default unit/stateless test suite",
				"  ship test --integration": "Include integration-tagged tests",
			},
		},
		{
			name: "check",
			out:  captureHelp(t, PrintCheckHelp),
			want: map[string]string{
				"  ship check": "Run fast compile/unit checks",
			},
		},
		{
			name: "templ",
			out:  captureHelp(t, PrintTemplHelp),
			want: map[string]string{
				"  ship templ generate [--path <dir>] [--file <file.templ>]": "Generate templ code",
			},
		},
		{
			name: "upgrade",
			out:  captureHelp(t, PrintUpgradeHelp),
			want: map[string]string{
				"  ship upgrade --to <version> [--dry-run]": "Update pinned CLI tooling references",
			},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			for prefix, desc := range tc.want {
				line := findLineByPrefix(tc.out, prefix)
				if line == "" {
					t.Fatalf("%s help missing line prefix: %q\n%s", tc.name, prefix, tc.out)
				}
				if !strings.Contains(line, desc) {
					t.Fatalf("%s help line %q missing description %q", tc.name, line, desc)
				}
			}
		})
	}
}

func TestPrintDevHelp_CanonicalFlagsOnly_RedSpec(t *testing.T) {
	t.Skip("red-spec only (TKT-255): enable in TKT-256 when positional dev help entries are removed")
}
