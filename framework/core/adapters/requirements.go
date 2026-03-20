package adapters

import "github.com/leomorpho/goship/config"

// RequirementsFromConfig derives capability requirements from runtime/process topology.
func RequirementsFromConfig(cfg *config.Config) Requirements {
	if cfg == nil {
		return Requirements{}
	}

	req := Requirements{}

	if cfg.Processes.Scheduler {
		req.Jobs.Cron = true
	}

	if cfg.Runtime.Profile == config.RuntimeProfileDistributed {
		req.Jobs.Retries = true
		req.Jobs.Delayed = true
	}

	return req
}
