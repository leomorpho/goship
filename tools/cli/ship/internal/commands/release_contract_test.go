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
