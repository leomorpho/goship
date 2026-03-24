package commands

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
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
	ID                string
	ModulePath        string
	LocalPath         string
	ContainerSnippet  string
	RouterSnippets    map[string]string
	EnvExampleSnippet string
	RequiredEnv       []requiredEnvVar
	InstallContract   moduleInstallContract
}

type requiredEnvVar struct {
	Name        string
	Description string
}

type moduleInstallContract struct {
	Routes     []string
	Config     []string
	Assets     []string
	Jobs       []string
	Templates  []string
	Migrations []string
	Tests      []string
}

func (c moduleInstallContract) IsEmpty() bool {
	return len(c.Routes) == 0 &&
		len(c.Config) == 0 &&
		len(c.Assets) == 0 &&
		len(c.Jobs) == 0 &&
		len(c.Templates) == 0 &&
		len(c.Migrations) == 0 &&
		len(c.Tests) == 0
}

func (m moduleInfo) installContract() moduleInstallContract {
	contract := m.InstallContract
	if len(contract.Config) == 0 {
		contract.Config = []string{"config/modules.yaml"}
	}
	if strings.TrimSpace(m.ContainerSnippet) != "" {
		contract.Config = appendUniqueStrings(contract.Config, "app/foundation/container.go")
	}
	for group, snippet := range m.RouterSnippets {
		if strings.TrimSpace(snippet) == "" {
			continue
		}
		contract.Routes = appendUniqueStrings(contract.Routes, "app/router.go ("+group+")")
	}
	if strings.TrimSpace(m.ModulePath) != "" {
		contract.Config = appendUniqueStrings(contract.Config, "go.mod", "go.work")
	}
	if strings.TrimSpace(m.LocalPath) != "" {
		contract.Migrations = appendUniqueStrings(contract.Migrations, filepath.ToSlash(filepath.Join(m.LocalPath, "db", "migrate", "migrations")))
	}
	if strings.TrimSpace(m.EnvExampleSnippet) != "" {
		contract.Config = appendUniqueStrings(contract.Config, ".env.example")
	}
	return contract
}

func appendUniqueStrings(dst []string, values ...string) []string {
	for _, value := range values {
		v := strings.TrimSpace(value)
		if v == "" {
			continue
		}
		exists := false
		for _, existing := range dst {
			if existing == v {
				exists = true
				break
			}
		}
		if !exists {
			dst = append(dst, v)
		}
	}
	return dst
}

var (
	paidSubscriptionsContainerSnippet = `
	// ship:module:paidsubscriptions
	// TODO: wire the paid subscriptions module (plans catalog, subscription store) here.
`
	paidSubscriptionsRouterSnippets = map[string]string{
		"auth": `
	// ship:module:paidsubscriptions
	// TODO: register pricing/session routes via modules/paidsubscriptions/routes.go.
`,
		"external": `
	// ship:module:paidsubscriptions
	// TODO: register public webhook handlers (e.g., Stripe) via modules/paidsubscriptions/routes.go.
`,
	}
	paidSubscriptionsEnvExampleSnippet = `# ship:module:paidsubscriptions
# Stripe settings for paid subscriptions.
STRIPE_KEY=
STRIPE_WEBHOOK_SECRET=
`
	paidSubscriptionsRequiredEnv = []requiredEnvVar{
		{
			Name:        "STRIPE_KEY",
			Description: "Stripe API key from your Stripe dashboard.",
		},
		{
			Name:        "STRIPE_WEBHOOK_SECRET",
			Description: "Stripe webhook signing secret for subscription events.",
		},
	}
)

