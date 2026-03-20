package flags

import (
	"context"
	"database/sql"
	"encoding/json"
)

type SQLStore struct {
	db *sql.DB
}

func NewSQLStore(db *sql.DB) *SQLStore {
	return &SQLStore{db: db}
}

func (s *SQLStore) Find(ctx context.Context, key string) (Flag, error) {
	var flag Flag
	var userIDsJSON string
	err := s.db.QueryRowContext(ctx, `
SELECT key, enabled, rollout_pct, COALESCE(user_ids, ''), COALESCE(description, '')
FROM feature_flags
WHERE key = ?`,
		key,
	).Scan(&flag.Key, &flag.Enabled, &flag.RolloutPct, &userIDsJSON, &flag.Description)
	if err != nil {
		return Flag{}, err
	}
	flag.UserIDs = parseUserIDs(userIDsJSON)
	return flag, nil
}

func (s *SQLStore) List(ctx context.Context) ([]Flag, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT key, enabled, rollout_pct, COALESCE(user_ids, ''), COALESCE(description, '')
FROM feature_flags
ORDER BY key`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	flags := make([]Flag, 0)
	for rows.Next() {
		var flag Flag
		var userIDsJSON string
		if err := rows.Scan(&flag.Key, &flag.Enabled, &flag.RolloutPct, &userIDsJSON, &flag.Description); err != nil {
			return nil, err
		}
		flag.UserIDs = parseUserIDs(userIDsJSON)
		flags = append(flags, flag)
	}
	return flags, rows.Err()
}

func (s *SQLStore) Create(ctx context.Context, flag Flag) error {
	_, err := s.db.ExecContext(ctx, `
INSERT INTO feature_flags (key, enabled, rollout_pct, user_ids, description, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
		flag.Key,
		flag.Enabled,
		flag.RolloutPct,
		encodeUserIDs(flag.UserIDs),
		flag.Description,
	)
	return err
}

func (s *SQLStore) Update(ctx context.Context, flag Flag) error {
	_, err := s.db.ExecContext(ctx, `
UPDATE feature_flags
SET enabled = ?, rollout_pct = ?, user_ids = ?, description = ?, updated_at = CURRENT_TIMESTAMP
WHERE key = ?`,
		flag.Enabled,
		flag.RolloutPct,
		encodeUserIDs(flag.UserIDs),
		flag.Description,
		flag.Key,
	)
	return err
}

func (s *SQLStore) Delete(ctx context.Context, key string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM feature_flags WHERE key = ?`, key)
	return err
}

func encodeUserIDs(userIDs []int64) string {
	if len(userIDs) == 0 {
		return ""
	}
	payload, err := json.Marshal(userIDs)
	if err != nil {
		return ""
	}
	return string(payload)
}

func parseUserIDs(raw string) []int64 {
	if raw == "" {
		return nil
	}
	out := make([]int64, 0)
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil
	}
	return out
}
