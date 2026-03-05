//go:build integration

package paidsubscriptions

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
)

func TestSQLStore_WithModuleMigration_EndToEnd(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "paidsubscriptions-integration.db")
	db, err := sql.Open("sqlite3", dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	applyModuleMigration(t, db)

	ctx := context.Background()
	store := NewSQLStore(db, "sqlite3", 15, 3)

	require.NoError(t, store.CreateSubscription(ctx, nil, 1))
	product, expiry, isTrial, err := store.GetCurrentlyActiveProduct(ctx, 1)
	require.NoError(t, err)
	require.NotNil(t, product)
	require.Equal(t, ProductTypePro.Value, product.Value)
	require.NotNil(t, expiry)
	require.True(t, isTrial)

	require.NoError(t, store.UpdateToPaidPro(ctx, 1))
	product, expiry, isTrial, err = store.GetCurrentlyActiveProduct(ctx, 1)
	require.NoError(t, err)
	require.NotNil(t, product)
	require.Equal(t, ProductTypePro.Value, product.Value)
	require.Nil(t, expiry)
	require.False(t, isTrial)

	require.NoError(t, store.StoreStripeCustomerID(ctx, 1, "cus_abc"))
	profileID, err := store.GetProfileIDFromStripeCustomerID(ctx, "cus_abc")
	require.NoError(t, err)
	require.Equal(t, 1, profileID)

	cancelAt := time.Now().UTC().Add(12 * time.Hour)
	require.NoError(t, store.CancelOrRenew(ctx, 1, &cancelAt))
	require.NoError(t, store.CancelOrRenew(ctx, 1, nil))

	require.NoError(t, store.UpdateToFree(ctx, 1))
	product, expiry, isTrial, err = store.GetCurrentlyActiveProduct(ctx, 1)
	require.NoError(t, err)
	require.NotNil(t, product)
	require.Equal(t, ProductTypeFree.Value, product.Value)
	require.Nil(t, expiry)
	require.False(t, isTrial)
}

func applyModuleMigration(t *testing.T, db *sql.DB) {
	t.Helper()

	migrationPath := filepath.Join("db", "migrate", "migrations", "20260305183000_init_paid_subscriptions.sql")
	migrationBytes, err := os.ReadFile(migrationPath)
	require.NoError(t, err)

	upSQL := sqliteCompatibleDDL(extractGooseUp(string(migrationBytes)))
	_, err = db.Exec(upSQL)
	require.NoError(t, err)
}

func extractGooseUp(content string) string {
	lines := strings.Split(content, "\n")
	var builder strings.Builder
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		switch trimmed {
		case "-- +goose Up":
			continue
		case "-- +goose Down":
			return builder.String()
		default:
			builder.WriteString(line)
			builder.WriteString("\n")
		}
	}
	return builder.String()
}

func sqliteCompatibleDDL(v string) string {
	replacements := []struct {
		old string
		new string
	}{
		{"BIGSERIAL PRIMARY KEY", "INTEGER PRIMARY KEY AUTOINCREMENT"},
		{"TIMESTAMPTZ", "DATETIME"},
	}
	out := v
	for _, replacement := range replacements {
		out = strings.ReplaceAll(out, replacement.old, replacement.new)
	}
	return out
}
