package profiles

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

func (p *ProfileService) PreferredLanguage(ctx context.Context, userID int) (lang string, ok bool, err error) {
	if p.db == nil {
		return "", false, ErrProfileDBNotConfigured
	}
	if userID <= 0 {
		return "", false, nil
	}

	query := "SELECT preferred_language FROM profiles WHERE user_profile = ? LIMIT 1"
	if strings.EqualFold(strings.TrimSpace(p.dbDialect), "postgres") {
		query = "SELECT preferred_language FROM profiles WHERE user_profile = $1 LIMIT 1"
	}

	var preferred sql.NullString
	if err := p.db.QueryRowContext(ctx, query, userID).Scan(&preferred); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", false, nil
		}
		return "", false, err
	}
	if !preferred.Valid {
		return "", false, nil
	}

	clean := strings.TrimSpace(preferred.String)
	if clean == "" {
		return "", false, nil
	}
	return clean, true, nil
}

func (p *ProfileService) SetPreferredLanguage(ctx context.Context, userID int, lang string) error {
	if p.db == nil {
		return ErrProfileDBNotConfigured
	}
	if userID <= 0 {
		return errors.New("invalid user id")
	}
	clean := strings.TrimSpace(strings.ToLower(lang))
	if clean == "" {
		return errors.New("language is required")
	}

	query := "UPDATE profiles SET preferred_language = ? WHERE user_profile = ?"
	args := []any{clean, userID}
	if strings.EqualFold(strings.TrimSpace(p.dbDialect), "postgres") {
		query = "UPDATE profiles SET preferred_language = $2 WHERE user_profile = $1"
		args = []any{userID, clean}
	}

	result, err := p.db.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}
	if rows, rowsErr := result.RowsAffected(); rowsErr == nil && rows == 0 {
		return fmt.Errorf("profile not found for user id %d", userID)
	}
	return nil
}
