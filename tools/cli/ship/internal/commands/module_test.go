package commands

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestParseAddArgs(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		want    string
		wantDry bool
		wantErr bool
	}{
		{name: "simple", args: []string{"Notifications"}, want: "notifications"},
		{name: "dry run", args: []string{"notifications", "--dry-run"}, want: "notifications", wantDry: true},
		{name: "unknown option", args: []string{"notifications", "--unknown"}, wantErr: true},
		{name: "missing name", args: []string{"--dry-run"}, wantErr: true},
		{name: "extra positional", args: []string{"notifications", "extra"}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, dry, err := parseModuleArgs(tt.args)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseAddArgs error = %v, wantErr=%v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if got != tt.want {
				t.Fatalf("module = %q, want %q", got, tt.want)
			}
			if dry != tt.wantDry {
				t.Fatalf("dry run = %v, want %v", dry, tt.wantDry)
			}
		})
	}
}

func TestInsertBetweenMarkers(t *testing.T) {
	content := "start\n// ship:marker:start\nexisting\n// ship:marker:end\nend\n"
	snippet := "\t// ship:module:test\n"
	updated, changed, err := insertBetweenMarkers(content, "// ship:marker:start", "// ship:marker:end", snippet)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !changed {
		t.Fatalf("expected change")
	}
	if !strings.Contains(updated, snippet) {
		t.Fatalf("snippet missing")
	}

	// second insertion should be no-op.
	updated2, changed2, err := insertBetweenMarkers(updated, "// ship:marker:start", "// ship:marker:end", snippet)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if changed2 {
		t.Fatalf("expected no change on reinsert")
	}
	if updated2 != updated {
		t.Fatalf("content mutated unexpectedly")
	}
}

func TestBuildModulesManifest(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "modules.yaml")
	changed, content, err := buildModulesManifest(path, "notifications")
	if err != nil {
		t.Fatalf("build manifest: %v", err)
	}
	if !changed {
		t.Fatalf("expected manifest changed")
	}
	if !strings.Contains(content, "- notifications") {
		t.Fatalf("module entry missing: %s", content)
	}
}

func TestRemoveSnippetFromContent(t *testing.T) {
	content := "start\n\t// ship:module:test\n\t// TODO: remove me\nend\n"
	updated, removed := removeSnippetFromContent(content, `
	// ship:module:test
	// TODO: remove me
`)
	if !removed {
		t.Fatal("expected snippet removal")
	}
	if strings.Contains(updated, "remove me") {
		t.Fatalf("snippet not removed: %s", updated)
	}
}

func TestRemoveModuleFromManifest(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "modules.yaml")
	if err := os.WriteFile(path, []byte(modulesManifestHeader+"  - notifications\n  - jobs\n"), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	removed, content, err := removeModuleFromManifest(path, "notifications")
	if err != nil {
		t.Fatalf("remove manifest: %v", err)
	}
	if !removed {
		t.Fatal("expected manifest change")
	}
	if strings.Contains(content, "- notifications") {
		t.Fatalf("module still present: %s", content)
	}
	if !strings.Contains(content, "- jobs") {
		t.Fatalf("unexpected manifest: %s", content)
	}
}

func TestNotificationsModuleCatalog_UsesConcreteWiring(t *testing.T) {
	info, ok := moduleCatalog["notifications"]
	if !ok {
		t.Fatal("expected notifications in module catalog")
	}
	if strings.Contains(info.ContainerSnippet, "TODO:") {
		t.Fatalf("container snippet still contains TODO text: %q", info.ContainerSnippet)
	}
	if strings.Contains(info.RouterSnippets["auth"], "TODO:") {
		t.Fatalf("router snippet still contains TODO text: %q", info.RouterSnippets["auth"])
	}
	if !strings.Contains(info.ContainerSnippet, "initNotifier") {
		t.Fatalf("container snippet missing initNotifier wiring: %q", info.ContainerSnippet)
	}
	for _, want := range []string{
		"notificationsModule := notificationroutes.NewRouteModule",
		"RegisterOnboardingRoutes(onboardingGroup)",
		"RegisterRoutes(onboardedGroup)",
	} {
		if !strings.Contains(info.RouterSnippets["auth"], want) {
			t.Fatalf("router snippet missing %q: %q", want, info.RouterSnippets["auth"])
		}
	}
}

func TestNotificationsModuleCatalog_InstallContractCoversRoutesViewsAndJobs(t *testing.T) {
	t.Parallel()

	info, ok := moduleCatalog["notifications"]
	if !ok {
		t.Fatal("expected notifications in module catalog")
	}

	contract := info.installContract()
	for _, configPath := range []string{
		"config/modules.yaml",
		"app/foundation/container.go",
		"go.mod",
		"go.work",
	} {
		if !containsExactString(contract.Config, configPath) {
			t.Fatalf("notifications contract config missing %q: %#v", configPath, contract.Config)
		}
	}
	for _, route := range []string{
		"app/router.go (auth)",
		"modules/notifications/routes/routes.go",
	} {
		if !containsExactString(contract.Routes, route) {
			t.Fatalf("notifications contract routes missing %q: %#v", route, contract.Routes)
		}
	}
	for _, tmpl := range []string{
		"framework/web/pages/gen/notifications_templ.go",
	} {
		if !containsExactString(contract.Templates, tmpl) {
			t.Fatalf("notifications contract templates missing %q: %#v", tmpl, contract.Templates)
		}
	}
	for _, jobSurface := range []string{
		"modules/notifications/planned_notifications.go",
		"modules/notifications/planned_notifications_store_sql.go",
	} {
		if !containsExactString(contract.Jobs, jobSurface) {
			t.Fatalf("notifications contract jobs missing %q: %#v", jobSurface, contract.Jobs)
		}
	}
	if !containsExactString(contract.Migrations, "modules/notifications/db/migrate/migrations") {
		t.Fatalf("notifications contract migrations missing module migration path: %#v", contract.Migrations)
	}
	for _, testSurface := range []string{
		"modules/notifications/routes/routes_contract_test.go",
		"modules/notifications/module_sql_mode_test.go",
		"modules/notifications/planned_notifications_store_sql_test.go",
	} {
		if !containsExactString(contract.Tests, testSurface) {
			t.Fatalf("notifications contract tests missing %q: %#v", testSurface, contract.Tests)
		}
	}
}

func TestModuleCatalog_FirstPartyConfigOwnershipHasNoNonCanonicalCollisions(t *testing.T) {
	t.Parallel()

	allowedSharedConfigSurfaces := map[string]struct{}{
		"config/modules.yaml":         {},
		"app/foundation/container.go": {},
		"go.mod":                      {},
		"go.work":                     {},
		".env.example":                {},
	}

	moduleByID := map[string]moduleInfo{}
	ids := make([]string, 0, len(moduleCatalog))
	for _, info := range moduleCatalog {
		if strings.TrimSpace(info.ID) == "" {
			continue
		}
		if _, exists := moduleByID[info.ID]; exists {
			continue
		}
		moduleByID[info.ID] = info
		ids = append(ids, info.ID)
	}
	sort.Strings(ids)

	configOwners := map[string][]string{}
	for _, id := range ids {
		contract := moduleByID[id].installContract()
		for _, cfg := range contract.Config {
			cfg = strings.TrimSpace(cfg)
			if cfg == "" {
				continue
			}
			if _, ok := allowedSharedConfigSurfaces[cfg]; ok {
				continue
			}
			configOwners[cfg] = append(configOwners[cfg], id)
		}
	}

	var collisions []string
	for cfg, owners := range configOwners {
		owners = dedupeStrings(owners)
		if len(owners) < 2 {
			continue
		}
		sort.Strings(owners)
		collisions = append(collisions, fmt.Sprintf("%s -> %s", cfg, strings.Join(owners, ", ")))
	}
	if len(collisions) == 0 {
		return
	}
	sort.Strings(collisions)
	t.Fatalf("non-canonical config ownership collisions detected:\n%s", strings.Join(collisions, "\n"))
}

