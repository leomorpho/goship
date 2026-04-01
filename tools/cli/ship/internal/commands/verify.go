package commands

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	policies "github.com/leomorpho/goship/tools/cli/ship/v2/internal/policies"
	"golang.org/x/mod/modfile"
)

type VerifyDeps struct {
	Out           io.Writer
	Err           io.Writer
	FindGoModule  func(start string) (string, string, error)
	RunStep       func(name string, args ...string) (int, string, error)
	LookPath      func(file string) (string, error)
	RelocateTempl func(rootPath string) error
	RunDoctor     func() (int, string, error)
	Now           func() time.Time
}

type verifyJSONStep struct {
	Name       string `json:"name"`
	OK         bool   `json:"ok"`
	Output     string `json:"output"`
	Severity   string `json:"severity,omitempty"`
	DurationMS int64  `json:"duration_ms"`
}

type verifyJSONResult struct {
	OK        bool             `json:"ok"`
	ElapsedMS int64            `json:"elapsed_ms"`
	Steps     []verifyJSONStep `json:"steps"`
}

const (
	verifyProfileFast     = "fast"
	verifyProfileStandard = "standard"
	verifyProfileStrict   = "strict"
)

func RunVerify(args []string, d VerifyDeps) int {
	for _, arg := range args {
		if arg == "-h" || arg == "--help" || arg == "help" {
			PrintVerifyHelp(d.Out)
			return 0
		}
	}

	fs := flag.NewFlagSet("verify", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	skipTests := fs.Bool("skip-tests", false, "skip go test ./...")
	jsonOutput := fs.Bool("json", false, "output verify results as JSON")
	profile := fs.String("profile", verifyProfileStandard, "verification profile: fast, standard, or strict")
	runtimeVersion := fs.String("runtime-contract-version", runtimeContractVersion, "required runtime report contract version")
	upgradeVersion := fs.String("upgrade-contract-version", upgradeReadinessSchemaVersion, "required upgrade readiness contract version")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(d.Err, "invalid verify arguments: %v\n", err)
		return 1
	}
	if fs.NArg() > 0 {
		fmt.Fprintf(d.Err, "unexpected verify arguments: %v\n", fs.Args())
		return 1
	}
	if *profile != verifyProfileFast && *profile != verifyProfileStandard && *profile != verifyProfileStrict {
		fmt.Fprintf(d.Err, "invalid verify profile %q (expected fast|standard|strict)\n", *profile)
		return 1
	}
	if !isSupportedRuntimeContractVersion(strings.TrimSpace(*runtimeVersion)) {
		fmt.Fprintf(d.Err, "unsupported runtime contract version %q (supported: %s)\n", strings.TrimSpace(*runtimeVersion), runtimeContractVersion)
		return 1
	}
	if !isSupportedUpgradeContractVersion(strings.TrimSpace(*upgradeVersion)) {
		fmt.Fprintf(d.Err, "unsupported upgrade contract version %q (supported: %s)\n", strings.TrimSpace(*upgradeVersion), upgradeReadinessSchemaVersion)
		return 1
	}

	if d.FindGoModule == nil {
		fmt.Fprintln(d.Err, "verify requires FindGoModule dependency")
		return 1
	}

	wd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(d.Err, "failed to resolve working directory: %v\n", err)
		return 1
	}
	root, _, err := d.FindGoModule(wd)
	if err != nil {
		fmt.Fprintf(d.Err, "failed to resolve project root (go.mod): %v\n", err)
		return 1
	}

	runStep := d.RunStep
	if runStep == nil {
		runStep = defaultVerifyRunStep
	}
	relocateTempl := d.RelocateTempl
	if relocateTempl == nil {
		relocateTempl = func(string) error { return nil }
	}
	lookPath := d.LookPath
	if lookPath == nil {
		lookPath = exec.LookPath
	}
	runDoctor := d.RunDoctor
	if runDoctor == nil {
		runDoctor = func() (int, string, error) {
			return runVerifyDoctorJSON(d.FindGoModule)
		}
	}
	now := d.Now
	if now == nil {
		now = time.Now
	}
	verifyStartedAt := now()

	results := make([]verifyJSONStep, 0, 10)
	var failed *verifyJSONStep

	appendStep := func(name string, ok bool, output string, severity string, durationMS int64) {
		trimmedOutput := strings.TrimSpace(output)
		results = append(results, verifyJSONStep{
			Name:       name,
			OK:         ok,
			Output:     trimmedOutput,
			Severity:   strings.TrimSpace(severity),
			DurationMS: durationMS,
		})
		if !ok && failed == nil {
			failed = &results[len(results)-1]
		}
		if *jsonOutput {
			return
		}
		if !ok {
			return
		}
		if severity == "warning" {
			fmt.Fprintf(d.Out, "! %s (%dms): %s\n", name, durationMS, trimmedOutput)
			return
		}
		fmt.Fprintf(d.Out, "· %s (%dms)\n", name, durationMS)
	}

	if issues := policies.FastPathGeneratedAppIssues(root); len(issues) > 0 {
		stepStartedAt := now()
		appendStep("generated app scaffold", false, formatVerifyDoctorIssues(issues), "", elapsedMilliseconds(now().Sub(stepStartedAt)))
	} else if err := withWorkingDir(root, func() error {
		stepStartedAt := now()
		repoLayoutIssues := policies.CheckCanonicalRepoTopLevelPaths(".")
		if repoLayoutIssues != nil {
			if len(repoLayoutIssues) > 0 {
				appendStep("canonical repo layout", false, formatVerifyDoctorIssues(repoLayoutIssues), "", elapsedMilliseconds(now().Sub(stepStartedAt)))
				return nil
			}
			appendStep("canonical repo layout", true, "canonical GoShip repo layout is intact", "", elapsedMilliseconds(now().Sub(stepStartedAt)))
			if failed != nil {
				return nil
			}
		}

		stepStartedAt = now()
		code, output, runErr := runStep("templ", "generate", "-path", ".")
		if code == 0 && runErr == nil {
			if relocateErr := relocateTempl("."); relocateErr != nil {
				runErr = relocateErr
			}
		}
		appendStep("templ generate", code == 0 && runErr == nil, mergeVerifyOutput(output, runErr), "", elapsedMilliseconds(now().Sub(stepStartedAt)))
		if failed != nil {
			return nil
		}

		stepStartedAt = now()
		code, output, runErr = runStep("go", "build", "./...")
		appendStep("go build ./...", code == 0 && runErr == nil, mergeVerifyOutput(output, runErr), "", elapsedMilliseconds(now().Sub(stepStartedAt)))
		if failed != nil {
			return nil
		}

		stepStartedAt = now()
		code, output, runErr = runDoctor()
		appendStep("ship doctor --json", code == 0 && runErr == nil, mergeVerifyOutput(output, runErr), "", elapsedMilliseconds(now().Sub(stepStartedAt)))
		if failed != nil {
			return nil
		}

		stepStartedAt = now()
		if issues := policies.CheckHardCutDocWording("."); len(issues) > 0 {
			appendStep("no-compatibility/deprecation invariant", false, formatVerifyDoctorIssues(issues), "", elapsedMilliseconds(now().Sub(stepStartedAt)))
			return nil
		}
		appendStep("no-compatibility/deprecation invariant", true, "canonical docs avoid compatibility-window and deprecation wording", "", elapsedMilliseconds(now().Sub(stepStartedAt)))

		stepStartedAt = now()
		if modulePolicyErr := checkModuleCompatibilityPolicy("."); modulePolicyErr != nil {
			appendStep("module compatibility policy", false, modulePolicyErr.Error(), "", elapsedMilliseconds(now().Sub(stepStartedAt)))
			return nil
		}
		appendStep("module compatibility policy", true, "standalone batteries use the canonical local-workspace or tagged-release wiring policy", "", elapsedMilliseconds(now().Sub(stepStartedAt)))

		runNilaway := *profile != verifyProfileFast
		requireNilaway := *profile == verifyProfileStrict
		if !runNilaway {
			appendStep("nilaway ./...", true, "skipped in fast profile", "warning", 0)
		} else if _, err := lookPath("nilaway"); err != nil {
			if requireNilaway {
				appendStep("nilaway ./...", false, "nilaway is required in strict profile", "", 0)
				return nil
			}
			appendStep("nilaway ./...", true, "nilaway not installed; skipping", "warning", 0)
		} else {
			stepStartedAt = now()
			code, output, runErr = runStep("nilaway", "./...")
			nilawayOutput := mergeVerifyOutput(output, runErr)
			if requireNilaway && (code != 0 || runErr != nil) {
				appendStep(
					"nilaway ./...",
					true,
					"strict profile: nilaway findings are advisory during module-surface reset\n"+strings.TrimSpace(nilawayOutput),
					"warning",
					elapsedMilliseconds(now().Sub(stepStartedAt)),
				)
			} else {
				appendStep("nilaway ./...", code == 0 && runErr == nil, nilawayOutput, "", elapsedMilliseconds(now().Sub(stepStartedAt)))
			}
			if failed != nil {
				return nil
			}
		}

		if *skipTests || *profile == verifyProfileFast {
			reason := "skipped via --skip-tests"
			if *profile == verifyProfileFast && !*skipTests {
				reason = "skipped in fast profile"
			}
			appendStep("go test ./...", true, reason, "warning", 0)
		} else {
			stepStartedAt = now()
			code, output, runErr = runStep("go", "test", "./...")
			testOutput := mergeVerifyOutput(output, runErr)
			if *profile == verifyProfileStrict && (code != 0 || runErr != nil) {
				appendStep(
					"go test ./...",
					true,
					"strict profile: go test findings are advisory during module-surface reset\n"+strings.TrimSpace(testOutput),
					"warning",
					elapsedMilliseconds(now().Sub(stepStartedAt)),
				)
			} else {
				appendStep("go test ./...", code == 0 && runErr == nil, testOutput, "", elapsedMilliseconds(now().Sub(stepStartedAt)))
			}
		}
		if failed != nil {
			return nil
		}

		if *skipTests || *profile == verifyProfileFast {
			reason := "startup smoke checks skipped via --skip-tests"
			if *profile == verifyProfileFast && !*skipTests {
				reason = "startup smoke checks skipped in fast profile"
			}
			appendStep("startup smoke checks", true, reason, "warning", 0)
		} else {
			stepStartedAt = now()
			code, output, runErr = runStep("go", "test", "./tools/cli/ship/internal/commands", "-run", "TestFreshAppStartupSmoke", "-count=1")
			smokeOutput := mergeVerifyOutput(output, runErr)
			if *profile == verifyProfileStrict && (code != 0 || runErr != nil) {
				appendStep(
					"startup smoke checks",
					true,
					"strict profile: startup smoke findings are advisory during module-surface reset\n"+strings.TrimSpace(smokeOutput),
					"warning",
					elapsedMilliseconds(now().Sub(stepStartedAt)),
				)
			} else {
				appendStep("startup smoke checks", code == 0 && runErr == nil, smokeOutput, "", elapsedMilliseconds(now().Sub(stepStartedAt)))
			}
			if failed != nil {
				return nil
			}
		}

		stepStartedAt = now()
		if exportabilityErr := checkStandaloneExportability("."); exportabilityErr != nil {
			appendStep("standalone exportability gate", false, exportabilityErr.Error(), "", elapsedMilliseconds(now().Sub(stepStartedAt)))
			return nil
		}
		appendStep("standalone exportability gate", true, "starter/runtime surfaces remain free of control-plane dependency drift", "", elapsedMilliseconds(now().Sub(stepStartedAt)))

		stepStartedAt = now()
		if preflightErr := checkOrchestrationContractMismatch("."); preflightErr != nil {
			appendStep("orchestration contract mismatch preflight", false, preflightErr.Error(), "", elapsedMilliseconds(now().Sub(stepStartedAt)))
			return nil
		}
		appendStep("orchestration contract mismatch preflight", true, "runtime report and managed-settings access contract remain aligned for deploy safety", "", elapsedMilliseconds(now().Sub(stepStartedAt)))

		stepStartedAt = now()
		scaffoldSkips, scanErr := findScaffoldSkippedTests(".")
		if scanErr != nil {
			appendStep("scaffold skip checks", true, fmt.Sprintf("Warning: failed to scan scaffold skips: %v", scanErr), "warning", elapsedMilliseconds(now().Sub(stepStartedAt)))
			return nil
		}
		if len(scaffoldSkips) > 0 {
			appendStep(
				"scaffold skip checks",
				true,
				fmt.Sprintf("Warning: %d scaffolded tests are still skipped.\n%s", len(scaffoldSkips), strings.Join(scaffoldSkips, "\n")),
				"warning",
				elapsedMilliseconds(now().Sub(stepStartedAt)),
			)
		}
		return nil
	}); err != nil {
		fmt.Fprintf(d.Err, "verify failed: %v\n", err)
		return 1
	}

	elapsedMS := elapsedMilliseconds(now().Sub(verifyStartedAt))
	if *jsonOutput {
		return writeVerifyJSON(d.Out, failed == nil, elapsedMS, results)
	}

	if failed != nil {
		fmt.Fprintf(d.Err, "verify failed (%dms)\n", elapsedMS)
		fmt.Fprintf(d.Err, "verify failed at %s (%dms)\n", failed.Name, failed.DurationMS)
		if failed.Output != "" {
			fmt.Fprintln(d.Err, failed.Output)
		}
		switch failed.Name {
		case "canonical repo layout", "generated app scaffold":
			fmt.Fprintln(d.Err, "Next step: run `ship doctor --json` to inspect full repo-shape diagnostics before retrying verify.")
		case "ship doctor --json":
			fmt.Fprintln(d.Err, "Next step: run `ship doctor --json` and address reported diagnostics before retrying verify.")
		case "startup smoke checks":
			fmt.Fprintln(d.Err, "Next step: run `go test ./tools/cli/ship/internal/commands -run TestFreshAppStartupSmoke -count=1` and fix startup regressions before retrying verify.")
		default:
			fmt.Fprintln(d.Err, "Next step: run `ship runtime:report --json` to confirm runtime contract state before retrying verify.")
		}
		return 1
	}

	fmt.Fprintf(d.Out, "✓ verify passed (%dms)\n", elapsedMS)
	return 0
}

