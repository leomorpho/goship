-- name: find_list_id_base
SELECT id
FROM email_subscription_types
WHERE name = ?

-- name: find_list_id_active_suffix
AND active = ?;

-- name: insert_list
INSERT INTO email_subscription_types (created_at, updated_at, name, active)
VALUES (?, ?, ?, ?);

-- name: find_subscription_by_email
SELECT id, email, verified, confirmation_code, latitude, longitude
FROM email_subscriptions
WHERE email = ?
LIMIT 1;

-- name: find_subscription_by_email_and_code
SELECT id, email, verified, confirmation_code, latitude, longitude
FROM email_subscriptions
WHERE email = ? AND confirmation_code = ?
LIMIT 1;

-- name: insert_subscription
INSERT INTO email_subscriptions (created_at, updated_at, email, verified, confirmation_code, latitude, longitude)
VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: find_subscription_link
SELECT 1
FROM email_subscription_subscriptions
WHERE email_subscription_id = ? AND email_subscription_type_id = ?
LIMIT 1;

-- name: insert_subscription_link
INSERT INTO email_subscription_subscriptions (email_subscription_id, email_subscription_type_id)
VALUES (?, ?);

-- name: update_subscription_location
UPDATE email_subscriptions
SET updated_at = ?, latitude = ?, longitude = ?
WHERE id = ?;

-- name: delete_subscription_link
DELETE FROM email_subscription_subscriptions
WHERE email_subscription_id = ? AND email_subscription_type_id = ?;

-- name: count_subscription_links
SELECT COUNT(*)
FROM email_subscription_subscriptions
WHERE email_subscription_id = ?;

-- name: delete_subscription_by_id
DELETE FROM email_subscriptions
WHERE id = ?;

-- name: rotate_subscription_code
UPDATE email_subscriptions
SET updated_at = ?, confirmation_code = ?
WHERE id = ?;

-- name: find_subscription_by_confirmation_code
SELECT id, verified
FROM email_subscriptions
WHERE confirmation_code = ?
LIMIT 1;

-- name: mark_subscription_verified
UPDATE email_subscriptions
SET updated_at = ?, verified = ?, confirmation_code = ?
WHERE id = ?;
