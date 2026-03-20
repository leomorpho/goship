-- name: delete_permissions_by_profile_base
DELETE FROM notification_permissions WHERE profile_id = ?

-- name: list_permissions_by_profile
SELECT permission, platform, token
FROM notification_permissions
WHERE profile_id = ?;

-- name: insert_or_upsert_permission
INSERT INTO notification_permissions (
	created_at, updated_at, permission, platform, profile_id, token
) VALUES (?, ?, ?, ?, ?, ?)
ON CONFLICT(profile_id, permission, platform) DO UPDATE SET
	updated_at = excluded.updated_at,
	token = excluded.token;

-- name: delete_permission_base
DELETE FROM notification_permissions
WHERE profile_id = ? AND permission = ?

-- name: count_permissions_for_platform
SELECT COUNT(*)
FROM notification_permissions
WHERE profile_id = ? AND platform = ?;

