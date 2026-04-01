package adapters

import (
	"fmt"
	"sort"
	"strings"

	"github.com/leomorpho/goship/v2/framework/core"
)

// Selection describes configured adapter names for runtime seams.
type Selection struct {
	DB     string
	Cache  string
	Jobs   string
	PubSub string
}

// Requirements describes capabilities required by runtime/process topology.
type Requirements struct {
	Jobs core.JobCapabilities
}

// Registry stores known adapter names and capability metadata.
type Registry struct {
	allowed map[string]map[string]struct{}
	jobsCap map[string]core.JobCapabilities
}

// NewDefaultRegistry returns the canonical adapter registry for current GoShip runtime.
func NewDefaultRegistry() Registry {
	return Registry{
		allowed: map[string]map[string]struct{}{
			"db": {
				"postgres": {},
				"mysql":    {},
				"sqlite":   {},
			},
			"cache": {
				"memory": {},
				"otter":  {},
				"redis":  {},
			},
			"jobs": {
				"inproc":   {},
				"dbqueue":  {},
				"asynq":    {},
				"backlite": {},
			},
			"pubsub": {
				"inproc": {},
				"redis":  {},
				"nats":   {},
			},
		},
		jobsCap: map[string]core.JobCapabilities{
			"inproc": {
				Delayed: true,
				Retries: true,
				Cron:    true,
			},
			"dbqueue": {
				Delayed:    true,
				Retries:    true,
				Cron:       true,
				DeadLetter: true,
			},
			"asynq": {
				Delayed:    true,
				Retries:    true,
				Cron:       true,
				Priority:   true,
				DeadLetter: true,
				Dashboard:  true,
			},
			"backlite": {
				Delayed:    true,
				Retries:    true,
				DeadLetter: true,
			},
		},
	}
}

// ValidateSelection ensures configured adapters are known.
func (r Registry) ValidateSelection(sel Selection) error {
	checks := []struct {
		kind string
		name string
	}{
		{kind: "db", name: sel.DB},
		{kind: "cache", name: sel.Cache},
		{kind: "jobs", name: sel.Jobs},
		{kind: "pubsub", name: sel.PubSub},
	}
	for _, c := range checks {
		if err := r.validateOne(c.kind, c.name); err != nil {
			return err
		}
	}
	return nil
}

// ValidateRequirements ensures selected adapters satisfy runtime capability needs.
func (r Registry) ValidateRequirements(sel Selection, req Requirements) error {
	cap, ok := r.JobsCapabilities(sel.Jobs)
	if !ok {
		return fmt.Errorf("unknown jobs adapter %q", sel.Jobs)
	}
	if err := core.ValidateJobCapabilities(req.Jobs, cap); err != nil {
		return fmt.Errorf("jobs adapter %q: %w", sel.Jobs, err)
	}
	return nil
}

// JobsCapabilities returns the capability metadata for a jobs adapter.
func (r Registry) JobsCapabilities(name string) (core.JobCapabilities, bool) {
	cap, ok := r.jobsCap[name]
	return cap, ok
}

func (r Registry) validateOne(kind, name string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("missing %s adapter name", kind)
	}
	allowed, ok := r.allowed[kind]
	if !ok {
		return fmt.Errorf("unknown adapter kind %q", kind)
	}
	if _, ok := allowed[name]; ok {
		return nil
	}
	return fmt.Errorf("unknown %s adapter %q (allowed: %s)", kind, name, strings.Join(sortedKeys(allowed), ", "))
}

func sortedKeys(m map[string]struct{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
