-- +goose Up
CREATE TABLE IF NOT EXISTS starter_bootstrap (
    id INTEGER PRIMARY KEY,
    app_name TEXT NOT NULL,
    created_at TEXT NOT NULL
);

INSERT INTO starter_bootstrap (id, app_name, created_at)
VALUES (1, 'GoShip Starter', CURRENT_TIMESTAMP)
ON CONFLICT(id) DO NOTHING;

-- +goose Down
DROP TABLE IF EXISTS starter_bootstrap;
