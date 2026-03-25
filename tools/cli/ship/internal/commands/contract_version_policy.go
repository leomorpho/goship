package commands

const (
	SupportedRuntimeContractVersion    = "runtime-contract-v1"
	SupportedRuntimeHandshakeVersion   = "runtime-handshake-v1"
	SupportedUpgradeReadinessVersion   = "upgrade-readiness-v1"
	SupportedManagedHookKeyVersion     = "managed-hook-key-version-v1"
	LegacyRuntimeContractVersion       = "runtime-contract-v0"
	LegacyRuntimeHandshakeVersion      = "runtime-handshake-v0"
	BlockerUnsupportedRuntimeContract  = "unsupported_runtime_contract_version"
	BlockerUnsupportedRuntimeHandshake = "unsupported_runtime_handshake_version"
	BlockerUnsupportedRuntimePair      = "unsupported_runtime_contract_pair"
	BlockerUnsupportedUpgradeReadiness = "unsupported_upgrade_readiness_version"
	BlockerUnsupportedManagedHookKey   = "unsupported_managed_hook_key_version"
	ContractViolationSecurityEvent     = "contract_violation"
)

type ContractVersionPolicyResult struct {
	OK             bool
	Blockers       []ContractVersionBlocker
	SecurityEvents []ContractVersionSecurityEvent
}

type ContractVersionBlocker struct {
	ID          string
	Expected    string
	Actual      string
	Remediation string
}

type ContractVersionSecurityEvent struct {
	Kind string
	Code string
}

// These helpers pin the exact version-policy contract for deploy/upgrade orchestration.
// Wiring can reuse them later without redefining blocker IDs or accepted versions.
func EvaluateDeployContractVersionPolicy(runtimeContractVersion, runtimeHandshakeVersion string) ContractVersionPolicyResult {
	if _, _, ok := negotiateRuntimeContractPair(runtimeContractVersion, runtimeHandshakeVersion); ok {
		return ContractVersionPolicyResult{OK: true}
	}

	result := ContractVersionPolicyResult{OK: true}

	if isSupportedRuntimeContract(runtimeContractVersion) && isSupportedRuntimeHandshake(runtimeHandshakeVersion) {
		result.OK = false
		result.Blockers = append(result.Blockers, ContractVersionBlocker{
			ID:          BlockerUnsupportedRuntimePair,
			Expected:    SupportedRuntimeContractVersion + " + " + SupportedRuntimeHandshakeVersion + " or " + LegacyRuntimeContractVersion + " + " + LegacyRuntimeHandshakeVersion,
			Actual:      runtimeContractVersion + " + " + runtimeHandshakeVersion,
			Remediation: "Use a supported runtime-contract/runtime-handshake pair from ship runtime:report --json before deploy orchestration proceeds.",
		})
		result.SecurityEvents = append(result.SecurityEvents, ContractVersionSecurityEvent{
			Kind: ContractViolationSecurityEvent,
			Code: BlockerUnsupportedRuntimePair,
		})
		return result
	}

	if runtimeContractVersion != SupportedRuntimeContractVersion {
		result.OK = false
		result.Blockers = append(result.Blockers, ContractVersionBlocker{
			ID:          BlockerUnsupportedRuntimeContract,
			Expected:    SupportedRuntimeContractVersion,
			Actual:      runtimeContractVersion,
			Remediation: "Re-run ship runtime:report --json from a supported runtime build before deploy orchestration proceeds.",
		})
		result.SecurityEvents = append(result.SecurityEvents, ContractVersionSecurityEvent{
			Kind: ContractViolationSecurityEvent,
			Code: BlockerUnsupportedRuntimeContract,
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
		result.SecurityEvents = append(result.SecurityEvents, ContractVersionSecurityEvent{
			Kind: ContractViolationSecurityEvent,
			Code: BlockerUnsupportedRuntimeHandshake,
		})
	}

	return result
}

func negotiateRuntimeContractPair(runtimeContractVersion, runtimeHandshakeVersion string) (string, string, bool) {
	pairs := map[string]string{
		SupportedRuntimeContractVersion: SupportedRuntimeHandshakeVersion,
		LegacyRuntimeContractVersion:    LegacyRuntimeHandshakeVersion,
	}
	if handshake, ok := pairs[runtimeContractVersion]; ok && handshake == runtimeHandshakeVersion {
		return runtimeContractVersion, runtimeHandshakeVersion, true
	}
	return "", "", false
}

func isSupportedRuntimeContract(version string) bool {
	return version == SupportedRuntimeContractVersion || version == LegacyRuntimeContractVersion
}

func isSupportedRuntimeHandshake(version string) bool {
	return version == SupportedRuntimeHandshakeVersion || version == LegacyRuntimeHandshakeVersion
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
	result.SecurityEvents = append(result.SecurityEvents, ContractVersionSecurityEvent{
		Kind: ContractViolationSecurityEvent,
		Code: BlockerUnsupportedUpgradeReadiness,
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
	result.SecurityEvents = append(result.SecurityEvents, ContractVersionSecurityEvent{
		Kind: ContractViolationSecurityEvent,
		Code: BlockerUnsupportedManagedHookKey,
	})
	return result
}