func TestModuleCatalog_FirstPartyAssetOwnershipHasNoCollisions(t *testing.T) {
	t.Parallel()

	moduleByID := map[string]moduleInfo{}
	ids := make([]string, 0, len(moduleCatalog))
	for _, info := range moduleCatalog {
		if strings.TrimSpace(info.ID) == "" {
			continue
		}
		if _, exists := moduleByID[info.ID]; exists {
			continue
		}
		moduleByID[info.ID] = info
		ids = append(ids, info.ID)
	}
	sort.Strings(ids)

	assetOwners := map[string][]string{}
	for _, id := range ids {
		contract := moduleByID[id].installContract()
		for _, asset := range contract.Assets {
			asset = strings.TrimSpace(asset)
			if asset == "" {
				continue
			}
			assetOwners[asset] = append(assetOwners[asset], id)
		}
	}

	var collisions []string
	for asset, owners := range assetOwners {
		owners = dedupeStrings(owners)
		if len(owners) < 2 {
			continue
		}
		sort.Strings(owners)
		collisions = append(collisions, fmt.Sprintf("%s -> %s", asset, strings.Join(owners, ", ")))
	}
	if len(collisions) == 0 {
		return
	}
	sort.Strings(collisions)
	t.Fatalf("asset ownership collisions detected across first-party modules:\n%s", strings.Join(collisions, "\n"))
}

func TestModuleCatalogUsesConcreteJobsAndStorageSeams(t *testing.T) {
	tests := []struct {
		name    string
		snippet string
		needles []string
	}{
		{
			name:    "jobs",
			snippet: moduleCatalog["jobs"].ContainerSnippet,
			needles: []string{"framework/bootstrap.WireJobsRuntime", "c.CoreJobs", "c.CoreJobsInspector"},
		},
		{
			name:    "storage",
			snippet: moduleCatalog["storage"].ContainerSnippet,
			needles: []string{"modules/storage.New", "core.BlobStorage"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if strings.Contains(tt.snippet, "TODO") {
				t.Fatalf("snippet still contains TODO text:\n%s", tt.snippet)
			}
			for _, needle := range tt.needles {
				if !strings.Contains(tt.snippet, needle) {
					t.Fatalf("snippet missing %q:\n%s", needle, tt.snippet)
				}
			}
		})
	}
}

func TestJobsModuleCatalog_InstallContractCoversWorkerQueueScheduleTests(t *testing.T) {
	t.Parallel()

	info, ok := moduleCatalog["jobs"]
	if !ok {
		t.Fatal("expected jobs in module catalog")
	}
	contract := info.installContract()

	for _, configPath := range []string{
		"config/modules.yaml",
		"app/foundation/container.go",
		"go.mod",
		"go.work",
	} {
		if !containsExactString(contract.Config, configPath) {
			t.Fatalf("jobs contract config missing %q: %#v", configPath, contract.Config)
		}
	}
	if !containsExactString(contract.Migrations, "modules/jobs/db/migrate/migrations") {
		t.Fatalf("jobs contract missing migration ownership: %#v", contract.Migrations)
	}
	for _, jobSurface := range []string{
		"modules/jobs/core_jobs_sql.go",
		"modules/jobs/core_jobs_redis.go",
		"modules/jobs/core_jobs_backlite.go",
		"modules/jobs/queue_priority.go",
	} {
		if !containsExactString(contract.Jobs, jobSurface) {
			t.Fatalf("jobs contract missing runtime surface %q: %#v", jobSurface, contract.Jobs)
		}
	}
	for _, testSurface := range []string{
		"modules/jobs/core_jobs_sql_test.go",
		"modules/jobs/drivers/sql/client_integration_test.go",
		"modules/jobs/core_jobs_redis_schedule_test.go",
	} {
		if !containsExactString(contract.Tests, testSurface) {
			t.Fatalf("jobs contract missing integration test surface %q: %#v", testSurface, contract.Tests)
		}
	}
}

func TestStorageModuleCatalog_InstallContractCoversAdapterConfigAndFilePathTests(t *testing.T) {
	t.Parallel()

	info, ok := moduleCatalog["storage"]
	if !ok {
		t.Fatal("expected storage in module catalog")
	}
	contract := info.installContract()

	for _, configPath := range []string{
		"config/modules.yaml",
		"app/foundation/container.go",
		"go.mod",
		"go.work",
	} {
		if !containsExactString(contract.Config, configPath) {
			t.Fatalf("storage contract config missing %q: %#v", configPath, contract.Config)
		}
	}
	if !containsExactString(contract.Migrations, "modules/storage/db/migrate/migrations") {
		t.Fatalf("storage contract missing migration path ownership: %#v", contract.Migrations)
	}
	for _, runtimeSurface := range []string{
		"modules/storage/module.go",
	} {
		if !containsExactString(contract.Jobs, runtimeSurface) {
			t.Fatalf("storage contract missing adapter/runtime surface %q: %#v", runtimeSurface, contract.Jobs)
		}
	}
	for _, testSurface := range []string{
		"modules/storage/module_test.go",
	} {
		if !containsExactString(contract.Tests, testSurface) {
			t.Fatalf("storage contract missing integration test surface %q: %#v", testSurface, contract.Tests)
		}
	}
}

func TestModuleCatalog_InstallContractCoverage(t *testing.T) {
	for id, info := range moduleCatalog {
		if info.installContract().IsEmpty() {
			t.Fatalf("module %q must define an install contract", id)
		}
	}
}

func TestRunModuleAdd_DryRunPrintsInstallContract(t *testing.T) {
	root := t.TempDir()
	writeModuleFixtureFiles(t, root)

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	code := RunModule([]string{"add", "jobs", "--dry-run"}, ModuleDeps{
		Out: out,
		Err: errOut,
		FindGoModule: func(start string) (string, string, error) {
			return root, "example.com/demo", nil
		},
	})
	if code != 0 {
		t.Fatalf("exit code = %d, stderr=%s", code, errOut.String())
	}
	for _, token := range []string{
		"install contract for module",
		"routes:",
		"config:",
		"assets:",
		"jobs:",
		"templates:",
		"migrations:",
		"tests:",
	} {
		if !strings.Contains(strings.ToLower(out.String()), token) {
			t.Fatalf("dry-run output missing %q:\n%s", token, out.String())
		}
	}
}

func TestRunModuleAdd_DryRunExplainsOwnershipMap(t *testing.T) {
	root := t.TempDir()
	writeModuleFixtureFiles(t, root)
	writeLocalModuleGoModFixture(t, root, filepath.Join("modules", "jobs"), "github.com/leomorpho/goship-modules/jobs")

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	code := RunModule([]string{"add", "jobs", "--dry-run"}, ModuleDeps{
		Out: out,
		Err: errOut,
		FindGoModule: func(start string) (string, string, error) {
			return root, "example.com/demo", nil
		},
	})
	if code != 0 {
		t.Fatalf("exit code = %d, stderr=%s", code, errOut.String())
	}

	output := out.String()
	if !strings.Contains(output, "ownership:") {
		t.Fatalf("dry-run output missing ownership section:\n%s", output)
	}
	for _, token := range []string{
		"app/foundation/container.go -> config",
		"config/modules.yaml -> config",
		"go.mod -> config",
		"go.work -> config",
		"modules/jobs/db/migrate/migrations -> migrations",
	} {
		if !strings.Contains(output, token) {
			t.Fatalf("dry-run output missing ownership token %q:\n%s", token, output)
		}
	}
}

