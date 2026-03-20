-- name: select_daily_update_audience
SELECT p.id
FROM profiles p
WHERE EXISTS (
	SELECT 1
	FROM notification_permissions np
	WHERE np.profile_id = p.id
	  AND np.permission = ?
)
AND NOT EXISTS (
	SELECT 1
	FROM sent_emails se
	WHERE se.profile_sent_emails = p.id
	  AND se.created_at >= ?
	  AND se.type IN (?, ?)
);

-- name: select_partner_update_audience
SELECT p.id
FROM profiles p
WHERE EXISTS (
	SELECT 1
	FROM notification_permissions np_partner
	WHERE np_partner.profile_id = p.id
	  AND np_partner.permission = ?
)
AND NOT EXISTS (
	SELECT 1
	FROM notification_permissions np_daily
	WHERE np_daily.profile_id = p.id
	  AND np_daily.permission = ?
)
AND EXISTS (
	SELECT 1
	FROM notifications n
	WHERE n.profile_notifications = p.id
	  AND n.read = ?
	  AND n.type = ?
)
AND NOT EXISTS (
	SELECT 1
	FROM sent_emails se
	WHERE se.profile_sent_emails = p.id
	  AND se.created_at >= ?
	  AND se.type IN (?, ?)
);

-- name: insert_sent_email
INSERT INTO sent_emails (created_at, updated_at, type, profile_sent_emails)
VALUES (?, ?, ?, ?);

-- name: delete_notifications_before
DELETE FROM notifications
WHERE created_at < ?;

-- name: delete_daily_notifications_before
DELETE FROM notifications
WHERE created_at < ? AND type = ?;

