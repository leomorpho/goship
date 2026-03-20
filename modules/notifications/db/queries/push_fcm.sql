-- name: insert_fcm_subscription
INSERT INTO fcm_subscriptions (
	created_at, updated_at, token, profile_id
) VALUES (?, ?, ?, ?);

-- name: list_fcm_subscriptions_by_profile
SELECT profile_id, token
FROM fcm_subscriptions
WHERE profile_id = ?;

-- name: delete_fcm_subscription_by_token
DELETE FROM fcm_subscriptions
WHERE profile_id = ? AND token = ?;

-- name: count_fcm_subscriptions_by_profile
SELECT COUNT(*)
FROM fcm_subscriptions
WHERE profile_id = ?;

-- name: count_fcm_subscription_by_token
SELECT COUNT(*)
FROM fcm_subscriptions
WHERE profile_id = ? AND token = ?;