var moduleCatalog = map[string]moduleInfo{
	"notifications": {
		ID:         "notifications",
		ModulePath: "github.com/leomorpho/goship-modules/notifications",
		LocalPath:  filepath.Join("modules", "notifications"),
		InstallContract: moduleInstallContract{
			Routes: []string{
				"modules/notifications/routes/routes.go",
			},
			Jobs: []string{
				"modules/notifications/planned_notifications.go",
				"modules/notifications/planned_notifications_store_sql.go",
			},
			Templates: []string{
				"framework/web/pages/gen/notifications_templ.go",
			},
			Tests: []string{
				"modules/notifications/routes/routes_contract_test.go",
				"modules/notifications/module_sql_mode_test.go",
				"modules/notifications/planned_notifications_store_sql_test.go",
			},
		},
		ContainerSnippet: `	// ship:module:notifications
	c.initNotifier()
`,
		RouterSnippets: map[string]string{
			"auth": `	// ship:module:notifications
	notificationsModule := notificationroutes.NewRouteModule(notificationroutes.RouteModuleDeps{
		Controller:                    ctr,
		ProfileService:                deps.ProfileService,
		NotifierService:               deps.NotifierService,
		PwaPushService:                deps.PwaPushService,
		FcmPushService:                deps.FcmPushService,
		NotificationPermissionService: deps.NotificationPermissionService,
	})
	if err := notificationsModule.RegisterOnboardingRoutes(onboardingGroup); err != nil {
		return err
	}
	if err := notificationsModule.RegisterRoutes(onboardedGroup); err != nil {
		return err
	}
`,
		},
	},
	"paidsubscriptions": {
		ID:                "paidsubscriptions",
		ModulePath:        "github.com/leomorpho/goship-modules/paidsubscriptions",
		LocalPath:         filepath.Join("modules", "paidsubscriptions"),
		ContainerSnippet:  paidSubscriptionsContainerSnippet,
		RouterSnippets:    paidSubscriptionsRouterSnippets,
		EnvExampleSnippet: paidSubscriptionsEnvExampleSnippet,
		RequiredEnv:       paidSubscriptionsRequiredEnv,
	},
	"billing": {
		ID:                "paidsubscriptions",
		ModulePath:        "github.com/leomorpho/goship-modules/paidsubscriptions",
		LocalPath:         filepath.Join("modules", "paidsubscriptions"),
		ContainerSnippet:  paidSubscriptionsContainerSnippet,
		RouterSnippets:    paidSubscriptionsRouterSnippets,
		EnvExampleSnippet: paidSubscriptionsEnvExampleSnippet,
		RequiredEnv:       paidSubscriptionsRequiredEnv,
	},
	"emailsubscriptions": {
		ID:         "emailsubscriptions",
		ModulePath: "github.com/leomorpho/goship-modules/emailsubscriptions",
		LocalPath:  filepath.Join("modules", "emailsubscriptions"),
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
		ID:         "jobs",
		ModulePath: "github.com/leomorpho/goship-modules/jobs",
		LocalPath:  filepath.Join("modules", "jobs"),
		ContainerSnippet: `
	// ship:module:jobs
	// Wire framework/bootstrap.WireJobsRuntime into c.CoreJobs and c.CoreJobsInspector.
`,
		RouterSnippets: map[string]string{},
	},
	"2fa": {
		ID: "2fa",
		ContainerSnippet: `
	// ship:module:2fa
	// TODO: wire the two-factor authentication module services.
`,
		RouterSnippets: map[string]string{
			"auth": `
	// ship:module:2fa
	// TODO: register 2FA setup/verify routes via modules/2fa/routes.go.
`,
		},
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
		InstallContract: moduleInstallContract{
			Routes: []string{
				"modules/admin/routes.go",
			},
			Templates: []string{
				"modules/admin/views/web/components/gen/admin_layout_templ.go",
				"modules/admin/views/web/components/gen/admin_form_templ.go",
				"modules/admin/views/web/components/gen/admin_list_templ.go",
			},
		},
		RouterSnippets: map[string]string{
			"auth": `
	adminPanelModule := adminmodule.New(adminmodule.ModuleDeps{
		Controller: ctr,
		DB:         c.Database,
		AuditLogs:  c.AuditLogs,
		Flags:      c.Flags,
	})
	if err := adminPanelModule.RegisterRoutes(onboardedGroup); err != nil {
		return err
	}
`,
		},
	},
	"storage": {
		ID:         "storage",
		ModulePath: "github.com/leomorpho/goship-modules/storage",
		LocalPath:  filepath.Join("modules", "storage"),
		ContainerSnippet: `
	// ship:module:storage
	// Wire modules/storage.New around the app-facing core.BlobStorage seam.
`,
		RouterSnippets: map[string]string{},
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
	case "remove":
		return runModuleRemove(rest, d)
	default:
		fmt.Fprintf(d.Err, "unknown module command: %s\n", sub)
		return 1
	}
}

