package config

func normalizedDefaultConfigForReporting() Config {
	return normalizedConfigForReporting(defaultConfig())
}

func normalizedConfigForReporting(cfg Config) Config {
	snapshot := cfg
	applyDatabaseDriverConfig(&snapshot)
	applyBackupDefaults(&snapshot)
	applyRuntimeDefaults(&snapshot)
	applyProcessesProfileIfUnset(&snapshot, hasAnyProcessSelection(snapshot.Processes))
	return snapshot
}

func hasAnyProcessSelection(processes ProcessesConfig) bool {
	return processes.Web || processes.Worker || processes.Scheduler || processes.CoLocated
}
