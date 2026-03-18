-- branch: postgres-only migration branch
-- name: create_schema_migrations_table_postgres
CREATE TABLE IF NOT EXISTS goship_schema_migrations (
	version VARCHAR(255) PRIMARY KEY,
	applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
)

-- branch: sqlite migration branch
-- name: create_schema_migrations_table_sqlite
CREATE TABLE IF NOT EXISTS goship_schema_migrations (
	version TEXT PRIMARY KEY,
	applied_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
)

-- branch: postgres-only migration branch
-- name: select_schema_migration_version_postgres
SELECT 1 FROM goship_schema_migrations WHERE version = $1

-- branch: sqlite migration branch
-- name: select_schema_migration_version_sqlite
SELECT 1 FROM goship_schema_migrations WHERE version = ?

-- branch: postgres-only migration branch
-- name: insert_schema_migration_version_postgres
INSERT INTO goship_schema_migrations (version) VALUES ($1)

-- branch: sqlite migration branch
-- name: insert_schema_migration_version_sqlite
INSERT INTO goship_schema_migrations (version) VALUES (?)

-- branch: postgres-only
-- name: drop_database_postgres
DROP DATABASE 

-- branch: postgres-only
-- name: create_database_postgres
CREATE DATABASE 

-- branch: postgres-only
-- name: create_pgvector_extension_postgres
CREATE EXTENSION IF NOT EXISTS vector
