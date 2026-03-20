package authsupport

import (
	"context"
	"errors"
	"time"
)

var ErrAuthStoreUnavailable = errors.New("auth store unavailable: database connection required")

type unavailableAuthStore struct{}

func newUnavailableAuthStore() *unavailableAuthStore {
	return &unavailableAuthStore{}
}

func (s *unavailableAuthStore) GetIdentityByUserID(context.Context, int) (*AuthIdentity, error) {
	return nil, ErrAuthStoreUnavailable
}

func (s *unavailableAuthStore) GetUserRecordByEmail(context.Context, string) (*AuthUserRecord, error) {
	return nil, ErrAuthStoreUnavailable
}

func (s *unavailableAuthStore) GetUserDisplayNameByUserID(context.Context, int) (string, error) {
	return "", ErrAuthStoreUnavailable
}

func (s *unavailableAuthStore) UpdateUserDisplayNameByUserID(context.Context, int, string) error {
	return ErrAuthStoreUnavailable
}

func (s *unavailableAuthStore) UpdateUserPasswordHashByUserID(context.Context, int, string) error {
	return ErrAuthStoreUnavailable
}

func (s *unavailableAuthStore) MarkUserVerifiedByUserID(context.Context, int) error {
	return ErrAuthStoreUnavailable
}

func (s *unavailableAuthStore) CreateLastSeenOnline(context.Context, int, time.Time) error {
	return ErrAuthStoreUnavailable
}

func (s *unavailableAuthStore) CreatePasswordToken(context.Context, int, string) (int, error) {
	return 0, ErrAuthStoreUnavailable
}

func (s *unavailableAuthStore) GetPasswordTokenHash(context.Context, int, int, time.Time) (string, error) {
	return "", ErrAuthStoreUnavailable
}

func (s *unavailableAuthStore) DeletePasswordTokens(context.Context, int) error {
	return ErrAuthStoreUnavailable
}
