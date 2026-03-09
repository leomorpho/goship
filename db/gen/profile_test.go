package gen

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func TestGetProfileFullyOnboardedByUserID_SQLite(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()

	if _, err := db.Exec(`
CREATE TABLE users (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL,
  email TEXT NOT NULL
);
CREATE TABLE profiles (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  fully_onboarded BOOLEAN NOT NULL DEFAULT 0,
  user_profile INTEGER NOT NULL
);
INSERT INTO users (name, email) VALUES ('Alice', 'alice@example.com');
INSERT INTO profiles (fully_onboarded, user_profile) VALUES (1, 1);
`); err != nil {
		t.Fatalf("seed sqlite: %v", err)
	}

	onboarded, err := GetProfileFullyOnboardedByUserID(context.Background(), db, "sqlite", 1)
	if err != nil {
		t.Fatalf("GetProfileFullyOnboardedByUserID error: %v", err)
	}
	if !onboarded {
		t.Fatalf("expected onboarded=true, got false")
	}
}

func TestGetProfileThumbnailObjectKeyByUserID_SQLite(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()

	if _, err := db.Exec(`
CREATE TABLE users (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL,
  email TEXT NOT NULL
);
CREATE TABLE images (
  id INTEGER PRIMARY KEY AUTOINCREMENT
);
CREATE TABLE file_storages (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  object_key TEXT NOT NULL
);
CREATE TABLE image_sizes (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  size TEXT NOT NULL,
  image_sizes INTEGER,
  image_size_file INTEGER NOT NULL
);
CREATE TABLE profiles (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_profile INTEGER NOT NULL,
  profile_profile_image INTEGER
);
INSERT INTO users (name, email) VALUES ('Bob', 'bob@example.com');
INSERT INTO images (id) VALUES (10);
INSERT INTO file_storages (id, object_key) VALUES (20, 'profiles/bob-thumb.jpg');
INSERT INTO image_sizes (id, size, image_sizes, image_size_file) VALUES (30, 'thumbnail', 10, 20);
INSERT INTO profiles (user_profile, profile_profile_image) VALUES (1, 10);
`); err != nil {
		t.Fatalf("seed sqlite: %v", err)
	}

	objectKey, err := GetProfileThumbnailObjectKeyByUserID(context.Background(), db, "sqlite", 1)
	if err != nil {
		t.Fatalf("GetProfileThumbnailObjectKeyByUserID error: %v", err)
	}
	if objectKey != "profiles/bob-thumb.jpg" {
		t.Fatalf("unexpected object key: %q", objectKey)
	}
}

func TestProfileSettingsQueries_SQLite(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()

	if _, err := db.Exec(`
CREATE TABLE profiles (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  bio TEXT,
  birthdate DATETIME,
  country_code TEXT,
  phone_number_e164 TEXT,
  phone_verified BOOLEAN NOT NULL DEFAULT 0,
  fully_onboarded BOOLEAN NOT NULL DEFAULT 0
);
INSERT INTO profiles (bio, birthdate, country_code, phone_number_e164, phone_verified, fully_onboarded)
VALUES ('hello', '2000-01-01T00:00:00Z', 'CA', '+16045551234', 1, 0);
`); err != nil {
		t.Fatalf("seed sqlite: %v", err)
	}

	settings, err := GetProfileSettingsByID(context.Background(), db, "sqlite", 1)
	if err != nil {
		t.Fatalf("GetProfileSettingsByID error: %v", err)
	}
	if settings.ID != 1 || settings.Bio != "hello" || !settings.PhoneVerified || settings.FullyOnboarded {
		t.Fatalf("unexpected settings row: %+v", settings)
	}

	if err := UpdateProfileBioByID(context.Background(), db, "sqlite", 1, "updated bio"); err != nil {
		t.Fatalf("UpdateProfileBioByID error: %v", err)
	}
	if err := UpdateProfilePhoneByID(context.Background(), db, "sqlite", 1, "US", "+12065550123"); err != nil {
		t.Fatalf("UpdateProfilePhoneByID error: %v", err)
	}
	if err := MarkProfileFullyOnboardedByID(context.Background(), db, "sqlite", 1); err != nil {
		t.Fatalf("MarkProfileFullyOnboardedByID error: %v", err)
	}
	if err := MarkProfilePhoneVerifiedByID(context.Background(), db, "sqlite", 1); err != nil {
		t.Fatalf("MarkProfilePhoneVerifiedByID error: %v", err)
	}

	settings, err = GetProfileSettingsByID(context.Background(), db, "sqlite", 1)
	if err != nil {
		t.Fatalf("GetProfileSettingsByID (after updates) error: %v", err)
	}
	if settings.Bio != "updated bio" {
		t.Fatalf("unexpected bio after update: %q", settings.Bio)
	}
	if !settings.CountryCode.Valid || settings.CountryCode.String != "US" {
		t.Fatalf("unexpected country code after update: %+v", settings.CountryCode)
	}
	if !settings.PhoneNumberE164.Valid || settings.PhoneNumberE164.String != "+12065550123" {
		t.Fatalf("unexpected phone number after update: %+v", settings.PhoneNumberE164)
	}
	if !settings.FullyOnboarded {
		t.Fatalf("expected fully onboarded=true")
	}
	if !settings.PhoneVerified {
		t.Fatalf("expected phone verified=true")
	}
}

