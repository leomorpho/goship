-- name: get_auth_user_record_by_email_postgres
SELECT id, name, email, password, verified
FROM users
WHERE email = $1
LIMIT 1;

-- name: get_auth_user_record_by_email_sqlite
SELECT id, name, email, password, verified
FROM users
WHERE email = ?
LIMIT 1;

-- name: get_auth_identity_by_user_id_postgres
SELECT u.id, u.name, u.email, p.id, p.fully_onboarded
FROM users u
LEFT JOIN profiles p ON p.user_profile = u.id
WHERE u.id = $1
LIMIT 1;

-- name: get_auth_identity_by_user_id_sqlite
SELECT u.id, u.name, u.email, p.id, p.fully_onboarded
FROM users u
LEFT JOIN profiles p ON p.user_profile = u.id
WHERE u.id = ?
LIMIT 1;

-- name: get_user_display_name_by_user_id_postgres
SELECT name
FROM users
WHERE id = $1
LIMIT 1;

-- name: get_user_display_name_by_user_id_sqlite
SELECT name
FROM users
WHERE id = ?
LIMIT 1;

-- name: insert_last_seen_online_postgres
INSERT INTO last_seen_onlines (seen_at, user_last_seen_at)
VALUES ($1, $2);

-- name: insert_last_seen_online_sqlite
INSERT INTO last_seen_onlines (seen_at, user_last_seen_at)
VALUES (?, ?);

-- name: update_user_password_hash_by_user_id_postgres
UPDATE users
SET password = $1
WHERE id = $2;

-- name: update_user_password_hash_by_user_id_sqlite
UPDATE users
SET password = ?
WHERE id = ?;

-- name: update_user_display_name_by_user_id_postgres
UPDATE users
SET name = $1
WHERE id = $2;

-- name: update_user_display_name_by_user_id_sqlite
UPDATE users
SET name = ?
WHERE id = ?;

-- name: mark_user_verified_by_user_id_postgres
UPDATE users
SET verified = true
WHERE id = $1;

-- name: mark_user_verified_by_user_id_sqlite
UPDATE users
SET verified = 1
WHERE id = ?;

-- name: insert_password_token_postgres
INSERT INTO password_tokens (hash, created_at, password_token_user)
VALUES ($1, $2, $3)
RETURNING id;

-- name: insert_password_token_sqlite
INSERT INTO password_tokens (hash, created_at, password_token_user)
VALUES (?, ?, ?)
RETURNING id;

-- name: get_password_token_hash_postgres
SELECT hash
FROM password_tokens
WHERE id = $1
  AND password_token_user = $2
  AND created_at >= $3
LIMIT 1;

-- name: get_password_token_hash_sqlite
SELECT hash
FROM password_tokens
WHERE id = ?
  AND password_token_user = ?
  AND created_at >= ?
LIMIT 1;

-- name: delete_password_tokens_by_user_id_postgres
DELETE FROM password_tokens
WHERE password_token_user = $1;

-- name: delete_password_tokens_by_user_id_sqlite
DELETE FROM password_tokens
WHERE password_token_user = ?;
