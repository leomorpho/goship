package sql

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/leomorpho/goship/db/ent"
)

type Config struct {
	EntClient *ent.Client
}

type Client struct {
	entClient *ent.Client
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
	if cfg.EntClient == nil {
		return nil, errors.New("sql jobs driver requires Ent client")
	}
	client := &Client{entClient: cfg.EntClient}
	if err := client.ensureSchema(context.Background()); err != nil {
		return nil, err
	}
	return client, nil
}

func (c *Client) ensureSchema(ctx context.Context) error {
	_, err := c.entClient.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS goship_jobs (
  id TEXT PRIMARY KEY,
  queue TEXT NOT NULL,
  name TEXT NOT NULL,
  payload TEXT NOT NULL,
  status TEXT NOT NULL,
  run_at TIMESTAMP NOT NULL,
  attempt INTEGER NOT NULL DEFAULT 0,
  max_retries INTEGER NOT NULL DEFAULT 0,
  last_error TEXT,
  locked_by TEXT,
  lock_until TIMESTAMP,
  created_at TIMESTAMP NOT NULL,
  updated_at TIMESTAMP NOT NULL
);`)
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
	_, err := c.entClient.ExecContext(
		ctx,
		`INSERT INTO goship_jobs
		(id, queue, name, payload, status, run_at, attempt, max_retries, created_at, updated_at)
		VALUES (?, ?, ?, ?, 'queued', ?, 0, ?, ?, ?)`,
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
	res, err := c.entClient.ExecContext(
		ctx,
		`UPDATE goship_jobs
		SET status = 'running', locked_by = ?, lock_until = ?, updated_at = ?
		WHERE id = (
			SELECT id FROM goship_jobs
			WHERE status = 'queued'
			  AND run_at <= ?
			  AND (lock_until IS NULL OR lock_until <= ?)
			ORDER BY run_at ASC, created_at ASC
			LIMIT 1
		)`,
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

	rows, err := c.entClient.QueryContext(
		ctx,
		`SELECT id, queue, name, payload, status, attempt, max_retries, run_at, created_at, updated_at, COALESCE(last_error, '')
		FROM goship_jobs
		WHERE status = 'running' AND locked_by = ?
		ORDER BY updated_at DESC
		LIMIT 1`,
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
	_, err := c.entClient.ExecContext(
		ctx,
		`UPDATE goship_jobs
		SET status = 'done', locked_by = NULL, lock_until = NULL, updated_at = ?
		WHERE id = ?`,
		time.Now().UTC(),
		id,
	)
	return err
}

func (c *Client) MarkRetry(ctx context.Context, id string, runAt time.Time, lastError string) error {
	_, err := c.entClient.ExecContext(
		ctx,
		`UPDATE goship_jobs
		SET status = 'queued',
		    attempt = attempt + 1,
		    run_at = ?,
		    last_error = ?,
		    locked_by = NULL,
		    lock_until = NULL,
		    updated_at = ?
		WHERE id = ?`,
		runAt.UTC(),
		lastError,
		time.Now().UTC(),
		id,
	)
	return err
}

func (c *Client) MarkFailed(ctx context.Context, id string, lastError string) error {
	_, err := c.entClient.ExecContext(
		ctx,
		`UPDATE goship_jobs
		SET status = 'failed',
		    attempt = attempt + 1,
		    last_error = ?,
		    locked_by = NULL,
		    lock_until = NULL,
		    updated_at = ?
		WHERE id = ?`,
		lastError,
		time.Now().UTC(),
		id,
	)
	return err
}

func (c *Client) List(ctx context.Context, queue string, statuses []string, limit int, offset int) ([]Job, error) {
	query := `SELECT id, queue, name, payload, status, attempt, max_retries, run_at, created_at, updated_at, COALESCE(last_error, '')
	FROM goship_jobs`
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

	rows, err := c.entClient.QueryContext(ctx, query, args...)
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
	rows, err := c.entClient.QueryContext(
		ctx,
		`SELECT id, queue, name, payload, status, attempt, max_retries, run_at, created_at, updated_at, COALESCE(last_error, '')
		FROM goship_jobs WHERE id = ? LIMIT 1`,
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