func TestRunModuleRemove_AbsentModuleIsNoOp(t *testing.T) {
	root := t.TempDir()
	writeModuleFixtureFiles(t, root)

	runOnce := func(t *testing.T) string {
		t.Helper()
		out := &bytes.Buffer{}
		errOut := &bytes.Buffer{}
		code := RunModule([]string{"remove", "storage"}, ModuleDeps{
			Out: out,
			Err: errOut,
			FindGoModule: func(start string) (string, string, error) {
				return root, "example.com/demo", nil
			},
		})
		if code != 0 {
			t.Fatalf("exit code = %d, stderr=%s", code, errOut.String())
		}
		return out.String()
	}

	first := runOnce(t)
	second := runOnce(t)
	for _, log := range []string{first, second} {
		if !strings.Contains(log, "Module was not wired; no changes needed.") {
			t.Fatalf("expected no-op message, got:\n%s", log)
		}
		if !strings.Contains(log, "Reminder: module:remove does not roll back related DB migrations.") {
			t.Fatalf("expected reminder message, got:\n%s", log)
		}
	}
}

func TestApplyModuleAddRemove_NotificationsIdempotent(t *testing.T) {
	root := t.TempDir()
	info, ok := moduleCatalog["notifications"]
	if !ok {
		t.Fatal("expected notifications in module catalog")
	}
	writeNotificationsModuleFixtureFiles(t, root)

	containerPath := filepath.Join(root, "app", "foundation", "container.go")
	routerPath := filepath.Join(root, "app", "router.go")
	manifestPath := filepath.Join(root, "config", "modules.yaml")
	goModPath := filepath.Join(root, "go.mod")
	goWorkPath := filepath.Join(root, "go.work")

	originalContainer := readTestFile(t, containerPath)
	originalRouter := readTestFile(t, routerPath)
	originalManifest := readTestFile(t, manifestPath)
	originalGoMod := readTestFile(t, goModPath)
	originalGoWork := readTestFile(t, goWorkPath)

	if err := applyModuleAdd(root, info, false, io.Discard); err != nil {
		t.Fatalf("applyModuleAdd first pass error: %v", err)
	}
	addedContainer := readTestFile(t, containerPath)
	addedRouter := readTestFile(t, routerPath)
	addedManifest := readTestFile(t, manifestPath)
	addedGoMod := readTestFile(t, goModPath)
	addedGoWork := readTestFile(t, goWorkPath)

	for _, want := range []string{
		"c.initNotifier()",
		"notificationsModule := notificationroutes.NewRouteModule",
		"RegisterOnboardingRoutes(onboardingGroup)",
		"RegisterRoutes(onboardedGroup)",
	} {
		if !strings.Contains(addedContainer+addedRouter, want) {
			t.Fatalf("missing notifications wiring token %q after add:\ncontainer:\n%s\nrouter:\n%s", want, addedContainer, addedRouter)
		}
	}
	if strings.Contains(addedContainer+addedRouter, "TODO:") {
		t.Fatalf("notifications wiring still contains TODO text after add:\ncontainer:\n%s\nrouter:\n%s", addedContainer, addedRouter)
	}
	if !strings.Contains(addedManifest, "- notifications") {
		t.Fatalf("expected notifications in modules manifest, got:\n%s", addedManifest)
	}
	if !strings.Contains(addedGoMod, "github.com/leomorpho/goship-modules/notifications v0.0.0") {
		t.Fatalf("expected notifications require in go.mod, got:\n%s", addedGoMod)
	}
	if !strings.Contains(addedGoWork, "./modules/notifications") {
		t.Fatalf("expected notifications in go.work use list, got:\n%s", addedGoWork)
	}

	if err := applyModuleAdd(root, info, false, io.Discard); err != nil {
		t.Fatalf("applyModuleAdd second pass error: %v", err)
	}
	if got := readTestFile(t, containerPath); got != addedContainer {
		t.Fatalf("container mutated on re-add:\nfirst:\n%s\nsecond:\n%s", addedContainer, got)
	}
	if got := readTestFile(t, routerPath); got != addedRouter {
		t.Fatalf("router mutated on re-add:\nfirst:\n%s\nsecond:\n%s", addedRouter, got)
	}
	if got := readTestFile(t, manifestPath); got != addedManifest {
		t.Fatalf("manifest mutated on re-add:\nfirst:\n%s\nsecond:\n%s", addedManifest, got)
	}
	if got := readTestFile(t, goModPath); got != addedGoMod {
		t.Fatalf("go.mod mutated on re-add:\nfirst:\n%s\nsecond:\n%s", addedGoMod, got)
	}
	if got := readTestFile(t, goWorkPath); got != addedGoWork {
		t.Fatalf("go.work mutated on re-add:\nfirst:\n%s\nsecond:\n%s", addedGoWork, got)
	}

	if err := applyModuleRemove(root, info, false, io.Discard); err != nil {
		t.Fatalf("applyModuleRemove first pass error: %v", err)
	}
	if got := readTestFile(t, containerPath); got != originalContainer {
		t.Fatalf("container did not restore after remove:\nwant:\n%s\ngot:\n%s", originalContainer, got)
	}
	if got := readTestFile(t, routerPath); got != originalRouter {
		t.Fatalf("router did not restore after remove:\nwant:\n%s\ngot:\n%s", originalRouter, got)
	}
	if got := readTestFile(t, manifestPath); got != originalManifest {
		t.Fatalf("manifest did not restore after remove:\nwant:\n%s\ngot:\n%s", originalManifest, got)
	}
	if got := readTestFile(t, goModPath); got != originalGoMod {
		t.Fatalf("go.mod did not restore after remove:\nwant:\n%s\ngot:\n%s", originalGoMod, got)
	}
	if got := readTestFile(t, goWorkPath); got != originalGoWork {
		t.Fatalf("go.work did not restore after remove:\nwant:\n%s\ngot:\n%s", originalGoWork, got)
	}

	if err := applyModuleRemove(root, info, false, io.Discard); err != nil {
		t.Fatalf("applyModuleRemove second pass error: %v", err)
	}
}

func TestApplyModuleAddRemove_FirstPartyBatteriesGolden(t *testing.T) {
	firstPartyBatteries := []string{
		"notifications",
		"paidsubscriptions",
		"emailsubscriptions",
		"realtime",
		"pwa",
		"jobs",
		"storage",
	}

	for _, battery := range firstPartyBatteries {
		t.Run(battery, func(t *testing.T) {
			info, ok := moduleCatalog[battery]
			if !ok {
				t.Fatalf("expected %q in module catalog", battery)
			}

			root := t.TempDir()
			writeNotificationsModuleFixtureFiles(t, root)
			writeLocalModuleGoModFixture(t, root, info.LocalPath, info.ModulePath)

			tracked := []string{
				filepath.Join(root, "app", "foundation", "container.go"),
				filepath.Join(root, "app", "router.go"),
				filepath.Join(root, "config", "modules.yaml"),
				filepath.Join(root, "go.mod"),
				filepath.Join(root, "go.work"),
				filepath.Join(root, ".env.example"),
			}
			original := map[string]string{}
			for _, path := range tracked {
				original[path] = readTestFile(t, path)
			}

			if err := applyModuleAdd(root, info, false, io.Discard); err != nil {
				t.Fatalf("applyModuleAdd error: %v", err)
			}
			if manifest := readTestFile(t, filepath.Join(root, "config", "modules.yaml")); !strings.Contains(manifest, "- "+info.ID) {
				t.Fatalf("expected %q in modules manifest after add, got:\n%s", info.ID, manifest)
			}

			if err := applyModuleRemove(root, info, false, io.Discard); err != nil {
				t.Fatalf("applyModuleRemove error: %v", err)
			}

			for _, path := range tracked {
				if got := readTestFile(t, path); got != original[path] {
					t.Fatalf("%s did not restore after add/remove for %s:\nwant:\n%s\ngot:\n%s", path, battery, original[path], got)
				}
			}
		})
	}
}

