package commands

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestSvelteKitExampleDoesNotClaimHandwrittenContractIsStable(t *testing.T) {
	t.Parallel()

	readme := readRepoFile(t, "examples/sveltekit-api-only/README.md")
	assertNotContains(t, "examples/sveltekit-api-only/README.md", readme, "stable TypeScript-facing contract surface")
	assertContains(t, "examples/sveltekit-api-only/README.md", readme, "generated contract")
}

func TestSvelteKitShimReexportsGeneratedContract(t *testing.T) {
	t.Parallel()

	shim := readRepoFile(t, "examples/sveltekit-api-only/src/lib/server/goship-contract.ts")
	assertContains(t, "examples/sveltekit-api-only/src/lib/server/goship-contract.ts", shim, `../../../generated/goship-contract`)

	generated := readRepoFile(t, "examples/sveltekit-api-only/generated/goship-contract.ts")
	assertContains(t, "examples/sveltekit-api-only/generated/goship-contract.ts", generated, "goshipContractVersion")
	assertContains(t, "examples/sveltekit-api-only/generated/goship-contract.ts", generated, "/api/v1/status")
	assertContains(t, "examples/sveltekit-api-only/generated/goship-contract.ts", generated, "get_api_v1_status")
}

func TestSvelteKitGeneratedContractCanBeRegeneratedDeterministically(t *testing.T) {
	t.Parallel()

	root := repoRootFromCommandsTest(t)
	tmp := filepath.Join(t.TempDir(), "goship-contract.ts")
	cmd := exec.Command("go", "run", "./tools/scripts/generate_sveltekit_contract.go", "--output", tmp)
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("generate_sveltekit_contract failed: %v\n%s", err, out)
	}

	want, err := os.ReadFile(filepath.Join(root, "examples", "sveltekit-api-only", "generated", "goship-contract.ts"))
	if err != nil {
		t.Fatalf("os.ReadFile(want) error = %v", err)
	}
	got, err := os.ReadFile(tmp)
	if err != nil {
		t.Fatalf("os.ReadFile(got) error = %v", err)
	}
	if string(got) != string(want) {
		t.Fatalf("generated contract drifted\nwant:\n%s\n\ngot:\n%s", want, got)
	}
}

func TestFrontendLaneWordingIsConsistent(t *testing.T) {
	t.Parallel()

	readme := readRepoFile(t, "README.md")
	apiGuide := readRepoFile(t, "docs/guides/08-building-an-api.md")
	cliRef := readRepoFile(t, "docs/reference/01-cli.md")
	exampleReadme := readRepoFile(t, "examples/sveltekit-api-only/README.md")

	for _, content := range []string{readme, apiGuide, cliRef, exampleReadme} {
		if !containsAll(content,
			"api-only-same-origin-sveltekit-v1",
			"SvelteKit-first",
		) {
			t.Fatal("frontend lane wording drifted")
		}
	}
	assertContains(t, "examples/sveltekit-api-only/README.md", exampleReadme, "generated contract package")
}

func TestGeneratedContractIncludesSameOriginMetadata(t *testing.T) {
	t.Parallel()

	generated := readRepoFile(t, "examples/sveltekit-api-only/generated/goship-contract.ts")
	assertContains(t, "examples/sveltekit-api-only/generated/goship-contract.ts", generated, `authMode: "same-origin auth/session"`)
	assertContains(t, "examples/sveltekit-api-only/generated/goship-contract.ts", generated, `csrfHeaderName: "X-CSRF-Token"`)
	assertContains(t, "examples/sveltekit-api-only/generated/goship-contract.ts", generated, `cookieMode: "include"`)
}

func containsAll(content string, needles ...string) bool {
	for _, needle := range needles {
		if !strings.Contains(content, needle) {
			return false
		}
	}
	return true
}
