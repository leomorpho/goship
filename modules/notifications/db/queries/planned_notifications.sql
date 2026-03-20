-- name: list_profiles_for_permission
SELECT np.profile_id, MAX(nt.updated_at) AS nt_updated_at
FROM notification_permissions np
LEFT JOIN notification_times nt
  ON nt.profile_id = np.profile_id AND nt.type = ?
WHERE np.permission = ?
GROUP BY np.profile_id;

-- name: delete_stale_last_seen_before
DELETE FROM last_seen_onlines
WHERE seen_at <= ?;

-- name: list_last_seen_for_profile
SELECT lso.seen_at
FROM last_seen_onlines lso
JOIN users u ON lso.user_last_seen_at = u.id
JOIN profiles p ON p.user_profile = u.id
WHERE p.id = ?;

-- name: upsert_notification_time
INSERT INTO notification_times (
	created_at, updated_at, type, send_minute, profile_id
) VALUES (?, ?, ?, ?, ?)
ON CONFLICT(profile_id, type) DO UPDATE SET
	send_minute = excluded.send_minute,
	updated_at = excluded.updated_at;

-- name: list_profile_ids_can_get_planned_notification_now_base
SELECT nt.profile_id
FROM notification_times nt
WHERE nt.type = ?
  AND nt.send_minute >= 0
  AND nt.send_minute <= ?
  AND NOT EXISTS (
	  SELECT 1
	  FROM notifications n
	  WHERE n.profile_id = nt.profile_id
		AND n.created_at >= ?
		AND n.type = ?
  )