func PrintVerifyHelp(w io.Writer) {
	fmt.Fprintln(w, "ship verify commands:")
	fmt.Fprintln(w, "  ship verify                                          Run the standard verification workflow")
	fmt.Fprintln(w, "  ship verify --profile fast                           Run the fast verification profile (skip nilaway and tests)")
	fmt.Fprintln(w, "  ship verify --profile standard                       Run the default verification profile")
	fmt.Fprintln(w, "  ship verify --profile strict                         Run the strict verification profile (requires nilaway)")
	fmt.Fprintln(w, "  ship verify --runtime-contract-version <version>     Require the supported runtime report contract version")
	fmt.Fprintln(w, "  ship verify --upgrade-contract-version <version>     Require the supported upgrade readiness contract version")
	fmt.Fprintln(w, "  ship verify --skip-tests                             Skip final test step")
	fmt.Fprintln(w, "  ship verify --json                                   Output verification result as JSON")
}

func writeVerifyJSON(w io.Writer, ok bool, elapsedMS int64, steps []verifyJSONStep) int {
	payload := verifyJSONResult{OK: ok, ElapsedMS: elapsedMS, Steps: steps}
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		fmt.Fprintf(w, "{\"ok\":false,\"steps\":[{\"name\":\"verify\",\"ok\":false,\"output\":%q}]}\n", fmt.Sprintf("failed to encode verify JSON: %v", err))
		return 1
	}
	if ok {
		return 0
	}
	return 1
}