func TestProfileFriendsQueries_SQLite(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()

	if _, err := db.Exec(`
CREATE TABLE users (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL,
  email TEXT NOT NULL
);
CREATE TABLE profiles (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_profile INTEGER NOT NULL,
  age INTEGER,
  bio TEXT,
  phone_number_e164 TEXT,
  country_code TEXT
);
CREATE TABLE profile_friends (
  profile_id INTEGER NOT NULL,
  friend_id INTEGER NOT NULL
);
INSERT INTO users (id, name, email) VALUES (1, 'Alice', 'alice@example.com');
INSERT INTO users (id, name, email) VALUES (2, 'Bob', 'bob@example.com');
INSERT INTO profiles (id, user_profile, age, bio, phone_number_e164, country_code) VALUES (10, 1, 30, 'A bio', '+111', 'CA');
INSERT INTO profiles (id, user_profile, age, bio, phone_number_e164, country_code) VALUES (20, 2, 31, 'B bio', '+222', 'US');
INSERT INTO profile_friends (profile_id, friend_id) VALUES (10, 20);
`); err != nil {
		t.Fatalf("seed sqlite: %v", err)
	}

	friends, err := GetFriendsByProfileID(context.Background(), db, "sqlite", 10)
	if err != nil {
		t.Fatalf("GetFriendsByProfileID error: %v", err)
	}
	if len(friends) != 1 {
		t.Fatalf("expected 1 friend, got %d", len(friends))
	}
	if friends[0].ProfileID != 20 || friends[0].UserID != 2 || friends[0].Name != "Bob" {
		t.Fatalf("unexpected friend row: %+v", friends[0])
	}

	exists, err := AreProfilesFriends(context.Background(), db, "sqlite", 10, 20)
	if err != nil {
		t.Fatalf("AreProfilesFriends error: %v", err)
	}
	if !exists {
		t.Fatalf("expected friends relation to exist")
	}

	exists, err = AreProfilesFriends(context.Background(), db, "sqlite", 20, 10)
	if err != nil {
		t.Fatalf("AreProfilesFriends reverse error: %v", err)
	}
	if exists {
		t.Fatalf("expected reverse relation to be false")
	}
}

func TestProfileCoreAndImageQueries_SQLite(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()

	if _, err := db.Exec(`
CREATE TABLE users (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL,
  email TEXT NOT NULL
);
CREATE TABLE profiles (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_profile INTEGER NOT NULL,
  age INTEGER,
  bio TEXT,
  phone_number_e164 TEXT,
  country_code TEXT,
  profile_profile_image INTEGER
);
CREATE TABLE images (
  id INTEGER PRIMARY KEY AUTOINCREMENT
);
CREATE TABLE file_storages (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  object_key TEXT NOT NULL
);
CREATE TABLE image_sizes (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  size TEXT NOT NULL,
  width INTEGER NOT NULL,
  height INTEGER NOT NULL,
  image_sizes INTEGER,
  image_size_file INTEGER NOT NULL
);
INSERT INTO users (id, name, email) VALUES (1, 'Alice', 'alice@example.com');
INSERT INTO images (id) VALUES (100);
INSERT INTO file_storages (id, object_key) VALUES (200, 'profile/thumb.jpg');
INSERT INTO image_sizes (id, size, width, height, image_sizes, image_size_file) VALUES (300, 'thumbnail', 64, 64, 100, 200);
INSERT INTO profiles (id, user_profile, age, bio, phone_number_e164, country_code, profile_profile_image)
VALUES (10, 1, 30, 'hi', '+123', 'CA', 100);
`); err != nil {
		t.Fatalf("seed sqlite: %v", err)
	}

	core, err := GetProfileCoreByID(context.Background(), db, "sqlite", 10)
	if err != nil {
		t.Fatalf("GetProfileCoreByID error: %v", err)
	}
	if core.ProfileID != 10 || core.Name != "Alice" {
		t.Fatalf("unexpected core row: %+v", core)
	}

	imageRows, err := GetProfileImageByProfileID(context.Background(), db, "sqlite", 10)
	if err != nil {
		t.Fatalf("GetProfileImageByProfileID error: %v", err)
	}
	if len(imageRows) != 1 {
		t.Fatalf("expected 1 image row, got %d", len(imageRows))
	}
	if imageRows[0].ImageID != 100 || imageRows[0].ObjectKey != "profile/thumb.jpg" {
		t.Fatalf("unexpected image row: %+v", imageRows[0])
	}
}

