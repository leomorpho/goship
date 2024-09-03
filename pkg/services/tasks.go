package services

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/hibiken/asynq"
	"github.com/mikestefanello/pagoda/config"
)

type (
	// TaskClient is that client that allows you to queue or schedule task execution
	TaskClient struct {
		// client stores the asynq client
		client *asynq.Client

		// scheduler stores the asynq scheduler
		scheduler *asynq.Scheduler

		// inspector allows one to look at which tasks are running
		inspector *asynq.Inspector
	}

	// task handles task creation operations
	task struct {
		client     *TaskClient
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

// NewTaskClient creates a new task client
func NewTaskClient(cfg *config.Config) *TaskClient {
	// Determine the database based on the environment
	db := cfg.Cache.Database
	if cfg.App.Environment == config.EnvTest {
		db = cfg.Cache.TestDatabase
	}

	conn := asynq.RedisClientOpt{
		Addr:     fmt.Sprintf("%s:%d", cfg.Cache.Hostname, cfg.Cache.Port),
		Password: cfg.Cache.Password,
		DB:       db,
	}

	return &TaskClient{
		client:    asynq.NewClient(conn),
		scheduler: asynq.NewScheduler(conn, nil),
	}
}

// Close closes the connection to the task service
func (t *TaskClient) Close() error {
	return t.client.Close()
}

// StartScheduler starts the scheduler service which adds scheduled tasks to the queue
// This must be running in order to queue tasks set for periodic execution
func (t *TaskClient) StartScheduler() error {
	return t.scheduler.Run()
}

// CountPendingTasks returns the count of tasks of the given type in the pending queue
func (t *TaskClient) CountPendingTasks(taskType string) int {
	return t.countTasksInQueue(t.inspector.ListPendingTasks, taskType)
}

// CountActiveTasks returns the count of tasks of the given type in the active queue
func (t *TaskClient) CountActiveTasks(taskType string) int {
	return t.countTasksInQueue(t.inspector.ListActiveTasks, taskType)
}

// CountRetryTasks returns the count of tasks of the given type in the retry queue
func (t *TaskClient) CountRetryTasks(taskType string) int {
	return t.countTasksInQueue(t.inspector.ListRetryTasks, taskType)
}

// CountScheduledTasks returns the count of tasks of the given type in the retry queue
func (t *TaskClient) CountScheduledTasks(taskType string) int {
	return t.countTasksInQueue(t.inspector.ListScheduledTasks, taskType)
}

// countTasksInQueue is a helper function to count tasks of a specific type in a specific state
func (t *TaskClient) countTasksInQueue(
	listTasksFunc func(queue string, opts ...asynq.ListOption) ([]*asynq.TaskInfo, error),
	taskType string,
) int {
	queues, err := t.inspector.Queues()
	if err != nil {
		log.Fatalf("Failed to list queues: %v", err)
	}

	count := 0
	for _, q := range queues {
		tasks, err := listTasksFunc(q)
		if err != nil {
			log.Fatalf("Failed to list tasks for queue %s: %v", q, err)
		}
		for _, task := range tasks {
			if task.Type == taskType {
				count++
			}
		}
	}
	return count
}

// New starts a task creation operation
func (t *TaskClient) New(typ string) *task {
	return &task{
		client: t,
		typ:    typ,
	}
}

// Payload sets the task payload data which will be sent to the task handler
func (t *task) Payload(payload any) *task {
	t.payload = payload
	return t
}

// Periodic sets the task to execute periodically according to a given interval
// The interval can be either in cron form ("*/5 * * * *") or "@every 30s"
func (t *task) Periodic(interval string) *task {
	t.periodic = &interval
	return t
}

// Queue specifies the name of the queue to add the task to
// The default queue will be used if this is not set
func (t *task) Queue(queue string) *task {
	t.queue = &queue
	return t
}

// Timeout sets the task timeout, meaning the task must execute within a given duration
func (t *task) Timeout(timeout time.Duration) *task {
	t.timeout = &timeout
	return t
}

// Deadline sets the task execution deadline to a specific date and time
func (t *task) Deadline(deadline time.Time) *task {
	t.deadline = &deadline
	return t
}

// At sets the exact date and time the task should be executed
func (t *task) At(processAt time.Time) *task {
	t.at = &processAt
	return t
}

// Wait instructs the task to wait a given duration before it is executed
func (t *task) Wait(duration time.Duration) *task {
	t.wait = &duration
	return t
}

// Retain instructs the task service to retain the task data for a given duration after execution is complete
func (t *task) Retain(duration time.Duration) *task {
	t.retain = &duration
	return t
}

// MaxRetries sets the maximum amount of times to retry executing the task in the event of a failure
func (t *task) MaxRetries(retries int) *task {
	t.maxRetries = &retries
	return t
}

// Save saves the task so it can be executed
func (t *task) Save() error {
	var err error

	// Build the payload
	var payload []byte
	if t.payload != nil {
		if payload, err = json.Marshal(t.payload); err != nil {
			return err
		}
	}

	// Build the task options
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

	// Build the task
	task := asynq.NewTask(t.typ, payload, opts...)

	// Schedule, if needed
	if t.periodic != nil {
		_, err = t.client.scheduler.Register(*t.periodic, task)
	} else {
		_, err = t.client.client.Enqueue(task)
	}
	return err
}
