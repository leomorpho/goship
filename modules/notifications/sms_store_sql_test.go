package notifications

import (
	"context"
	"database/sql"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/sns"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
)

func TestSMSSender_SQLStore_CreateAndVerifyCode(t *testing.T) {
	db := openSMSTestDB(t)
	store := newSQLSMSCodeStore(db, "sqlite3")
	sender := newSMSSender(store, nil, "goship", 30)
	sender.sendSMS = func(context.Context, string, string) (*sns.PublishOutput, error) {
		return &sns.PublishOutput{}, nil
	}

	code, err := sender.CreateConfirmationCode(context.Background(), 123, "+12065550100")
	require.NoError(t, err)
	require.Len(t, code, 4)

	ok, err := sender.VerifyConfirmationCode(context.Background(), 123, code)
	require.NoError(t, err)
	require.True(t, ok)
}

func TestSMSSender_SQLStore_WrongCode(t *testing.T) {
	db := openSMSTestDB(t)
	store := newSQLSMSCodeStore(db, "sqlite3")
	sender := newSMSSender(store, nil, "goship", 30)
	sender.sendSMS = func(context.Context, string, string) (*sns.PublishOutput, error) {
		return &sns.PublishOutput{}, nil
	}

	_, err := sender.CreateConfirmationCode(context.Background(), 321, "+12065550101")
	require.NoError(t, err)

	ok, err := sender.VerifyConfirmationCode(context.Background(), 321, "0000")
	require.Error(t, err)
	require.False(t, ok)
}

func openSMSTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	_, err = NewSQLNotificationStoreWithSchema(db, "sqlite3")
	require.NoError(t, err)
	return db
}
