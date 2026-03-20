package twofa

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"image/png"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/pquerna/otp/totp"
	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	store     Store
	issuer    string
	secretKey string
}

func NewService(store Store, issuer, secretKey string) *Service {
	return &Service{
		store:     store,
		issuer:    issuer,
		secretKey: secretKey,
	}
}

func (s *Service) IsEnabled(ctx context.Context, userID int) (bool, error) {
	settings, err := s.store.GetSettingsByUserID(ctx, userID)
	if err != nil {
		return false, err
	}
	return settings.TOTPEnabled, nil
}

func (s *Service) BeginPendingLogin(ctx echo.Context, userID int) error {
	return SetPendingUserCookie(ctx, s.secretKey, userID)
}

func (s *Service) GenerateSecret(accountName string) (secret, qrCodeDataURL string, err error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      s.issuer,
		AccountName: accountName,
	})
	if err != nil {
		return "", "", err
	}
	image, err := key.Image(256, 256)
	if err != nil {
		return "", "", err
	}
	var buf strings.Builder
	encoder := base64.NewEncoder(base64.StdEncoding, stringWriter{builder: &buf})
	if err := png.Encode(encoder, image); err != nil {
		return "", "", err
	}
	if err := encoder.Close(); err != nil {
		return "", "", err
	}
	return key.Secret(), "data:image/png;base64," + buf.String(), nil
}

func (s *Service) ValidateCode(secret, code string) bool {
	return totp.Validate(strings.TrimSpace(code), strings.TrimSpace(secret))
}

func (s *Service) GenerateBackupCodes() []string {
	out := make([]string, 0, 10)
	for len(out) < 10 {
		code := "BK-" + randomAlphaNumeric(4) + "-" + randomAlphaNumeric(4)
		out = append(out, code)
	}
	return out
}

func (s *Service) HashBackupCode(code string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(strings.TrimSpace(code)), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func (s *Service) Enable(ctx context.Context, userID int, secret string, backupCodes []string) error {
	encryptedSecret, err := encryptSecret(s.secretKey, secret)
	if err != nil {
		return err
	}
	hashes := make([]string, 0, len(backupCodes))
	for _, code := range backupCodes {
		hash, err := s.HashBackupCode(code)
		if err != nil {
			return err
		}
		hashes = append(hashes, hash)
	}
	return s.store.Enable(ctx, userID, encryptedSecret, hashes)
}

func (s *Service) ValidateStoredCode(ctx context.Context, userID int, code string) (bool, error) {
	settings, err := s.store.GetSettingsByUserID(ctx, userID)
	if err != nil {
		return false, err
	}
	if !settings.TOTPEnabled {
		return false, nil
	}
	if strings.HasPrefix(strings.ToUpper(strings.TrimSpace(code)), "BK-") {
		return s.useBackupCode(ctx, userID, settings.BackupCodeHashes, code)
	}
	secret, err := decryptSecret(s.secretKey, settings.EncryptedSecret)
	if err != nil {
		return false, err
	}
	return s.ValidateCode(secret, code), nil
}

func (s *Service) RegenerateBackupCodes(ctx context.Context, userID int) ([]string, error) {
	codes := s.GenerateBackupCodes()
	hashes := make([]string, 0, len(codes))
	for _, code := range codes {
		hash, err := s.HashBackupCode(code)
		if err != nil {
			return nil, err
		}
		hashes = append(hashes, hash)
	}
	if err := s.store.ReplaceBackupCodes(ctx, userID, hashes); err != nil {
		return nil, err
	}
	return codes, nil
}

func (s *Service) useBackupCode(ctx context.Context, userID int, hashes []string, code string) (bool, error) {
	remaining := make([]string, 0, len(hashes))
	used := false
	for _, hash := range hashes {
		if !used && bcrypt.CompareHashAndPassword([]byte(hash), []byte(strings.TrimSpace(code))) == nil {
			used = true
			continue
		}
		remaining = append(remaining, hash)
	}
	if !used {
		return false, nil
	}
	if err := s.store.UseBackupCode(ctx, userID, remaining); err != nil {
		return false, err
	}
	return true, nil
}

func randomAlphaNumeric(length int) string {
	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	b := make([]byte, length)
	rnd := make([]byte, length)
	if _, err := rand.Read(rnd); err != nil {
		panic(err)
	}
	for i := range b {
		b[i] = alphabet[int(rnd[i])%len(alphabet)]
	}
	return string(b)
}

func encryptSecret(secretKey, plaintext string) (string, error) {
	sum := sha256.Sum256([]byte(secretKey))
	block, err := aes.NewCipher(sum[:])
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func decryptSecret(secretKey, ciphertext string) (string, error) {
	raw, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256([]byte(secretKey))
	block, err := aes.NewCipher(sum[:])
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(raw) < gcm.NonceSize() {
		return "", fmt.Errorf("invalid encrypted secret")
	}
	nonce := raw[:gcm.NonceSize()]
	plaintext, err := gcm.Open(nil, nonce, raw[gcm.NonceSize():], nil)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

type stringWriter struct {
	builder *strings.Builder
}

func (w stringWriter) Write(p []byte) (int, error) {
	return w.builder.Write(p)
}

func ManualEntryKey(secret string) string {
	return secret
}
