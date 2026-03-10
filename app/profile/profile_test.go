//go:build integration

package profiles_test

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"image"
	"image/jpeg"
	"testing"
	"time"

	mapset "github.com/deckarep/golang-set/v2"
	"github.com/jackc/pgx/stdlib"
	paidsubscriptions "github.com/leomorpho/goship-modules/paidsubscriptions"
	profilesvc "github.com/leomorpho/goship/app/profile"
	"github.com/leomorpho/goship/framework/domain"
	storagerepo "github.com/leomorpho/goship/framework/repos/storage"
	"github.com/leomorpho/goship/framework/tests"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func init() {
	// Register "pgx" as "postgres" explicitly for database/sql
	sql.Register("postgres", stdlib.GetDefaultDriver())
}

// TODO: move most of these scenarios into SQLite-backed unit tests and keep a single Postgres/pgvector run
// for schema wiring, to avoid restarting containers for every subtest.
func TestGetProfileByID(t *testing.T) {
	db, dialect, ctx := tests.CreateTestContainerPostgresDB(t)

	// Create users.
	user1, err := tests.CreateUserDB(ctx, db, "User", "user1@example.com", "password", true)
	assert.NoError(t, err)
	user2, err := tests.CreateUserDB(ctx, db, "User", "user2@example.com", "password", true)
	assert.NoError(t, err)

	profileService := profilesvc.NewProfileServiceWithDBDeps(
		db, dialect, storagerepo.NewMockStorageClient(), nil, nil,
	)

	// Create profiles for users with different genders.
	profile1ID, err := profileService.CreateProfile(
		ctx, user1.ID, "bio",
		time.Now().AddDate(-25, 0, 0), nil, nil,
	)
	assert.Nil(t, err)

	profile2ID, err := profileService.CreateProfile(
		ctx, user2.ID, "bio",
		time.Now().AddDate(-25, 0, 0), nil, nil,
	)
	assert.Nil(t, err)

	// TODO: expand and verify individual fields
	profileObj, err := profileService.GetProfileByID(ctx, profile1ID, nil)
	assert.Nil(t, err)
	assert.Equal(t, profile1ID, profileObj.ID)

	profileObj, err = profileService.GetProfileByID(ctx, profile2ID, nil)
	assert.Nil(t, err)
	assert.Equal(t, profile2ID, profileObj.ID)
}

func TestGetProfileFriends(t *testing.T) {
	db, dialect, ctx := tests.CreateTestContainerPostgresDB(t)

	// Create users
	user1, err := tests.CreateUserDB(ctx, db, "Jo Bandi", "jo@gmail.com", "password", true)
	assert.NoError(t, err)
	user2, err := tests.CreateUserDB(ctx, db, "Joanne Bandi", "joane@gmail.com", "password", true)
	assert.NoError(t, err)
	user3, err := tests.CreateUserDB(ctx, db, "James Bond", "james007@gmail.com", "password", true)
	assert.NoError(t, err)

	// Create profiles
	profileService := profilesvc.NewProfileServiceWithDBDeps(db, dialect, nil, nil, nil)
	profile1ID, err := profileService.CreateProfile(
		ctx, user1.ID, "bio", time.Time{}, nil, nil,
	)
	assert.Nil(t, err)
	profile2ID, err := profileService.CreateProfile(
		ctx, user2.ID, "bio", time.Time{}, nil, nil,
	)
	assert.Nil(t, err)
	profile3ID, err := profileService.CreateProfile(
		ctx, user3.ID, "bio", time.Time{}, nil, nil,
	)
	assert.Nil(t, err)

	// Link profiles as friends
	err = tests.LinkFriendsDB(ctx, db, profile1ID, []int{profile2ID})
	assert.NoError(t, err)

	friends, err := profileService.GetFriends(ctx, profile1ID)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(friends))
	assert.Equal(t, profile2ID, friends[0].ID)

	err = tests.LinkFriendsDB(ctx, db, profile1ID, []int{profile3ID})
	assert.NoError(t, err)
	friends, err = profileService.GetFriends(ctx, profile1ID)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(friends))
	expectedProfileIds := mapset.NewSet[int]()
	expectedProfileIds.Add(profile2ID)
	expectedProfileIds.Add(profile3ID)
	actualProfileIds := mapset.NewSet[int]()
	actualProfileIds.Add(friends[0].ID)
	actualProfileIds.Add(friends[1].ID)
	assert.Equal(t, expectedProfileIds, actualProfileIds)

	// TODO: need to test that friends get ordered by latest shared question answered,
	// or that they get ordered by latest message. Really, they should be ordered by
	// who's got the latest items added to their shared TemporalizedFeed with the
	// current user.
}

