package profiles

import (
	"context"
	"strings"
	"time"

	paidsubscriptions "github.com/leomorpho/goship-modules/paidsubscriptions"
	dbgen "github.com/leomorpho/goship/db/gen"
	dbqueries "github.com/leomorpho/goship/db/queries"
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

	userInsertQuery, profileInsertQuery, err := p.registerInsertQueries()
	if err != nil {
		return nil, err
	}

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

func (p *ProfileService) registerInsertQueries() (userInsert, profileInsert string, err error) {
	switch normalizeDialect(p.dbDialect) {
	case "postgres", "postgresql", "pgx":
		userInsert, err = dbqueries.Get("insert_user_returning_id_postgres")
		if err != nil {
			return "", "", err
		}
		profileInsert, err = dbqueries.Get("insert_profile_returning_id_postgres")
		if err != nil {
			return "", "", err
		}
	default:
		userInsert, err = dbqueries.Get("insert_user_returning_id_sqlite")
		if err != nil {
			return "", "", err
		}
		profileInsert, err = dbqueries.Get("insert_profile_returning_id_sqlite")
		if err != nil {
			return "", "", err
		}
	}
	return userInsert, profileInsert, nil
}

func normalizeDialect(dialect string) string {
	return strings.ToLower(strings.TrimSpace(dialect))
}

func (p *ProfileService) MarkPhoneVerified(ctx context.Context, profileID int) error {
	if p.db == nil {
		return ErrProfileDBNotConfigured
	}
	return dbgen.MarkProfilePhoneVerifiedByID(ctx, p.db, p.dbDialect, profileID)
}
