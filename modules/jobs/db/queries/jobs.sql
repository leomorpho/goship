-- name: insert_job
INSERT INTO goship_jobs
	(id, queue, name, payload, status, run_at, attempt, max_retries, created_at, updated_at)
VALUES (?, ?, ?, ?, 'queued', ?, 0, ?, ?, ?);

-- name: claim_next_job
UPDATE goship_jobs
SET status = 'running', locked_by = ?, lock_until = ?, updated_at = ?
WHERE id = (
	SELECT id FROM goship_jobs
	WHERE status = 'queued'
	  AND run_at <= ?
	  AND (lock_until IS NULL OR lock_until <= ?)
	ORDER BY run_at ASC, created_at ASC
	LIMIT 1
);

-- name: select_claimed_job
SELECT id, queue, name, payload, status, attempt, max_retries, run_at, created_at, updated_at, COALESCE(last_error, '')
FROM goship_jobs
WHERE status = 'running' AND locked_by = ?
ORDER BY updated_at DESC
LIMIT 1;

-- name: mark_done
UPDATE goship_jobs
SET status = 'done', locked_by = NULL, lock_until = NULL, updated_at = ?
WHERE id = ?;

-- name: mark_retry
UPDATE goship_jobs
SET status = 'queued',
	attempt = attempt + 1,
	run_at = ?,
	last_error = ?,
	locked_by = NULL,
	lock_until = NULL,
	updated_at = ?
WHERE id = ?;

-- name: mark_failed
UPDATE goship_jobs
SET status = 'failed',
	attempt = attempt + 1,
	last_error = ?,
	locked_by = NULL,
	lock_until = NULL,
	updated_at = ?
WHERE id = ?;

-- name: list_base
SELECT id, queue, name, payload, status, attempt, max_retries, run_at, created_at, updated_at, COALESCE(last_error, '')
FROM goship_jobs

-- name: get_by_id
SELECT id, queue, name, payload, status, attempt, max_retries, run_at, created_at, updated_at, COALESCE(last_error, '')
FROM goship_jobs
WHERE id = ?
LIMIT 1;