func runModuleAdd(args []string, d ModuleDeps) int {
	name, dryRun, err := parseModuleArgs(args)
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
	printModuleInstallContract(d.Out, info)
	if !dryRun {
		if err := warnMissingModuleEnv(root, info, d.Out); err != nil {
			fmt.Fprintf(d.Err, "module:add env checks failed: %v\n", err)
			return 1
		}
	}
	if dryRun {
		fmt.Fprintln(d.Out, "Dry-run mode: no files were written.")
	}
	return 0
}

func printModuleInstallContract(out io.Writer, info moduleInfo) {
	contract := info.installContract()
	fmt.Fprintf(out, "Install contract for module %q:\n", info.ID)
	printInstallContractSection(out, "routes", contract.Routes)
	printInstallContractSection(out, "config", contract.Config)
	printInstallContractSection(out, "assets", contract.Assets)
	printInstallContractSection(out, "jobs", contract.Jobs)
	printInstallContractSection(out, "templates", contract.Templates)
	printInstallContractSection(out, "migrations", contract.Migrations)
	printInstallContractSection(out, "tests", contract.Tests)
	printInstallContractOwnership(out, contract)
}

func printInstallContractSection(out io.Writer, label string, values []string) {
	fmt.Fprintf(out, "  %s:\n", label)
	if len(values) == 0 {
		fmt.Fprintln(out, "    - (none)")
		return
	}
	for _, value := range values {
		fmt.Fprintf(out, "    - %s\n", value)
	}
}

func printInstallContractOwnership(out io.Writer, contract moduleInstallContract) {
	ownership := map[string][]string{}
	appendOwnership := func(owner string, values []string) {
		for _, value := range values {
			v := strings.TrimSpace(value)
			if v == "" {
				continue
			}
			ownership[v] = appendUniqueStrings(ownership[v], owner)
		}
	}

	appendOwnership("routes", contract.Routes)
	appendOwnership("config", contract.Config)
	appendOwnership("assets", contract.Assets)
	appendOwnership("jobs", contract.Jobs)
	appendOwnership("templates", contract.Templates)
	appendOwnership("migrations", contract.Migrations)
	appendOwnership("tests", contract.Tests)

	fmt.Fprintln(out, "  ownership:")
	if len(ownership) == 0 {
		fmt.Fprintln(out, "    - (none)")
		return
	}

	files := make([]string, 0, len(ownership))
	for file := range ownership {
		files = append(files, file)
	}
	sort.Strings(files)
	for _, file := range files {
		owners := ownership[file]
		sort.Strings(owners)
		fmt.Fprintf(out, "    - %s -> %s\n", file, strings.Join(owners, ", "))
	}
}

