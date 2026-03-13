package twofa

import "context"

type Store interface {
	GetSettingsByUserID(ctx context.Context, userID int) (UserSettings, error)
	Enable(ctx context.Context, userID int, encryptedSecret string, backupCodeHashes []string) error
	ReplaceBackupCodes(ctx context.Context, userID int, backupCodeHashes []string) error
	UseBackupCode(ctx context.Context, userID int, remainingBackupCodeHashes []string) error
}

type UserSettings struct {
	UserID           int
	Email            string
	TOTPEnabled      bool
	EncryptedSecret  string
	BackupCodeHashes []string
}
