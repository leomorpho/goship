package jobs

import (
	"context"

	sqldriver "github.com/leomorpho/goship-modules/jobs/drivers/sql"
)

type sqlJobsInspector struct {
	client *sqldriver.Client
}

func newSQLJobsInspector(client *sqldriver.Client) JobsInspector {
	return &sqlJobsInspector{client: client}
}

func (s *sqlJobsInspector) List(ctx context.Context, filter JobListFilter) ([]JobRecord, error) {
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
	out := make([]JobRecord, 0, len(rows))
	for _, row := range rows {
		out = append(out, toCoreJobRecord(row))
	}
	return out, nil
}

func (s *sqlJobsInspector) Get(ctx context.Context, id string) (JobRecord, bool, error) {
	if s == nil || s.client == nil {
		return JobRecord{}, false, nil
	}
	row, ok, err := s.client.Get(ctx, id)
	if err != nil {
		return JobRecord{}, false, err
	}
	if !ok {
		return JobRecord{}, false, nil
	}
	return toCoreJobRecord(row), true, nil
}

func toCoreJobRecord(row sqldriver.Job) JobRecord {
	return JobRecord{
		ID:         row.ID,
		Queue:      row.Queue,
		Name:       row.Name,
		Payload:    []byte(row.Payload),
		Status:     JobStatus(row.Status),
		Attempt:    row.Attempt,
		MaxRetries: row.MaxRetries,
		RunAt:      row.RunAt,
		CreatedAt:  row.CreatedAt,
		UpdatedAt:  row.UpdatedAt,
		LastError:  row.LastError,
	}
}
