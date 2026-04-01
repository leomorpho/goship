package sql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	dbmigrate "github.com/leomorpho/goship/v2-modules/jobs/db/migrate"
	dbqueries "github.com/leomorpho/goship/v2-modules/jobs/db/queries"
)

type Config struct {
	SQLDB *sql.DB
}

type Client struct {
	db sqlExecutor
}

type sqlExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

type Job struct {
	ID         string
	Queue      string
	Name       string
	Payload    string
	Status     string
	Attempt    int
	MaxRetries int
	RunAt      time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
	LastError  string
}

func New(cfg Config) (*Client, error) {
	if cfg.SQLDB == nil {
		return nil, errors.New("sql jobs driver requires SQLDB")
	}
	client := &Client{db: cfg.SQLDB}
	if err := client.ensureSchema(context.Background()); err != nil {
		return nil, err
	}
	return client, nil
}

func (c *Client) ensureSchema(ctx context.Context) error {
	ddl, err := dbmigrate.LoadInitJobsUpSQL()
	if err != nil {
		return err
	}
	ddl = strings.ReplaceAll(ddl, "TIMESTAMPTZ", "TIMESTAMP")
	_, err = c.db.ExecContext(ctx, ddl)
	return err
}

func (c *Client) Enqueue(
	ctx context.Context,
	id string,
	queue string,
	name string,
	payload string,
	runAt time.Time,
	maxRetries int,
) error {
	now := time.Now().UTC()
	query, err := dbqueries.Get("insert_job")
	if err != nil {
		return err
	}
	_, err = c.db.ExecContext(
		ctx,
		query,
		id,
		queue,
		name,
		payload,
		runAt.UTC(),
		maxRetries,
		now,
		now,
	)
	return err
}

func (c *Client) ClaimNext(ctx context.Context, workerID string, lockUntil time.Time) (Job, bool, error) {
	now := time.Now().UTC()
	claimQuery, err := dbqueries.Get("claim_next_job")
	if err != nil {
		return Job{}, false, err
	}
	res, err := c.db.ExecContext(
		ctx,
		claimQuery,
		workerID,
		lockUntil.UTC(),
		now,
		now,
		now,
	)
	if err != nil {
		return Job{}, false, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return Job{}, false, err
	}
	if affected == 0 {
		return Job{}, false, nil
	}

	selectClaimedQuery, err := dbqueries.Get("select_claimed_job")
	if err != nil {
		return Job{}, false, err
	}
	rows, err := c.db.QueryContext(
		ctx,
		selectClaimedQuery,
		workerID,
	)
	if err != nil {
		return Job{}, false, err
	}
	defer rows.Close()
	if !rows.Next() {
		return Job{}, false, nil
	}

	var job Job
	if err := rows.Scan(
		&job.ID,
		&job.Queue,
		&job.Name,
		&job.Payload,
		&job.Status,
		&job.Attempt,
		&job.MaxRetries,
		&job.RunAt,
		&job.CreatedAt,
		&job.UpdatedAt,
		&job.LastError,
	); err != nil {
		return Job{}, false, err
	}
	return job, true, nil
}

func (c *Client) MarkDone(ctx context.Context, id string) error {
	query, err := dbqueries.Get("mark_done")
	if err != nil {
		return err
	}
	_, err = c.db.ExecContext(
		ctx,
		query,
		time.Now().UTC(),
		id,
	)
	return err
}

func (c *Client) MarkRetry(ctx context.Context, id string, runAt time.Time, lastError string) error {
	query, err := dbqueries.Get("mark_retry")
	if err != nil {
		return err
	}
	_, err = c.db.ExecContext(
		ctx,
		query,
		runAt.UTC(),
		lastError,
		time.Now().UTC(),
		id,
	)
	return err
}

func (c *Client) MarkFailed(ctx context.Context, id string, lastError string) error {
	query, err := dbqueries.Get("mark_failed")
	if err != nil {
		return err
	}
	_, err = c.db.ExecContext(
		ctx,
		query,
		lastError,
		time.Now().UTC(),
		id,
	)
	return err
}

func (c *Client) List(ctx context.Context, queue string, statuses []string, limit int, offset int) ([]Job, error) {
	query, err := dbqueries.Get("list_base")
	if err != nil {
		return nil, err
	}
	where := make([]string, 0, 2)
	args := make([]any, 0, 6)

	if strings.TrimSpace(queue) != "" {
		where = append(where, "queue = ?")
		args = append(args, queue)
	}
	if len(statuses) > 0 {
		placeholders := make([]string, 0, len(statuses))
		for _, status := range statuses {
			placeholders = append(placeholders, "?")
			args = append(args, status)
		}
		where = append(where, fmt.Sprintf("status IN (%s)", strings.Join(placeholders, ", ")))
	}
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}
	query += " ORDER BY run_at ASC, created_at ASC"

	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	if offset < 0 {
		offset = 0
	}
	query += " LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := c.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]Job, 0, limit)
	for rows.Next() {
		var job Job
		if err := rows.Scan(
			&job.ID,
			&job.Queue,
			&job.Name,
			&job.Payload,
			&job.Status,
			&job.Attempt,
			&job.MaxRetries,
			&job.RunAt,
			&job.CreatedAt,
			&job.UpdatedAt,
			&job.LastError,
		); err != nil {
			return nil, err
		}
		out = append(out, job)
	}
	return out, rows.Err()
}

func (c *Client) Get(ctx context.Context, id string) (Job, bool, error) {
	query, err := dbqueries.Get("get_by_id")
	if err != nil {
		return Job{}, false, err
	}
	rows, err := c.db.QueryContext(
		ctx,
		query,
		id,
	)
	if err != nil {
		return Job{}, false, err
	}
	defer rows.Close()
	if !rows.Next() {
		return Job{}, false, nil
	}
	var job Job
	if err := rows.Scan(
		&job.ID,
		&job.Queue,
		&job.Name,
		&job.Payload,
		&job.Status,
		&job.Attempt,
		&job.MaxRetries,
		&job.RunAt,
		&job.CreatedAt,
		&job.UpdatedAt,
		&job.LastError,
	); err != nil {
		return Job{}, false, err
	}
	return job, true, nil
}
