package gen

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/leomorpho/goship/db/queries"
)

type AuthUserRecord struct {
	UserID     int
	Name       string
	Email      string
	Password   string
	IsVerified bool
}

type AuthIdentity struct {
	UserID                int
	UserName              string
	UserEmail             string
	HasProfile            bool
	ProfileID             int
	ProfileFullyOnboarded bool
}

// Execer is the minimal exec contract used by generated write helpers.
type Execer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

// QueryExecRower is the minimal combined contract for read/write helpers that
// need both QueryRowContext and ExecContext support.
type QueryExecRower interface {
	QueryRower
	Execer
}

func GetAuthUserRecordByEmail(
	ctx context.Context,
	db QueryRower,
	dialect string,
	email string,
) (*AuthUserRecord, error) {
	if db == nil {
		return nil, errors.New("query runner is nil")
	}

	query, args := getAuthUserRecordByEmailQuery(strings.ToLower(strings.TrimSpace(dialect)), email)
	var row AuthUserRecord
	if err := db.QueryRowContext(ctx, query, args...).Scan(
		&row.UserID,
		&row.Name,
		&row.Email,
		&row.Password,
		&row.IsVerified,
	); err != nil {
		return nil, err
	}
	return &row, nil
}

func GetAuthIdentityByUserID(
	ctx context.Context,
	db QueryRower,
	dialect string,
	userID int,
) (*AuthIdentity, error) {
	if db == nil {
		return nil, errors.New("query runner is nil")
	}

	query, args := getAuthIdentityByUserIDQuery(strings.ToLower(strings.TrimSpace(dialect)), userID)

	var row AuthIdentity
	var profileID sql.NullInt64
	var fullyOnboarded sql.NullBool
	if err := db.QueryRowContext(ctx, query, args...).Scan(
		&row.UserID,
		&row.UserName,
		&row.UserEmail,
		&profileID,
		&fullyOnboarded,
	); err != nil {
		return nil, err
	}

	if profileID.Valid {
		row.HasProfile = true
		row.ProfileID = int(profileID.Int64)
		row.ProfileFullyOnboarded = fullyOnboarded.Valid && fullyOnboarded.Bool
	}

	return &row, nil
}

func GetUserDisplayNameByUserID(
	ctx context.Context,
	db QueryRower,
	dialect string,
	userID int,
) (string, error) {
	if db == nil {
		return "", errors.New("query runner is nil")
	}
	query, args := getUserDisplayNameByUserIDQuery(strings.ToLower(strings.TrimSpace(dialect)), userID)
	var name string
	if err := db.QueryRowContext(ctx, query, args...).Scan(&name); err != nil {
		return "", err
	}
	return name, nil
}

func InsertLastSeenOnline(
	ctx context.Context,
	db Execer,
	dialect string,
	userID int,
	seenAt time.Time,
) error {
	if db == nil {
		return errors.New("exec runner is nil")
	}
	query, args := insertLastSeenOnlineQuery(strings.ToLower(strings.TrimSpace(dialect)), userID, seenAt)
	_, err := db.ExecContext(ctx, query, args...)
	return err
}

func UpdateUserPasswordHashByUserID(
	ctx context.Context,
	db Execer,
	dialect string,
	userID int,
	passwordHash string,
) error {
	if db == nil {
		return errors.New("exec runner is nil")
	}
	query, args := updateUserPasswordHashByUserIDQuery(strings.ToLower(strings.TrimSpace(dialect)), userID, passwordHash)
	_, err := db.ExecContext(ctx, query, args...)
	return err
}

func MarkUserVerifiedByUserID(
	ctx context.Context,
	db Execer,
	dialect string,
	userID int,
) error {
	if db == nil {
		return errors.New("exec runner is nil")
	}
	query, args := markUserVerifiedByUserIDQuery(strings.ToLower(strings.TrimSpace(dialect)), userID)
	_, err := db.ExecContext(ctx, query, args...)
	return err
}

