CREATE TABLE IF NOT EXISTS feature_flags (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    key TEXT NOT NULL UNIQUE,
    enabled BOOLEAN NOT NULL DEFAULT FALSE,
    rollout_pct INTEGER NOT NULL DEFAULT 0,
    user_ids TEXT,
    description TEXT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_feature_flags_key ON feature_flags(key);

DROP INDEX IF EXISTS idx_feature_flags_key;
DROP TABLE IF EXISTS feature_flags;
