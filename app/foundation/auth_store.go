package foundation

import (
	"context"
	"strings"
	"time"

	profilesvc "github.com/leomorpho/goship/app/profile"
	"github.com/leomorpho/goship/db/ent"
	"github.com/leomorpho/goship/db/ent/passwordtoken"
	"github.com/leomorpho/goship/db/ent/user"
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

type entAuthStore struct {
	orm *ent.Client
}

func newEntAuthStore(orm *ent.Client) *entAuthStore {
	return &entAuthStore{orm: orm}
}

func (s *entAuthStore) GetIdentityByUserID(ctx context.Context, userID int) (*AuthIdentity, error) {
	u, err := s.orm.User.Query().
		Where(user.ID(userID)).
		WithProfile(func(q *ent.ProfileQuery) {
			q.WithProfileImage(func(pi *ent.ImageQuery) {
				pi.WithSizes(func(sz *ent.ImageSizeQuery) {
					sz.WithFile()
				})
			})
		}).
		Only(ctx)
	if err != nil {
		return nil, err
	}

	identity := &AuthIdentity{
		UserID:    u.ID,
		UserName:  u.Name,
		UserEmail: u.Email,
	}
	if u.Edges.Profile != nil {
		identity.HasProfile = true
		identity.ProfileID = u.Edges.Profile.ID
		identity.ProfileFullyOnboarded = profilesvc.IsProfileFullyOnboarded(u.Edges.Profile)
	}
	return identity, nil
}

func (s *entAuthStore) GetUserRecordByEmail(ctx context.Context, email string) (*AuthUserRecord, error) {
	u, err := s.orm.User.Query().
		Where(user.Email(strings.ToLower(email))).
		Only(ctx)
	if err != nil {
		return nil, err
	}
	return &AuthUserRecord{
		UserID:     u.ID,
		Name:       u.Name,
		Email:      u.Email,
		Password:   u.Password,
		IsVerified: u.Verified,
	}, nil
}

func (s *entAuthStore) CreateLastSeenOnline(ctx context.Context, userID int, seenAt time.Time) error {
	_, err := s.orm.LastSeenOnline.
		Create().
		SetUserID(userID).
		SetSeenAt(seenAt).
		Save(ctx)
	return err
}

func (s *entAuthStore) GetUserDisplayNameByUserID(ctx context.Context, userID int) (string, error) {
	u, err := s.orm.User.Get(ctx, userID)
	if err != nil {
		return "", err
	}
	return u.Name, nil
}

func (s *entAuthStore) UpdateUserDisplayNameByUserID(ctx context.Context, userID int, displayName string) error {
	_, err := s.orm.User.
		UpdateOneID(userID).
		SetName(displayName).
		Save(ctx)
	return err
}

func (s *entAuthStore) UpdateUserPasswordHashByUserID(ctx context.Context, userID int, passwordHash string) error {
	_, err := s.orm.User.
		UpdateOneID(userID).
		SetPassword(passwordHash).
		Save(ctx)
	return err
}

func (s *entAuthStore) MarkUserVerifiedByUserID(ctx context.Context, userID int) error {
	_, err := s.orm.User.
		UpdateOneID(userID).
		SetVerified(true).
		Save(ctx)
	return err
}

func (s *entAuthStore) CreatePasswordToken(ctx context.Context, userID int, hash string) (int, error) {
	pt, err := s.orm.PasswordToken.
		Create().
		SetHash(hash).
		SetUserID(userID).
		Save(ctx)
	if err != nil {
		return 0, err
	}
	return pt.ID, nil
}

func (s *entAuthStore) GetPasswordTokenHash(ctx context.Context, userID, tokenID int, notBefore time.Time) (string, error) {
	pt, err := s.orm.PasswordToken.
		Query().
		Where(passwordtoken.ID(tokenID)).
		Where(passwordtoken.HasUserWith(user.ID(userID))).
		Where(passwordtoken.CreatedAtGTE(notBefore)).
		Only(ctx)
	if err != nil {
		return "", err
	}
	return pt.Hash, nil
}

func (s *entAuthStore) DeletePasswordTokens(ctx context.Context, userID int) error {
	_, err := s.orm.PasswordToken.
		Delete().
		Where(passwordtoken.HasUserWith(user.ID(userID))).
		Exec(ctx)
	return err
}
