-- name: delete_file_storage_by_object_key_postgres
DELETE FROM file_storages
WHERE object_key = $1

-- name: delete_file_storage_by_object_key_sqlite
DELETE FROM file_storages
WHERE object_key = ?

-- name: insert_file_storage_returning_id_postgres
INSERT INTO file_storages (created_at, updated_at, bucket_name, object_key, file_size, file_hash)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id

-- name: insert_file_storage_sqlite
INSERT INTO file_storages (created_at, updated_at, bucket_name, object_key, file_size, file_hash)
VALUES (?, ?, ?, ?, ?, ?)
