package notifications

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sns"
)

type sqlSMSCodeStore struct {
	db         *sql.DB
	postgresql bool
}

// NewSQLSMSSender initializes a new SMSSender with SQL-backed storage and AWS SNS.
func NewSQLSMSSender(db *sql.DB, dialect, region, senderID string, validationTextExpirationMinutes int) (*SMSSender, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("configuration error: %w", err)
	}
	client := sns.NewFromConfig(cfg)
	return newSMSSender(
		newSQLSMSCodeStore(db, dialect),
		client,
		senderID,
		validationTextExpirationMinutes,
	), nil
}

func newSQLSMSCodeStore(db *sql.DB, dialect string) *sqlSMSCodeStore {
	d := strings.ToLower(strings.TrimSpace(dialect))
	return &sqlSMSCodeStore{
		db:         db,
		postgresql: d == "postgres" || d == "postgresql" || d == "pgx",
	}
}

func (s *sqlSMSCodeStore) deleteCodesByProfileID(ctx context.Context, profileID int) error {
	_, err := s.db.ExecContext(ctx, s.bind(`
		DELETE FROM phone_verification_codes
		WHERE profile_id = ?
	`), profileID)
	return err
}

func (s *sqlSMSCodeStore) createCode(ctx context.Context, profileID int, code string) error {
	now := time.Now().UTC()
	_, err := s.db.ExecContext(ctx, s.bind(`
		INSERT INTO phone_verification_codes (
			created_at, updated_at, code, profile_id
		) VALUES (?, ?, ?, ?)
	`), now, now, code, profileID)
	return err
}

func (s *sqlSMSCodeStore) findLatestValidCode(
	ctx context.Context, profileID int, minCreatedAt time.Time,
) (*phoneVerificationCodeRecord, error) {
	rec := &phoneVerificationCodeRecord{}
	err := s.db.QueryRowContext(ctx, s.bind(`
		SELECT id, code
		FROM phone_verification_codes
		WHERE profile_id = ? AND created_at >= ?
		ORDER BY created_at DESC
		LIMIT 1
	`), profileID, minCreatedAt).Scan(&rec.ID, &rec.Code)
	if err != nil {
		return nil, err
	}
	return rec, nil
}

func (s *sqlSMSCodeStore) deleteCodeByID(ctx context.Context, id int) error {
	_, err := s.db.ExecContext(ctx, s.bind(`
		DELETE FROM phone_verification_codes
		WHERE id = ?
	`), id)
	return err
}

func (s *sqlSMSCodeStore) bind(query string) string {
	if !s.postgresql || strings.Count(query, "?") == 0 {
		return query
	}
	var b strings.Builder
	b.Grow(len(query) + 8)
	arg := 1
	for _, r := range query {
		if r == '?' {
			b.WriteByte('$')
			b.WriteString(strconv.Itoa(arg))
			arg++
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}
