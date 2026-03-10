package profiles

import (
	"context"
	"database/sql"
	"io"
	"testing"
	"time"

	"github.com/leomorpho/goship/framework/domain"
	storagerepo "github.com/leomorpho/goship/framework/repos/storage"
	_ "github.com/mattn/go-sqlite3"
)

type cleanupStorageFake struct {
	deleted []string
}

func (f *cleanupStorageFake) CreateBucket(bucketName string, location string) error {
	return nil
}

func (f *cleanupStorageFake) UploadFile(bucket storagerepo.Bucket, objectName string, fileStream io.Reader) (*int, error) {
	id := 1
	return &id, nil
}

func (f *cleanupStorageFake) DeleteFile(bucket storagerepo.Bucket, objectName string) error {
	f.deleted = append(f.deleted, objectName)
	return nil
}

func (f *cleanupStorageFake) GetPresignedURL(bucket storagerepo.Bucket, objectName string, expiry time.Duration) (string, error) {
	return "https://example.com/" + objectName, nil
}

func (f *cleanupStorageFake) GetImageObjectFromFile(file *storagerepo.ImageFile) (*domain.Photo, error) {
	return &domain.Photo{}, nil
}

func (f *cleanupStorageFake) GetImageObjectsFromFiles(files []*storagerepo.ImageFile) ([]domain.Photo, error) {
	return []domain.Photo{}, nil
}

func TestDeleteProfileStorageFiles_SQLPath(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()

	_, err = db.Exec(`
CREATE TABLE profiles (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  profile_profile_image INTEGER
);
CREATE TABLE images (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  profile_photos INTEGER,
  created_at DATETIME
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
INSERT INTO images (id, profile_photos, created_at) VALUES (100, NULL, '2026-01-01T00:00:00Z');
INSERT INTO images (id, profile_photos, created_at) VALUES (101, 1, '2026-01-02T00:00:00Z');
INSERT INTO file_storages (id, object_key) VALUES (200, 'profile-thumb.jpg');
INSERT INTO file_storages (id, object_key) VALUES (201, 'gallery-preview.jpg');
INSERT INTO image_sizes (id, size, width, height, image_sizes, image_size_file) VALUES (300, 'thumbnail', 64, 64, 100, 200);
INSERT INTO image_sizes (id, size, width, height, image_sizes, image_size_file) VALUES (301, 'preview', 320, 240, 101, 201);
INSERT INTO profiles (id, profile_profile_image) VALUES (1, 100);
`)
	if err != nil {
		t.Fatalf("seed sqlite: %v", err)
	}

	fake := &cleanupStorageFake{}
	svc := NewProfileServiceWithDBDeps(db, "sqlite", fake, nil, nil)

	if err := svc.deleteProfileStorageFiles(context.Background(), 1); err != nil {
		t.Fatalf("deleteProfileStorageFiles returned error: %v", err)
	}

	if len(fake.deleted) != 2 {
		t.Fatalf("expected 2 deleted object keys, got %d (%v)", len(fake.deleted), fake.deleted)
	}

	foundProfile := false
	foundGallery := false
	for _, key := range fake.deleted {
		if key == "profile-thumb.jpg" {
			foundProfile = true
		}
		if key == "gallery-preview.jpg" {
			foundGallery = true
		}
	}
	if !foundProfile || !foundGallery {
		t.Fatalf("unexpected deleted keys: %v", fake.deleted)
	}
}

func TestDeleteUserData_SQLPath_RemovesBenefactorAndUser(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()

	_, err = db.Exec(`
CREATE TABLE users (
  id INTEGER PRIMARY KEY AUTOINCREMENT
);
CREATE TABLE profiles (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_profile INTEGER NOT NULL,
  profile_profile_image INTEGER
);
CREATE TABLE images (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  profile_photos INTEGER,
  created_at DATETIME
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
CREATE TABLE monthly_subscriptions (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  paying_profile_id INTEGER NOT NULL
);
CREATE TABLE monthly_subscription_benefactors (
  monthly_subscription_id INTEGER NOT NULL,
  profile_id INTEGER NOT NULL
);

INSERT INTO users (id) VALUES (1);
INSERT INTO users (id) VALUES (2);
INSERT INTO profiles (id, user_profile, profile_profile_image) VALUES (10, 1, 100);
INSERT INTO profiles (id, user_profile, profile_profile_image) VALUES (20, 2, NULL);

INSERT INTO images (id, profile_photos, created_at) VALUES (100, NULL, '2026-01-01T00:00:00Z');
INSERT INTO images (id, profile_photos, created_at) VALUES (101, 10, '2026-01-02T00:00:00Z');
INSERT INTO file_storages (id, object_key) VALUES (200, 'profile-thumb.jpg');
INSERT INTO file_storages (id, object_key) VALUES (201, 'gallery-preview.jpg');
INSERT INTO image_sizes (id, size, width, height, image_sizes, image_size_file) VALUES (300, 'thumbnail', 64, 64, 100, 200);
INSERT INTO image_sizes (id, size, width, height, image_sizes, image_size_file) VALUES (301, 'preview', 320, 240, 101, 201);

INSERT INTO monthly_subscriptions (id, paying_profile_id) VALUES (500, 20);
INSERT INTO monthly_subscription_benefactors (monthly_subscription_id, profile_id) VALUES (500, 20);
INSERT INTO monthly_subscription_benefactors (monthly_subscription_id, profile_id) VALUES (500, 10);
`)
	if err != nil {
		t.Fatalf("seed sqlite: %v", err)
	}

	fake := &cleanupStorageFake{}
	svc := NewProfileServiceWithDBDeps(db, "sqlite", fake, nil, nil)
	if err := svc.DeleteUserData(context.Background(), 10); err != nil {
		t.Fatalf("DeleteUserData returned error: %v", err)
	}

	if len(fake.deleted) != 2 {
		t.Fatalf("expected 2 deleted object keys, got %d (%v)", len(fake.deleted), fake.deleted)
	}

	var usersLeft int
	if err := db.QueryRow(`SELECT COUNT(1) FROM users WHERE id = 1`).Scan(&usersLeft); err != nil {
		t.Fatalf("count users error: %v", err)
	}
	if usersLeft != 0 {
		t.Fatalf("expected user 1 to be deleted, got %d rows", usersLeft)
	}

	var benefactorRows int
	if err := db.QueryRow(`SELECT COUNT(1) FROM monthly_subscription_benefactors WHERE monthly_subscription_id = 500 AND profile_id = 10`).Scan(&benefactorRows); err != nil {
		t.Fatalf("count benefactors error: %v", err)
	}
	if benefactorRows != 0 {
		t.Fatalf("expected benefactor profile 10 removed from subscription, got %d rows", benefactorRows)
	}

	var subscriptionLeft int
	if err := db.QueryRow(`SELECT COUNT(1) FROM monthly_subscriptions WHERE id = 500`).Scan(&subscriptionLeft); err != nil {
		t.Fatalf("count subscriptions error: %v", err)
	}
	if subscriptionLeft != 1 {
		t.Fatalf("expected subscription to remain for payer profile, got %d rows", subscriptionLeft)
	}
}