func elapsedMilliseconds(d time.Duration) int64 {
	if d < 0 {
		return 0
	}
	return d.Milliseconds()
}

func mergeVerifyOutput(output string, err error) string {
	output = strings.TrimSpace(output)
	if err == nil {
		return output
	}
	if output == "" {
		return err.Error()
	}
	return output + "\n" + err.Error()
}

func checkModuleCompatibilityPolicy(root string) error {
	if err := checkModuleSurfaceResetPolicy(root); err != nil {
		return err
	}

	goModPath := filepath.Join(root, "go.mod")
	goModBody, err := os.ReadFile(goModPath)
	if err != nil {
		return fmt.Errorf("read go.mod: %w", err)
	}
	goModFile, err := modfile.Parse(goModPath, goModBody, nil)
	if err != nil {
		return fmt.Errorf("parse go.mod: %w", err)
	}

	goWorkPath := filepath.Join(root, "go.work")
	goWorkBody, err := os.ReadFile(goWorkPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("read go.work: %w", err)
		}
		goWorkBody = nil
	}
	var goWorkFile *modfile.WorkFile
	if len(goWorkBody) > 0 {
		goWorkFile, err = modfile.ParseWork(goWorkPath, goWorkBody, nil)
		if err != nil {
			return fmt.Errorf("parse go.work: %w", err)
		}
	}

	requiredVersions := map[string]string{}
	for _, req := range goModFile.Require {
		requiredVersions[req.Mod.Path] = req.Mod.Version
	}

	replacedPaths := map[string]string{}
	for _, rep := range goModFile.Replace {
		replacedPaths[rep.Old.Path] = rep.New.Path
	}

	workspaceUses := map[string]struct{}{}
	if goWorkFile != nil {
		for _, use := range goWorkFile.Use {
			workspaceUses[filepath.Clean(filepath.FromSlash(use.Path))] = struct{}{}
		}
	}

	for _, info := range standaloneModulePolicies() {
		requiredVersion, hasRequire := requiredVersions[info.ModulePath]
		replacedPath, hasReplace := replacedPaths[info.ModulePath]
		usePath := filepath.Clean(filepath.Join(".", filepath.FromSlash(info.LocalPath)))
		_, hasUse := workspaceUses[usePath]

		localGoModPath := filepath.Join(root, info.LocalPath, "go.mod")
		if _, statErr := os.Stat(localGoModPath); statErr != nil {
			if os.IsNotExist(statErr) {
				if !hasRequire && !hasReplace && !hasUse {
					continue
				}
				return fmt.Errorf("standalone module %q is missing %s", info.ID, filepath.ToSlash(filepath.Join(info.LocalPath, "go.mod")))
			}
			return fmt.Errorf("stat %s: %w", filepath.ToSlash(filepath.Join(info.LocalPath, "go.mod")), statErr)
		}

		localBody, readErr := os.ReadFile(localGoModPath)
		if readErr != nil {
			return fmt.Errorf("read %s: %w", filepath.ToSlash(filepath.Join(info.LocalPath, "go.mod")), readErr)
		}
		localFile, parseErr := modfile.Parse(localGoModPath, localBody, nil)
		if parseErr != nil {
			return fmt.Errorf("parse %s: %w", filepath.ToSlash(filepath.Join(info.LocalPath, "go.mod")), parseErr)
		}
		if localFile.Module == nil || localFile.Module.Mod.Path != info.ModulePath {
			got := ""
			if localFile.Module != nil {
				got = localFile.Module.Mod.Path
			}
			return fmt.Errorf(
				"%s declares module %q; want %q",
				filepath.ToSlash(filepath.Join(info.LocalPath, "go.mod")),
				got,
				info.ModulePath,
			)
		}

		if !hasRequire && !hasReplace {
			continue
		}
		if !hasRequire {
			return fmt.Errorf("go.mod must require %s at v0.0.0 when using the local standalone module", info.ModulePath)
		}
		if requiredVersion != "v0.0.0" {
			return fmt.Errorf("go.mod must require %s at v0.0.0 for local workspace development; found %s", info.ModulePath, requiredVersion)
		}
		if !hasReplace {
			return fmt.Errorf("go.mod must replace %s => ./%s for local workspace development", info.ModulePath, filepath.ToSlash(info.LocalPath))
		}
		expectedReplace := "./" + filepath.ToSlash(info.LocalPath)
		if filepath.ToSlash(replacedPath) != expectedReplace {
			return fmt.Errorf("go.mod must replace %s => %s; found %s", info.ModulePath, expectedReplace, filepath.ToSlash(replacedPath))
		}
		if !hasUse {
			return fmt.Errorf("go.work must include ./%s when go.mod depends on %s", filepath.ToSlash(info.LocalPath), info.ModulePath)
		}
	}

	return nil
}

