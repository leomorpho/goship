package authsupport

import (
	"context"
	"database/sql"
	"errors"
	"time"

	dbgen "github.com/leomorpho/goship/v2/db/gen"
)

var errBobAuthStoreNotImplemented = errors.New("bob auth store not implemented")

type bobAuthStore struct {
	db      *sql.DB
	dialect string
}

func newBobAuthStore(db *sql.DB, dialect string) *bobAuthStore {
	return &bobAuthStore{
		db:      db,
		dialect: dialect,
	}
}

func (s *bobAuthStore) GetIdentityByUserID(ctx context.Context, userID int) (*AuthIdentity, error) {
	if s.db == nil {
		return nil, errBobAuthStoreNotImplemented
	}
	row, err := dbgen.GetAuthIdentityByUserID(ctx, s.db, s.dialect, userID)
	if err != nil {
		return nil, err
	}
	return &AuthIdentity{
		UserID:                row.UserID,
		UserName:              row.UserName,
		UserEmail:             row.UserEmail,
		HasProfile:            row.HasProfile,
		ProfileID:             row.ProfileID,
		ProfileFullyOnboarded: row.ProfileFullyOnboarded,
	}, nil
}

func (s *bobAuthStore) GetUserRecordByEmail(ctx context.Context, email string) (*AuthUserRecord, error) {
	if s.db == nil {
		return nil, errBobAuthStoreNotImplemented
	}
	row, err := dbgen.GetAuthUserRecordByEmail(ctx, s.db, s.dialect, email)
	if err != nil {
		return nil, err
	}
	return &AuthUserRecord{
		UserID:     row.UserID,
		Name:       row.Name,
		Email:      row.Email,
		Password:   row.Password,
		IsVerified: row.IsVerified,
	}, nil
}

func (s *bobAuthStore) CreateLastSeenOnline(ctx context.Context, userID int, seenAt time.Time) error {
	if s.db == nil {
		return errBobAuthStoreNotImplemented
	}
	return dbgen.InsertLastSeenOnline(ctx, s.db, s.dialect, userID, seenAt)
}

func (s *bobAuthStore) UpdateUserPasswordHashByUserID(ctx context.Context, userID int, passwordHash string) error {
	if s.db == nil {
		return errBobAuthStoreNotImplemented
	}
	return dbgen.UpdateUserPasswordHashByUserID(ctx, s.db, s.dialect, userID, passwordHash)
}

func (s *bobAuthStore) GetUserDisplayNameByUserID(ctx context.Context, userID int) (string, error) {
	if s.db == nil {
		return "", errBobAuthStoreNotImplemented
	}
	return dbgen.GetUserDisplayNameByUserID(ctx, s.db, s.dialect, userID)
}

func (s *bobAuthStore) UpdateUserDisplayNameByUserID(ctx context.Context, userID int, displayName string) error {
	if s.db == nil {
		return errBobAuthStoreNotImplemented
	}
	return dbgen.UpdateUserDisplayNameByUserID(ctx, s.db, s.dialect, userID, displayName)
}

func (s *bobAuthStore) MarkUserVerifiedByUserID(ctx context.Context, userID int) error {
	if s.db == nil {
		return errBobAuthStoreNotImplemented
	}
	return dbgen.MarkUserVerifiedByUserID(ctx, s.db, s.dialect, userID)
}

func (s *bobAuthStore) CreatePasswordToken(ctx context.Context, userID int, hash string) (int, error) {
	if s.db == nil {
		return 0, errBobAuthStoreNotImplemented
	}
	return dbgen.InsertPasswordToken(ctx, s.db, s.dialect, userID, hash, time.Now().UTC())
}

func (s *bobAuthStore) GetPasswordTokenHash(ctx context.Context, userID int, tokenID int, notBefore time.Time) (string, error) {
	if s.db == nil {
		return "", errBobAuthStoreNotImplemented
	}
	return dbgen.GetPasswordTokenHash(ctx, s.db, s.dialect, userID, tokenID, notBefore)
}

func (s *bobAuthStore) DeletePasswordTokens(ctx context.Context, userID int) error {
	if s.db == nil {
		return errBobAuthStoreNotImplemented
	}
	return dbgen.DeletePasswordTokensByUserID(ctx, s.db, s.dialect, userID)
}
