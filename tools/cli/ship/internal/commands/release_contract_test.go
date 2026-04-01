package commands

import "testing"

func TestBetaReadinessChecklistUsesRealCurrentProofCommands(t *testing.T) {
	t.Parallel()

	checklist := readRepoFile(t, "docs/beta-readiness.md")
	for _, needle := range []string{
		"TestFreshApp -count=1",
		"TestStarterJobsModuleRoundTripStaysBuildable -count=1",
		"TestFreshAppAuthFlow -count=1",
		"TestFreshAppNoInfraDefaultPath -count=1",
		"TestGettingStartedUsesFreshCloneBuildInstallPath -count=1",
	} {
		assertContains(t, "docs/beta-readiness.md", checklist, needle)
	}
	assertNotContains(t, "docs/beta-readiness.md", checklist, "npx playwright test tests/auth_golden_flow.spec.ts")
	assertNotContains(t, "docs/beta-readiness.md", checklist, "go test ./modules/jobs ./framework/repos/cache -count=1")
}

func TestWorkflowAndDocsExposeFrontendAsTopLevelReleaseSurface(t *testing.T) {
	t.Parallel()

	workflow := readRepoFile(t, ".github/workflows/test.yml")
	assertContains(t, ".github/workflows/test.yml", workflow, "top_level_frontend")
	assertContains(t, ".github/workflows/test.yml", workflow, "needs: [split_frontend_contract]")

	guide := readRepoFile(t, "docs/guides/02-development-workflows.md")
	assertContains(t, "docs/guides/02-development-workflows.md", guide, "top_level_frontend")
	assertContains(t, "docs/guides/02-development-workflows.md", guide, "split_frontend_contract")
}

func TestReleaseProofTargetExists(t *testing.T) {
	t.Parallel()

	makefile := readRepoFile(t, "Makefile")
	assertContains(t, "Makefile", makefile, "test-release-proof")

	script := readRepoFile(t, "tools/scripts/check-release-proof.sh")
	for _, needle := range []string{
		"TestFreshApp$",
		"TestFreshAppStartupSmoke$",
		"TestFreshAppNoInfraDefaultPath$",
		"TestFreshAppAuthFlow$",
		"TestFreshAppAPI$",
		"TestFreshAppAPIStartupSmoke$",
	} {
		assertContains(t, "tools/scripts/check-release-proof.sh", script, needle)
	}
}

func TestGettingStartedProofTargetExists(t *testing.T) {
	t.Parallel()

	makefile := readRepoFile(t, "Makefile")
	assertContains(t, "Makefile", makefile, "test-getting-started")

	script := readRepoFile(t, "tools/scripts/check-getting-started.sh")
	assertContains(t, "tools/scripts/check-getting-started.sh", script, "new myapp --module example.com/myapp --no-i18n")
	assertContains(t, "tools/scripts/check-getting-started.sh", script, "db:migrate")
	assertContains(t, "tools/scripts/check-getting-started.sh", script, "test >/dev/null")
	assertContains(t, "tools/scripts/check-getting-started.sh", script, "verify --profile fast")
}

func TestPublishedInstallContractMatchesOnboarding(t *testing.T) {
	t.Parallel()

	gettingStarted := readRepoFile(t, "docs/guides/01-getting-started.md")
	assertContains(t, "docs/guides/01-getting-started.md", gettingStarted, "go build -o ./bin/ship ./tools/cli/ship/cmd/ship")
	assertNotContains(t, "docs/guides/01-getting-started.md", gettingStarted, "tools/cli/ship/v2/cmd/ship@v2.0.5")

	readme := readRepoFile(t, "README.md")
	assertNotContains(t, "README.md", readme, "tools/cli/ship/v2/cmd/ship@v2.0.5")
}

func TestBootstrapBudgetTargetIsDocumentedAndWired(t *testing.T) {
	t.Parallel()

	makefile := readRepoFile(t, "Makefile")
	assertContains(t, "Makefile", makefile, "test-bootstrap-budget")

	script := readRepoFile(t, "tools/scripts/check-bootstrap-budget.sh")
	assertContains(t, "tools/scripts/check-bootstrap-budget.sh", script, "ship new")
	assertContains(t, "tools/scripts/check-bootstrap-budget.sh", script, "db:migrate")
	assertContains(t, "tools/scripts/check-bootstrap-budget.sh", script, "/health/readiness")

	guide := readRepoFile(t, "docs/guides/02-development-workflows.md")
	assertContains(t, "docs/guides/02-development-workflows.md", guide, "make test-bootstrap-budget")
}

func TestGoldenSuiteDocsDistinguishV1FromLegacyAlphaSurface(t *testing.T) {
	t.Parallel()

	guide := readRepoFile(t, "docs/guides/02-development-workflows.md")
	assertContains(t, "docs/guides/02-development-workflows.md", guide, "test:golden")
	assertContains(t, "docs/guides/02-development-workflows.md", guide, "legacy")
	assertContains(t, "docs/guides/02-development-workflows.md", guide, "not the canonical v1 release-proof lane")

	cliRef := readRepoFile(t, "docs/reference/01-cli.md")
	assertContains(t, "docs/reference/01-cli.md", cliRef, "legacy CI lane")
	assertContains(t, "docs/reference/01-cli.md", cliRef, "not as the primary v1 release-proof surface")
}

func TestKamalGuideMatchesDistributedDeploymentTopology(t *testing.T) {
	t.Parallel()

	guide := readRepoFile(t, "docs/guides/04-deployment-kamal.md")
	assertContains(t, "docs/guides/04-deployment-kamal.md", guide, "separate worker host")
	assertContains(t, "docs/guides/04-deployment-kamal.md", guide, "external Postgres")
	assertContains(t, "docs/guides/04-deployment-kamal.md", guide, "external Redis")
	assertContains(t, "docs/guides/04-deployment-kamal.md", guide, "verify --profile fast")
	assertContains(t, "docs/guides/04-deployment-kamal.md", guide, "does **not** document the single-binary local path")
}
