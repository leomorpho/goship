package authsupport

import (
	"context"
	"time"
)

type authStore interface {
	GetIdentityByUserID(ctx context.Context, userID int) (*AuthIdentity, error)
	GetUserRecordByEmail(ctx context.Context, email string) (*AuthUserRecord, error)
	GetUserDisplayNameByUserID(ctx context.Context, userID int) (string, error)
	UpdateUserDisplayNameByUserID(ctx context.Context, userID int, displayName string) error
	UpdateUserPasswordHashByUserID(ctx context.Context, userID int, passwordHash string) error
	MarkUserVerifiedByUserID(ctx context.Context, userID int) error
	CreateLastSeenOnline(ctx context.Context, userID int, seenAt time.Time) error
	CreatePasswordToken(ctx context.Context, userID int, hash string) (int, error)
	GetPasswordTokenHash(ctx context.Context, userID, tokenID int, notBefore time.Time) (string, error)
	DeletePasswordTokens(ctx context.Context, userID int) error
}
