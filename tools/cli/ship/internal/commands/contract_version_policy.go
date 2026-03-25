package commands

const (
	SupportedRuntimeContractVersion    = "runtime-contract-v1"
	SupportedRuntimeHandshakeVersion   = "runtime-handshake-v1"
	SupportedUpgradeReadinessVersion   = "upgrade-readiness-v1"
	SupportedManagedHookKeyVersion     = "managed-hook-key-version-v1"
	BlockerUnsupportedRuntimeContract  = "unsupported_runtime_contract_version"
	BlockerUnsupportedRuntimeHandshake = "unsupported_runtime_handshake_version"
	BlockerUnsupportedUpgradeReadiness = "unsupported_upgrade_readiness_version"
	BlockerUnsupportedManagedHookKey   = "unsupported_managed_hook_key_version"
)

type ContractVersionPolicyResult struct {
	OK       bool
	Blockers []ContractVersionBlocker
}

type ContractVersionBlocker struct {
	ID          string
	Expected    string
	Actual      string
	Remediation string
}

// These helpers pin the exact version-policy contract for deploy/upgrade orchestration.
// Wiring can reuse them later without redefining blocker IDs or accepted versions.
func EvaluateDeployContractVersionPolicy(runtimeContractVersion, runtimeHandshakeVersion string) ContractVersionPolicyResult {
	result := ContractVersionPolicyResult{OK: true}

	if runtimeContractVersion != SupportedRuntimeContractVersion {
		result.OK = false
		result.Blockers = append(result.Blockers, ContractVersionBlocker{
			ID:          BlockerUnsupportedRuntimeContract,
			Expected:    SupportedRuntimeContractVersion,
			Actual:      runtimeContractVersion,
			Remediation: "Re-run ship runtime:report --json from a supported runtime build before deploy orchestration proceeds.",
		})
	}

	if runtimeHandshakeVersion != SupportedRuntimeHandshakeVersion {
		result.OK = false
		result.Blockers = append(result.Blockers, ContractVersionBlocker{
			ID:          BlockerUnsupportedRuntimeHandshake,
			Expected:    SupportedRuntimeHandshakeVersion,
			Actual:      runtimeHandshakeVersion,
			Remediation: "Refresh the runtime handshake payload via ship runtime:report --json so deploy orchestration uses a supported contract version.",
		})
	}

	return result
}

func EvaluateUpgradeContractVersionPolicy(upgradeReadinessVersion string) ContractVersionPolicyResult {
	result := ContractVersionPolicyResult{OK: true}
	if upgradeReadinessVersion == SupportedUpgradeReadinessVersion {
		return result
	}

	result.OK = false
	result.Blockers = append(result.Blockers, ContractVersionBlocker{
		ID:          BlockerUnsupportedUpgradeReadiness,
		Expected:    SupportedUpgradeReadinessVersion,
		Actual:      upgradeReadinessVersion,
		Remediation: "Re-run ship upgrade --json from a supported CLI build before upgrade orchestration proceeds.",
	})
	return result
}

func EvaluateManagedHookKeyVersionPolicy(keyVersion string) ContractVersionPolicyResult {
	result := ContractVersionPolicyResult{OK: true}
	if keyVersion == SupportedManagedHookKeyVersion {
		return result
	}
	result.OK = false
	result.Blockers = append(result.Blockers, ContractVersionBlocker{
		ID:          BlockerUnsupportedManagedHookKey,
		Expected:    SupportedManagedHookKeyVersion,
		Actual:      keyVersion,
		Remediation: "Refresh managed hook caller configuration so requests use the supported key-version contract.",
	})
	return result
}
