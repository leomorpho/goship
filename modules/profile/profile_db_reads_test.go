package profiles

import (
	"context"
	"database/sql"
	"testing"
	"time"

	storagerepo "github.com/leomorpho/goship/framework/repos/storage"
	_ "github.com/mattn/go-sqlite3"
)

func TestIsProfileFullyOnboardedByUserID_UsesDBStore(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()

	if _, err := db.Exec(`
CREATE TABLE users (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL, email TEXT NOT NULL);
CREATE TABLE profiles (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_profile INTEGER NOT NULL,
  fully_onboarded BOOLEAN NOT NULL DEFAULT 0
);
INSERT INTO users (name, email) VALUES ('Alice', 'alice@example.com');
INSERT INTO profiles (user_profile, fully_onboarded) VALUES (1, 1);
`); err != nil {
		t.Fatalf("seed sqlite: %v", err)
	}

	svc := NewProfileServiceWithDBDeps(db, "sqlite", nil, nil, nil)
	got, err := svc.IsProfileFullyOnboardedByUserID(context.Background(), 1)
	if err != nil {
		t.Fatalf("IsProfileFullyOnboardedByUserID error: %v", err)
	}
	if !got {
		t.Fatalf("expected fully onboarded true, got false")
	}
}

func TestIsProfileFullyOnboardedByUserID_FailsWithoutDB(t *testing.T) {
	svc := NewProfileServiceWithDBDeps(nil, "sqlite", nil, nil, nil)
	_, err := svc.IsProfileFullyOnboardedByUserID(context.Background(), 1)
	if err == nil {
		t.Fatal("expected error when db is missing")
	}
	if err != ErrProfileDBNotConfigured {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetProfilePhotoThumbnailURL_UsesDBStore(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()

	if _, err := db.Exec(`
CREATE TABLE users (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL, email TEXT NOT NULL);
CREATE TABLE images (id INTEGER PRIMARY KEY AUTOINCREMENT);
CREATE TABLE file_storages (id INTEGER PRIMARY KEY AUTOINCREMENT, object_key TEXT NOT NULL);
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

	svc := NewProfileServiceWithDBDeps(db, "sqlite", storagerepo.NewMockStorageClient(), nil, nil)
	got := svc.GetProfilePhotoThumbnailURL(1)
	if got == "" {
		t.Fatalf("expected signed URL, got empty string")
	}
}

func TestGetProfilePhotoThumbnailURL_DefaultWhenDepsMissing(t *testing.T) {
	svc := NewProfileServiceWithDBDeps(nil, "sqlite", nil, nil, nil)
	got := svc.GetProfilePhotoThumbnailURL(1)
	if got == "" {
		t.Fatal("expected fallback profile URL")
	}
}

func TestCreateAndUpdateProfile_SQLPath(t *testing.T) {
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
  phone_number_e164 TEXT,
  phone_verified BOOLEAN NOT NULL DEFAULT 0,
  fully_onboarded BOOLEAN NOT NULL DEFAULT 0,
  user_profile INTEGER NOT NULL
);
INSERT INTO users (id, name, email) VALUES (1, 'Alice', 'alice@example.com');
`); err != nil {
		t.Fatalf("seed sqlite: %v", err)
	}

	svc := NewProfileServiceWithDBDeps(db, "sqlite", nil, nil, nil)
	cc := "US"
	phone := "+12065550000"
	profileID, err := svc.CreateProfile(
		context.Background(),
		1,
		"initial bio",
		time.Now().UTC().AddDate(-30, 0, 0),
		&cc,
		&phone,
	)
	if err != nil {
		t.Fatalf("CreateProfile error: %v", err)
	}
	if profileID <= 0 {
		t.Fatalf("expected valid profile id, got %d", profileID)
	}

	updatedCC := "CA"
	updatedPhone := "+16045550000"
	err = svc.UpdateProfile(
		context.Background(),
		profileID,
		false,
		"updated bio",
		time.Now().UTC().AddDate(-31, 0, 0),
		&updatedCC,
		&updatedPhone,
	)
	if err != nil {
		t.Fatalf("UpdateProfile error: %v", err)
	}

	settings, err := svc.GetProfileSettingsByID(context.Background(), profileID)
	if err != nil {
		t.Fatalf("GetProfileSettingsByID error: %v", err)
	}
	if settings.Bio != "updated bio" {
		t.Fatalf("expected updated bio, got %q", settings.Bio)
	}
	if settings.CountryCode != "CA" {
		t.Fatalf("expected country code CA, got %q", settings.CountryCode)
	}
	if settings.PhoneNumberE164 != "+16045550000" {
		t.Fatalf("expected updated phone, got %q", settings.PhoneNumberE164)
	}
}

func TestGetProfileByID_DerivesAgeFromBirthdate(t *testing.T) {
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
  birthdate DATETIME,
  age INTEGER,
  bio TEXT,
  phone_number_e164 TEXT,
  country_code TEXT,
  profile_profile_image INTEGER
);
CREATE TABLE images (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  profile_photos INTEGER,
  created_at DATETIME
);
CREATE TABLE image_sizes (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  size TEXT NOT NULL,
  width INTEGER NOT NULL,
  height INTEGER NOT NULL,
  image_sizes INTEGER,
  image_size_file INTEGER NOT NULL
);
CREATE TABLE file_storages (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  object_key TEXT NOT NULL
);
INSERT INTO users (id, name, email) VALUES (1, 'Alice', 'alice@example.com');
INSERT INTO profiles (id, user_profile, birthdate, age, bio) VALUES (1, 1, '2000-03-27T00:00:00Z', 99, 'bio');
`); err != nil {
		t.Fatalf("seed sqlite: %v", err)
	}

	svc := NewProfileServiceWithDBDeps(db, "sqlite", storagerepo.NewMockStorageClient(), nil, nil)
	profile, err := svc.GetProfileByID(context.Background(), 1, nil)
	if err != nil {
		t.Fatalf("GetProfileByID error: %v", err)
	}
	now := time.Now().UTC()
	expected := now.Year() - 2000
	if now.Month() < time.March || (now.Month() == time.March && now.Day() < 27) {
		expected--
	}
	if profile.Age != expected {
		t.Fatalf("profile age = %d, want %d", profile.Age, expected)
	}
}