func TestApplyModuleAdd_IdempotentAcrossFirstPartyBatteries(t *testing.T) {
	firstPartyBatteries := []string{
		"notifications",
		"paidsubscriptions",
		"emailsubscriptions",
		"realtime",
		"pwa",
		"jobs",
		"storage",
	}

	for _, battery := range firstPartyBatteries {
		t.Run(battery, func(t *testing.T) {
			info, ok := moduleCatalog[battery]
			if !ok {
				t.Fatalf("expected %q in module catalog", battery)
			}

			root := t.TempDir()
			writeNotificationsModuleFixtureFiles(t, root)
			writeLocalModuleGoModFixture(t, root, info.LocalPath, info.ModulePath)

			tracked := []string{
				filepath.Join(root, "app", "foundation", "container.go"),
				filepath.Join(root, "app", "router.go"),
				filepath.Join(root, "config", "modules.yaml"),
				filepath.Join(root, "go.mod"),
				filepath.Join(root, "go.work"),
				filepath.Join(root, ".env.example"),
			}

			if err := applyModuleAdd(root, info, false, io.Discard); err != nil {
				t.Fatalf("applyModuleAdd first pass error: %v", err)
			}
			firstPass := map[string]string{}
			for _, path := range tracked {
				firstPass[path] = readTestFile(t, path)
			}

			if err := applyModuleAdd(root, info, false, io.Discard); err != nil {
				t.Fatalf("applyModuleAdd second pass error: %v", err)
			}
			for _, path := range tracked {
				if got := readTestFile(t, path); got != firstPass[path] {
					t.Fatalf("%s changed on second add for %s:\nfirst:\n%s\nsecond:\n%s", path, battery, firstPass[path], got)
				}
			}
		})
	}
}

func TestApplyModuleAdd_ComposesSupportedFirstPartyPairs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		first  string
		second string
	}{
		{name: "notifications and jobs", first: "notifications", second: "jobs"},
		{name: "notifications and storage", first: "notifications", second: "storage"},
		{name: "notifications and realtime", first: "notifications", second: "realtime"},
		{name: "notifications and pwa", first: "notifications", second: "pwa"},
		{name: "paidsubscriptions and emailsubscriptions", first: "paidsubscriptions", second: "emailsubscriptions"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			writeNotificationsModuleFixtureFiles(t, root)

			firstInfo := moduleCatalog[tt.first]
			secondInfo := moduleCatalog[tt.second]
			writeLocalModuleGoModFixture(t, root, firstInfo.LocalPath, firstInfo.ModulePath)
			writeLocalModuleGoModFixture(t, root, secondInfo.LocalPath, secondInfo.ModulePath)

			if err := applyModuleAdd(root, firstInfo, false, io.Discard); err != nil {
				t.Fatalf("applyModuleAdd(%s) error: %v", tt.first, err)
			}
			if err := applyModuleAdd(root, secondInfo, false, io.Discard); err != nil {
				t.Fatalf("applyModuleAdd(%s) error: %v", tt.second, err)
			}

			manifest := readTestFile(t, filepath.Join(root, "config", "modules.yaml"))
			if !strings.Contains(manifest, "- "+firstInfo.ID) {
				t.Fatalf("modules manifest missing %q after composition, got:\n%s", firstInfo.ID, manifest)
			}
			if !strings.Contains(manifest, "- "+secondInfo.ID) {
				t.Fatalf("modules manifest missing %q after composition, got:\n%s", secondInfo.ID, manifest)
			}

			goMod := readTestFile(t, filepath.Join(root, "go.mod"))
			for _, info := range []moduleInfo{firstInfo, secondInfo} {
				if strings.TrimSpace(info.ModulePath) == "" {
					continue
				}
				if !strings.Contains(goMod, info.ModulePath+" v0.0.0") {
					t.Fatalf("go.mod missing require for %q after composition, got:\n%s", info.ModulePath, goMod)
				}
				if !strings.Contains(goMod, "replace "+info.ModulePath+" => ./"+filepath.ToSlash(info.LocalPath)) {
					t.Fatalf("go.mod missing replace for %q after composition, got:\n%s", info.ModulePath, goMod)
				}
			}

			goWork := readTestFile(t, filepath.Join(root, "go.work"))
			for _, info := range []moduleInfo{firstInfo, secondInfo} {
				if strings.TrimSpace(info.LocalPath) == "" {
					continue
				}
				if !strings.Contains(goWork, "./"+filepath.ToSlash(info.LocalPath)) {
					t.Fatalf("go.work missing use for %q after composition, got:\n%s", info.LocalPath, goWork)
				}
			}
		})
	}
}

func TestApplyModuleAddRemove_SupportedBatteryMatrixLeavesCleanScaffold(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		batteries []string
	}{
		{name: "notifications and jobs", batteries: []string{"notifications", "jobs"}},
		{name: "notifications and storage", batteries: []string{"notifications", "storage"}},
		{name: "notifications realtime and pwa", batteries: []string{"notifications", "realtime", "pwa"}},
		{name: "paidsubscriptions and emailsubscriptions", batteries: []string{"paidsubscriptions", "emailsubscriptions"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			writeNotificationsModuleFixtureFiles(t, root)

			infos := make([]moduleInfo, 0, len(tt.batteries))
			for _, battery := range tt.batteries {
				info, ok := moduleCatalog[battery]
				if !ok {
					t.Fatalf("expected %q in module catalog", battery)
				}
				writeLocalModuleGoModFixture(t, root, info.LocalPath, info.ModulePath)
				if err := applyModuleAdd(root, info, false, io.Discard); err != nil {
					t.Fatalf("applyModuleAdd(%s) error: %v", battery, err)
				}
				infos = append(infos, info)
			}

			for i := len(infos) - 1; i >= 0; i-- {
				info := infos[i]
				if err := applyModuleRemove(root, info, false, io.Discard); err != nil {
					t.Fatalf("applyModuleRemove(%s) error: %v", info.ID, err)
				}
			}

			manifest := readTestFile(t, filepath.Join(root, "config", "modules.yaml"))
			if manifest != modulesManifestHeader {
				t.Fatalf("modules manifest should reset to baseline after matrix remove, got:\n%s", manifest)
			}

			container := readTestFile(t, filepath.Join(root, "app", "foundation", "container.go"))
			router := readTestFile(t, filepath.Join(root, "app", "router.go"))
			goMod := readTestFile(t, filepath.Join(root, "go.mod"))
			goWork := readTestFile(t, filepath.Join(root, "go.work"))
			for _, info := range infos {
				if snippet := strings.TrimSpace(info.ContainerSnippet); snippet != "" && strings.Contains(container, snippet) {
					t.Fatalf("container still contains %q snippet after remove matrix", info.ID)
				}
				for group, snippet := range info.RouterSnippets {
					if snippet = strings.TrimSpace(snippet); snippet != "" && strings.Contains(router, snippet) {
						t.Fatalf("router still contains %q group snippet for %q after remove matrix", group, info.ID)
					}
				}
				if strings.TrimSpace(info.ModulePath) != "" {
					if strings.Contains(goMod, info.ModulePath+" v0.0.0") {
						t.Fatalf("go.mod still contains require for %q after remove matrix", info.ModulePath)
					}
					if strings.Contains(goMod, "replace "+info.ModulePath+" => ./"+filepath.ToSlash(info.LocalPath)) {
						t.Fatalf("go.mod still contains replace for %q after remove matrix", info.ModulePath)
					}
				}
				if strings.TrimSpace(info.LocalPath) != "" && strings.Contains(goWork, "./"+filepath.ToSlash(info.LocalPath)) {
					t.Fatalf("go.work still contains use for %q after remove matrix", info.LocalPath)
				}
			}
		})
	}
}

