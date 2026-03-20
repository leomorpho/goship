-- name: create_subscription
INSERT INTO monthly_subscriptions
	(created_at, updated_at, product, is_active, paid, is_trial, started_at, expired_on, cancelled_at, paying_profile_id)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: create_subscription_benefactor_link
INSERT INTO monthly_subscription_benefactors (monthly_subscription_id, profile_id)
SELECT id, ?
FROM monthly_subscriptions
WHERE paying_profile_id = ? AND is_active = ?
ORDER BY id DESC
LIMIT 1;

-- name: deactivate_expired_subscriptions
UPDATE monthly_subscriptions
SET is_active = ?, expired_on = ?
WHERE is_active = ? AND expired_on IS NOT NULL AND expired_on <= ?;

-- name: update_active_plan_existing
UPDATE monthly_subscriptions
SET
	updated_at = ?,
	product = ?,
	paid = ?,
	is_trial = ?,
	is_active = ?,
	started_at = ?,
	expired_on = ?,
	cancelled_at = NULL
WHERE paying_profile_id = ? AND is_active = ?;

-- name: get_currently_active_product
SELECT product, expired_on, is_trial
FROM monthly_subscriptions
WHERE is_active = ? AND paying_profile_id = ?
ORDER BY id DESC
LIMIT 1;

-- name: get_stripe_customer_id_by_profile
SELECT stripe_customer_id
FROM subscription_customers
WHERE profile_id = ?;

-- name: insert_stripe_customer_id
INSERT INTO subscription_customers (profile_id, stripe_customer_id)
VALUES (?, ?);

-- name: update_stripe_customer_id
UPDATE subscription_customers
SET stripe_customer_id = ?
WHERE profile_id = ?;

-- name: get_profile_id_by_stripe_customer
SELECT profile_id
FROM subscription_customers
WHERE stripe_customer_id = ?;

-- name: get_latest_expired_on
SELECT expired_on
FROM monthly_subscriptions
WHERE paying_profile_id = ? AND is_active = ?
ORDER BY id DESC
LIMIT 1;

-- name: update_expired_on_for_active
UPDATE monthly_subscriptions
SET expired_on = ?
WHERE paying_profile_id = ? AND is_active = ?;

-- name: cancel_or_renew_clear
UPDATE monthly_subscriptions
SET cancelled_at = NULL, expired_on = NULL
WHERE paying_profile_id = ? AND is_active = ?;

-- name: cancel_or_renew_set
UPDATE monthly_subscriptions
SET expired_on = ?, cancelled_at = ?
WHERE paying_profile_id = ? AND is_active = ?;

-- name: update_to_free
UPDATE monthly_subscriptions
SET expired_on = ?, is_active = ?
WHERE paying_profile_id = ? AND is_active = ?;

-- name: count_active_subscriptions
SELECT COUNT(*)
FROM monthly_subscriptions
WHERE paying_profile_id = ? AND is_active = ?;
