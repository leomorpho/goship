-- name: insert_notification
INSERT INTO notifications (
	created_at, updated_at, type, title, text, link, read, read_at, profile_id,
	profile_id_who_caused_notification, resource_id_tied_to_notif, read_in_notifications_center
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: select_notifications_by_profile_base
SELECT id, type, title, text, link, created_at, read, read_at,
	   profile_id, profile_id_who_caused_notification, resource_id_tied_to_notif, read_in_notifications_center
FROM notifications
WHERE profile_id = ?

-- name: select_notification_type_by_id
SELECT type FROM notifications WHERE id = ?;

-- name: delete_notification_by_id
DELETE FROM notifications WHERE id = ?;

-- name: mark_notification_read_by_id
UPDATE notifications
SET read = ?, read_at = ?, updated_at = ?
WHERE id = ?;

-- name: mark_all_notifications_read_by_profile
UPDATE notifications
SET read = ?, read_at = ?, updated_at = ?
WHERE profile_id = ?;

-- name: mark_notification_unread_by_id
UPDATE notifications
SET read = ?, read_at = NULL, updated_at = ?
WHERE id = ?;

-- name: count_notifications_for_type_since_base
SELECT COUNT(*)
FROM notifications
WHERE type = ? AND created_at > ?

-- name: count_notification_belongs_to_profile
SELECT COUNT(*)
FROM notifications
WHERE id = ? AND profile_id = ?;

