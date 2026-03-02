package runtimeplan

import (
	"fmt"

	"github.com/mikestefanello/pagoda/config"
)

// Plan describes the resolved runtime/process topology without mutating startup behavior.
type Plan struct {
	Profile      string
	RunWeb       bool
	RunWorker    bool
	RunScheduler bool
	CoLocated    bool
	Adapters     Adapters
}

// Adapters contains selected backend implementations for major pluggable capabilities.
type Adapters struct {
	DB     string
	Cache  string
	Jobs   string
	PubSub string
}

// Resolve builds a normalized runtime plan from config and validates obvious topology mistakes.
func Resolve(cfg *config.Config) (Plan, error) {
	if cfg == nil {
		return Plan{}, fmt.Errorf("nil config")
	}

	profile := string(cfg.Runtime.Profile)
	switch cfg.Runtime.Profile {
	case "", config.RuntimeProfileServerDB:
		profile = string(config.RuntimeProfileServerDB)
	case config.RuntimeProfileSingleNode, config.RuntimeProfileDistributed:
	default:
		return Plan{}, fmt.Errorf("unknown runtime profile: %s", cfg.Runtime.Profile)
	}

	p := Plan{
		Profile:      profile,
		RunWeb:       cfg.Processes.Web,
		RunWorker:    cfg.Processes.Worker,
		RunScheduler: cfg.Processes.Scheduler,
		CoLocated:    cfg.Processes.CoLocated,
		Adapters: Adapters{
			DB:     cfg.Adapters.DB,
			Cache:  cfg.Adapters.Cache,
			Jobs:   cfg.Adapters.Jobs,
			PubSub: cfg.Adapters.PubSub,
		},
	}

	if !p.RunWeb && !p.RunWorker && !p.RunScheduler {
		return Plan{}, fmt.Errorf("invalid processes: at least one of web/worker/scheduler must be enabled")
	}

	// Capability guardrails for current scaffold.
	if cfg.Runtime.Profile == config.RuntimeProfileDistributed && p.Adapters.Jobs == "inproc" {
		return Plan{}, fmt.Errorf("invalid distributed jobs backend: inproc")
	}

	return p, nil
}
