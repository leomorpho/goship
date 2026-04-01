package bootstrap

import (
	"context"

	jobsmodule "github.com/leomorpho/goship/v2-modules/jobs"
	"github.com/leomorpho/goship/v2/framework/core"
)

type moduleJobsBridge struct {
	inner jobsmodule.Jobs
}

type moduleJobsInspectorBridge struct {
	inner jobsmodule.JobsInspector
}

// AdaptModuleJobs exposes module jobs behind the app core jobs seam.
func AdaptModuleJobs(inner jobsmodule.Jobs) core.Jobs {
	if inner == nil {
		return nil
	}
	return moduleJobsBridge{inner: inner}
}

// AdaptModuleJobsInspector exposes module job inspector behind the app core seam.
func AdaptModuleJobsInspector(inner jobsmodule.JobsInspector) core.JobsInspector {
	if inner == nil {
		return nil
	}
	return moduleJobsInspectorBridge{inner: inner}
}

func (b moduleJobsBridge) Register(name string, handler core.JobHandler) error {
	return b.inner.Register(name, func(ctx context.Context, payload []byte) error {
		return handler(ctx, payload)
	})
}

func (b moduleJobsBridge) Enqueue(ctx context.Context, name string, payload []byte, opts core.EnqueueOptions) (string, error) {
	return b.inner.Enqueue(ctx, name, payload, jobsmodule.EnqueueOptions{
		Queue:      opts.Queue,
		RunAt:      opts.RunAt,
		MaxRetries: opts.MaxRetries,
		Timeout:    opts.Timeout,
		Retention:  opts.Retention,
		Priority:   opts.Priority,
	})
}

func (b moduleJobsBridge) StartWorker(ctx context.Context) error {
	return b.inner.StartWorker(ctx)
}

func (b moduleJobsBridge) StartScheduler(ctx context.Context) error {
	return b.inner.StartScheduler(ctx)
}

func (b moduleJobsBridge) Stop(ctx context.Context) error {
	return b.inner.Stop(ctx)
}

func (b moduleJobsBridge) Capabilities() core.JobCapabilities {
	c := b.inner.Capabilities()
	return core.JobCapabilities{
		Delayed:    c.Delayed,
		Retries:    c.Retries,
		Cron:       c.Cron,
		Priority:   c.Priority,
		DeadLetter: c.DeadLetter,
		Dashboard:  c.Dashboard,
	}
}

func (b moduleJobsInspectorBridge) List(ctx context.Context, filter core.JobListFilter) ([]core.JobRecord, error) {
	statuses := make([]jobsmodule.JobStatus, 0, len(filter.Statuses))
	for _, s := range filter.Statuses {
		statuses = append(statuses, jobsmodule.JobStatus(s))
	}
	rows, err := b.inner.List(ctx, jobsmodule.JobListFilter{
		Queue:    filter.Queue,
		Statuses: statuses,
		Limit:    filter.Limit,
		Offset:   filter.Offset,
	})
	if err != nil {
		return nil, err
	}
	out := make([]core.JobRecord, 0, len(rows))
	for _, row := range rows {
		out = append(out, core.JobRecord{
			ID:         row.ID,
			Queue:      row.Queue,
			Name:       row.Name,
			Payload:    row.Payload,
			Status:     core.JobStatus(row.Status),
			Attempt:    row.Attempt,
			MaxRetries: row.MaxRetries,
			RunAt:      row.RunAt,
			CreatedAt:  row.CreatedAt,
			UpdatedAt:  row.UpdatedAt,
			LastError:  row.LastError,
		})
	}
	return out, nil
}

func (b moduleJobsInspectorBridge) Get(ctx context.Context, id string) (core.JobRecord, bool, error) {
	row, found, err := b.inner.Get(ctx, id)
	if err != nil || !found {
		return core.JobRecord{}, found, err
	}
	return core.JobRecord{
		ID:         row.ID,
		Queue:      row.Queue,
		Name:       row.Name,
		Payload:    row.Payload,
		Status:     core.JobStatus(row.Status),
		Attempt:    row.Attempt,
		MaxRetries: row.MaxRetries,
		RunAt:      row.RunAt,
		CreatedAt:  row.CreatedAt,
		UpdatedAt:  row.UpdatedAt,
		LastError:  row.LastError,
	}, true, nil
}
