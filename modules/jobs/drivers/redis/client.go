package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/hibiken/asynq"
)

type Config struct {
	Addr     string
	Password string
	DB       int
}

type (
	// Client queues and schedules tasks through Asynq/Redis.
	Client struct {
		client    *asynq.Client
		scheduler *asynq.Scheduler
		inspector *asynq.Inspector
	}

	TaskBuilder struct {
		client     *Client
		typ        string
		payload    any
		periodic   *string
		queue      *string
		maxRetries *int
		timeout    *time.Duration
		deadline   *time.Time
		at         *time.Time
		wait       *time.Duration
		retain     *time.Duration
	}
)

func New(cfg Config) *Client {
	conn := asynq.RedisClientOpt{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	}

	return &Client{
		client:    asynq.NewClient(conn),
		scheduler: asynq.NewScheduler(conn, nil),
		inspector: asynq.NewInspector(conn),
	}
}

func (c *Client) Close() error {
	return c.client.Close()
}

func (c *Client) StartScheduler() error {
	return c.scheduler.Run()
}

func (c *Client) CountPendingTasks(taskType string) int {
	return c.countTasksInQueue(c.inspector.ListPendingTasks, taskType)
}

func (c *Client) CountActiveTasks(taskType string) int {
	return c.countTasksInQueue(c.inspector.ListActiveTasks, taskType)
}

func (c *Client) CountRetryTasks(taskType string) int {
	return c.countTasksInQueue(c.inspector.ListRetryTasks, taskType)
}

func (c *Client) CountScheduledTasks(taskType string) int {
	return c.countTasksInQueue(c.inspector.ListScheduledTasks, taskType)
}

func (c *Client) countTasksInQueue(
	listTasksFunc func(queue string, opts ...asynq.ListOption) ([]*asynq.TaskInfo, error),
	taskType string,
) int {
	queues, err := c.inspector.Queues()
	if err != nil {
		slog.Error("failed to list queues", "error", err)
		return 0
	}

	count := 0
	for _, q := range queues {
		tasks, err := listTasksFunc(q)
		if err != nil {
			slog.Error("failed to list tasks for queue", "queue", q, "error", err)
			return 0
		}
		for _, task := range tasks {
			if task.Type == taskType {
				count++
			}
		}
	}
	return count
}

func (c *Client) EnqueueContext(ctx context.Context, typ string, payload []byte, opts ...asynq.Option) (*asynq.TaskInfo, error) {
	job := asynq.NewTask(typ, payload, opts...)
	return c.client.EnqueueContext(ctx, job)
}

func (c *Client) NewTask(typ string) *TaskBuilder {
	return &TaskBuilder{
		client: c,
		typ:    typ,
	}
}

func (t *TaskBuilder) Payload(payload any) *TaskBuilder {
	t.payload = payload
	return t
}

func (t *TaskBuilder) Periodic(interval string) *TaskBuilder {
	t.periodic = &interval
	return t
}

func (t *TaskBuilder) Queue(queue string) *TaskBuilder {
	t.queue = &queue
	return t
}

func (t *TaskBuilder) Timeout(timeout time.Duration) *TaskBuilder {
	t.timeout = &timeout
	return t
}

func (t *TaskBuilder) Deadline(deadline time.Time) *TaskBuilder {
	t.deadline = &deadline
	return t
}

func (t *TaskBuilder) At(processAt time.Time) *TaskBuilder {
	t.at = &processAt
	return t
}

func (t *TaskBuilder) Wait(duration time.Duration) *TaskBuilder {
	t.wait = &duration
	return t
}

func (t *TaskBuilder) Retain(duration time.Duration) *TaskBuilder {
	t.retain = &duration
	return t
}

func (t *TaskBuilder) MaxRetries(retries int) *TaskBuilder {
	t.maxRetries = &retries
	return t
}

func (t *TaskBuilder) Save() error {
	var err error

	var payload []byte
	if t.payload != nil {
		if payload, err = json.Marshal(t.payload); err != nil {
			return err
		}
	}

	opts := make([]asynq.Option, 0)
	if t.queue != nil {
		opts = append(opts, asynq.Queue(*t.queue))
	}
	if t.maxRetries != nil {
		opts = append(opts, asynq.MaxRetry(*t.maxRetries))
	}
	if t.timeout != nil {
		opts = append(opts, asynq.Timeout(*t.timeout))
	}
	if t.deadline != nil {
		opts = append(opts, asynq.Deadline(*t.deadline))
	}
	if t.wait != nil {
		opts = append(opts, asynq.ProcessIn(*t.wait))
	}
	if t.retain != nil {
		opts = append(opts, asynq.Retention(*t.retain))
	}
	if t.at != nil {
		opts = append(opts, asynq.ProcessAt(*t.at))
	}

	job := asynq.NewTask(t.typ, payload, opts...)

	if t.periodic != nil {
		_, err = t.client.scheduler.Register(*t.periodic, job)
	} else {
		_, err = t.client.client.Enqueue(job)
	}
	return err
}

func (c Config) String() string {
	return fmt.Sprintf("%s/%d", c.Addr, c.DB)
}