func TestApplyModuleRemove_ComposedPairLeavesRemainingBatteryIntact(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeNotificationsModuleFixtureFiles(t, root)

	notifications := moduleCatalog["notifications"]
	jobs := moduleCatalog["jobs"]
	writeLocalModuleGoModFixture(t, root, notifications.LocalPath, notifications.ModulePath)
	writeLocalModuleGoModFixture(t, root, jobs.LocalPath, jobs.ModulePath)

	if err := applyModuleAdd(root, notifications, false, io.Discard); err != nil {
		t.Fatalf("applyModuleAdd(notifications) error: %v", err)
	}
	if err := applyModuleAdd(root, jobs, false, io.Discard); err != nil {
		t.Fatalf("applyModuleAdd(jobs) error: %v", err)
	}

	if err := applyModuleRemove(root, notifications, false, io.Discard); err != nil {
		t.Fatalf("applyModuleRemove(notifications) error: %v", err)
	}

	manifest := readTestFile(t, filepath.Join(root, "config", "modules.yaml"))
	if strings.Contains(manifest, "- notifications") {
		t.Fatalf("notifications should be removed from modules manifest, got:\n%s", manifest)
	}
	if !strings.Contains(manifest, "- jobs") {
		t.Fatalf("jobs should remain in modules manifest, got:\n%s", manifest)
	}

	container := readTestFile(t, filepath.Join(root, "app", "foundation", "container.go"))
	if strings.Contains(container, "c.initNotifier()") {
		t.Fatalf("notifications wiring should be removed from container, got:\n%s", container)
	}
	for _, token := range []string{"framework/bootstrap.WireJobsRuntime", "c.CoreJobs", "c.CoreJobsInspector"} {
		if !strings.Contains(container, token) {
			t.Fatalf("jobs wiring token %q missing after removing notifications, got:\n%s", token, container)
		}
	}

	goMod := readTestFile(t, filepath.Join(root, "go.mod"))
	if strings.Contains(goMod, "github.com/leomorpho/goship-modules/notifications") {
		t.Fatalf("notifications dependency should be removed from go.mod, got:\n%s", goMod)
	}
	if !strings.Contains(goMod, "github.com/leomorpho/goship-modules/jobs v0.0.0") {
		t.Fatalf("jobs dependency should remain in go.mod, got:\n%s", goMod)
	}
}

func TestApplyModuleAdd_WiresLocalModuleDependencyContract_RedSpec(t *testing.T) {
	root := t.TempDir()
	writeModuleFixtureFiles(t, root)

	info, ok := moduleCatalog["notifications"]
	if !ok {
		t.Fatal("expected notifications in module catalog")
	}
	if err := applyModuleAdd(root, info, false, io.Discard); err != nil {
		t.Fatalf("applyModuleAdd error: %v", err)
	}

	goMod := readTestFile(t, filepath.Join(root, "go.mod"))
	if !strings.Contains(goMod, "github.com/leomorpho/goship-modules/notifications v0.0.0") {
		t.Fatalf("expected notifications require in go.mod, got:\n%s", goMod)
	}
	if !strings.Contains(goMod, "replace github.com/leomorpho/goship-modules/notifications => ./modules/notifications") {
		t.Fatalf("expected notifications replace in go.mod, got:\n%s", goMod)
	}

	goWork := readTestFile(t, filepath.Join(root, "go.work"))
	if !strings.Contains(goWork, "./modules/notifications") {
		t.Fatalf("expected notifications in go.work use list, got:\n%s", goWork)
	}
}

func TestApplyModuleRemove_FailsWithReferenceBlockers_RedSpec(t *testing.T) {
	root := t.TempDir()
	writeModuleFixtureFiles(t, root)

	info, ok := moduleCatalog["notifications"]
	if !ok {
		t.Fatal("expected notifications in module catalog")
	}
	err := applyModuleRemove(root, info, false, io.Discard)
	if err == nil {
		t.Fatal("expected remove blocker error")
	}
	if !strings.Contains(err.Error(), "module remove blocked") {
		t.Fatalf("expected blocker error, got %v", err)
	}
	if !strings.Contains(err.Error(), "app/router.go") {
		t.Fatalf("expected router blocker in error, got %v", err)
	}
}

func TestApplyModuleRemove_AbsentModuleIsIdempotentNoOp(t *testing.T) {
	root := t.TempDir()
	writeModuleFixtureFiles(t, root)

	info, ok := moduleCatalog["storage"]
	if !ok {
		t.Fatal("expected storage in module catalog")
	}

	tracked := []string{
		filepath.Join(root, "app", "foundation", "container.go"),
		filepath.Join(root, "app", "router.go"),
		filepath.Join(root, "config", "modules.yaml"),
		filepath.Join(root, "go.mod"),
		filepath.Join(root, "go.work"),
		filepath.Join(root, ".env.example"),
	}
	original := map[string]string{}
	for _, path := range tracked {
		original[path] = readTestFile(t, path)
	}

	for i := 0; i < 2; i++ {
		out := &bytes.Buffer{}
		if err := applyModuleRemove(root, info, false, out); err != nil {
			t.Fatalf("applyModuleRemove pass %d error: %v", i+1, err)
		}
		if !strings.Contains(out.String(), "Module was not wired; no changes needed.") {
			t.Fatalf("expected no-op message on pass %d, got:\n%s", i+1, out.String())
		}
	}

	for _, path := range tracked {
		if got := readTestFile(t, path); got != original[path] {
			t.Fatalf("%s changed after removing absent module:\nwant:\n%s\ngot:\n%s", path, original[path], got)
		}
	}
}

func TestPaidSubscriptionsModuleCatalog_InstallContractCoversRoutesConfigAndTests(t *testing.T) {
	t.Parallel()

	info, ok := moduleCatalog["paidsubscriptions"]
	if !ok {
		t.Fatal("expected paidsubscriptions in module catalog")
	}
	contract := info.installContract()

	for _, route := range []string{
		"app/router.go (auth)",
		"app/router.go (external)",
		"modules/paidsubscriptions/routes/routes.go",
	} {
		if !containsExactString(contract.Routes, route) {
			t.Fatalf("paidsubscriptions contract routes missing %q: %#v", route, contract.Routes)
		}
	}
	for _, configPath := range []string{
		"config/modules.yaml",
		"app/foundation/container.go",
		"go.mod",
		"go.work",
		".env.example",
	} {
		if !containsExactString(contract.Config, configPath) {
			t.Fatalf("paidsubscriptions contract config missing %q: %#v", configPath, contract.Config)
		}
	}
	if !containsExactString(contract.Migrations, "modules/paidsubscriptions/db/migrate/migrations") {
		t.Fatalf("paidsubscriptions contract missing migration path ownership: %#v", contract.Migrations)
	}
	for _, runtimeSurface := range []string{
		"modules/paidsubscriptions/service.go",
		"modules/paidsubscriptions/store_sql.go",
		"modules/paidsubscriptions/plan_catalog.go",
	} {
		if !containsExactString(contract.Jobs, runtimeSurface) {
			t.Fatalf("paidsubscriptions contract missing runtime surface %q: %#v", runtimeSurface, contract.Jobs)
		}
	}
	for _, testSurface := range []string{
		"modules/paidsubscriptions/service_test.go",
		"modules/paidsubscriptions/store_sql_test.go",
		"modules/paidsubscriptions/store_sql_integration_test.go",
	} {
		if !containsExactString(contract.Tests, testSurface) {
			t.Fatalf("paidsubscriptions contract missing test ownership %q: %#v", testSurface, contract.Tests)
		}
	}
}

