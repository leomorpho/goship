package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDeployContractVersionPolicy_RedSpec(t *testing.T) {
	t.Run("accepts supported runtime contract versions", func(t *testing.T) {
		result := EvaluateDeployContractVersionPolicy("runtime-contract-v1", "runtime-handshake-v1")
		if !result.OK {
			t.Fatalf("expected supported deploy contract versions to pass, got %+v", result)
		}
		if len(result.Blockers) != 0 {
			t.Fatalf("expected no blockers for supported deploy contract versions, got %+v", result.Blockers)
		}
	})

	t.Run("rejects unsupported runtime contract version with stable diagnostic", func(t *testing.T) {
		result := EvaluateDeployContractVersionPolicy("runtime-contract-v2", "runtime-handshake-v1")
		if result.OK {
			t.Fatalf("expected unsupported runtime contract version to fail, got %+v", result)
		}
		if len(result.Blockers) != 1 {
			t.Fatalf("expected one blocker, got %+v", result.Blockers)
		}
		if got := result.Blockers[0].ID; got != "unsupported_runtime_contract_version" {
			t.Fatalf("blocker id=%q want unsupported_runtime_contract_version", got)
		}
		if got := result.Blockers[0].Expected; got != "runtime-contract-v1" {
			t.Fatalf("expected version=%q want runtime-contract-v1", got)
		}
		if got := result.Blockers[0].Actual; got != "runtime-contract-v2" {
			t.Fatalf("actual version=%q want runtime-contract-v2", got)
		}
	})

	t.Run("rejects unsupported runtime handshake version with stable diagnostic", func(t *testing.T) {
		result := EvaluateDeployContractVersionPolicy("runtime-contract-v1", "runtime-handshake-v2")
		if result.OK {
			t.Fatalf("expected unsupported runtime handshake version to fail, got %+v", result)
		}
		if len(result.Blockers) != 1 {
			t.Fatalf("expected one blocker, got %+v", result.Blockers)
		}
		if got := result.Blockers[0].ID; got != "unsupported_runtime_handshake_version" {
			t.Fatalf("blocker id=%q want unsupported_runtime_handshake_version", got)
		}
		if got := result.Blockers[0].Expected; got != "runtime-handshake-v1" {
			t.Fatalf("expected version=%q want runtime-handshake-v1", got)
		}
		if got := result.Blockers[0].Actual; got != "runtime-handshake-v2" {
			t.Fatalf("actual version=%q want runtime-handshake-v2", got)
		}
	})
}

func TestUpgradeContractVersionPolicy_RedSpec(t *testing.T) {
	result := EvaluateUpgradeContractVersionPolicy("upgrade-readiness-v2")
	if result.OK {
		t.Fatalf("expected unsupported upgrade contract version to fail, got %+v", result)
	}
	if len(result.Blockers) != 1 {
		t.Fatalf("expected one blocker, got %+v", result.Blockers)
	}
	if got := result.Blockers[0].ID; got != "unsupported_upgrade_readiness_version" {
		t.Fatalf("blocker id=%q want unsupported_upgrade_readiness_version", got)
	}
	if got := result.Blockers[0].Expected; got != "upgrade-readiness-v1" {
		t.Fatalf("expected version=%q want upgrade-readiness-v1", got)
	}
	if got := result.Blockers[0].Actual; got != "upgrade-readiness-v2" {
		t.Fatalf("actual version=%q want upgrade-readiness-v2", got)
	}
}

func TestManagedHookKeyVersionPolicy_RedSpec(t *testing.T) {
	result := EvaluateManagedHookKeyVersionPolicy("managed-hook-key-version-v2")
	if result.OK {
		t.Fatalf("expected unsupported managed hook key-version contract to fail, got %+v", result)
	}
	if len(result.Blockers) != 1 {
		t.Fatalf("expected one blocker, got %+v", result.Blockers)
	}
	if got := result.Blockers[0].ID; got != "unsupported_managed_hook_key_version" {
		t.Fatalf("blocker id=%q want unsupported_managed_hook_key_version", got)
	}
	if got := result.Blockers[0].Expected; got != "managed-hook-key-version-v1" {
		t.Fatalf("expected=%q want managed-hook-key-version-v1", got)
	}
	if got := result.Blockers[0].Actual; got != "managed-hook-key-version-v2" {
		t.Fatalf("actual=%q want managed-hook-key-version-v2", got)
	}
}

func TestContractVersionPolicyDocs_RedSpec(t *testing.T) {
	root := repoRootFromCommandsTest(t)

	roadmap, err := os.ReadFile(filepath.Join(root, "docs", "roadmap", "01-framework-plan.md"))
	if err != nil {
		t.Fatal(err)
	}
	risks, err := os.ReadFile(filepath.Join(root, "docs", "architecture", "06-known-gaps-and-risks.md"))
	if err != nil {
		t.Fatal(err)
	}

	roadmapText := string(roadmap)
	for _, token := range []string{
		"runtime-contract-v1",
		"runtime-handshake-v1",
		"upgrade-readiness-v1",
		"unsupported_runtime_contract_version",
		"unsupported_runtime_handshake_version",
		"unsupported_upgrade_readiness_version",
		"ship runtime:report --json",
		"ship upgrade --json",
		"ship verify",
	} {
		if !strings.Contains(roadmapText, token) {
			t.Fatalf("framework roadmap should include contract version policy token %q", token)
		}
	}

	risksText := string(risks)
	for _, token := range []string{
		"contract-version policy",
		"unsupported_runtime_contract_version",
		"unsupported_runtime_handshake_version",
		"unsupported_upgrade_readiness_version",
		"blocking mismatch",
		"remediation",
	} {
		if !strings.Contains(risksText, token) {
			t.Fatalf("known gaps doc should include contract version policy token %q", token)
		}
	}
}
