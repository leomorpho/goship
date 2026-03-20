-- +goose Up
CREATE TABLE IF NOT EXISTS goship_jobs (
  id TEXT PRIMARY KEY,
  queue TEXT NOT NULL,
  name TEXT NOT NULL,
  payload TEXT NOT NULL,
  status TEXT NOT NULL,
  run_at TIMESTAMPTZ NOT NULL,
  attempt INTEGER NOT NULL DEFAULT 0,
  max_retries INTEGER NOT NULL DEFAULT 0,
  last_error TEXT,
  locked_by TEXT,
  lock_until TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS goship_jobs_status_run_at_idx
  ON goship_jobs (status, run_at, created_at);

CREATE INDEX IF NOT EXISTS goship_jobs_queue_status_idx
  ON goship_jobs (queue, status, run_at, created_at);

-- +goose Down
DROP INDEX IF EXISTS goship_jobs_queue_status_idx;
DROP INDEX IF EXISTS goship_jobs_status_run_at_idx;
DROP TABLE IF EXISTS goship_jobs;
