package commands

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	gen "github.com/leomorpho/goship/tools/cli/ship/internal/generators"
)

func TestMakeIslandContract_RedSpec(t *testing.T) {
	root := repoRootFromCommandsTest(t)

	makeHelp := mustReadText(t, filepath.Join(root, "tools", "cli", "ship", "internal", "commands", "help.go"))
	cliDispatch := mustReadText(t, filepath.Join(root, "tools", "cli", "ship", "internal", "cli", "cli.go"))
	cliRef := mustReadText(t, filepath.Join(root, "docs", "reference", "01-cli.md"))
	gaps := mustReadText(t, filepath.Join(root, "docs", "architecture", "06-known-gaps-and-risks.md"))

	for _, required := range []string{
		"ship make:island <Name>",
		"Generate a frontend island scaffold",
	} {
		if !strings.Contains(makeHelp, required) {
			t.Fatalf("make help should advertise %q for the make:island generator", required)
		}
		if !strings.Contains(cliRef, required) {
			t.Fatalf("cli reference should advertise %q for the make:island generator", required)
		}
	}

	for _, required := range []string{
		`case "island":`,
		"return c.runMakeIsland(args[1:])",
		"func (c CLI) runMakeIsland(args []string) int {",
	} {
		if !strings.Contains(cliDispatch, required) {
			t.Fatalf("cli dispatch should include %q for ship make:island", required)
		}
	}

	for _, required := range []string{
		"ship make:island <Name>",
		"frontend/islands/<Name>.js",
		"app/views/web/components/<name>_island.templ",
		"make build-js",
		"ship templ generate --file",
	} {
		if !strings.Contains(cliRef, required) {
			t.Fatalf("cli reference should describe %q for make:island", required)
		}
		if !strings.Contains(gaps, required) {
			t.Fatalf("known gaps doc should describe %q for make:island", required)
		}
	}
}

func TestRunMakeIsland_GeneratesCanonicalScaffold_RedSpec(t *testing.T) {
	root := t.TempDir()
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}

	code := gen.RunMakeIsland([]string{"StatusBadge"}, gen.MakeIslandDeps{
		Out: out,
		Err: errOut,
		Cwd: root,
	})
	if code != 0 {
		t.Fatalf("exit code = %d, stderr=%s", code, errOut.String())
	}

	islandPath := filepath.Join(root, "frontend", "islands", "StatusBadge.js")
	templPath := filepath.Join(root, "app", "views", "web", "components", "status_badge_island.templ")

	islandText := mustReadText(t, islandPath)
	templText := mustReadText(t, templPath)

	for _, token := range []string{
		"export function mount(el, props = {})",
		`container.dataset.component = "status-badge-island"`,
		`return typeof props.label === "string" && props.label.length > 0 ? props.label : fallback;`,
	} {
		if !strings.Contains(islandText, token) {
			t.Fatalf("generated island file missing %q:\n%s", token, islandText)
		}
	}

	for _, token := range []string{
		"templ StatusBadgeIsland(props map[string]any)",
		`data-component="status-badge-island"`,
		`data-island="StatusBadge"`,
		`data-props={ templ.JSONString(props) }`,
	} {
		if !strings.Contains(templText, token) {
			t.Fatalf("generated templ seam missing %q:\n%s", token, templText)
		}
	}

	for _, token := range []string{
		"make:island result",
		"frontend/islands/StatusBadge.js",
		"app/views/web/components/status_badge_island.templ",
		"make build-js",
		"ship templ generate --file app/views/web/components/status_badge_island.templ",
		"@components.StatusBadgeIsland(",
	} {
		if !strings.Contains(out.String(), token) {
			t.Fatalf("generator output missing %q:\n%s", token, out.String())
		}
	}
}

func TestRunMakeIsland_RefusesOverwrite_RedSpec(t *testing.T) {
	root := t.TempDir()
	islandPath := filepath.Join(root, "frontend", "islands", "StatusBadge.js")
	if err := os.MkdirAll(filepath.Dir(islandPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(islandPath, []byte("existing"), 0o644); err != nil {
		t.Fatal(err)
	}

	errOut := &bytes.Buffer{}
	code := gen.RunMakeIsland([]string{"StatusBadge"}, gen.MakeIslandDeps{
		Out: &bytes.Buffer{},
		Err: errOut,
		Cwd: root,
	})
	if code == 0 {
		t.Fatal("expected non-zero exit code")
	}
	if !strings.Contains(errOut.String(), "refusing to overwrite existing island file") {
		t.Fatalf("stderr missing overwrite protection:\n%s", errOut.String())
	}
}
