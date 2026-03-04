package emailsubscriptions

import (
	"crypto/rand"
	"encoding/base64"
)

func generateUniqueCode() (string, error) {
	const tokenSize = 32
	tokenBytes := make([]byte, tokenSize)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(tokenBytes), nil
}
