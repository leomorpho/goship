-- +goose Up
CREATE TABLE IF NOT EXISTS notifications (
    id BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    type TEXT NOT NULL,
    title TEXT NOT NULL,
    text TEXT NOT NULL,
    link TEXT NULL,
    read BOOLEAN NOT NULL DEFAULT FALSE,
    read_at TIMESTAMPTZ NULL,
    profile_id BIGINT NOT NULL,
    profile_id_who_caused_notification BIGINT NOT NULL DEFAULT 0,
    resource_id_tied_to_notif BIGINT NOT NULL DEFAULT 0,
    read_in_notifications_center BOOLEAN NOT NULL DEFAULT TRUE
);

CREATE INDEX IF NOT EXISTS notifications_profile_id_created_at_idx
    ON notifications (profile_id, created_at DESC);

CREATE INDEX IF NOT EXISTS notifications_type_created_at_idx
    ON notifications (type, created_at DESC);

CREATE TABLE IF NOT EXISTS notification_permissions (
    id BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    permission TEXT NOT NULL,
    platform TEXT NOT NULL,
    profile_id BIGINT NOT NULL,
    token TEXT NOT NULL,
    UNIQUE(profile_id, permission, platform)
);

CREATE INDEX IF NOT EXISTS notification_permissions_profile_platform_idx
    ON notification_permissions (profile_id, platform);

CREATE TABLE IF NOT EXISTS phone_verification_codes (
    id BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    code TEXT NOT NULL,
    profile_id BIGINT NOT NULL
);

CREATE INDEX IF NOT EXISTS phone_verification_codes_profile_id_idx
    ON phone_verification_codes (profile_id, created_at DESC);

CREATE TABLE IF NOT EXISTS pwa_push_subscriptions (
    id BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    endpoint TEXT NOT NULL,
    p256dh TEXT NOT NULL,
    auth TEXT NOT NULL,
    profile_id BIGINT NOT NULL
);

CREATE INDEX IF NOT EXISTS pwa_push_subscriptions_profile_endpoint_idx
    ON pwa_push_subscriptions (profile_id, endpoint);

CREATE TABLE IF NOT EXISTS fcm_subscriptions (
    id BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    token TEXT NOT NULL,
    profile_id BIGINT NOT NULL
);

CREATE INDEX IF NOT EXISTS fcm_subscriptions_profile_token_idx
    ON fcm_subscriptions (profile_id, token);

-- +goose Down
DROP INDEX IF EXISTS fcm_subscriptions_profile_token_idx;
DROP TABLE IF EXISTS fcm_subscriptions;
DROP INDEX IF EXISTS pwa_push_subscriptions_profile_endpoint_idx;
DROP TABLE IF EXISTS pwa_push_subscriptions;
DROP INDEX IF EXISTS phone_verification_codes_profile_id_idx;
DROP TABLE IF EXISTS phone_verification_codes;
DROP INDEX IF EXISTS notification_permissions_profile_platform_idx;
DROP TABLE IF EXISTS notification_permissions;
DROP INDEX IF EXISTS notifications_type_created_at_idx;
DROP INDEX IF EXISTS notifications_profile_id_created_at_idx;
DROP TABLE IF EXISTS notifications;
