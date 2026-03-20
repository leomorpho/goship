package auditlog

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

type Store interface {
	Insert(ctx context.Context, entry Log) error
	List(ctx context.Context, filters ListFilters) ([]Log, error)
}

type SQLStore struct {
	db *sql.DB
}

func NewSQLStore(db *sql.DB) *SQLStore {
	return &SQLStore{db: db}
}

func (s *SQLStore) Insert(ctx context.Context, entry Log) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("audit log store is not configured")
	}

	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO audit_logs (user_id, action, resource_type, resource_id, changes, ip_address, user_agent)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		entry.UserID,
		entry.Action,
		entry.ResourceType,
		entry.ResourceID,
		nullIfEmpty(entry.Changes),
		nullIfEmpty(entry.IPAddress),
		nullIfEmpty(entry.UserAgent),
	)
	return err
}

func (s *SQLStore) List(ctx context.Context, filters ListFilters) ([]Log, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("audit log store is not configured")
	}

	query := `SELECT id, user_id, action, resource_type, resource_id, COALESCE(changes, ''), COALESCE(ip_address, ''), COALESCE(user_agent, ''), created_at
		FROM audit_logs`
	args := make([]any, 0, 5)
	where := make([]string, 0, 4)

	if filters.UserID != nil {
		where = append(where, "user_id = ?")
		args = append(args, *filters.UserID)
	}
	if filters.Action != "" {
		where = append(where, "action = ?")
		args = append(args, filters.Action)
	}
	if filters.ResourceType != "" {
		where = append(where, "resource_type = ?")
		args = append(args, filters.ResourceType)
	}
	if filters.ResourceID != "" {
		where = append(where, "resource_id = ?")
		args = append(args, filters.ResourceID)
	}
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}

	limit := filters.Limit
	if limit <= 0 {
		limit = 100
	}
	query += " ORDER BY created_at DESC, id DESC LIMIT ?"
	args = append(args, limit)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	logs := make([]Log, 0, limit)
	for rows.Next() {
		var entry Log
		var userID sql.NullInt64
		if err := rows.Scan(
			&entry.ID,
			&userID,
			&entry.Action,
			&entry.ResourceType,
			&entry.ResourceID,
			&entry.Changes,
			&entry.IPAddress,
			&entry.UserAgent,
			&entry.CreatedAt,
		); err != nil {
			return nil, err
		}
		if userID.Valid {
			entry.UserID = &userID.Int64
		}
		logs = append(logs, entry)
	}
	return logs, rows.Err()
}

func nullIfEmpty(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}