func runModuleRemove(args []string, d ModuleDeps) int {
	name, dryRun, err := parseModuleArgs(args)
	if err != nil {
		fmt.Fprintf(d.Err, "invalid module:remove arguments: %v\n", err)
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

	if err := applyModuleRemove(root, info, dryRun, d.Out); err != nil {
		fmt.Fprintf(d.Err, "module:remove failed: %v\n", err)
		return 1
	}
	fmt.Fprintln(d.Out, "Reminder: module:remove does not roll back related DB migrations.")
	if dryRun {
		fmt.Fprintln(d.Out, "Dry-run mode: no files were written.")
	}
	return 0
}

func parseModuleArgs(args []string) (string, bool, error) {
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

	if dependencyChanged, err := syncLocalModuleDependency(root, info, dryRun, out); err != nil {
		return err
	} else if dependencyChanged {
		changed = true
	}

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

	if envChanged, err := appendModuleEnvExample(root, info, dryRun, out); err != nil {
		return err
	} else if envChanged {
		changed = true
	}

	if !changed {
		fmt.Fprintln(out, "Module already wired; no changes needed.")
	}
	return nil
}

func appendModuleEnvExample(root string, info moduleInfo, dryRun bool, out io.Writer) (bool, error) {
	trimmed := strings.TrimSpace(info.EnvExampleSnippet)
	if trimmed == "" {
		return false, nil
	}

	envExamplePath := filepath.Join(root, ".env.example")
	body, err := os.ReadFile(envExamplePath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("read %s: %w", envExamplePath, err)
	}
	current := string(body)
	if strings.Contains(current, trimmed) {
		return false, nil
	}

	next := current
	if strings.TrimSpace(next) != "" && !strings.HasSuffix(next, "\n") {
		next += "\n"
	}
	if strings.TrimSpace(next) != "" {
		next += "\n"
	}
	next += trimmed + "\n"
	if err := writeOrDiff(envExamplePath, next, dryRun, out); err != nil {
		return false, err
	}
	return true, nil
}

func warnMissingModuleEnv(root string, info moduleInfo, out io.Writer) error {
	if len(info.RequiredEnv) == 0 {
		return nil
	}

	dotEnvKeys, err := readDotEnvKeys(filepath.Join(root, ".env"))
	if err != nil {
		return err
	}
	missing := make([]requiredEnvVar, 0)
	for _, requirement := range info.RequiredEnv {
		if _, ok := dotEnvKeys[requirement.Name]; ok {
			continue
		}
		if _, ok := os.LookupEnv(requirement.Name); ok {
			continue
		}
		missing = append(missing, requirement)
	}
	if len(missing) == 0 {
		return nil
	}

	fmt.Fprintf(out, "Warning: missing environment variables for module %q (installation continues):\n", info.ID)
	for _, requirement := range missing {
		fmt.Fprintf(out, "- %s: %s\n", requirement.Name, requirement.Description)
	}
	fmt.Fprintln(out, "Set these in .env or your shell before exercising module runtime paths.")
	return nil
}

func readDotEnvKeys(path string) (map[string]struct{}, error) {
	keys := map[string]struct{}{}
	body, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return keys, nil
		}
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	for _, raw := range strings.Split(string(body), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		eq := strings.Index(line, "=")
		if eq <= 0 {
			continue
		}
		key := strings.TrimSpace(line[:eq])
		if key == "" {
			continue
		}
		keys[key] = struct{}{}
	}
	return keys, nil
}

