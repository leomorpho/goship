package twofa

import (
	"context"
	"testing"
)

type fakeStore struct {
	settings UserSettings
	enabled  bool
	hashes   []string
}

func (f *fakeStore) GetSettingsByUserID(context.Context, int) (UserSettings, error) {
	return f.settings, nil
}

func (f *fakeStore) Enable(_ context.Context, _ int, _ string, backupCodeHashes []string) error {
	f.enabled = true
	f.hashes = append([]string{}, backupCodeHashes...)
	return nil
}

func (f *fakeStore) ReplaceBackupCodes(_ context.Context, _ int, backupCodeHashes []string) error {
	f.hashes = append([]string{}, backupCodeHashes...)
	return nil
}

func (f *fakeStore) UseBackupCode(_ context.Context, _ int, remainingBackupCodeHashes []string) error {
	f.hashes = append([]string{}, remainingBackupCodeHashes...)
	return nil
}

func TestGenerateBackupCodes_CountAndPrefix(t *testing.T) {
	svc := NewService(&fakeStore{}, "GoShip", "secret")
	codes := svc.GenerateBackupCodes()
	if len(codes) != 10 {
		t.Fatalf("expected 10 codes, got %d", len(codes))
	}
	for _, code := range codes {
		if len(code) != len("BK-XXXX-XXXX") || code[:3] != "BK-" {
			t.Fatalf("unexpected code format: %q", code)
		}
	}
}

func TestEnableAndValidateBackupCode(t *testing.T) {
	store := &fakeStore{}
	svc := NewService(store, "GoShip", "secret")
	codes := []string{"BK-ABCD-EFGH"}
	if err := svc.Enable(context.Background(), 1, "secret-key", codes); err != nil {
		t.Fatalf("Enable() error = %v", err)
	}
	if !store.enabled || len(store.hashes) != 1 {
		t.Fatalf("expected hashed backup codes to be stored")
	}
	store.settings = UserSettings{
		UserID:           1,
		TOTPEnabled:      true,
		EncryptedSecret:  mustEncrypt(t, "secret", "secret-key"),
		BackupCodeHashes: append([]string{}, store.hashes...),
	}
	valid, err := svc.ValidateStoredCode(context.Background(), 1, "BK-ABCD-EFGH")
	if err != nil {
		t.Fatalf("ValidateStoredCode() error = %v", err)
	}
	if !valid {
		t.Fatal("expected backup code to validate")
	}
	if len(store.hashes) != 0 {
		t.Fatalf("expected used backup code to be removed, got %d remaining", len(store.hashes))
	}
}

func mustEncrypt(t *testing.T, secretKey, plaintext string) string {
	t.Helper()
	value, err := encryptSecret(secretKey, plaintext)
	if err != nil {
		t.Fatalf("encryptSecret() error = %v", err)
	}
	return value
}
