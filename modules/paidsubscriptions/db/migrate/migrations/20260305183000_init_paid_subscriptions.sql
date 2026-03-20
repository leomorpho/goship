-- +goose Up
CREATE TABLE IF NOT EXISTS monthly_subscriptions (
    id BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    product TEXT NOT NULL,
    is_active BOOLEAN NOT NULL,
    paid BOOLEAN NOT NULL,
    is_trial BOOLEAN NOT NULL,
    started_at TIMESTAMPTZ NULL,
    expired_on TIMESTAMPTZ NULL,
    cancelled_at TIMESTAMPTZ NULL,
    paying_profile_id BIGINT NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS monthlysubscription_paying_profile_id_is_active
    ON monthly_subscriptions (paying_profile_id, is_active);

CREATE TABLE IF NOT EXISTS monthly_subscription_benefactors (
    monthly_subscription_id BIGINT NOT NULL,
    profile_id BIGINT NOT NULL,
    PRIMARY KEY (monthly_subscription_id, profile_id)
);

CREATE TABLE IF NOT EXISTS subscription_customers (
    profile_id BIGINT PRIMARY KEY,
    stripe_customer_id TEXT NOT NULL UNIQUE
);

-- +goose Down
DROP TABLE IF EXISTS subscription_customers;
DROP TABLE IF EXISTS monthly_subscription_benefactors;
DROP INDEX IF EXISTS monthlysubscription_paying_profile_id_is_active;
DROP TABLE IF EXISTS monthly_subscriptions;
