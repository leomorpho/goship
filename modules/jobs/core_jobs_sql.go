package jobs

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	sqldriver "github.com/leomorpho/goship-modules/jobs/drivers/sql"
	"github.com/leomorpho/goship/framework/core"
)

var _ core.Jobs = (*sqlCoreJobs)(nil)

type sqlCoreJobs struct {
	client       *sqldriver.Client
	capabilities core.JobCapabilities

	mu       sync.RWMutex
	handlers map[string]core.JobHandler
}

func newSQLCoreJobs(client *sqldriver.Client) core.Jobs {
	return &sqlCoreJobs{
		client: client,
		capabilities: core.JobCapabilities{
			Delayed:    true,
			Retries:    true,
			Cron:       false,
			DeadLetter: true,
		},
		handlers: make(map[string]core.JobHandler),
	}
}

const (
	sqlWorkerPollInterval = 100 * time.Millisecond
	sqlWorkerLockDuration = 30 * time.Second
)

func (s *sqlCoreJobs) Register(name string, handler core.JobHandler) error {
	if name == "" {
		return errors.New("job name is required")
	}
	if handler == nil {
		return errors.New("job handler is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handlers[name] = handler
	return nil
}

func (s *sqlCoreJobs) Enqueue(ctx context.Context, name string, payload []byte, opts core.EnqueueOptions) (string, error) {
	if s == nil || s.client == nil {
		return "", errors.New("sql jobs client is not initialized")
	}
	id, err := generateJobID()
	if err != nil {
		return "", err
	}

	queue := opts.Queue
	if queue == "" {
		if opts.Priority > 0 {
			queue = priorityToQueue(opts.Priority)
		} else {
			queue = "default"
		}
	}
	runAt := opts.RunAt
	if runAt.IsZero() {
		runAt = time.Now().UTC()
	}
	maxRetries := opts.MaxRetries
	if maxRetries < 0 {
		maxRetries = 0
	}
	payloadStr := string(payload)
	if !json.Valid(payload) {
		encoded, marshalErr := json.Marshal(map[string]string{"data": payloadStr})
		if marshalErr != nil {
			return "", marshalErr
		}
		payloadStr = string(encoded)
	}

	if err := s.client.Enqueue(ctx, id, queue, name, payloadStr, runAt, maxRetries); err != nil {
		return "", err
	}
	return id, nil
}

func (s *sqlCoreJobs) StartWorker(ctx context.Context) error {
	if s == nil || s.client == nil {
		return errors.New("sql jobs client is not initialized")
	}
	workerID, err := generateJobID()
	if err != nil {
		return fmt.Errorf("generate worker id: %w", err)
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		job, found, claimErr := s.client.ClaimNext(ctx, workerID, time.Now().UTC().Add(sqlWorkerLockDuration))
		if claimErr != nil {
			return claimErr
		}
		if !found {
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(sqlWorkerPollInterval):
			}
			continue
		}
		s.processOne(ctx, job)
	}
}

func (s *sqlCoreJobs) StartScheduler(context.Context) error { return nil }
func (s *sqlCoreJobs) Stop(context.Context) error           { return nil }
func (s *sqlCoreJobs) Capabilities() core.JobCapabilities {
	if s == nil {
		return core.JobCapabilities{}
	}
	return s.capabilities
}

func (s *sqlCoreJobs) processOne(ctx context.Context, job sqldriver.Job) {
	handler := s.handlerFor(job.Name)
	if handler == nil {
		_ = s.client.MarkFailed(ctx, job.ID, fmt.Sprintf("no registered handler for job %q", job.Name))
		return
	}
	if err := handler(ctx, []byte(job.Payload)); err != nil {
		if job.Attempt+1 > job.MaxRetries {
			_ = s.client.MarkFailed(ctx, job.ID, err.Error())
			return
		}
		_ = s.client.MarkRetry(ctx, job.ID, retryRunAt(job.Attempt+1), err.Error())
		return
	}
	_ = s.client.MarkDone(ctx, job.ID)
}

func (s *sqlCoreJobs) handlerFor(name string) core.JobHandler {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.handlers[name]
}

func retryRunAt(nextAttempt int) time.Time {
	if nextAttempt < 1 {
		nextAttempt = 1
	}
	delay := time.Duration(nextAttempt*nextAttempt) * time.Second
	if delay > 30*time.Second {
		delay = 30 * time.Second
	}
	return time.Now().UTC().Add(delay)
}

func generateJobID() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
