package twofa

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
)

type SQLStore struct {
	db      *sql.DB
	dialect string
}

func NewSQLStore(db *sql.DB, dialect string) *SQLStore {
	return &SQLStore{db: db, dialect: dialect}
}

func (s *SQLStore) GetSettingsByUserID(ctx context.Context, userID int) (UserSettings, error) {
	query := `SELECT id, email, COALESCE(totp_enabled, false), COALESCE(totp_secret, ''), COALESCE(totp_backup_codes, '[]')
FROM users WHERE id = ` + placeholder(s.dialect, 1) + ` LIMIT 1`
	var settings UserSettings
	var rawBackup string
	err := s.db.QueryRowContext(ctx, query, userID).Scan(
		&settings.UserID,
		&settings.Email,
		&settings.TOTPEnabled,
		&settings.EncryptedSecret,
		&rawBackup,
	)
	if err != nil {
		return UserSettings{}, err
	}
	if strings.TrimSpace(rawBackup) == "" {
		rawBackup = "[]"
	}
	if err := json.Unmarshal([]byte(rawBackup), &settings.BackupCodeHashes); err != nil {
		return UserSettings{}, err
	}
	return settings, nil
}

func (s *SQLStore) Enable(ctx context.Context, userID int, encryptedSecret string, backupCodeHashes []string) error {
	raw, err := json.Marshal(backupCodeHashes)
	if err != nil {
		return err
	}
	query := `UPDATE users SET totp_secret = ` + placeholder(s.dialect, 1) + `,
totp_enabled = true,
totp_backup_codes = ` + placeholder(s.dialect, 2) + `
WHERE id = ` + placeholder(s.dialect, 3)
	_, err = s.db.ExecContext(ctx, query, encryptedSecret, string(raw), userID)
	return err
}

func (s *SQLStore) ReplaceBackupCodes(ctx context.Context, userID int, backupCodeHashes []string) error {
	raw, err := json.Marshal(backupCodeHashes)
	if err != nil {
		return err
	}
	query := `UPDATE users SET totp_backup_codes = ` + placeholder(s.dialect, 1) + ` WHERE id = ` + placeholder(s.dialect, 2)
	_, err = s.db.ExecContext(ctx, query, string(raw), userID)
	return err
}

func (s *SQLStore) UseBackupCode(ctx context.Context, userID int, remainingBackupCodeHashes []string) error {
	return s.ReplaceBackupCodes(ctx, userID, remainingBackupCodeHashes)
}

func placeholder(dialect string, index int) string {
	if strings.Contains(strings.ToLower(strings.TrimSpace(dialect)), "post") {
		return fmt.Sprintf("$%d", index)
	}
	return "?"
}