func UpdateUserDisplayNameByUserID(
	ctx context.Context,
	db Execer,
	dialect string,
	userID int,
	displayName string,
) error {
	if db == nil {
		return errors.New("exec runner is nil")
	}
	query, args := updateUserDisplayNameByUserIDQuery(strings.ToLower(strings.TrimSpace(dialect)), userID, displayName)
	_, err := db.ExecContext(ctx, query, args...)
	return err
}

func InsertPasswordToken(
	ctx context.Context,
	db QueryExecRower,
	dialect string,
	userID int,
	hash string,
	createdAt time.Time,
) (int, error) {
	if db == nil {
		return 0, errors.New("query runner is nil")
	}
	query, args := insertPasswordTokenQuery(strings.ToLower(strings.TrimSpace(dialect)), userID, hash, createdAt)
	var tokenID int
	if err := db.QueryRowContext(ctx, query, args...).Scan(&tokenID); err != nil {
		return 0, err
	}
	return tokenID, nil
}

func GetPasswordTokenHash(
	ctx context.Context,
	db QueryRower,
	dialect string,
	userID int,
	tokenID int,
	notBefore time.Time,
) (string, error) {
	if db == nil {
		return "", errors.New("query runner is nil")
	}
	query, args := getPasswordTokenHashQuery(strings.ToLower(strings.TrimSpace(dialect)), userID, tokenID, notBefore)
	var hash string
	if err := db.QueryRowContext(ctx, query, args...).Scan(&hash); err != nil {
		return "", err
	}
	return hash, nil
}

func DeletePasswordTokensByUserID(
	ctx context.Context,
	db Execer,
	dialect string,
	userID int,
) error {
	if db == nil {
		return errors.New("exec runner is nil")
	}
	query, args := deletePasswordTokensByUserIDQuery(strings.ToLower(strings.TrimSpace(dialect)), userID)
	_, err := db.ExecContext(ctx, query, args...)
	return err
}

func getAuthUserRecordByEmailQuery(dialect, email string) (string, []any) {
	email = strings.ToLower(strings.TrimSpace(email))
	query := mustQuery("get_auth_user_record_by_email", dialect)
	return query, []any{email}
}

func getAuthIdentityByUserIDQuery(dialect string, userID int) (string, []any) {
	return mustQuery("get_auth_identity_by_user_id", dialect), []any{userID}
}

func getUserDisplayNameByUserIDQuery(dialect string, userID int) (string, []any) {
	return mustQuery("get_user_display_name_by_user_id", dialect), []any{userID}
}

func insertLastSeenOnlineQuery(dialect string, userID int, seenAt time.Time) (string, []any) {
	return mustQuery("insert_last_seen_online", dialect), []any{seenAt, userID}
}

func insertPasswordTokenQuery(dialect string, userID int, hash string, createdAt time.Time) (string, []any) {
	return mustQuery("insert_password_token", dialect), []any{hash, createdAt, userID}
}

func updateUserPasswordHashByUserIDQuery(dialect string, userID int, passwordHash string) (string, []any) {
	return mustQuery("update_user_password_hash_by_user_id", dialect), []any{passwordHash, userID}
}

func updateUserDisplayNameByUserIDQuery(dialect string, userID int, displayName string) (string, []any) {
	return mustQuery("update_user_display_name_by_user_id", dialect), []any{displayName, userID}
}

func markUserVerifiedByUserIDQuery(dialect string, userID int) (string, []any) {
	return mustQuery("mark_user_verified_by_user_id", dialect), []any{userID}
}

func getPasswordTokenHashQuery(dialect string, userID int, tokenID int, notBefore time.Time) (string, []any) {
	return mustQuery("get_password_token_hash", dialect), []any{tokenID, userID, notBefore}
}

func deletePasswordTokensByUserIDQuery(dialect string, userID int) (string, []any) {
	return mustQuery("delete_password_tokens_by_user_id", dialect), []any{userID}
}

func mustQuery(baseName, dialect string) string {
	key := baseName + "_" + dialectSuffix(dialect)
	query, err := queries.Get(key)
	if err != nil {
		panic(err)
	}
	return query
}

func dialectSuffix(dialect string) string {
	switch strings.ToLower(strings.TrimSpace(dialect)) {
	case "postgres", "postgresql", "pgx":
		return "postgres"
	default:
		return "sqlite"
	}
}
