package jobs

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/hibiken/asynq"
	redisdriver "github.com/leomorpho/goship-modules/jobs/drivers/redis"
	"github.com/leomorpho/goship/framework/core"
)

var _ core.Jobs = (*redisCoreJobs)(nil)

type redisCoreJobs struct {
	client       *redisdriver.Client
	capabilities core.JobCapabilities

	mu       sync.RWMutex
	handlers map[string]core.JobHandler
}

func newRedisCoreJobs(client *redisdriver.Client) core.Jobs {
	return &redisCoreJobs{
		client: client,
		capabilities: core.JobCapabilities{
			Delayed:    true,
			Retries:    true,
			Cron:       true,
			Priority:   true,
			DeadLetter: true,
			Dashboard:  true,
		},
		handlers: make(map[string]core.JobHandler),
	}
}

func (a *redisCoreJobs) Register(name string, handler core.JobHandler) error {
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

func (a *redisCoreJobs) Enqueue(ctx context.Context, name string, payload []byte, opts core.EnqueueOptions) (string, error) {
	if a == nil || a.client == nil {
		return "", errors.New("jobs client is not initialized")
	}
	info, err := a.client.EnqueueContext(ctx, name, payload, toAsynqOptions(opts)...)
	if err != nil {
		return "", err
	}
	return info.ID, nil
}

func (a *redisCoreJobs) StartWorker(context.Context) error {
	// Worker startup is handled by cmd/worker runtime wiring.
	return nil
}

func (a *redisCoreJobs) StartScheduler(context.Context) error {
	if a == nil || a.client == nil {
		return errors.New("jobs scheduler is not initialized")
	}
	return a.client.StartScheduler()
}

func (a *redisCoreJobs) Stop(context.Context) error {
	if a == nil || a.client == nil {
		return nil
	}
	return a.client.Close()
}

func (a *redisCoreJobs) Capabilities() core.JobCapabilities {
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
	if opts.Retention > 0 {
		converted = append(converted, asynq.Retention(opts.Retention))
	}
	if !opts.RunAt.IsZero() {
		delay := time.Until(opts.RunAt)
		if delay > 0 {
			converted = append(converted, asynq.ProcessIn(delay))
		}
	}
	return converted
}
