package commands

import (
	"encoding/json"
	"os/exec"
	"strings"
	"testing"

	frameworkapi "github.com/leomorpho/goship/framework/api"
	"github.com/leomorpho/goship/framework/testutil"
)

func TestAPIGuideMatchesFrameworkAPIPackage(t *testing.T) {
	t.Parallel()

	guide := readRepoFile(t, "docs/guides/08-building-an-api.md")
	assertContains(t, "docs/guides/08-building-an-api.md", guide, "framework/api")
	assertContains(t, "docs/guides/08-building-an-api.md", guide, "api.OK")
	assertContains(t, "docs/guides/08-building-an-api.md", guide, "api.Fail")
	assertContains(t, "docs/guides/08-building-an-api.md", guide, "api.IsAPIRequest")

	_ = frameworkapi.NotFound
	_ = frameworkapi.Unauthorized
	_ = frameworkapi.Validation
	_ = frameworkapi.OK
	_ = frameworkapi.Fail
	_ = frameworkapi.IsAPIRequest
}

func TestAPIV1StatusRouteReturnsCanonicalEnvelope(t *testing.T) {
	server := testutil.NewTestServer(t)
	resp := server.Get("/api/v1/status")
	resp.AssertStatus(200)

	var payload map[string]any
	resp.AssertJSON(&payload)
	if _, ok := payload["data"]; !ok {
		t.Fatalf("payload missing data: %v", payload)
	}
}

func TestRoutesJSONIncludesAPIV1Status(t *testing.T) {
	root := repoRootFromCommandsTest(t)
	cmd := exec.Command("go", "run", "./tools/cli/ship/cmd/ship", "routes", "--json")
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("routes --json failed: %v\n%s", err, out)
	}

	if !strings.Contains(string(out), `/api/v1/status`) {
		t.Fatalf("routes output missing /api/v1/status\n%s", out)
	}

	var rows []map[string]any
	if err := json.Unmarshal(out, &rows); err != nil {
		t.Fatalf("json.Unmarshal() error = %v\n%s", err, out)
	}
}

func TestRoutesJSONIncludesEndpointMetadataForAPIV1Status(t *testing.T) {
	root := repoRootFromCommandsTest(t)
	cmd := exec.Command("go", "run", "./tools/cli/ship/cmd/ship", "routes", "--json")
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("routes --json failed: %v\n%s", err, out)
	}

	var rows []map[string]any
	if err := json.Unmarshal(out, &rows); err != nil {
		t.Fatalf("json.Unmarshal() error = %v\n%s", err, out)
	}

	for _, row := range rows {
		if row["path"] == "/api/v1/status" {
			if row["operation_id"] != "get_api_v1_status" {
				t.Fatalf("operation_id = %#v", row["operation_id"])
			}
			if row["response_contract"] != "api.status.v1" {
				t.Fatalf("response_contract = %#v", row["response_contract"])
			}
			return
		}
	}
	t.Fatal("missing /api/v1/status route")
}

func TestBackendContractGuideIsCanonicalSource(t *testing.T) {
	t.Parallel()

	guide := readRepoFile(t, "docs/guides/08-building-an-api.md")
	readme := readRepoFile(t, "README.md")
	cliRef := readRepoFile(t, "docs/reference/01-cli.md")

	assertContains(t, "docs/guides/08-building-an-api.md", guide, "canonical backend contract")
	assertContains(t, "README.md", readme, "docs/guides/08-building-an-api.md")
	assertContains(t, "docs/reference/01-cli.md", cliRef, "canonical backend contract document")
}

func TestRuntimeReportIncludesContractVersionsAndModuleAdoption(t *testing.T) {
	t.Parallel()

	shipbin := buildShipBinary(t)
	appPath := scaffoldFreshAppViaShip(t, shipbin, false)
	runCmd(t, appPath, shipbin, "module:add", "jobs")

	report := runCmd(t, appPath, shipbin, "runtime:report", "--json")
	assertContains(t, "runtime report", report, `"contract_version": "runtime-contract-v1"`)
	assertContains(t, "runtime report", report, `"schema_version": "runtime-handshake-v1"`)
	assertContains(t, "runtime report", report, `"id": "jobs"`)
	assertContains(t, "runtime report", report, `"installed": true`)
}
