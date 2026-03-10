package commands

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	rt "github.com/leomorpho/goship/tools/cli/ship/internal/runtime"
)

const modulesManifestHeader = `# Workspace-level module enablement for the monolith.
# Modules are installed globally for the app workspace, not per mini-app.
modules:
`

// ModuleDeps capture the runtime needs of the module namespace.
type ModuleDeps struct {
	Out          io.Writer
	Err          io.Writer
	FindGoModule func(start string) (string, string, error)
}

// moduleInfo describes how a module should update the app markers.
type moduleInfo struct {
	ID               string
	ContainerSnippet string
	RouterSnippets   map[string]string
}

var moduleCatalog = map[string]moduleInfo{
	"notifications": {
		ID: "notifications",
		ContainerSnippet: `
	// ship:module:notifications
	// TODO: wire the notifications module (db, pubsub, push, sms) here.
`,
		RouterSnippets: map[string]string{
			"auth": `
	// ship:module:notifications
	// TODO: register notification preferences + push subscription routes via modules/notifications/routes.go.
`,
		},
	},
	"paidsubscriptions": {
		ID: "paidsubscriptions",
		ContainerSnippet: `
	// ship:module:paidsubscriptions
	// TODO: wire the paid subscriptions module (plans catalog, subscription store) here.
`,
		RouterSnippets: map[string]string{
			"auth": `
	// ship:module:paidsubscriptions
	// TODO: register pricing/session routes via modules/paidsubscriptions/routes.go.
`,
			"external": `
	// ship:module:paidsubscriptions
	// TODO: register public webhook handlers (e.g., Stripe) via modules/paidsubscriptions/routes.go.
`,
		},
	},
	"emailsubscriptions": {
		ID: "emailsubscriptions",
		ContainerSnippet: `
	// ship:module:emailsubscriptions
	// TODO: wire the email subscriptions module (store, confirmation) here.
`,
		RouterSnippets: map[string]string{
			"public": `
	// ship:module:emailsubscriptions
	// TODO: register email subscription routes via modules/emailsubscriptions/routes.go.
`,
		},
	},
	"jobs": {
		ID: "jobs",
		ContainerSnippet: `
	// ship:module:jobs
	// TODO: wire background job processors via modules/jobs.
`,
		RouterSnippets: map[string]string{},
	},
	"pwa": {
		ID: "pwa",
		ContainerSnippet: `
	// ship:module:pwa
	// TODO: wire the PWA install/push helpers.
`,
		RouterSnippets: map[string]string{
			"public": `
	// ship:module:pwa
	// TODO: register PWA install/uninstall routes via modules/pwa/routes.go.
`,
		},
	},
	"admin": {
		ID: "admin",
		ContainerSnippet: `
	// ship:module:admin
	// TODO: wire admin console services.
`,
		RouterSnippets: map[string]string{
			"auth": `
	// ship:module:admin
	// TODO: register admin routes via modules/admin/routes.go.
`,
		},
	},
}

// RunModule dispatches the module namespace commands.
func RunModule(args []string, d ModuleDeps) int {
	if len(args) == 0 {
		fmt.Fprintln(d.Err, "usage: ship module:<add> <name> [--dry-run]")
		return 1
	}

	sub := args[0]
	rest := args[1:]
	switch sub {
	case "add":
		return runModuleAdd(rest, d)
	default:
		fmt.Fprintf(d.Err, "unknown module command: %s\n", sub)
		return 1
	}
}

func runModuleAdd(args []string, d ModuleDeps) int {
	name, dryRun, err := parseAddArgs(args)
	if err != nil {
		fmt.Fprintf(d.Err, "invalid module:add arguments: %v\n", err)
		return 1
	}
	info, ok := moduleCatalog[name]
	if !ok {
		fmt.Fprintf(d.Err, "unknown module %q\n", name)
		return 1
	}
	root, _, err := d.FindGoModule(".")
	if err != nil {
		fmt.Fprintf(d.Err, "failed to locate project root: %v\n", err)
		return 1
	}

	if err := applyModuleAdd(root, info, dryRun, d.Out); err != nil {
		fmt.Fprintf(d.Err, "module:add failed: %v\n", err)
		return 1
	}
	if dryRun {
		fmt.Fprintln(d.Out, "Dry-run mode: no files were written.")
	}
	return 0
}

func parseAddArgs(args []string) (string, bool, error) {
	var name string
	var dryRun bool
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--dry-run":
			dryRun = true
		default:
			if strings.HasPrefix(args[i], "-") {
				return "", false, fmt.Errorf("unknown option: %s", args[i])
			}
			if name != "" {
				return "", false, fmt.Errorf("unexpected argument: %s", args[i])
			}
			name = strings.ToLower(strings.TrimSpace(args[i]))
		}
	}
	if name == "" {
		return "", false, errors.New("module name is required")
	}
	return name, dryRun, nil
}

