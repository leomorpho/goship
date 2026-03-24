package commands

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
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
	} {
		if !strings.Contains(strings.ToLower(out.String()), token) {
			t.Fatalf("dry-run output missing %q:\n%s", token, out.String())
		}
	}
}

func TestApplyModuleAdd_TwoFactor(t *testing.T) {
	root := t.TempDir()

	containerPath := filepath.Join(root, "app", "foundation", "container.go")
	if err := os.MkdirAll(filepath.Dir(containerPath), 0o755); err != nil {
		t.Fatal(err)
	}
	containerContent := `package foundation

func NewContainer() *Container {
	c := &Container{}
	// ship:container:start
	// ship:container:end
	return c
}

type Container struct{}
`
	if err := os.WriteFile(containerPath, []byte(containerContent), 0o644); err != nil {
		t.Fatal(err)
	}

	routerPath := filepath.Join(root, "app", "router.go")
	if err := os.MkdirAll(filepath.Dir(routerPath), 0o755); err != nil {
		t.Fatal(err)
	}
	routerContent := `package goship

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
`
	if err := os.WriteFile(routerPath, []byte(routerContent), 0o644); err != nil {
		t.Fatal(err)
	}

	manifestPath := filepath.Join(root, "config", "modules.yaml")
	if err := os.MkdirAll(filepath.Dir(manifestPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manifestPath, []byte(modulesManifestHeader), 0o644); err != nil {
		t.Fatal(err)
	}

	info, ok := moduleCatalog["2fa"]
	if !ok {
		t.Fatal("expected 2fa in module catalog")
	}
	if err := applyModuleAdd(root, info, false, io.Discard); err != nil {
		t.Fatalf("applyModuleAdd error: %v", err)
	}

	manifest, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(manifest), "- 2fa") {
		t.Fatalf("expected 2fa in modules manifest, got:\n%s", string(manifest))
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

func TestAdminModuleCatalog_UsesConcreteWiring(t *testing.T) {
	info, ok := moduleCatalog["admin"]
	if !ok {
		t.Fatal("expected admin in module catalog")
	}

	snippet := strings.TrimSpace(info.RouterSnippets["auth"])
	if snippet == "" {
		t.Fatal("expected admin auth router snippet")
	}
	if !strings.Contains(snippet, "adminmodule.New(adminmodule.ModuleDeps{") {
		t.Fatalf("expected concrete admin module constructor, got:\n%s", snippet)
	}
	if !strings.Contains(snippet, "RegisterRoutes(onboardedGroup)") {
		t.Fatalf("expected concrete admin route registration, got:\n%s", snippet)
	}
	if strings.Contains(snippet, "TODO") {
		t.Fatalf("admin router snippet still uses TODO placeholder: %q", snippet)
	}
}

func TestApplyModuleAdd_AdminIdempotent(t *testing.T) {
	root := t.TempDir()

	containerPath := filepath.Join(root, "app", "foundation", "container.go")
	if err := os.MkdirAll(filepath.Dir(containerPath), 0o755); err != nil {
		t.Fatal(err)
	}
	containerContent := `package foundation

func NewContainer() *Container {
	c := &Container{}
	// ship:container:start
	// ship:container:end
	return c
}

type Container struct{}
`
	if err := os.WriteFile(containerPath, []byte(containerContent), 0o644); err != nil {
		t.Fatal(err)
	}

	routerPath := filepath.Join(root, "app", "router.go")
	if err := os.MkdirAll(filepath.Dir(routerPath), 0o755); err != nil {
		t.Fatal(err)
	}
	routerContent := `package goship

import (
	adminmodule "github.com/leomorpho/goship/modules/admin"
)

func registerAuthRoutes() {
	// ship:routes:auth:start
	// ship:routes:auth:end

	_ = adminmodule.ModuleDeps{}
}
`
	if err := os.WriteFile(routerPath, []byte(routerContent), 0o644); err != nil {
		t.Fatal(err)
	}

	manifestPath := filepath.Join(root, "config", "modules.yaml")
	if err := os.MkdirAll(filepath.Dir(manifestPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manifestPath, []byte(modulesManifestHeader), 0o644); err != nil {
		t.Fatal(err)
	}

	info, ok := moduleCatalog["admin"]
	if !ok {
		t.Fatal("expected admin in module catalog")
	}
	if err := applyModuleAdd(root, info, false, io.Discard); err != nil {
		t.Fatalf("applyModuleAdd error: %v", err)
	}

	firstRouter := readTestFile(t, routerPath)
	if !strings.Contains(firstRouter, "adminmodule.New(adminmodule.ModuleDeps{") {
		t.Fatalf("expected admin router wiring, got:\n%s", firstRouter)
	}
	if !strings.Contains(firstRouter, "RegisterRoutes(onboardedGroup)") {
		t.Fatalf("expected admin route registration, got:\n%s", firstRouter)
	}

	if err := applyModuleAdd(root, info, false, io.Discard); err != nil {
		t.Fatalf("applyModuleAdd second pass error: %v", err)
	}
	secondRouter := readTestFile(t, routerPath)
	if secondRouter != firstRouter {
		t.Fatalf("admin router changed on reapply:\nfirst:\n%s\nsecond:\n%s", firstRouter, secondRouter)
	}

	if err := applyModuleRemove(root, info, false, io.Discard); err != nil {
		t.Fatalf("applyModuleRemove error: %v", err)
	}
	removedRouter := readTestFile(t, routerPath)
	if strings.Contains(removedRouter, "adminmodule.New(adminmodule.ModuleDeps{") {
		t.Fatalf("expected admin router wiring removed, got:\n%s", removedRouter)
	}
	if strings.Contains(removedRouter, "RegisterRoutes(onboardedGroup)") {
		t.Fatalf("expected admin route registration removed, got:\n%s", removedRouter)
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

func TestApplyModuleAdd_BillingAppendsStripeEnvExample(t *testing.T) {
	root := t.TempDir()
	writeModuleFixtureFiles(t, root)
	if err := os.MkdirAll(filepath.Join(root, "modules", "paidsubscriptions"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "modules", "paidsubscriptions", "go.mod"), []byte("module github.com/leomorpho/goship-modules/paidsubscriptions\n\ngo 1.24.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	info, ok := moduleCatalog["billing"]
	if !ok {
		t.Fatal("expected billing in module catalog")
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

func TestApplyModuleAddRemove_BillingRestoresEnvExample(t *testing.T) {
	root := t.TempDir()
	writeModuleFixtureFiles(t, root)
	if err := os.MkdirAll(filepath.Join(root, "modules", "paidsubscriptions"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "modules", "paidsubscriptions", "go.mod"), []byte("module github.com/leomorpho/goship-modules/paidsubscriptions\n\ngo 1.24.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	info, ok := moduleCatalog["billing"]
	if !ok {
		t.Fatal("expected billing in module catalog")
	}

	envExamplePath := filepath.Join(root, ".env.example")
	original := readTestFile(t, envExamplePath)

	if err := applyModuleAdd(root, info, false, io.Discard); err != nil {
		t.Fatalf("applyModuleAdd error: %v", err)
	}
	withBilling := readTestFile(t, envExamplePath)
	if !strings.Contains(withBilling, "STRIPE_KEY=") || !strings.Contains(withBilling, "STRIPE_WEBHOOK_SECRET=") {
		t.Fatalf("expected stripe env vars in .env.example after add, got:\n%s", withBilling)
	}

	if err := applyModuleRemove(root, info, false, io.Discard); err != nil {
		t.Fatalf("applyModuleRemove error: %v", err)
	}
	afterRemove := readTestFile(t, envExamplePath)
	if afterRemove != original {
		t.Fatalf(".env.example did not restore after remove:\nwant:\n%s\ngot:\n%s", original, afterRemove)
	}
}

func TestWarnMissingModuleEnv_BillingWarningsAreNonBlocking(t *testing.T) {
	root := t.TempDir()
	writeModuleFixtureFiles(t, root)

	out := &bytes.Buffer{}
	if err := warnMissingModuleEnv(root, moduleCatalog["billing"], out); err != nil {
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
	if err := warnMissingModuleEnv(root, moduleCatalog["billing"], out); err != nil {
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
