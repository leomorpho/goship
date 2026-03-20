package adapters

import (
	"context"
	"errors"
	"sync"

	"github.com/leomorpho/goship/config"
	"github.com/leomorpho/goship/framework/core"
)

var _ core.Jobs = (*CoreJobsAdapter)(nil)

// CoreJobsAdapter provides a capability-aware fallback/delegating core.Jobs adapter.
type CoreJobsAdapter struct {
	delegate     core.Jobs
	capabilities core.JobCapabilities

	mu       sync.RWMutex
	handlers map[string]core.JobHandler
}

func NewCoreJobsAdapter(delegate core.Jobs, capabilities core.JobCapabilities) *CoreJobsAdapter {
	return &CoreJobsAdapter{
		delegate:     delegate,
		capabilities: capabilities,
		handlers:     make(map[string]core.JobHandler),
	}
}

func NewCoreJobsAdapterFromConfig(delegate core.Jobs, cfg *config.Config) (*CoreJobsAdapter, error) {
	resolved, err := ResolveFromConfig(cfg)
	if err != nil {
		return nil, err
	}
	return NewCoreJobsAdapter(delegate, resolved.JobsCapabilities), nil
}

func (a *CoreJobsAdapter) Register(name string, handler core.JobHandler) error {
	if name == "" {
		return errors.New("job name is required")
	}
	if handler == nil {
		return errors.New("job handler is required")
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.handlers[name] = handler
	return nil
}

func (a *CoreJobsAdapter) Enqueue(ctx context.Context, name string, payload []byte, opts core.EnqueueOptions) (string, error) {
	if a == nil || a.delegate == nil {
		return "", errors.New("jobs client is not initialized")
	}
	return a.delegate.Enqueue(ctx, name, payload, opts)
}

func (a *CoreJobsAdapter) StartWorker(context.Context) error {
	// Worker startup is currently handled by cmd/worker runtime wiring.
	return nil
}

func (a *CoreJobsAdapter) StartScheduler(ctx context.Context) error {
	if a == nil || a.delegate == nil {
		return errors.New("jobs scheduler is not initialized")
	}
	return a.delegate.StartScheduler(ctx)
}

func (a *CoreJobsAdapter) Stop(ctx context.Context) error {
	if a == nil || a.delegate == nil {
		return nil
	}
	return a.delegate.Stop(ctx)
}

func (a *CoreJobsAdapter) Capabilities() core.JobCapabilities {
	if a == nil {
		return core.JobCapabilities{}
	}
	return a.capabilities
}
