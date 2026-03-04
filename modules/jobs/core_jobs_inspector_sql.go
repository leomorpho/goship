package jobs

import (
	"context"

	sqldriver "github.com/leomorpho/goship-modules/jobs/drivers/sql"
	"github.com/leomorpho/goship/framework/core"
)

type sqlJobsInspector struct {
	client *sqldriver.Client
}

func newSQLJobsInspector(client *sqldriver.Client) core.JobsInspector {
	return &sqlJobsInspector{client: client}
}

func (s *sqlJobsInspector) List(ctx context.Context, filter core.JobListFilter) ([]core.JobRecord, error) {
	if s == nil || s.client == nil {
		return nil, nil
	}
	statuses := make([]string, 0, len(filter.Statuses))
	for _, status := range filter.Statuses {
		statuses = append(statuses, string(status))
	}
	rows, err := s.client.List(ctx, filter.Queue, statuses, filter.Limit, filter.Offset)
	if err != nil {
		return nil, err
	}
	out := make([]core.JobRecord, 0, len(rows))
	for _, row := range rows {
		out = append(out, toCoreJobRecord(row))
	}
	return out, nil
}

func (s *sqlJobsInspector) Get(ctx context.Context, id string) (core.JobRecord, bool, error) {
	if s == nil || s.client == nil {
		return core.JobRecord{}, false, nil
	}
	row, ok, err := s.client.Get(ctx, id)
	if err != nil {
		return core.JobRecord{}, false, err
	}
	if !ok {
		return core.JobRecord{}, false, nil
	}
	return toCoreJobRecord(row), true, nil
}

func toCoreJobRecord(row sqldriver.Job) core.JobRecord {
	return core.JobRecord{
		ID:         row.ID,
		Queue:      row.Queue,
		Name:       row.Name,
		Payload:    []byte(row.Payload),
		Status:     core.JobStatus(row.Status),
		Attempt:    row.Attempt,
		MaxRetries: row.MaxRetries,
		RunAt:      row.RunAt,
		CreatedAt:  row.CreatedAt,
		UpdatedAt:  row.UpdatedAt,
		LastError:  row.LastError,
	}
}
