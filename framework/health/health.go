package health

import (
	"context"
	"fmt"
	"sort"
	"slices"
	"strings"
	"sync"
)

const (
	StatusOK    = "ok"
	StatusError = "error"
)

var defaultStartupChecks = []string{"db", "cache", "jobs"}

type Checker interface {
	Name() string
	Check(ctx context.Context) CheckResult
}

type CheckResult struct {
	Status     string         `json:"status"`
	LatencyMs  int64          `json:"latency_ms,omitempty"`
	Error      string         `json:"error,omitempty"`
	QueueDepth int            `json:"queue_depth,omitempty"`
	Extra      map[string]any `json:"extra,omitempty"`
}

type Registry struct {
	mu       sync.RWMutex
	checkers []Checker
}

type StartupSummary struct {
	Ready      bool     `json:"ready"`
	Required   []string `json:"required"`
	Registered []string `json:"registered"`
	Missing    []string `json:"missing"`
}

func NewRegistry(checkers ...Checker) *Registry {
	registry := &Registry{}
	for _, checker := range checkers {
		registry.Register(checker)
	}
	return registry
}

func (r *Registry) Register(checker Checker) {
	if r == nil || checker == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.checkers = append(r.checkers, checker)
}

func (r *Registry) ValidateStartupContract(requiredChecks ...string) error {
	summary := r.StartupSummary(requiredChecks...)
	if !summary.Ready {
		return fmt.Errorf(
			"health startup contract: ready=%t required=%v registered=%v missing=%v",
			summary.Ready,
			summary.Required,
			summary.Registered,
			summary.Missing,
		)
	}
	return nil
}

func (r *Registry) StartupSummary(requiredChecks ...string) StartupSummary {
	required := slices.Clone(requiredChecks)
	if len(required) == 0 {
		required = slices.Clone(defaultStartupChecks)
	}

	summary := StartupSummary{
		Required: required,
	}

	if r == nil {
		summary.Missing = slices.Clone(required)
		return summary
	}

	r.mu.RLock()
	checkers := append([]Checker(nil), r.checkers...)
	r.mu.RUnlock()

	registered := make([]string, 0, len(checkers))
	for _, checker := range checkers {
		name := strings.TrimSpace(checker.Name())
		if name == "" || slices.Contains(registered, name) {
			continue
		}
		registered = append(registered, name)
	}
	sort.Strings(registered)
	summary.Registered = registered

	missing := make([]string, 0)
	for _, name := range required {
		if !slices.Contains(registered, name) {
			missing = append(missing, name)
		}
	}
	summary.Missing = missing
	summary.Ready = len(missing) == 0
	return summary
}

func (r *Registry) Run(ctx context.Context) (map[string]CheckResult, bool) {
	results := make(map[string]CheckResult)
	if r == nil {
		return results, true
	}

	r.mu.RLock()
	checkers := append([]Checker(nil), r.checkers...)
	r.mu.RUnlock()

	allOK := true
	for _, checker := range checkers {
		result := checker.Check(ctx)
		if result.Status == "" {
			result.Status = StatusError
		}
		if result.Status != StatusOK {
			allOK = false
		}
		results[checker.Name()] = result
	}

	return results, allOK
}
