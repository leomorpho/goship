-- name: insert_user_returning_id_postgres
INSERT INTO users (created_at, updated_at, name, email, password, verified)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id;

-- name: insert_user_returning_id_sqlite
INSERT INTO users (created_at, updated_at, name, email, password, verified)
VALUES (?, ?, ?, ?, ?, ?)
RETURNING id;

-- name: insert_profile_returning_id_postgres
INSERT INTO profiles (
	created_at,
	updated_at,
	bio,
	birthdate,
	age,
	fully_onboarded,
	phone_verified,
	user_profile
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id;

-- name: insert_profile_returning_id_sqlite
INSERT INTO profiles (
	created_at,
	updated_at,
	bio,
	birthdate,
	age,
	fully_onboarded,
	phone_verified,
	user_profile
)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
RETURNING id;

-- name: insert_profile_returning_id_sqlite_legacy
INSERT INTO profiles (
	created_at,
	updated_at,
	user_profile,
	fully_onboarded
)
VALUES (?, ?, ?, ?)
RETURNING id;