func applyModuleAdd(root string, info moduleInfo, dryRun bool, out io.Writer) error {
	var changed bool

	manifestPath := filepath.Join(root, "config", "modules.yaml")
	manifestChanged, manifestContent, err := buildModulesManifest(manifestPath, info.ID)
	if err != nil {
		return err
	}
	if manifestChanged {
		if err := writeOrDiff(manifestPath, manifestContent, dryRun, out); err != nil {
			return err
		}
		changed = true
	}

	containerPath := filepath.Join(root, "app", "foundation", "container.go")
	containerContent, err := os.ReadFile(containerPath)
	if err != nil {
		return fmt.Errorf("read container: %w", err)
	}
	containerUpdated, containerChanged, err := insertBetweenMarkers(
		string(containerContent),
		"// ship:container:start",
		"// ship:container:end",
		info.ContainerSnippet,
	)
	if err != nil {
		return fmt.Errorf("update container: %w", err)
	}
	if containerChanged {
		if err := writeOrDiff(containerPath, containerUpdated, dryRun, out); err != nil {
			return err
		}
		changed = true
	}

	routerPath := filepath.Join(root, "app", "router.go")
	routerContent, err := os.ReadFile(routerPath)
	if err != nil {
		return fmt.Errorf("read router: %w", err)
	}
	currentRouter := string(routerContent)
	routerChanged := false
	for group, snippet := range info.RouterSnippets {
		start, end, err := routeMarkerPair(group)
		if err != nil {
			return fmt.Errorf("router marker: %w", err)
		}
		currentRouter, changed, err = insertBetweenMarkers(currentRouter, start, end, snippet)
		if err != nil {
			return fmt.Errorf("update router %s: %w", group, err)
		}
		if changed {
			routerChanged = true
		}
	}
	if routerChanged {
		if err := writeOrDiff(routerPath, currentRouter, dryRun, out); err != nil {
			return err
		}
		changed = true
	}

	if !changed {
		fmt.Fprintln(out, "Module already wired; no changes needed.")
	}
	return nil
}

func buildModulesManifest(path, moduleID string) (bool, string, error) {
	manifest := rt.ModulesManifest{}
	modules := []string{}
	body, err := os.ReadFile(path)
	if err == nil {
		m, err := rt.LoadModulesManifest(path)
		if err != nil {
			return false, "", fmt.Errorf("parse %s: %w", path, err)
		}
		manifest = m
		modules = manifest.Modules
	} else if !os.IsNotExist(err) {
		return false, "", fmt.Errorf("read %s: %w", path, err)
	}
	normalized := strings.TrimSpace(strings.ToLower(moduleID))
	for _, existing := range modules {
		if existing == normalized {
			return false, string(body), nil
		}
	}
	modules = append(modules, normalized)
	normalizedModules, err := rt.NormalizeModules(modules)
	if err != nil {
		return false, "", fmt.Errorf("normalize modules: %w", err)
	}
	result := modulesManifestHeader
	for _, mod := range normalizedModules {
		result += fmt.Sprintf("  - %s\n", mod)
	}
	return true, result, nil
}

func insertBetweenMarkers(src, start, end, snippet string) (string, bool, error) {
	startIdx := strings.Index(src, start)
	endIdx := strings.Index(src, end)
	if startIdx == -1 || endIdx == -1 {
		return "", false, fmt.Errorf("marker pair %q / %q not found", start, end)
	}
	if endIdx <= startIdx {
		return "", false, fmt.Errorf("marker %q appears after %q", end, start)
	}
	block := src[startIdx:endIdx]
	trimmed := strings.TrimSpace(snippet)
	if trimmed == "" {
		return src, false, nil
	}
	if strings.Contains(block, trimmed) {
		return src, false, nil
	}

	insert := snippet
	if !strings.HasSuffix(block, "\n") {
		insert = "\n" + insert
	}
	if !strings.HasSuffix(insert, "\n") {
		insert += "\n"
	}

	return src[:endIdx] + insert + src[endIdx:], true, nil
}

func writeOrDiff(path, content string, dryRun bool, out io.Writer) error {
	if dryRun {
		fmt.Fprintf(out, "Diff for %s:\n", path)
		return diffContent(path, content, out)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

func diffContent(path, newContent string, out io.Writer) error {
	oldExists := true
	oldPath := path
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			oldExists = false
		} else {
			return err
		}
	}
	if !oldExists {
		tmp, err := os.CreateTemp("", "ship-module-old-")
		if err != nil {
			return err
		}
		defer os.Remove(tmp.Name())
		if _, err := tmp.WriteString(""); err != nil {
			return err
		}
		oldPath = tmp.Name()
		tmp.Close()
	}

	newTmp, err := os.CreateTemp("", "ship-module-new-")
	if err != nil {
		return err
	}
	newTmpPath := newTmp.Name()
	defer os.Remove(newTmpPath)
	if _, err := newTmp.WriteString(newContent); err != nil {
		return err
	}
	newTmp.Close()

	cmd := exec.Command("diff", "-u", oldPath, newTmpPath)
	cmd.Stdout = out
	cmd.Stderr = out
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return nil
		}
		return err
	}
	return nil
}

func routeMarkerPair(auth string) (string, string, error) {
	switch auth {
	case "public":
		return "// ship:routes:public:start", "// ship:routes:public:end", nil
	case "auth":
		return "// ship:routes:auth:start", "// ship:routes:auth:end", nil
	case "external":
		return "// ship:routes:external:start", "// ship:routes:external:end", nil
	default:
		return "", "", fmt.Errorf("unknown router group %q", auth)
	}
}
