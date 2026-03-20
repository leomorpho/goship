package commands

const (
	runtimeContractVersion        = "runtime-contract-v1"
	runtimeHandshakeSchemaVersion = "runtime-handshake-v1"
	upgradeReadinessSchemaVersion = "upgrade-readiness-v1"
)

func isSupportedRuntimeContractVersion(version string) bool {
	return version == runtimeContractVersion
}

func isSupportedUpgradeContractVersion(version string) bool {
	return version == upgradeReadinessSchemaVersion
}
