package runtimeplan

import (
	"fmt"

	"github.com/leomorpho/goship/config"
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
	if cfg.Runtime.Profile != "" && normalizeProfile(cfg) == string(cfg.Runtime.Profile) &&
		cfg.Runtime.Profile != config.RuntimeProfileServerDB &&
		cfg.Runtime.Profile != config.RuntimeProfileSingleNode &&
		cfg.Runtime.Profile != config.RuntimeProfileDistributed {
		return Plan{}, fmt.Errorf("unknown runtime profile: %s", cfg.Runtime.Profile)
	}

	p := Plan{
		Profile:      normalizeProfile(cfg),
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
	if p.Profile == string(config.RuntimeProfileDistributed) && p.Adapters.Jobs == "inproc" {
		return Plan{}, fmt.Errorf("invalid distributed jobs backend: inproc")
	}

	return p, nil
}

func normalizeProfile(cfg *config.Config) string {
	switch cfg.Runtime.Profile {
	case config.RuntimeProfileServerDB:
		return string(config.RuntimeProfileServerDB)
	case config.RuntimeProfileSingleNode:
		return string(config.RuntimeProfileSingleNode)
	case config.RuntimeProfileDistributed:
		return string(config.RuntimeProfileDistributed)
	case "":
		if looksLikeSingleNodeDefault(cfg) {
			return string(config.RuntimeProfileSingleNode)
		}
		return string(config.RuntimeProfileServerDB)
	default:
		return string(cfg.Runtime.Profile)
	}
}

func looksLikeSingleNodeDefault(cfg *config.Config) bool {
	if cfg == nil {
		return false
	}

	return cfg.Processes.Web &&
		cfg.Processes.Worker &&
		cfg.Processes.Scheduler &&
		cfg.Processes.CoLocated &&
		cfg.Adapters.DB == "sqlite" &&
		cfg.Adapters.Cache == "otter" &&
		cfg.Adapters.Jobs == "backlite" &&
		cfg.Adapters.PubSub == "inproc"
}
