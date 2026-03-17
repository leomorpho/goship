package emailsubscriptions

import (
	"crypto/rand"
	"encoding/base64"
	"math/big"
)

func generateUniqueCode() (string, error) {
	const tokenSize = 32

	tokenBytes := make([]byte, tokenSize)
	_, err := rand.Read(tokenBytes)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(tokenBytes), nil
}

// generateInvitationCode generates a unique code of a specified length containing only letters and numbers.
func generateInvitationCode(length int) (string, error) {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	code := make([]byte, length)

	for i := 0; i < length; i++ {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", err
		}
		code[i] = charset[num.Int64()]
	}

	return string(code), nil
}