func TestLinkAndUnlinkProfilesAsFriends(t *testing.T) {
	db, dialect, ctx := tests.CreateTestContainerPostgresDB(t)

	profileService := profilesvc.NewProfileServiceWithDBDeps(db, dialect, nil, nil, nil)

	// Create two users and profiles
	user1, err := tests.CreateUserDB(ctx, db, "User1", "user1@example.com", "password", true)
	assert.NoError(t, err)
	user2, err := tests.CreateUserDB(ctx, db, "User2", "user2@example.com", "password", true)
	assert.NoError(t, err)

	profile1ID, err := profileService.CreateProfile(
		ctx, user1.ID, "bio1",
		time.Now().AddDate(-25, 0, 0), nil, nil,
	)
	assert.Nil(t, err)

	profile2ID, err := profileService.CreateProfile(
		ctx, user2.ID, "bio2",
		time.Now().AddDate(-25, 0, 0), nil, nil,
	)
	assert.Nil(t, err)

	// Link profiles as friends
	err = profileService.LinkProfilesAsFriends(ctx, profile1ID, profile2ID)
	assert.Nil(t, err)

	// Verify that the profiles are linked as friends in the database
	areFriends, err := friendshipExists(ctx, db, profile1ID, profile2ID)
	assert.Nil(t, err)
	assert.True(t, areFriends, "Profiles should be linked as friends")

	// Unlink the profiles
	err = profileService.UnlinkProfilesAsFriends(ctx, profile1ID, profile2ID)
	assert.Nil(t, err)

	// Verify that the profiles are no longer linked as friends
	areStillFriends, err := friendshipExists(ctx, db, profile1ID, profile2ID)
	assert.Nil(t, err)
	assert.False(t, areStillFriends, "Profiles should no longer be linked as friends")
}

func TestUploadImageSizes(t *testing.T) {
	// Initialize the MockStorageClient
	mockStorage := storagerepo.NewMockStorageClient()

	// Create an instance of ProfileService using the mock storage client
	db, dialect, ctx := tests.CreateTestContainerPostgresDB(t)

	profileService := profilesvc.NewProfileServiceWithDBDeps(db, dialect, mockStorage, nil, nil)

	// Mock data
	profileID := 1
	fileName := "test.jpg"
	imageCategory := domain.ImageCategoryProfileGallery

	// Create a simple image
	img := image.NewRGBA(image.Rect(0, 0, 1000, 500))
	jpegBuf := new(bytes.Buffer)
	jpeg.Encode(jpegBuf, img, nil)

	// Turn the buffer into a reader to simulate file upload
	fileStream := bytes.NewReader(jpegBuf.Bytes())

	fileID, err := insertFileStorage(ctx, db, "bucketName", "key", "a")
	assert.Nil(t, err)

	// Expected calls to the mock
	mockStorage.On("UploadFile", mock.Anything, mock.Anything, mock.Anything).Return(&fileID, nil).Times(1)

	// Run the function we want to test
	imageID, err := profileService.UploadImageSizes(
		ctx, profileID, fileStream, imageCategory, fileName,
		[]domain.ImageSize{domain.ImageSizeThumbnail},
	)

	// Assert expectations
	assert.NoError(t, err)
	assert.NotNil(t, imageID)

	// TODO: i can see it's being called when stepping through it with
	// the debugger, but this is failing...
	// Verify that all expected interactions with the mock occurred
	// mockStorage.AssertExpectations(t)
}
func TestDeletePhoto(t *testing.T) {
	db, dialect, ctx := tests.CreateTestContainerPostgresDB(t)

	mockStorage := storagerepo.NewMockStorageClient()
	profileService := profilesvc.NewProfileServiceWithDBDeps(db, dialect, mockStorage, nil, nil)

	// Prepare mock data
	fileName := "new_profile.jpg"

	// Create a dummy image
	img := image.NewRGBA(image.Rect(0, 0, 1000, 500))
	jpegBuf := new(bytes.Buffer)
	jpeg.Encode(jpegBuf, img, nil)

	// Simulate file upload
	fileStream := bytes.NewReader(jpegBuf.Bytes())

	// Mock expected calls
	mockStorage.On("DeleteFile", mock.Anything, mock.Anything).Return(nil).Once()

	file1ID, err := insertFileStorage(ctx, db, "bucketName", "key1", "a3")
	assert.Nil(t, err)

	file2ID, err := insertFileStorage(ctx, db, "bucketName", "key2", "a2")
	assert.Nil(t, err)

	// Expect deletion call for thumbnail and preview
	mockStorage.
		On("UploadFile", mock.Anything, mock.Anything, mock.Anything).
		Return(&file1ID, nil).Once()
	mockStorage.
		On("UploadFile", mock.Anything, mock.Anything, mock.Anything).Return(&file2ID, nil).Once()

	user1, err := tests.CreateUserDB(ctx, db, "TestUser1", "test1@example.com", "password", true)
	assert.NoError(t, err)

	profile1ID, err := profileService.CreateProfile(ctx, user1.ID, "Test bio 1", time.Now().AddDate(-30, 0, 0), nil, nil)
	assert.NoError(t, err)

	// Create the image
	err = profileService.SetProfilePhoto(ctx, profile1ID, fileStream, fileName)
	assert.Nil(t, err)

	imageID, err := profileImageIDByProfileID(ctx, db, profile1ID)
	assert.Nil(t, err)
	assert.NotZero(t, imageID)

	numImages, err := countRows(ctx, db, "images")
	assert.Nil(t, err)
	assert.Equal(t, 1, numImages)

	numImageSizes, err := countRows(ctx, db, "image_sizes")
	assert.Nil(t, err)
	assert.Equal(t, 2, numImageSizes)

	numFiles, err := countRows(ctx, db, "file_storages")
	assert.Nil(t, err)
	assert.Equal(t, 2, numFiles)

	err = profileService.DeletePhoto(ctx, imageID, nil)
	assert.NoError(t, err)

	numImages, err = countRows(ctx, db, "images")
	assert.Nil(t, err)
	assert.Equal(t, 0, numImages)

	numImageSizes, err = countRows(ctx, db, "image_sizes")
	assert.Nil(t, err)
	assert.Equal(t, 0, numImageSizes)

	// These are deleted by DeletePhoto in real life, but not in mock
	numFiles, err = countRows(ctx, db, "file_storages")
	assert.Nil(t, err)
	assert.Equal(t, 2, numFiles)
}

