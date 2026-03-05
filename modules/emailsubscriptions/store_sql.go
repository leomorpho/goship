package emailsubscriptions

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/leomorpho/goship-modules/emailsubscriptions/db/gen"
	"strings"
	"time"
)

// SQLStore is a module-local SQL implementation of Store.
type SQLStore struct {
	db         *sql.DB
	postgresQL bool
}

func NewSQLStore(db *sql.DB, dialect string) *SQLStore {
	normalized := strings.ToLower(strings.TrimSpace(dialect))
	return &SQLStore{
		db:         db,
		postgresQL: normalized == "postgres" || normalized == "postgresql" || normalized == "pgx",
	}
}

func (s *SQLStore) CreateList(ctx context.Context, emailList List) error {
	_, err := gen.FindListID(ctx, s.db, s.dialect(), string(emailList), false)
	if err == nil {
		return nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return err
	}

	now := time.Now().UTC()
	err = gen.InsertList(ctx, s.db, s.dialect(), string(emailList), now)
	if err != nil && isUniqueConstraint(err) {
		return nil
	}
	return err
}

func (s *SQLStore) Subscribe(
	ctx context.Context, email string, emailList List, latitude, longitude *float64,
) (*Subscription, error) {
	if (latitude != nil && longitude == nil) || (latitude == nil && longitude != nil) {
		return nil, fmt.Errorf("both latitude and longitude must be set or omitted")
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	listID, err := gen.FindListID(ctx, tx, s.dialect(), string(emailList), true)
	if err != nil {
		return nil, err
	}

	sub, err := gen.FindSubscriptionByEmail(ctx, tx, s.dialect(), email)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		confirmationCode, genErr := generateUniqueCode()
		if genErr != nil {
			return nil, genErr
		}

		now := time.Now().UTC()
		id, idErr := gen.InsertSubscription(
			ctx, tx, s.dialect(), now, email, confirmationCode, nullableFloat(latitude), nullableFloat(longitude),
		)
		if idErr != nil {
			return nil, idErr
		}

		sub = gen.SQLSubscription{
			ID:               int(id),
			Email:            email,
			Verified:         false,
			ConfirmationCode: confirmationCode,
			Lat:              nullableFloat(latitude),
			Lon:              nullableFloat(longitude),
		}
	case err != nil:
		return nil, err
	}

	alreadySubscribed, err := gen.FindSubscriptionLink(ctx, tx, s.dialect(), sub.ID, listID)
	switch {
	case err != nil:
		return nil, err
	case alreadySubscribed:
		return nil, &ErrAlreadySubscribed{EmailList: string(emailList)}
	}

	err = gen.InsertSubscriptionLink(ctx, tx, s.dialect(), sub.ID, listID)
	if err != nil {
		return nil, err
	}

	if latitude != nil && longitude != nil {
		now := time.Now().UTC()
		err = gen.UpdateSubscriptionLocation(ctx, tx, s.dialect(), now, sub.ID, *latitude, *longitude)
		if err != nil {
			return nil, err
		}
		sub.Lat = sql.NullFloat64{Float64: *latitude, Valid: true}
		sub.Lon = sql.NullFloat64{Float64: *longitude, Valid: true}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return toDomainSubscription(sub), nil
}

func (s *SQLStore) Unsubscribe(ctx context.Context, email string, _ string, emailList List) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	sub, err := gen.FindSubscriptionByEmail(ctx, tx, s.dialect(), email)
	if err != nil {
		return err
	}

	listID, err := gen.FindListID(ctx, tx, s.dialect(), string(emailList), false)
	if err != nil {
		return err
	}

	err = gen.DeleteSubscriptionLink(ctx, tx, s.dialect(), sub.ID, listID)
	if err != nil {
		return err
	}

	count, err := gen.CountSubscriptionLinks(ctx, tx, s.dialect(), sub.ID)
	if err != nil {
		return err
	}

	if count == 0 {
		if err := gen.DeleteSubscriptionByID(ctx, tx, s.dialect(), sub.ID); err != nil {
			return err
		}
		return tx.Commit()
	}

	confirmationCode, err := generateUniqueCode()
	if err != nil {
		return err
	}
	err = gen.RotateSubscriptionCode(ctx, tx, s.dialect(), time.Now().UTC(), sub.ID, confirmationCode)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (s *SQLStore) Confirm(ctx context.Context, code string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	id, verified, err := gen.FindSubscriptionByConfirmationCode(ctx, tx, s.dialect(), code)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrInvalidEmailConfirmationCode
	}
	if err != nil {
		return err
	}
	if verified {
		return tx.Commit()
	}

	confirmationCode, err := generateUniqueCode()
	if err != nil {
		return err
	}
	err = gen.MarkSubscriptionVerified(ctx, tx, s.dialect(), time.Now().UTC(), id, confirmationCode)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (s *SQLStore) dialect() string {
	if s.postgresQL {
		return "postgres"
	}
	return "sqlite3"
}

func isUniqueConstraint(err error) bool {
	if err == nil {
		return false
	}
	v := strings.ToLower(err.Error())
	return strings.Contains(v, "unique") || strings.Contains(v, "duplicate")
}

func nullableFloat(value *float64) sql.NullFloat64 {
	if value == nil {
		return sql.NullFloat64{}
	}
	return sql.NullFloat64{Float64: *value, Valid: true}
}

func toDomainSubscription(s gen.SQLSubscription) *Subscription {
	out := &Subscription{
		ID:               s.ID,
		Email:            s.Email,
		Verified:         s.Verified,
		ConfirmationCode: s.ConfirmationCode,
	}
	if s.Lat.Valid {
		out.Lat = s.Lat.Float64
	}
	if s.Lon.Valid {
		out.Lon = s.Lon.Float64
	}
	return out
}
