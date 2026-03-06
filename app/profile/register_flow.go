package profiles

import (
	"context"
	"strings"
	"time"

	paidsubscriptions "github.com/leomorpho/goship-modules/paidsubscriptions"
	dbgen "github.com/leomorpho/goship/db/gen"
	"github.com/leomorpho/goship/framework/domain"
)

type RegistrationResult struct {
	UserID    int
	UserName  string
	UserEmail string
	ProfileID int
}

// RegisterUserWithProfile creates user+profile inside one transaction and optionally
// creates the initial trial subscription in the same transaction boundary.
func (p *ProfileService) RegisterUserWithProfile(
	ctx context.Context,
	name string,
	email string,
	passwordHash string,
	birthdate time.Time,
	subscriptionsService *paidsubscriptions.Service,
) (*RegistrationResult, error) {
	if p.db == nil {
		return nil, ErrProfileDBNotConfigured
	}
	return p.registerUserWithProfileSQL(ctx, name, email, passwordHash, birthdate, subscriptionsService)
}

func (p *ProfileService) registerUserWithProfileSQL(
	ctx context.Context,
	name string,
	email string,
	passwordHash string,
	birthdate time.Time,
	subscriptionsService *paidsubscriptions.Service,
) (*RegistrationResult, error) {
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	now := time.Now().UTC()
	normalizedEmail := strings.ToLower(strings.TrimSpace(email))

	userInsertQuery, profileInsertQuery := p.registerInsertQueries()

	var userID int
	if err := tx.QueryRowContext(
		ctx,
		userInsertQuery,
		name,
		normalizedEmail,
		passwordHash,
		false,
	).Scan(&userID); err != nil {
		return nil, err
	}

	var profileID int
	if err := tx.QueryRowContext(
		ctx,
		profileInsertQuery,
		now,
		now,
		domain.DefaultBio,
		birthdate,
		CalculateAge(birthdate),
		false,
		false,
		userID,
	).Scan(&profileID); err != nil {
		return nil, err
	}

	if subscriptionsService != nil {
		if err := subscriptionsService.CreateSubscription(ctx, tx, profileID); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &RegistrationResult{
		UserID:    userID,
		UserName:  name,
		UserEmail: normalizedEmail,
		ProfileID: profileID,
	}, nil
}

func (p *ProfileService) registerInsertQueries() (userInsert, profileInsert string) {
	switch strings.ToLower(strings.TrimSpace(p.dbDialect)) {
	case "postgres", "postgresql", "pgx":
		userInsert = `
			INSERT INTO users (name, email, password, verified)
			VALUES ($1, $2, $3, $4)
			RETURNING id
		`
		profileInsert = `
			INSERT INTO profiles (
				created_at,
				updated_at,
				bio,
				birthdate,
				age,
				fully_onboarded,
				phone_verified,
				user_profile
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			RETURNING id
		`
	default:
		userInsert = `
			INSERT INTO users (name, email, password, verified)
			VALUES (?, ?, ?, ?)
			RETURNING id
		`
		profileInsert = `
			INSERT INTO profiles (
				created_at,
				updated_at,
				bio,
				birthdate,
				age,
				fully_onboarded,
				phone_verified,
				user_profile
			)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
			RETURNING id
		`
	}
	return userInsert, profileInsert
}

func (p *ProfileService) MarkPhoneVerified(ctx context.Context, profileID int) error {
	if p.db == nil {
		return ErrProfileDBNotConfigured
	}
	return dbgen.MarkProfilePhoneVerifiedByID(ctx, p.db, p.dbDialect, profileID)
}
