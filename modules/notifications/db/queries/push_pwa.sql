-- name: insert_pwa_subscription
INSERT INTO pwa_push_subscriptions (
	created_at, updated_at, endpoint, p256dh, auth, profile_id
) VALUES (?, ?, ?, ?, ?, ?);

-- name: list_pwa_subscriptions_by_profile
SELECT profile_id, endpoint, p256dh, auth
FROM pwa_push_subscriptions
WHERE profile_id = ?;

-- name: delete_pwa_subscription_by_endpoint
DELETE FROM pwa_push_subscriptions
WHERE profile_id = ? AND endpoint = ?;

-- name: count_pwa_subscriptions_by_profile
SELECT COUNT(*)
FROM pwa_push_subscriptions
WHERE profile_id = ?;

-- name: count_pwa_subscription_by_endpoint
SELECT COUNT(*)
FROM pwa_push_subscriptions
WHERE profile_id = ? AND endpoint = ?;