func applyModuleRemove(root string, info moduleInfo, dryRun bool, out io.Writer) error {
	var changed bool

	blockers, err := findModuleRemovalBlockers(root, info)
	if err != nil {
		return err
	}
	if len(blockers) > 0 {
		return fmt.Errorf("module remove blocked: %s", strings.Join(blockers, ", "))
	}

	if dependencyChanged, err := removeLocalModuleDependency(root, info, dryRun, out); err != nil {
		return err
	} else if dependencyChanged {
		changed = true
	}

	manifestPath := filepath.Join(root, "config", "modules.yaml")
	removed, manifestContent, err := removeModuleFromManifest(manifestPath, info.ID)
	if err != nil {
		return err
	}
	if removed {
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
	containerUpdated, containerChanged := removeSnippetFromContent(string(containerContent), info.ContainerSnippet)
	containerUpdated = normalizeEmptyMarkerGap(containerUpdated, "// ship:container:start", "// ship:container:end")
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
	var changedSnippet bool
	for group, snippet := range info.RouterSnippets {
		currentRouter, changedSnippet = removeSnippetFromContent(currentRouter, snippet)
		if changedSnippet {
			routerChanged = true
		}
		start, end, markerErr := routeMarkerPair(group)
		if markerErr != nil {
			return fmt.Errorf("router marker: %w", markerErr)
		}
		currentRouter = normalizeEmptyMarkerGap(currentRouter, start, end)
	}
	if routerChanged {
		if err := writeOrDiff(routerPath, currentRouter, dryRun, out); err != nil {
			return err
		}
		changed = true
	}

	if envChanged, err := removeModuleEnvExample(root, info, dryRun, out); err != nil {
		return err
	} else if envChanged {
		changed = true
	}

	if !changed {
		fmt.Fprintln(out, "Module was not wired; no changes needed.")
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
	return true, renderModulesManifest(normalizedModules), nil
}

func renderModulesManifest(modules []string) string {
	var b strings.Builder
	b.WriteString(modulesManifestHeader)
	for _, mod := range modules {
		b.WriteString("  - ")
		b.WriteString(mod)
		b.WriteByte('\n')
	}
	return b.String()
}

func removeModuleFromManifest(path, moduleID string) (bool, string, error) {
	modules := []string{}
	if _, err := os.Stat(path); err == nil {
		m, err := rt.LoadModulesManifest(path)
		if err != nil {
			return false, "", fmt.Errorf("parse %s: %w", path, err)
		}
		modules = m.Modules
	} else if !os.IsNotExist(err) {
		return false, "", fmt.Errorf("read %s: %w", path, err)
	}

	normalized := strings.TrimSpace(strings.ToLower(moduleID))
	var found bool
	var filtered []string
	for _, module := range modules {
		if module == normalized {
			found = true
			continue
		}
		filtered = append(filtered, module)
	}
	if !found {
		return false, "", nil
	}

	normalizedModules, err := rt.NormalizeModules(filtered)
	if err != nil {
		return false, "", fmt.Errorf("normalize modules: %w", err)
	}
	return true, renderModulesManifest(normalizedModules), nil
}

func removeSnippetFromContent(src, snippet string) (string, bool) {
	trimmed := strings.TrimSpace(snippet)
	if trimmed == "" {
		return src, false
	}
	idx := strings.Index(src, trimmed)
	if idx == -1 {
		return src, false
	}
	start := idx
	for start > 0 && (src[start-1] == '\n' || src[start-1] == '\r' || src[start-1] == ' ' || src[start-1] == '\t') {
		start--
		if src[start] == '\n' {
			break
		}
	}
	end := idx + len(trimmed)
	for end < len(src) && (src[end] == '\n' || src[end] == '\r' || src[end] == ' ' || src[end] == '\t') {
		end++
		if end > 0 && src[end-1] == '\n' {
			break
		}
	}
	return src[:start] + src[end:], true
}

func normalizeEmptyMarkerGap(src, startMarker, endMarker string) string {
	start := strings.Index(src, startMarker)
	end := strings.Index(src, endMarker)
	if start == -1 || end == -1 || end <= start {
		return src
	}
	gapStart := start + len(startMarker)
	gap := src[gapStart:end]
	if strings.TrimSpace(gap) != "" {
		return src
	}
	lineStart := strings.LastIndex(src[:start], "\n")
	if lineStart == -1 {
		lineStart = 0
	} else {
		lineStart++
	}
	indent := src[lineStart:start]
	return src[:gapStart] + "\n" + indent + endMarker + src[end+len(endMarker):]
}

func removeModuleEnvExample(root string, info moduleInfo, dryRun bool, out io.Writer) (bool, error) {
	trimmed := strings.TrimSpace(info.EnvExampleSnippet)
	if trimmed == "" {
		return false, nil
	}

	envExamplePath := filepath.Join(root, ".env.example")
	body, err := os.ReadFile(envExamplePath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("read %s: %w", envExamplePath, err)
	}

	updated, removed := removeSnippetFromContent(string(body), trimmed)
	if !removed {
		return false, nil
	}
	if err := writeOrDiff(envExamplePath, updated, dryRun, out); err != nil {
		return false, err
	}
	return true, nil
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