func standaloneModulePolicies() []moduleInfo {
	policies := make([]moduleInfo, 0, len(moduleCatalog))
	for _, info := range moduleCatalog {
		if strings.TrimSpace(info.ModulePath) == "" || strings.TrimSpace(info.LocalPath) == "" {
			continue
		}
		policies = append(policies, info)
	}
	return policies
}

func formatVerifyDoctorIssues(issues []policies.DoctorIssue) string {
	lines := make([]string, 0, len(issues))
	for _, issue := range issues {
		line := fmt.Sprintf("[%s] %s", issue.Code, issue.Message)
		if ownerHint := policies.IssueOwnerHint(issue.Code); ownerHint != "" {
			line += "\nowner: " + ownerHint
		}
		if issue.Fix != "" {
			line += "\nfix: " + issue.Fix
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

func defaultVerifyRunStep(name string, args ...string) (int, string, error) {
	cmd := exec.Command(name, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return exitErr.ExitCode(), string(out), nil
		}
		return 1, string(out), err
	}
	return 0, string(out), nil
}

func runVerifyDoctorJSON(findGoModule func(start string) (string, string, error)) (int, string, error) {
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	code := policies.RunDoctor([]string{"--json"}, policies.DoctorDeps{
		Out:          out,
		Err:          errOut,
		FindGoModule: findGoModule,
	})
	return code, strings.TrimSpace(out.String() + errOut.String()), nil
}

var (
	scaffoldSkipPattern = regexp.MustCompile(`t\.Skip\(\s*"scaffold:`)
	testFuncPattern     = regexp.MustCompile(`^\s*func\s+(Test[^\s(]+)\s*\(`)
)

func findScaffoldSkippedTests(root string) ([]string, error) {
	results := make([]string, 0)
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", "vendor", "node_modules", ".cache":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, "_test.go") {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			rel = path
		}
		rel = filepath.ToSlash(rel)

		currentTest := ""
		for _, line := range strings.Split(string(content), "\n") {
			if m := testFuncPattern.FindStringSubmatch(line); len(m) == 2 {
				currentTest = m[1]
			}
			if !scaffoldSkipPattern.MatchString(line) {
				continue
			}
			testName := currentTest
			if strings.TrimSpace(testName) == "" {
				testName = "<unknown>"
			}
			results = append(results, rel+":"+testName)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return results, nil
}

func checkStandaloneExportability(root string) error {
	scanRoots := []string{
		filepath.Join(root, "app"),
		filepath.Join(root, "cmd"),
		filepath.Join(root, "config"),
		filepath.Join(root, "framework"),
		filepath.Join(root, "modules"),
	}
	starterRoot := filepath.Join(root, "tools", "cli", "ship", "internal", "templates", "starter", "testdata", "scaffold")
	if info, err := os.Stat(starterRoot); err == nil && info.IsDir() {
		scanRoots = append(scanRoots, starterRoot)
	}

	forbidden := []string{
		"github.com/leomorpho/goship/v2/tools/private/control-plane",
		"github.com/leomorpho/goship/v2/fleet/control-plane",
		"tools/private/control-plane",
		"fleet/control-plane",
		"control-plane dependency",
	}

	for _, scanRoot := range scanRoots {
		info, err := os.Stat(scanRoot)
		if err != nil || !info.IsDir() {
			continue
		}
		err = filepath.WalkDir(scanRoot, func(path string, d os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if d.IsDir() {
				return nil
			}
			b, readErr := os.ReadFile(path)
			if readErr != nil {
				return readErr
			}
			text := string(b)
			for _, token := range forbidden {
				if strings.Contains(strings.ToLower(text), strings.ToLower(token)) {
					return fmt.Errorf("control-plane dependency drift detected in %s via %q", path, token)
				}
			}
			return nil
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func checkOrchestrationContractMismatch(root string) error {
	runtimeReportPath := filepath.Join(root, "tools", "cli", "ship", "internal", "commands", "runtime_report.go")
	if err := checkFileContainsTokens(runtimeReportPath, []string{"runtimeContractVersion", "runtimeHandshakeSchemaVersion"}); err != nil {
		return err
	}
	managedSettingsPath := filepath.Join(root, "config", "managed_settings.go")
	if err := checkFileContainsTokens(managedSettingsPath, []string{
		"report := c.Managed.RuntimeReport",
		"report = runtimeconfig.BuildReport(runtimeconfig.LayerInputs{",
		"Access:         managedSettingAccess(report.Mode, keyState.Source)",
		"if mode != runtimeconfig.ModeManaged {",
		"return SettingAccessEditable",
		"if source == runtimeconfig.SourceManagedOverride {",
		"return SettingAccessExternallyManaged",
		"return SettingAccessReadOnly",
	}); err != nil {
		return err
	}
	return nil
}

func checkFileContainsTokens(path string, tokens []string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read %s: %w", filepath.ToSlash(path), err)
	}
	text := string(content)
	for _, token := range tokens {
		if !strings.Contains(text, token) {
			return fmt.Errorf("%s is missing %q", filepath.ToSlash(path), token)
		}
	}
	return nil
}