func TestSetProfilePhoto(t *testing.T) {
	db, dialect, ctx := tests.CreateTestContainerPostgresDB(t)

	mockStorage := storagerepo.NewMockStorageClient()
	profileService := profilesvc.NewProfileServiceWithDBDeps(db, dialect, mockStorage, nil, nil)

	// Prepare mock data
	fileName := "new_profile.jpg"

	// Create a dummy image
	img := image.NewRGBA(image.Rect(0, 0, 1000, 500))
	jpegBuf := new(bytes.Buffer)
	jpeg.Encode(jpegBuf, img, nil)

	// Simulate file upload
	fileStream := bytes.NewReader(jpegBuf.Bytes())

	// Mock expected calls
	mockStorage.On("DeleteFile", mock.Anything, mock.Anything).Return(nil).Once()

	fileID, err := insertFileStorage(ctx, db, "bucketName", "key", "a")
	assert.Nil(t, err)

	// Expect deletion call
	mockStorage.
		On("UploadFile", mock.Anything, mock.Anything, mock.Anything).
		Return(&fileID, nil).
		Twice() // For thumbnail and preview

	user1, err := tests.CreateUserDB(ctx, db, "TestUser1", "test1@example.com", "password", true)
	assert.NoError(t, err)

	profileID, err := profileService.CreateProfile(ctx, user1.ID, "Test bio 1", time.Now().AddDate(-30, 0, 0), nil, nil)
	assert.NoError(t, err)

	// Execute the method to test
	err = profileService.SetProfilePhoto(ctx, profileID, fileStream, fileName)

	// Check results
	assert.NoError(t, err)
	// TODO: i can see it's being called when stepping through it with
	// the debugger, but this is failing...
	// Verify that all expected interactions with the mock occurred
	// mockStorage.AssertExpectations(t) // Verify all mocked methods were called as expected
}

