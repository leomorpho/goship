package jobs

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	backlitedriver "github.com/leomorpho/goship-modules/jobs/drivers/backlite"
	backlite "github.com/mikestefanello/backlite"
)

var _ Jobs = (*backliteCoreJobs)(nil)

type backliteCoreJobs struct {
	client       *backlitedriver.Client
	capabilities JobCapabilities

	mu       sync.RWMutex
	handlers map[string]JobHandler
	queues   map[string]struct{}
}

type backliteEnvelope struct {
	QueueName string `json:"queue_name"`
	Payload   []byte `json:"payload"`
}

func (t backliteEnvelope) Config() backlite.QueueConfig {
	return backlite.QueueConfig{Name: t.QueueName}
}

type backliteQueue struct {
	config  backlite.QueueConfig
	process func(context.Context, []byte) error
}

func (q *backliteQueue) Config() *backlite.QueueConfig {
	return &q.config
}

func (q *backliteQueue) Process(ctx context.Context, payload []byte) error {
	return q.process(ctx, payload)
}

func newBackliteCoreJobs(client *backlitedriver.Client) Jobs {
	return &backliteCoreJobs{
		client: client,
		capabilities: JobCapabilities{
			Delayed:    true,
			Retries:    true,
			Cron:       false,
			Priority:   false,
			DeadLetter: true,
			Dashboard:  false,
		},
		handlers: make(map[string]JobHandler),
		queues:   make(map[string]struct{}),
	}
}

func (b *backliteCoreJobs) Register(name string, handler JobHandler) error {
	if name == "" {
		return errors.New("job name is required")
	}
	if handler == nil {
		return errors.New("job handler is required")
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[name] = handler
	return nil
}

func (b *backliteCoreJobs) Enqueue(ctx context.Context, name string, payload []byte, opts EnqueueOptions) (string, error) {
	if b == nil || b.client == nil {
		return "", errors.New("backlite jobs client is not initialized")
	}
	queue, err := b.ensureQueue(name, opts)
	if err != nil {
		return "", err
	}
	return b.client.Add(ctx, backliteEnvelope{
		QueueName: queue,
		Payload:   payload,
	}, opts.RunAt)
}

func (b *backliteCoreJobs) StartWorker(ctx context.Context) error {
	if b == nil || b.client == nil {
		return errors.New("backlite jobs client is not initialized")
	}
	b.client.Start(ctx)
	<-ctx.Done()
	stopCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if b.client.Stop(stopCtx) {
		return nil
	}
	if err := stopCtx.Err(); err != nil {
		return err
	}
	return errors.New("backlite worker did not stop before timeout")
}

func (b *backliteCoreJobs) StartScheduler(context.Context) error { return nil }

func (b *backliteCoreJobs) Stop(ctx context.Context) error {
	if b == nil || b.client == nil {
		return nil
	}
	if b.client.Stop(ctx) {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return errors.New("backlite worker did not stop before timeout")
}

func (b *backliteCoreJobs) Capabilities() JobCapabilities {
	if b == nil {
		return JobCapabilities{}
	}
	return b.capabilities
}

func (b *backliteCoreJobs) ensureQueue(name string, opts EnqueueOptions) (string, error) {
	handler := b.handlerFor(name)
	if handler == nil {
		return "", fmt.Errorf("no registered handler for job %q", name)
	}

	cfg := backliteQueueConfig(name, opts)

	b.mu.Lock()
	defer b.mu.Unlock()
	if _, ok := b.queues[cfg.Name]; ok {
		return cfg.Name, nil
	}

	b.client.Register(&backliteQueue{
		config: cfg,
		process: func(ctx context.Context, payload []byte) error {
			current := b.handlerFor(name)
			if current == nil {
				return fmt.Errorf("no registered handler for job %q", name)
			}
			return current(ctx, payload)
		},
	})
	b.queues[cfg.Name] = struct{}{}
	return cfg.Name, nil
}

func (b *backliteCoreJobs) handlerFor(name string) JobHandler {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.handlers[name]
}

func backliteQueueConfig(name string, opts EnqueueOptions) backlite.QueueConfig {
	maxAttempts := opts.MaxRetries + 1
	if maxAttempts < 1 {
		maxAttempts = 1
	}
	cfg := backlite.QueueConfig{
		Name:        backliteQueueName(name, opts),
		MaxAttempts: maxAttempts,
		Timeout:     opts.Timeout,
		Backoff:     time.Second,
	}
	if opts.Retention > 0 {
		cfg.Retention = &backlite.Retention{
			Duration: opts.Retention,
			Data:     &backlite.RetainData{},
		}
	}
	return cfg
}

func backliteQueueName(name string, opts EnqueueOptions) string {
	baseQueue := opts.Queue
	if baseQueue == "" {
		if opts.Priority > 0 {
			baseQueue = priorityToQueue(opts.Priority)
		} else {
			baseQueue = "default"
		}
	}
	baseQueue = sanitizeBackliteName(baseQueue)
	jobName := sanitizeBackliteName(name)
	hashInput := fmt.Sprintf("%s|%s|%d|%d|%d", baseQueue, name, opts.MaxRetries, opts.Timeout, opts.Retention)
	sum := sha1.Sum([]byte(hashInput))
	return fmt.Sprintf("%s__%s__%s", baseQueue, jobName, hex.EncodeToString(sum[:6]))
}

func sanitizeBackliteName(v string) string {
	v = strings.ToLower(strings.TrimSpace(v))
	if v == "" {
		return "default"
	}
	var b strings.Builder
	b.Grow(len(v))
	lastDash := false
	for _, r := range v {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
			lastDash = false
		case r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		default:
			if !lastDash {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "default"
	}
	return out
}