func TestCountUnseenNotificationsByProfile_SQLite(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	_, err = db.Exec(`CREATE TABLE notifications (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		profile_notifications INTEGER NOT NULL,
		read BOOLEAN NOT NULL DEFAULT 0
	)`)
	if err != nil {
		t.Fatalf("create table: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO notifications (profile_notifications, read) VALUES
		(1, 0),
		(1, 0),
		(1, 1),
		(2, 0)`); err != nil {
		t.Fatalf("seed notifications: %v", err)
	}

	got, err := CountUnseenNotificationsByProfile(context.Background(), db, "sqlite", 1)
	if err != nil {
		t.Fatalf("count unseen notifications: %v", err)
	}
	if got != 2 {
		t.Fatalf("count = %d, want 2", got)
	}
}

func TestCountUnseenNotificationsByProfileQuery_PostgresPlaceholders(t *testing.T) {
	query, args := countUnseenNotificationsByProfileQuery("postgres", 12)
	if query != "SELECT COUNT(*)\nFROM notifications\nWHERE profile_notifications = $1 AND read = $2;" {
		t.Fatalf("query = %q", query)
	}
	if len(args) != 2 || args[0] != 12 || args[1] != false {
		t.Fatalf("args = %#v", args)
	}
}

func TestProfileFriendshipWriteQueries_SQLite(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()

	if _, err := db.Exec(`
CREATE TABLE profile_friends (
  profile_id INTEGER NOT NULL,
  friend_id INTEGER NOT NULL,
  PRIMARY KEY (profile_id, friend_id)
);
`); err != nil {
		t.Fatalf("seed sqlite: %v", err)
	}

	if err := LinkProfilesAsFriends(context.Background(), db, "sqlite", 1, 2); err != nil {
		t.Fatalf("LinkProfilesAsFriends error: %v", err)
	}
	// Should be idempotent due INSERT OR IGNORE.
	if err := LinkProfilesAsFriends(context.Background(), db, "sqlite", 1, 2); err != nil {
		t.Fatalf("LinkProfilesAsFriends second call error: %v", err)
	}

	exists, err := AreProfilesFriends(context.Background(), db, "sqlite", 1, 2)
	if err != nil {
		t.Fatalf("AreProfilesFriends after link error: %v", err)
	}
	if !exists {
		t.Fatalf("expected friendship to exist after link")
	}

	if err := UnlinkProfilesAsFriends(context.Background(), db, "sqlite", 1, 2); err != nil {
		t.Fatalf("UnlinkProfilesAsFriends error: %v", err)
	}
	exists, err = AreProfilesFriends(context.Background(), db, "sqlite", 1, 2)
	if err != nil {
		t.Fatalf("AreProfilesFriends after unlink error: %v", err)
	}
	if exists {
		t.Fatalf("expected friendship to be removed")
	}
}

func TestProfileMediaWriteQueries_SQLite(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()

	if _, err := db.Exec(`
CREATE TABLE profiles (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  profile_profile_image INTEGER
);
CREATE TABLE images (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL,
  type TEXT NOT NULL,
  profile_photos INTEGER
);
CREATE TABLE file_storages (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  object_key TEXT NOT NULL
);
CREATE TABLE image_sizes (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL,
  size TEXT NOT NULL,
  width INTEGER NOT NULL,
  height INTEGER NOT NULL,
  image_sizes INTEGER,
  image_size_file INTEGER NOT NULL
);
INSERT INTO profiles (id) VALUES (1);
INSERT INTO file_storages (id, object_key) VALUES (10, 'p/a-thumb.jpg');
INSERT INTO file_storages (id, object_key) VALUES (11, 'p/a-preview.jpg');
`); err != nil {
		t.Fatalf("seed sqlite: %v", err)
	}

	now := time.Now().UTC()
	imageID, err := InsertImage(context.Background(), db, "sqlite", "profile_photo", now, now)
	if err != nil {
		t.Fatalf("InsertImage error: %v", err)
	}
	if imageID <= 0 {
		t.Fatalf("expected valid image id, got %d", imageID)
	}

	if err := InsertImageSize(context.Background(), db, "sqlite", "thumbnail", 64, 64, imageID, 10, now, now); err != nil {
		t.Fatalf("InsertImageSize thumbnail error: %v", err)
	}
	if err := InsertImageSize(context.Background(), db, "sqlite", "preview", 256, 256, imageID, 11, now, now); err != nil {
		t.Fatalf("InsertImageSize preview error: %v", err)
	}
	if err := SetProfileImageID(context.Background(), db, "sqlite", 1, imageID); err != nil {
		t.Fatalf("SetProfileImageID error: %v", err)
	}

	gotImageID, err := GetProfileImageIDByProfileID(context.Background(), db, "sqlite", 1)
	if err != nil {
		t.Fatalf("GetProfileImageIDByProfileID error: %v", err)
	}
	if !gotImageID.Valid || int(gotImageID.Int64) != imageID {
		t.Fatalf("unexpected profile image id: %+v", gotImageID)
	}

	rows, err := GetImageStorageObjectsByImageID(context.Background(), db, "sqlite", imageID)
	if err != nil {
		t.Fatalf("GetImageStorageObjectsByImageID error: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 image rows, got %d", len(rows))
	}

	if err := AttachGalleryImageToProfile(context.Background(), db, "sqlite", imageID, 1); err != nil {
		t.Fatalf("AttachGalleryImageToProfile error: %v", err)
	}
	isGalleryOwned, err := ImageBelongsToProfileGallery(context.Background(), db, "sqlite", imageID, 1)
	if err != nil {
		t.Fatalf("ImageBelongsToProfileGallery error: %v", err)
	}
	if !isGalleryOwned {
		t.Fatalf("expected image to be attached to profile gallery")
	}

	if err := ClearProfileImageByImageID(context.Background(), db, "sqlite", imageID); err != nil {
		t.Fatalf("ClearProfileImageByImageID error: %v", err)
	}
	gotImageID, err = GetProfileImageIDByProfileID(context.Background(), db, "sqlite", 1)
	if err != nil {
		t.Fatalf("GetProfileImageIDByProfileID after clear error: %v", err)
	}
	if gotImageID.Valid {
		t.Fatalf("expected cleared profile image id to be null, got %+v", gotImageID)
	}

	if err := DeleteImageByID(context.Background(), db, "sqlite", imageID); err != nil {
		t.Fatalf("DeleteImageByID error: %v", err)
	}
	var remaining int
	if err := db.QueryRow(`SELECT COUNT(1) FROM images WHERE id = ?`, imageID).Scan(&remaining); err != nil {
		t.Fatalf("count images error: %v", err)
	}
	if remaining != 0 {
		t.Fatalf("expected image deleted, remaining=%d", remaining)
	}
}

func TestProfileSubscriptionCleanupAndDeleteUserQueries_SQLite(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()

	if _, err := db.Exec(`
CREATE TABLE users (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL
);
CREATE TABLE profiles (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_profile INTEGER NOT NULL
);
CREATE TABLE monthly_subscriptions (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  paying_profile_id INTEGER NOT NULL
);
CREATE TABLE monthly_subscription_benefactors (
  monthly_subscription_id INTEGER NOT NULL,
  profile_id INTEGER NOT NULL
);
INSERT INTO users (id, name) VALUES (1, 'payer-user');
INSERT INTO users (id, name) VALUES (2, 'benefactor-user');
INSERT INTO profiles (id, user_profile) VALUES (10, 1);
INSERT INTO profiles (id, user_profile) VALUES (20, 2);
INSERT INTO monthly_subscriptions (id, paying_profile_id) VALUES (100, 10);
INSERT INTO monthly_subscription_benefactors (monthly_subscription_id, profile_id) VALUES (100, 10);
INSERT INTO monthly_subscription_benefactors (monthly_subscription_id, profile_id) VALUES (100, 20);
`); err != nil {
		t.Fatalf("seed sqlite: %v", err)
	}

	sub, err := GetSubscriptionForBenefactorByProfileID(context.Background(), db, "sqlite", 20)
	if err != nil {
		t.Fatalf("GetSubscriptionForBenefactorByProfileID error: %v", err)
	}
	if sub.SubscriptionID != 100 || sub.PayingProfile != 10 {
		t.Fatalf("unexpected subscription record: %+v", sub)
	}

	count, err := CountSubscriptionBenefactorsBySubscriptionID(context.Background(), db, "sqlite", 100)
	if err != nil {
		t.Fatalf("CountSubscriptionBenefactorsBySubscriptionID error: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected benefactors count=2, got %d", count)
	}

	if err := RemoveSubscriptionBenefactorBySubscriptionAndProfile(context.Background(), db, "sqlite", 100, 20); err != nil {
		t.Fatalf("RemoveSubscriptionBenefactorBySubscriptionAndProfile error: %v", err)
	}
	count, err = CountSubscriptionBenefactorsBySubscriptionID(context.Background(), db, "sqlite", 100)
	if err != nil {
		t.Fatalf("CountSubscriptionBenefactorsBySubscriptionID after remove error: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected benefactors count=1 after remove, got %d", count)
	}

	if err := DeleteSubscriptionByID(context.Background(), db, "sqlite", 100); err != nil {
		t.Fatalf("DeleteSubscriptionByID error: %v", err)
	}
	var subsLeft int
	if err := db.QueryRow(`SELECT COUNT(1) FROM monthly_subscriptions WHERE id = 100`).Scan(&subsLeft); err != nil {
		t.Fatalf("count subscriptions error: %v", err)
	}
	if subsLeft != 0 {
		t.Fatalf("expected subscription deleted, left=%d", subsLeft)
	}

	if err := DeleteUserByProfileID(context.Background(), db, "sqlite", 20); err != nil {
		t.Fatalf("DeleteUserByProfileID error: %v", err)
	}
	var userLeft int
	if err := db.QueryRow(`SELECT COUNT(1) FROM users WHERE id = 2`).Scan(&userLeft); err != nil {
		t.Fatalf("count users error: %v", err)
	}
	if userLeft != 0 {
		t.Fatalf("expected user deleted, left=%d", userLeft)
	}
}

func TestProfileCreateAndUpdateDetailsQueries_SQLite(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()

	if _, err := db.Exec(`
CREATE TABLE users (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL,
  email TEXT NOT NULL
);
CREATE TABLE profiles (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL,
  bio TEXT,
  birthdate DATETIME,
  age INTEGER,
  country_code TEXT,
  phone_verified BOOLEAN NOT NULL DEFAULT 0,
  fully_onboarded BOOLEAN NOT NULL DEFAULT 0,
  phone_number_e164 TEXT,
  user_profile INTEGER NOT NULL
);
INSERT INTO users (id, name, email) VALUES (1, 'Alice', 'alice@example.com');
`); err != nil {
		t.Fatalf("seed sqlite: %v", err)
	}

	cc := "US"
	phone := "+12065550000"
	now := time.Now().UTC()
	birthdate := now.AddDate(-30, 0, 0)

	profileID, err := InsertProfile(
		context.Background(),
		db,
		"sqlite",
		1,
		"hello",
		birthdate,
		30,
		&cc,
		&phone,
		now,
		now,
	)
	if err != nil {
		t.Fatalf("InsertProfile error: %v", err)
	}
	if profileID <= 0 {
		t.Fatalf("expected valid profile id, got %d", profileID)
	}

	newCC := "CA"
	newPhone := "+16045550000"
	if err := UpdateProfileDetailsByID(
		context.Background(),
		db,
		"sqlite",
		profileID,
		"updated bio",
		birthdate.AddDate(-1, 0, 0),
		31,
		&newCC,
		&newPhone,
	); err != nil {
		t.Fatalf("UpdateProfileDetailsByID error: %v", err)
	}

	settings, err := GetProfileSettingsByID(context.Background(), db, "sqlite", profileID)
	if err != nil {
		t.Fatalf("GetProfileSettingsByID error: %v", err)
	}
	if settings.Bio != "updated bio" {
		t.Fatalf("expected updated bio, got %q", settings.Bio)
	}
	if !settings.CountryCode.Valid || settings.CountryCode.String != "CA" {
		t.Fatalf("expected country code CA, got %+v", settings.CountryCode)
	}
	if !settings.PhoneNumberE164.Valid || settings.PhoneNumberE164.String != "+16045550000" {
		t.Fatalf("expected phone +16045550000, got %+v", settings.PhoneNumberE164)
	}
}
