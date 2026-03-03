package services

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/hibiken/asynq"
	"github.com/leomorpho/goship/config"
	"github.com/leomorpho/goship/pkg/core"
	coreadapters "github.com/leomorpho/goship/pkg/core/adapters"
)

var _ core.Jobs = (*CoreJobsAdapter)(nil)

// CoreJobsAdapter adapts TaskClient to the core.Jobs interface.
type CoreJobsAdapter struct {
	client       *TaskClient
	capabilities core.JobCapabilities

	mu       sync.RWMutex
	handlers map[string]core.JobHandler
}

func NewCoreJobsAdapter(client *TaskClient, capabilities core.JobCapabilities) *CoreJobsAdapter {
	return &CoreJobsAdapter{
		client:       client,
		capabilities: capabilities,
		handlers:     make(map[string]core.JobHandler),
	}
}

func NewCoreJobsAdapterFromConfig(client *TaskClient, cfg *config.Config) (*CoreJobsAdapter, error) {
	resolved, err := coreadapters.ResolveFromConfig(cfg)
	if err != nil {
		return nil, err
	}
	return NewCoreJobsAdapter(client, resolved.JobsCapabilities), nil
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
	if a == nil || a.client == nil || a.client.client == nil {
		return "", errors.New("jobs client is not initialized")
	}
	task := asynq.NewTask(name, payload, toAsynqOptions(opts)...)
	info, err := a.client.client.EnqueueContext(ctx, task)
	if err != nil {
		return "", err
	}
	return info.ID, nil
}

func (a *CoreJobsAdapter) StartWorker(context.Context) error {
	// Worker startup is currently handled by cmd/worker runtime wiring.
	return nil
}

func (a *CoreJobsAdapter) StartScheduler(context.Context) error {
	if a == nil || a.client == nil || a.client.scheduler == nil {
		return errors.New("jobs scheduler is not initialized")
	}
	return a.client.StartScheduler()
}

func (a *CoreJobsAdapter) Stop(context.Context) error {
	if a == nil || a.client == nil {
		return nil
	}
	return a.client.Close()
}

func (a *CoreJobsAdapter) Capabilities() core.JobCapabilities {
	if a == nil {
		return core.JobCapabilities{}
	}
	return a.capabilities
}

func toAsynqOptions(opts core.EnqueueOptions) []asynq.Option {
	converted := make([]asynq.Option, 0, 5)

	if opts.Queue != "" {
		converted = append(converted, asynq.Queue(opts.Queue))
	} else if opts.Priority > 0 {
		converted = append(converted, asynq.Queue(priorityToQueue(opts.Priority)))
	}
	if opts.MaxRetries > 0 {
		converted = append(converted, asynq.MaxRetry(opts.MaxRetries))
	}
	if opts.Timeout > 0 {
		converted = append(converted, asynq.Timeout(opts.Timeout))
	}
	if !opts.RunAt.IsZero() {
		delay := time.Until(opts.RunAt)
		if delay > 0 {
			converted = append(converted, asynq.ProcessIn(delay))
		}
	}
	return converted
}

func priorityToQueue(priority int) string {
	switch {
	case priority >= 90:
		return "critical"
	case priority >= 50:
		return "default"
	default:
		return "low"
	}
}
