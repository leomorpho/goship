package security

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestManagedHookVerifier_SharedReplayStoreContract_RedSpec(t *testing.T) {
	root := repoRootForSecurityContractTest(t)

	securitySource := mustReadSecurityContractText(t, filepath.Join(root, "framework", "security", "managed_hooks.go"))
	managedDoc := mustReadSecurityContractText(t, filepath.Join(root, "docs", "architecture", "09-standalone-and-managed-mode.md"))
	risksDoc := mustReadSecurityContractText(t, filepath.Join(root, "docs", "architecture", "06-known-gaps-and-risks.md"))

	for _, token := range []string{"NonceStore", "WithNonceStore", "consumeNonce("} {
		if !strings.Contains(securitySource, token) {
			t.Fatalf("managed hook verifier should expose shared replay seam token %q", token)
		}
	}
	if !strings.Contains(managedDoc, "shared/distributed replay store") {
		t.Fatal("managed-mode architecture doc should describe shared/distributed replay store as the managed hook replay contract")
	}
	if strings.Contains(risksDoc, "nonce cache is currently in-memory per process") {
		t.Fatal("known risks doc should stop describing managed hook replay protection as process-local only once the shared-store contract is canonical")
	}
}

func repoRootForSecurityContractTest(t *testing.T) string {
	t.Helper()

	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "Makefile")); err == nil {
			if _, err := os.Stat(filepath.Join(dir, ".docket")); err == nil {
				return dir
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find repo root")
		}
		dir = parent
	}
}

func mustReadSecurityContractText(t *testing.T, path string) string {
	t.Helper()

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(content)
}