func TestEmailSubscriptionsModuleCatalog_UsesConcreteWiring(t *testing.T) {
	info, ok := moduleCatalog["emailsubscriptions"]
	if !ok {
		t.Fatal("expected emailsubscriptions in module catalog")
	}
	if strings.Contains(info.ContainerSnippet, "TODO") {
		t.Fatalf("emailsubscriptions container snippet still contains TODO text:\n%s", info.ContainerSnippet)
	}

	authSnippet := strings.TrimSpace(info.RouterSnippets["auth"])
	if authSnippet == "" {
		t.Fatal("expected emailsubscriptions auth router snippet")
	}
	if strings.Contains(authSnippet, "TODO") {
		t.Fatalf("emailsubscriptions auth router snippet still contains TODO text:\n%s", authSnippet)
	}
	for _, token := range []string{
		"modules/emailsubscriptions.New",
		"RouteNameDeleteEmailSubscriptionWithToken",
		"modules/notifications/routes/routes.go",
	} {
		if !strings.Contains(info.ContainerSnippet+authSnippet, token) {
			t.Fatalf("emailsubscriptions wiring missing %q:\ncontainer:\n%s\nauth:\n%s", token, info.ContainerSnippet, authSnippet)
		}
	}
}

func TestEmailSubscriptionsModuleCatalog_InstallContractCoversRoutesRuntimeAndVerificationTests(t *testing.T) {
	t.Parallel()

	info, ok := moduleCatalog["emailsubscriptions"]
	if !ok {
		t.Fatal("expected emailsubscriptions in module catalog")
	}
	contract := info.installContract()

	for _, route := range []string{
		"app/router.go (auth)",
		"modules/notifications/routes/routes.go",
	} {
		if !containsExactString(contract.Routes, route) {
			t.Fatalf("emailsubscriptions contract routes missing %q: %#v", route, contract.Routes)
		}
	}
	for _, configPath := range []string{
		"config/modules.yaml",
		"app/foundation/container.go",
		"go.mod",
		"go.work",
	} {
		if !containsExactString(contract.Config, configPath) {
			t.Fatalf("emailsubscriptions contract config missing %q: %#v", configPath, contract.Config)
		}
	}
	if !containsExactString(contract.Migrations, "modules/emailsubscriptions/db/migrate/migrations") {
		t.Fatalf("emailsubscriptions contract missing migration path ownership: %#v", contract.Migrations)
	}
	for _, runtimeSurface := range []string{
		"modules/emailsubscriptions/service.go",
		"modules/emailsubscriptions/store_sql.go",
		"modules/emailsubscriptions/catalog.go",
	} {
		if !containsExactString(contract.Jobs, runtimeSurface) {
			t.Fatalf("emailsubscriptions contract missing runtime surface %q: %#v", runtimeSurface, contract.Jobs)
		}
	}
	for _, testSurface := range []string{
		"modules/emailsubscriptions/service_test.go",
		"modules/emailsubscriptions/store_sql_test.go",
		"modules/emailsubscriptions/store_sql_integration_test.go",
		"modules/emailsubscriptions/catalog_test.go",
		"modules/notifications/routes/routes_contract_test.go",
	} {
		if !containsExactString(contract.Tests, testSurface) {
			t.Fatalf("emailsubscriptions contract missing verification test ownership %q: %#v", testSurface, contract.Tests)
		}
	}
}

func TestRealtimeModuleCatalog_UsesConcreteWiring(t *testing.T) {
	info, ok := moduleCatalog["realtime"]
	if !ok {
		t.Fatal("expected realtime in module catalog")
	}
	if strings.Contains(info.ContainerSnippet, "TODO") {
		t.Fatalf("realtime container snippet still contains TODO text:\n%s", info.ContainerSnippet)
	}

	authSnippet := strings.TrimSpace(info.RouterSnippets["auth"])
	if authSnippet == "" {
		t.Fatal("expected realtime auth router snippet")
	}
	if strings.Contains(authSnippet, "TODO") {
		t.Fatalf("realtime auth router snippet still contains TODO text:\n%s", authSnippet)
	}
	for _, token := range []string{
		"initSSEHub",
		"registerRealtimeRoutes",
		"RouteNameRealtime",
	} {
		if !strings.Contains(info.ContainerSnippet+authSnippet, token) {
			t.Fatalf("realtime wiring missing %q:\ncontainer:\n%s\nauth:\n%s", token, info.ContainerSnippet, authSnippet)
		}
	}
}

func TestRealtimeModuleCatalog_InstallContractCoversStarterAndRuntimeStartupTests(t *testing.T) {
	t.Parallel()

	info, ok := moduleCatalog["realtime"]
	if !ok {
		t.Fatal("expected realtime in module catalog")
	}
	contract := info.installContract()

	for _, route := range []string{
		"app/router.go (auth)",
		"router.go",
	} {
		if !containsExactString(contract.Routes, route) {
			t.Fatalf("realtime contract routes missing %q: %#v", route, contract.Routes)
		}
	}
	for _, configPath := range []string{
		"config/modules.yaml",
		"app/foundation/container.go",
	} {
		if !containsExactString(contract.Config, configPath) {
			t.Fatalf("realtime contract config missing %q: %#v", configPath, contract.Config)
		}
	}
	for _, runtimeSurface := range []string{
		"container.go",
		"router.go",
		"framework/runtimeplan/features.go",
		"framework/sse/hub.go",
	} {
		if !containsExactString(contract.Jobs, runtimeSurface) {
			t.Fatalf("realtime contract missing runtime/startup surface %q: %#v", runtimeSurface, contract.Jobs)
		}
	}
	for _, testSurface := range []string{
		"startup_contract_test.go",
		"router_guardrails_test.go",
		"router_contract_test.go",
		"framework/runtimeplan/features_test.go",
		"tools/cli/ship/internal/commands/cherie_ci_contract_test.go",
	} {
		if !containsExactString(contract.Tests, testSurface) {
			t.Fatalf("realtime contract missing starter/runtime verification test ownership %q: %#v", testSurface, contract.Tests)
		}
	}
}

func TestPWAModuleCatalog_UsesConcreteWiring(t *testing.T) {
	info, ok := moduleCatalog["pwa"]
	if !ok {
		t.Fatal("expected pwa in module catalog")
	}
	if strings.Contains(info.ContainerSnippet, "TODO") {
		t.Fatalf("pwa container snippet still contains TODO text:\n%s", info.ContainerSnippet)
	}

	publicSnippet := strings.TrimSpace(info.RouterSnippets["public"])
	if publicSnippet == "" {
		t.Fatal("expected pwa public router snippet")
	}
	if strings.Contains(publicSnippet, "TODO") {
		t.Fatalf("pwa public router snippet still contains TODO text:\n%s", publicSnippet)
	}
	for _, token := range []string{
		"RegisterStaticRoutes",
		"pwamodule.NewModule",
		"RegisterRoutes(g)",
		"modules/pwa/routes.go",
	} {
		if !strings.Contains(info.ContainerSnippet+publicSnippet, token) {
			t.Fatalf("pwa wiring missing %q:\ncontainer:\n%s\npublic:\n%s", token, info.ContainerSnippet, publicSnippet)
		}
	}
}

