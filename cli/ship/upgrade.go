package ship

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	atlasRefPattern = regexp.MustCompile(`(?m)^(\s*(?:const\s+)?atlasGoRunRef\s*=\s*"ariga\.io/atlas/cmd/atlas@)v[^"]+("\s*)$`)
)

func (c CLI) runUpgrade(args []string) int {
	for _, arg := range args {
		if arg == "-h" || arg == "--help" || arg == "help" {
			printUpgradeHelp(c.Out)
			return 0
		}
	}
	fs := flag.NewFlagSet("upgrade", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	to := fs.String("to", "", "target pinned version, e.g. v0.3.1001")
	dryRun := fs.Bool("dry-run", false, "print planned file changes without writing")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(c.Err, "invalid upgrade arguments: %v\n", err)
		return 1
	}
	if fs.NArg() > 0 {
		fmt.Fprintf(c.Err, "unexpected upgrade arguments: %v\n", fs.Args())
		printUpgradeHelp(c.Err)
		return 1
	}
	if strings.TrimSpace(*to) == "" {
		fmt.Fprintln(c.Err, "missing required --to version")
		return 1
	}
	if !strings.HasPrefix(*to, "v") {
		fmt.Fprintln(c.Err, "version must start with 'v' (example: v0.3.1001)")
		return 1
	}

	wd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(c.Err, "failed to resolve working directory: %v\n", err)
		return 1
	}
	root, _, err := findGoModule(wd)
	if err != nil {
		fmt.Fprintf(c.Err, "failed to resolve project root (go.mod): %v\n", err)
		return 1
	}

	return c.upgradeAtlas(root, *to, *dryRun)
}

func (c CLI) upgradeAtlas(root, version string, dryRun bool) int {
	path := filepath.Join(root, "cli", "ship", "cli.go")
	old, newText, changed, err := rewriteAtlasVersion(path, version)
	if err != nil {
		fmt.Fprintf(c.Err, "failed to update atlas version: %v\n", err)
		return 1
	}
	if !changed {
		fmt.Fprintf(c.Out, "atlas already pinned to %s in %s\n", version, path)
		return 0
	}
	if dryRun {
		fmt.Fprintf(c.Out, "dry-run: would update atlas in %s: %s -> %s\n", path, old, version)
		return 0
	}
	if err := os.WriteFile(path, []byte(newText), 0o644); err != nil {
		fmt.Fprintf(c.Err, "failed to write %s: %v\n", path, err)
		return 1
	}
	fmt.Fprintf(c.Out, "updated atlas pin in %s: %s -> %s\n", path, old, version)
	return 0
}

func rewriteAtlasVersion(path, target string) (oldVersion string, rewritten string, changed bool, err error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", "", false, err
	}
	text := string(b)
	match := atlasRefPattern.FindStringSubmatch(text)
	if len(match) == 0 {
		return "", "", false, fmt.Errorf("atlasGoRunRef constant not found in %s", path)
	}
	full := match[0]
	prefix := match[1]
	suffix := match[2]
	old := strings.TrimSuffix(strings.TrimPrefix(full, prefix), suffix)
	if old == target {
		return old, text, false, nil
	}
	replacement := prefix + target + suffix
	updated := atlasRefPattern.ReplaceAllString(text, replacement)
	return old, updated, true, nil
}
