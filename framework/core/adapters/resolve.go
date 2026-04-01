package adapters

import (
	"fmt"

	"github.com/leomorpho/goship/v2/config"
	"github.com/leomorpho/goship/v2/framework/core"
)

// Resolved contains normalized adapter information derived from config.
type Resolved struct {
	Selection        Selection
	Requirements     Requirements
	JobsCapabilities core.JobCapabilities
}

// ResolveFromConfig validates adapter selection and capability requirements.
func ResolveFromConfig(cfg *config.Config) (Resolved, error) {
	if cfg == nil {
		return Resolved{}, fmt.Errorf("nil config")
	}

	reg := NewDefaultRegistry()
	sel := Selection{
		DB:     cfg.Adapters.DB,
		Cache:  cfg.Adapters.Cache,
		Jobs:   cfg.Adapters.Jobs,
		PubSub: cfg.Adapters.PubSub,
	}
	if err := reg.ValidateSelection(sel); err != nil {
		return Resolved{}, err
	}
	if err := validateSelectionDependencies(sel); err != nil {
		return Resolved{}, err
	}

	req := RequirementsFromConfig(cfg)
	if err := reg.ValidateRequirements(sel, req); err != nil {
		return Resolved{}, err
	}

	cap, ok := reg.JobsCapabilities(sel.Jobs)
	if !ok {
		return Resolved{}, fmt.Errorf("unknown jobs adapter %q", sel.Jobs)
	}

	return Resolved{
		Selection:        sel,
		Requirements:     req,
		JobsCapabilities: cap,
	}, nil
}

func validateSelectionDependencies(sel Selection) error {
	if sel.PubSub == "redis" && sel.Cache != "redis" {
		return fmt.Errorf("pubsub adapter %q requires cache adapter %q", sel.PubSub, "redis")
	}
	return nil
}