func TestPWAModuleCatalog_InstallContractCoversInstallableAssetsAndBrowserTests(t *testing.T) {
	t.Parallel()

	info, ok := moduleCatalog["pwa"]
	if !ok {
		t.Fatal("expected pwa in module catalog")
	}
	contract := info.installContract()

	for _, route := range []string{
		"app/router.go (public)",
		"modules/pwa/routes.go",
		"framework/web/wiring.go",
	} {
		if !containsExactString(contract.Routes, route) {
			t.Fatalf("pwa contract routes missing %q: %#v", route, contract.Routes)
		}
	}
	for _, configPath := range []string{
		"config/modules.yaml",
		"app/foundation/container.go",
	} {
		if !containsExactString(contract.Config, configPath) {
			t.Fatalf("pwa contract config missing %q: %#v", configPath, contract.Config)
		}
	}
	for _, asset := range []string{
		"modules/pwa/static/manifest.json",
		"modules/pwa/static/service-worker.js",
	} {
		if !containsExactString(contract.Assets, asset) {
			t.Fatalf("pwa contract assets missing %q: %#v", asset, contract.Assets)
		}
	}
	for _, tmpl := range []string{
		"modules/pwa/views/web/pages/gen/install_app_templ.go",
		"modules/pwa/views/web/components/gen/pwa_install_templ.go",
	} {
		if !containsExactString(contract.Templates, tmpl) {
			t.Fatalf("pwa contract templates missing %q: %#v", tmpl, contract.Templates)
		}
	}
	for _, testSurface := range []string{
		"modules/pwa/module_test.go",
		"framework/web/controllers/route_smoke_test.go",
		"tools/cli/ship/internal/commands/doc_sync_contract_test.go",
	} {
		if !containsExactString(contract.Tests, testSurface) {
			t.Fatalf("pwa contract missing installable/browser verification test ownership %q: %#v", testSurface, contract.Tests)
		}
	}
}

