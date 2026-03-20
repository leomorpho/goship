package notifications

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"

	"github.com/leomorpho/goship/framework/domain"
)

func TestPlannedNotificationsService_SQLStore_CreateTimesAndSelectCandidates(t *testing.T) {
	db := openPlannedNotificationsTestDB(t)
	svc := NewPlannedNotificationsServiceWithStore(newSQLPlannedNotificationStore(db, "sqlite3"), nil)
	ctx := context.Background()

	seedPlannedProfile(t, db, 1, 11, true)
	seedPlannedProfile(t, db, 2, 22, false)

	_, err := db.ExecContext(ctx, `
		INSERT INTO notification_permissions (created_at, updated_at, permission, platform, profile_id, token)
		VALUES (?, ?, ?, ?, ?, ?), (?, ?, ?, ?, ?, ?)
	`,
		time.Now().UTC(), time.Now().UTC(), domain.NotificationPermissionDailyReminder.Value, domain.NotificationPlatformPush.Value, 1, "tok-1",
		time.Now().UTC(), time.Now().UTC(), domain.NotificationPermissionDailyReminder.Value, domain.NotificationPlatformPush.Value, 2, "tok-2",
	)
	require.NoError(t, err)

	err = svc.CreateNotificationTimeObjects(ctx, domain.NotificationTypeDailyConversationReminder, domain.NotificationPermissionDailyReminder)
	require.NoError(t, err)

	var countProfile1 int
	err = db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM notification_times
		WHERE profile_id = ? AND type = ?
	`, 1, domain.NotificationTypeDailyConversationReminder.Value).Scan(&countProfile1)
	require.NoError(t, err)
	require.Equal(t, 1, countProfile1)

	var countProfile2 int
	err = db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM notification_times
		WHERE profile_id = ? AND type = ?
	`, 2, domain.NotificationTypeDailyConversationReminder.Value).Scan(&countProfile2)
	require.NoError(t, err)
	require.Equal(t, 0, countProfile2)

	timestamp := time.Date(2026, 3, 5, 23, 0, 0, 0, time.UTC)
	candidates, err := svc.ProfileIDsCanGetPlannedNotificationNow(
		ctx, timestamp, domain.NotificationTypeDailyConversationReminder, nil,
	)
	require.NoError(t, err)
	require.Equal(t, []int{1}, candidates)
}

func TestPlannedNotificationsService_SQLStore_AvoidsDoubleSendSameDay(t *testing.T) {
	db := openPlannedNotificationsTestDB(t)
	svc := NewPlannedNotificationsServiceWithStore(newSQLPlannedNotificationStore(db, "sqlite3"), nil)
	ctx := context.Background()

	seedPlannedProfile(t, db, 3, 33, true)

	_, err := db.ExecContext(ctx, `
		INSERT INTO notification_times (created_at, updated_at, type, send_minute, profile_id)
		VALUES (?, ?, ?, ?, ?)
	`, time.Now().UTC(), time.Now().UTC(), domain.NotificationTypeDailyConversationReminder.Value, 60, 3)
	require.NoError(t, err)

	todayMidnight := time.Date(2026, 3, 5, 0, 0, 0, 0, time.UTC)
	_, err = db.ExecContext(ctx, `
		INSERT INTO notifications (created_at, updated_at, type, title, text, read, profile_id, profile_id_who_caused_notification, resource_id_tied_to_notif, read_in_notifications_center)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		todayMidnight.Add(2*time.Hour),
		todayMidnight.Add(2*time.Hour),
		domain.NotificationTypeDailyConversationReminder.Value,
		"daily",
		"text",
		false,
		3,
		0,
		0,
		true,
	)
	require.NoError(t, err)

	candidates, err := svc.ProfileIDsCanGetPlannedNotificationNow(
		ctx, todayMidnight.Add(5*time.Hour), domain.NotificationTypeDailyConversationReminder, nil,
	)
	require.NoError(t, err)
	require.Empty(t, candidates)
}

func seedPlannedProfile(t *testing.T, db *sql.DB, profileID, userID int, withLastSeen bool) {
	t.Helper()
	now := time.Now().UTC()
	_, err := db.Exec(`
		INSERT INTO users (id, created_at, updated_at, name, email, password, verified)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, userID, now, now, "user", "user@example.com", "x", true)
	require.NoError(t, err)
	_, err = db.Exec(`
		INSERT INTO profiles (id, created_at, updated_at, bio, birthdate, age, fully_onboarded, user_profile)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, profileID, now, now, "bio", now, 25, true, userID)
	require.NoError(t, err)

	if withLastSeen {
		_, err = db.Exec(`
			INSERT INTO last_seen_onlines (id, seen_at, user_last_seen_at)
			VALUES (?, ?, ?), (?, ?, ?), (?, ?, ?)
		`, profileID*10+1, now.Add(-3*time.Hour), userID, profileID*10+2, now.Add(-2*time.Hour), userID, profileID*10+3, now.Add(-2*time.Hour+15*time.Minute), userID)
		require.NoError(t, err)
	}
}

func openPlannedNotificationsTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	_, err = NewSQLNotificationStoreWithSchema(db, "sqlite3")
	require.NoError(t, err)

	ddl := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL,
			name TEXT NOT NULL,
			email TEXT NOT NULL,
			password TEXT NOT NULL,
			verified BOOLEAN NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS profiles (
			id INTEGER PRIMARY KEY,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL,
			bio TEXT,
			birthdate TIMESTAMP NOT NULL,
			age INTEGER NOT NULL,
			fully_onboarded BOOLEAN NOT NULL,
			user_profile INTEGER NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS last_seen_onlines (
			id INTEGER PRIMARY KEY,
			seen_at TIMESTAMP NOT NULL,
			user_last_seen_at INTEGER NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS notification_times (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL,
			type TEXT NOT NULL,
			send_minute INTEGER NOT NULL,
			profile_id INTEGER NOT NULL,
			UNIQUE(profile_id, type)
		)`,
	}
	for _, stmt := range ddl {
		_, err := db.Exec(stmt)
		require.NoError(t, err)
	}
	return db
}
