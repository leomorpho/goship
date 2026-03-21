package health

import (
	"context"
	"sync"
)

const (
	StatusOK    = "ok"
	StatusError = "error"
)

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