func TestApplyModuleAdd_StorageBatteryContract_RedSpec(t *testing.T) {
	root := t.TempDir()
	writeModuleFixtureFiles(t, root)
	if err := os.MkdirAll(filepath.Join(root, "modules", "storage"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "modules", "storage", "go.mod"), []byte("module github.com/leomorpho/goship-modules/storage\n\ngo 1.24.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	info, ok := moduleCatalog["storage"]
	if !ok {
		t.Fatal("expected storage in module catalog")
	}
	if err := applyModuleAdd(root, info, false, io.Discard); err != nil {
		t.Fatalf("applyModuleAdd error: %v", err)
	}

	manifest := readTestFile(t, filepath.Join(root, "config", "modules.yaml"))
	if !strings.Contains(manifest, "- storage") {
		t.Fatalf("expected storage in modules manifest, got:\n%s", manifest)
	}
	goMod := readTestFile(t, filepath.Join(root, "go.mod"))
	if !strings.Contains(goMod, "github.com/leomorpho/goship-modules/storage v0.0.0") {
		t.Fatalf("expected storage require in go.mod, got:\n%s", goMod)
	}
}

func TestApplyModuleRemove_RemovesSafeStorageDependencyContract_RedSpec(t *testing.T) {
	root := t.TempDir()
	writeModuleFixtureFiles(t, root)
	if err := os.MkdirAll(filepath.Join(root, "modules", "storage"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "modules", "storage", "go.mod"), []byte("module github.com/leomorpho/goship-modules/storage\n\ngo 1.24.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	info := moduleCatalog["storage"]
	if err := applyModuleAdd(root, info, false, io.Discard); err != nil {
		t.Fatalf("applyModuleAdd error: %v", err)
	}
	if err := applyModuleRemove(root, info, false, io.Discard); err != nil {
		t.Fatalf("applyModuleRemove error: %v", err)
	}

	goMod := readTestFile(t, filepath.Join(root, "go.mod"))
	if strings.Contains(goMod, "github.com/leomorpho/goship-modules/storage") {
		t.Fatalf("expected storage dependency removed from go.mod, got:\n%s", goMod)
	}
	manifest := readTestFile(t, filepath.Join(root, "config", "modules.yaml"))
	if strings.Contains(manifest, "- storage") {
		t.Fatalf("expected storage removed from modules manifest, got:\n%s", manifest)
	}
}

func TestApplyModuleAdd_JobsAndStorageConcreteSeams_Idempotent(t *testing.T) {
	tests := []struct {
		name    string
		module  string
		setup   func(t *testing.T, root string)
		needles []string
	}{
		{
			name:   "jobs",
			module: "jobs",
			setup: func(t *testing.T, root string) {
				writeModuleFixtureFiles(t, root)
				if err := os.MkdirAll(filepath.Join(root, "modules", "jobs"), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(root, "modules", "jobs", "go.mod"), []byte("module github.com/leomorpho/goship-modules/jobs\n\ngo 1.24.0\n"), 0o644); err != nil {
					t.Fatal(err)
				}
			},
			needles: []string{"framework/bootstrap.WireJobsRuntime", "c.CoreJobs", "c.CoreJobsInspector"},
		},
		{
			name:   "storage",
			module: "storage",
			setup: func(t *testing.T, root string) {
				writeModuleFixtureFiles(t, root)
				if err := os.MkdirAll(filepath.Join(root, "modules", "storage"), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(root, "modules", "storage", "go.mod"), []byte("module github.com/leomorpho/goship-modules/storage\n\ngo 1.24.0\n"), 0o644); err != nil {
					t.Fatal(err)
				}
			},
			needles: []string{"modules/storage.New", "core.BlobStorage"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			tt.setup(t, root)

			info, ok := moduleCatalog[tt.module]
			if !ok {
				t.Fatalf("expected %s in module catalog", tt.module)
			}

			if err := applyModuleAdd(root, info, false, io.Discard); err != nil {
				t.Fatalf("applyModuleAdd error: %v", err)
			}

			containerPath := filepath.Join(root, "app", "foundation", "container.go")
			first := readTestFile(t, containerPath)
			for _, needle := range tt.needles {
				if !strings.Contains(first, needle) {
					t.Fatalf("container missing %q after first apply:\n%s", needle, first)
				}
			}

			if err := applyModuleAdd(root, info, false, io.Discard); err != nil {
				t.Fatalf("applyModuleAdd second pass error: %v", err)
			}

			second := readTestFile(t, containerPath)
			if second != first {
				t.Fatalf("container changed on second apply:\nfirst:\n%s\nsecond:\n%s", first, second)
			}
		})
	}
}

func TestApplyModuleAdd_PaidSubscriptionsAppendsStripeEnvExample(t *testing.T) {
	root := t.TempDir()
	writeModuleFixtureFiles(t, root)
	if err := os.MkdirAll(filepath.Join(root, "modules", "paidsubscriptions"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "modules", "paidsubscriptions", "go.mod"), []byte("module github.com/leomorpho/goship-modules/paidsubscriptions\n\ngo 1.24.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	info, ok := moduleCatalog["paidsubscriptions"]
	if !ok {
		t.Fatal("expected paidsubscriptions in module catalog")
	}
	if err := applyModuleAdd(root, info, false, io.Discard); err != nil {
		t.Fatalf("applyModuleAdd error: %v", err)
	}

	envExamplePath := filepath.Join(root, ".env.example")
	first := readTestFile(t, envExamplePath)
	if !strings.Contains(first, "STRIPE_KEY=") || !strings.Contains(first, "STRIPE_WEBHOOK_SECRET=") {
		t.Fatalf("expected stripe env vars in .env.example, got:\n%s", first)
	}

	if err := applyModuleAdd(root, info, false, io.Discard); err != nil {
		t.Fatalf("applyModuleAdd second pass error: %v", err)
	}
	second := readTestFile(t, envExamplePath)
	if strings.Count(second, "STRIPE_KEY=") != 1 || strings.Count(second, "STRIPE_WEBHOOK_SECRET=") != 1 {
		t.Fatalf("expected stripe env vars exactly once in .env.example, got:\n%s", second)
	}
}

func TestApplyModuleAddRemove_PaidSubscriptionsRestoresEnvExample(t *testing.T) {
	root := t.TempDir()
	writeModuleFixtureFiles(t, root)
	if err := os.MkdirAll(filepath.Join(root, "modules", "paidsubscriptions"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "modules", "paidsubscriptions", "go.mod"), []byte("module github.com/leomorpho/goship-modules/paidsubscriptions\n\ngo 1.24.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	info, ok := moduleCatalog["paidsubscriptions"]
	if !ok {
		t.Fatal("expected paidsubscriptions in module catalog")
	}

	envExamplePath := filepath.Join(root, ".env.example")
	original := readTestFile(t, envExamplePath)

	if err := applyModuleAdd(root, info, false, io.Discard); err != nil {
		t.Fatalf("applyModuleAdd error: %v", err)
	}
	withPaidSubscriptions := readTestFile(t, envExamplePath)
	if !strings.Contains(withPaidSubscriptions, "STRIPE_KEY=") || !strings.Contains(withPaidSubscriptions, "STRIPE_WEBHOOK_SECRET=") {
		t.Fatalf("expected stripe env vars in .env.example after add, got:\n%s", withPaidSubscriptions)
	}

	if err := applyModuleRemove(root, info, false, io.Discard); err != nil {
		t.Fatalf("applyModuleRemove error: %v", err)
	}
	afterRemove := readTestFile(t, envExamplePath)
	if afterRemove != original {
		t.Fatalf(".env.example did not restore after remove:\nwant:\n%s\ngot:\n%s", original, afterRemove)
	}
}

func TestWarnMissingModuleEnv_PaidSubscriptionsWarningsAreNonBlocking(t *testing.T) {
	root := t.TempDir()
	writeModuleFixtureFiles(t, root)

	out := &bytes.Buffer{}
	if err := warnMissingModuleEnv(root, moduleCatalog["paidsubscriptions"], out); err != nil {
		t.Fatalf("warnMissingModuleEnv error: %v", err)
	}
	log := out.String()
	for _, token := range []string{
		`module "paidsubscriptions"`,
		"STRIPE_KEY",
		"STRIPE_WEBHOOK_SECRET",
		"Set these in .env or your shell",
	} {
		if !strings.Contains(log, token) {
			t.Fatalf("missing warning token %q:\n%s", token, log)
		}
	}
}

func TestWarnMissingModuleEnv_UsesDotEnvAndShellValues(t *testing.T) {
	root := t.TempDir()
	writeModuleFixtureFiles(t, root)

	dotEnvPath := filepath.Join(root, ".env")
	if err := os.WriteFile(dotEnvPath, []byte("STRIPE_KEY=sk_test_123\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("STRIPE_WEBHOOK_SECRET", "whsec_123")

	out := &bytes.Buffer{}
	if err := warnMissingModuleEnv(root, moduleCatalog["paidsubscriptions"], out); err != nil {
		t.Fatalf("warnMissingModuleEnv error: %v", err)
	}
	if out.Len() != 0 {
		t.Fatalf("expected no warning output when vars are configured, got:\n%s", out.String())
	}
}

func writeModuleFixtureFiles(t *testing.T, root string) {
	t.Helper()
	files := map[string]string{
		filepath.Join(root, "app", "foundation", "container.go"): `package foundation

func NewContainer() *Container {
	c := &Container{}
	// ship:container:start
	// ship:container:end
	return c
}

func TestCanonicalReferenceBatteryIDs_AreInstallableCatalogEntries(t *testing.T) {
	t.Parallel()

	for _, id := range canonicalReferenceBatteryIDs() {
		info, ok := moduleCatalog[id]
		if !ok {
			t.Fatalf("canonical reference battery %q missing from module catalog", id)
		}
		if strings.TrimSpace(info.ModulePath) == "" {
			t.Fatalf("canonical reference battery %q must declare ModulePath", id)
		}
		if strings.TrimSpace(info.LocalPath) == "" {
			t.Fatalf("canonical reference battery %q must declare LocalPath", id)
		}
		if contract := info.installContract(); contract.IsEmpty() {
			t.Fatalf("canonical reference battery %q must expose a non-empty install contract", id)
		}
	}
}

type Container struct{}
`,
		filepath.Join(root, "app", "router.go"): `package goship

import _ "github.com/leomorpho/goship-modules/notifications"

func registerPublicRoutes() {
	// ship:routes:public:start
	// ship:routes:public:end
}

func registerAuthRoutes() {
	// ship:routes:auth:start
	// ship:routes:auth:end
}

func registerExternalRoutes() {
	// ship:routes:external:start
	// ship:routes:external:end
}
`,
		filepath.Join(root, "config", "modules.yaml"): modulesManifestHeader,
		filepath.Join(root, "go.mod"): `module example.com/demo

go 1.24.0
`,
		filepath.Join(root, "go.work"): `go 1.25.6

use (
	.
)
`,
		filepath.Join(root, ".env.example"): `APP_KEY=
DATABASE_URL=sqlite://tmp/starter.db
CACHE_DRIVER=memory
QUEUE_DRIVER=backlite
`,
		filepath.Join(root, "modules", "notifications", "go.mod"): `module github.com/leomorpho/goship-modules/notifications

go 1.24.0
`,
	}
	for path, content := range files {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func writeNotificationsModuleFixtureFiles(t *testing.T, root string) {
	t.Helper()
	files := map[string]string{
		filepath.Join(root, "app", "foundation", "container.go"): `package foundation

func NewContainer() *Container {
	c := &Container{}
	// ship:container:start
	// ship:container:end
	return c
}

type Container struct{}
`,
		filepath.Join(root, "app", "router.go"): `package goship

func registerPublicRoutes() {
	// ship:routes:public:start
	// ship:routes:public:end
}

func registerAuthRoutes() {
	// ship:routes:auth:start
	// ship:routes:auth:end
}

func registerExternalRoutes() {
	// ship:routes:external:start
	// ship:routes:external:end
}
`,
		filepath.Join(root, "config", "modules.yaml"): modulesManifestHeader,
		filepath.Join(root, "go.mod"): `module example.com/demo

go 1.24.0
`,
		filepath.Join(root, "go.work"): `go 1.25.6

use .
`,
		filepath.Join(root, ".env.example"): `APP_KEY=
DATABASE_URL=sqlite://tmp/starter.db
CACHE_DRIVER=memory
QUEUE_DRIVER=backlite
`,
	}
	for path, content := range files {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func readTestFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func writeLocalModuleGoModFixture(t *testing.T, root, localPath, modulePath string) {
	t.Helper()
	if strings.TrimSpace(localPath) == "" || strings.TrimSpace(modulePath) == "" {
		return
	}
	path := filepath.Join(root, localPath, "go.mod")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	content := "module " + modulePath + "\n\ngo 1.24.0\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func containsExactString(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}

func dedupeStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}