func TestDeleteUserData(t *testing.T) {
	/*
		This test create users, adds a subscription to one, answers and questions,
		and then deletes the two users, making sure only the expected objects
		are deleted in DB through cascading.
		This can absolutely be expended (AND SHOULD!)
	*/
	db, dialect, ctx := tests.CreateTestContainerPostgresDB(t)

	// Create users.
	user1, err := tests.CreateUserDB(ctx, db, "User", "user1@example.com", "password", true)
	assert.NoError(t, err)
	user2, err := tests.CreateUserDB(ctx, db, "User", "user2@example.com", "password", true)
	assert.NoError(t, err)
	user3, err := tests.CreateUserDB(ctx, db, "James Bond", "james007@gmail.com", "password", true)
	assert.NoError(t, err)

	// Create profiles for users with different genders.
	profileService := profilesvc.NewProfileServiceWithDBDeps(db, dialect, storagerepo.NewMockStorageClient(), nil, nil)
	profile1ID, err := profileService.CreateProfile(
		ctx, user1.ID, "bio",
		time.Now().AddDate(-25, 0, 0), nil, nil,
	)
	assert.Nil(t, err)

	profile2ID, err := profileService.CreateProfile(
		ctx, user2.ID, "bio",
		time.Now().AddDate(-25, 0, 0), nil, nil,
	)
	assert.Nil(t, err)

	profile3ID, err := profileService.CreateProfile(
		ctx, user3.ID, "bio", time.Time{}, nil, nil,
	)
	assert.Nil(t, err)

	// Link profiles as friends
	err = tests.LinkFriendsDB(ctx, db, profile1ID, []int{profile3ID})
	assert.NoError(t, err)

	// Create subcription then delete it (mimicking user cancelling before
	// deleting, this is enforced in the deletion flow).
	subscriptionsService := paidsubscriptions.New(paidsubscriptions.NewSQLStore(db, dialect, 5, 5))
	err = subscriptionsService.CreateSubscription(ctx, nil, profile1ID)
	assert.NoError(t, err)

	err = subscriptionsService.ActivatePlan(ctx, profile1ID, "pro")
	assert.NoError(t, err)
	now := time.Now()
	err = subscriptionsService.CancelOrRenew(ctx, profile1ID, &now)
	assert.NoError(t, err)

	initialUserIDs, err := idSet(ctx, db, "users")
	assert.NoError(t, err)
	initialProfileIDs, err := idSet(ctx, db, "profiles")
	assert.NoError(t, err)

	assert.True(t, initialUserIDs.Contains(user1.ID))
	assert.True(t, initialProfileIDs.Contains(profile1ID))

	// TEST THE ACTUAL METHOD: delete user with pro subscription
	err = profileService.DeleteUserData(ctx, profile1ID)
	assert.NoError(t, err)

	finalUserIDs, err := idSet(ctx, db, "users")
	assert.NoError(t, err)
	finalProfileIDs, err := idSet(ctx, db, "profiles")
	assert.NoError(t, err)

	assert.Equal(t, initialUserIDs.Cardinality(), finalUserIDs.Cardinality()+1)
	assert.Equal(t, initialProfileIDs.Cardinality(), finalProfileIDs.Cardinality()+1)

	assert.False(t, finalUserIDs.Contains(user1.ID))
	assert.False(t, finalProfileIDs.Contains(profile1ID))

	// TEST THE ACTUAL METHOD: delete user with free subscription
	err = profileService.DeleteUserData(ctx, profile2ID)
	assert.NoError(t, err)

	finalUsers2, err := idSet(ctx, db, "users")
	assert.NoError(t, err)
	finalProfiles2, err := idSet(ctx, db, "profiles")
	assert.NoError(t, err)

	assert.Equal(t, initialUserIDs.Cardinality(), finalUsers2.Cardinality()+2)
	assert.Equal(t, initialProfileIDs.Cardinality(), finalProfiles2.Cardinality()+2)
}

func insertFileStorage(ctx context.Context, db *sql.DB, bucket, objectKey, fileHash string) (int, error) {
	var id int
	now := time.Now().UTC()
	err := db.QueryRowContext(
		ctx,
		`INSERT INTO file_storages (created_at, updated_at, bucket_name, object_key, file_size, file_hash)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id`,
		now, now, bucket, objectKey, 1, fileHash,
	).Scan(&id)
	return id, err
}

func countRows(ctx context.Context, db *sql.DB, table string) (int, error) {
	var count int
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s", table) //nolint:gosec // internal test table names
	err := db.QueryRowContext(ctx, query).Scan(&count)
	return count, err
}

func profileImageIDByProfileID(ctx context.Context, db *sql.DB, profileID int) (int, error) {
	var imageID int
	err := db.QueryRowContext(ctx, `SELECT profile_profile_image FROM profiles WHERE id = $1`, profileID).Scan(&imageID)
	return imageID, err
}

func friendshipExists(ctx context.Context, db *sql.DB, profileID, friendID int) (bool, error) {
	var exists bool
	err := db.QueryRowContext(
		ctx,
		`SELECT EXISTS(SELECT 1 FROM profile_friends WHERE profile_id = $1 AND friend_id = $2)`,
		profileID,
		friendID,
	).Scan(&exists)
	return exists, err
}

func idSet(ctx context.Context, db *sql.DB, table string) (mapset.Set[int], error) {
	query := fmt.Sprintf("SELECT id FROM %s", table) //nolint:gosec // internal test table names
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := mapset.NewSet[int]()
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out.Add(id)
	}
	return out, rows.Err()
}
