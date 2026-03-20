package jobs

import (
	"context"
	"time"
)

type JobHandler func(ctx context.Context, payload []byte) error

type EnqueueOptions struct {
	Queue      string
	RunAt      time.Time
	MaxRetries int
	Timeout    time.Duration
	Retention  time.Duration
	Priority   int
}

type JobCapabilities struct {
	Delayed    bool
	Retries    bool
	Cron       bool
	Priority   bool
	DeadLetter bool
	Dashboard  bool
}

type JobStatus string

const (
	JobStatusQueued  JobStatus = "queued"
	JobStatusRunning JobStatus = "running"
	JobStatusDone    JobStatus = "done"
	JobStatusFailed  JobStatus = "failed"
)

type JobRecord struct {
	ID         string
	Queue      string
	Name       string
	Payload    []byte
	Status     JobStatus
	Attempt    int
	MaxRetries int
	RunAt      time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
	LastError  string
}

type JobListFilter struct {
	Queue    string
	Statuses []JobStatus
	Limit    int
	Offset   int
}

type Jobs interface {
	Register(name string, handler JobHandler) error
	Enqueue(ctx context.Context, name string, payload []byte, opts EnqueueOptions) (jobID string, err error)
	StartWorker(ctx context.Context) error
	StartScheduler(ctx context.Context) error
	Stop(ctx context.Context) error
	Capabilities() JobCapabilities
}

type JobsInspector interface {
	List(ctx context.Context, filter JobListFilter) ([]JobRecord, error)
	Get(ctx context.Context, id string) (JobRecord, bool, error)
}

type MessageHandler func(ctx context.Context, payload []byte) error
