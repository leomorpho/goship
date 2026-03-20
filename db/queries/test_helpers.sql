-- name: insert_test_user_returning_id_postgres
INSERT INTO users (created_at, updated_at, name, email, password, verified)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id

-- name: insert_test_profile_friend_postgres
INSERT INTO profile_friends (profile_id, friend_id) VALUES ($1, $2)
