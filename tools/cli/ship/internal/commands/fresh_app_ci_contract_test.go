package commands

import (
	"strings"
	"testing"
)

func TestFreshAppCIScriptUsesRealProofTargets(t *testing.T) {
	t.Parallel()

	script := readRepoFile(t, "tools/scripts/check-fresh-app-ci.sh")
	assertContains(t, "tools/scripts/check-fresh-app-ci.sh", script, "TestFreshApp$")
	assertContains(t, "tools/scripts/check-fresh-app-ci.sh", script, "TestFreshAppStartupSmoke$")
	assertContains(t, "tools/scripts/check-fresh-app-ci.sh", script, "TestFreshAppAPI$")
	assertContains(t, "tools/scripts/check-fresh-app-ci.sh", script, "TestFreshAppAPIStartupSmoke$")
	assertContains(t, "tools/scripts/check-fresh-app-ci.sh", script, "TestFreshAppAdminDashboardRequiresAdmin$")
	assertContains(t, "tools/scripts/check-fresh-app-ci.sh", script, "TestFreshAppAdminDashboardCanManageGeneratedResource$")
	assertContains(t, "tools/scripts/check-fresh-app-ci.sh", script, "TestFreshAppMailerPreviewFlow$")
	assertContains(t, "tools/scripts/check-fresh-app-ci.sh", script, "TestFreshAppSupportedBatteryCombinationStaysBuildable$")
	assertContains(t, "tools/scripts/check-fresh-app-ci.sh", script, "TestFreshApp(StorageModuleEnablesProfileUpload|EmailSubscriptionsModuleEnablesProfileToggle|PaidSubscriptionsModuleEnablesProfileToggle|NotificationsModuleEnablesHomeFeedInbox)$")
	assertContains(t, "tools/scripts/check-fresh-app-ci.sh", script, "TestRuntimeReportIncludesContractVersionsAndModuleAdoption$")
	assertContains(t, "tools/scripts/check-fresh-app-ci.sh", script, "no tests to run")
	assertContains(t, "tools/scripts/check-fresh-app-ci.sh", script, "no test files")
	assertNotContains(t, "tools/scripts/check-fresh-app-ci.sh", script, "./framework/http/controllers")
}

func TestFreshAppGuideMatchesRealCILane(t *testing.T) {
	t.Parallel()

	guide := readRepoFile(t, "docs/guides/02-development-workflows.md")
	assertContains(t, "docs/guides/02-development-workflows.md", guide, "TestFreshApp")
	assertContains(t, "docs/guides/02-development-workflows.md", guide, "TestFreshAppStartupSmoke")
	assertNotContains(t, "docs/guides/02-development-workflows.md", guide, "go test ./app/...")
}

func TestGettingStartedUsesFreshCloneBuildInstallPath(t *testing.T) {
	t.Parallel()

	guide := readRepoFile(t, "docs/guides/01-getting-started.md")
	assertContains(t, "docs/guides/01-getting-started.md", guide, "git clone https://github.com/leomorpho/goship.git")
	assertContains(t, "docs/guides/01-getting-started.md", guide, "go build -o ./bin/ship ./tools/cli/ship/cmd/ship")
	assertNotContains(t, "docs/guides/01-getting-started.md", guide, "tools/cli/ship/v2/cmd/ship@v2.0.5")
}

func TestDescribeModuleAdoptionUsesManifestForGeneratedApps(t *testing.T) {
	t.Parallel()

	shipbin := buildShipBinary(t)
	appPath := scaffoldFreshAppViaShip(t, shipbin, false)
	runCmd(t, appPath, shipbin, "module:add", "jobs")
	runCmd(t, appPath, shipbin, "module:add", "storage")
	runCmd(t, appPath, shipbin, "module:add", "emailsubscriptions")

	describe := runCmd(t, appPath, shipbin, "describe", "--pretty")
	for _, id := range []string{"jobs", "storage", "emailsubscriptions"} {
		if !strings.Contains(describe, `"id": "`+id+`"`) || !strings.Contains(describe, `"installed": true`) {
			t.Fatalf("describe output missing installed %s adoption\n%s", id, describe)
		}
	}

	runtimeReport := runCmd(t, appPath, shipbin, "runtime:report", "--json")
	for _, id := range []string{"jobs", "storage", "emailsubscriptions"} {
		if !strings.Contains(runtimeReport, `"id": "`+id+`"`) || !strings.Contains(runtimeReport, `"installed": true`) {
			t.Fatalf("runtime report missing installed %s adoption\n%s", id, runtimeReport)
		}
	}
}
