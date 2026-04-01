package gen

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/leomorpho/goship/v2/db/queries"
)

type ProfileSettingsRecord struct {
	ID              int
	Bio             string
	Birthdate       sql.NullTime
	CountryCode     sql.NullString
	PhoneNumberE164 sql.NullString
	PhoneVerified   bool
	FullyOnboarded  bool
}

type ProfileFriendRecord struct {
	ProfileID       int
	UserID          int
	Name            string
	Age             sql.NullInt64
	Bio             sql.NullString
	PhoneNumberE164 sql.NullString
	CountryCode     sql.NullString
}

type ProfilePhotoSizeRecord struct {
	ImageID   int
	Size      string
	Width     int
	Height    int
	ObjectKey string
}

type ProfileCoreRecord struct {
	ProfileID       int
	Name            string
	Age             sql.NullInt64
	Bio             sql.NullString
	PhoneNumberE164 sql.NullString
	CountryCode     sql.NullString
}

type SubscriptionBenefactorRecord struct {
	SubscriptionID int
	PayingProfile  int
}

type Queryer interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

func GetProfileFullyOnboardedByUserID(
	ctx context.Context,
	db QueryRower,
	dialect string,
	userID int,
) (bool, error) {
	if db == nil {
		return false, errors.New("query runner is nil")
	}
	query, args := getProfileFullyOnboardedByUserIDQuery(strings.ToLower(strings.TrimSpace(dialect)), userID)
	var onboarded bool
	if err := db.QueryRowContext(ctx, query, args...).Scan(&onboarded); err != nil {
		return false, err
	}
	return onboarded, nil
}

func GetProfileThumbnailObjectKeyByUserID(
	ctx context.Context,
	db QueryRower,
	dialect string,
	userID int,
) (string, error) {
	if db == nil {
		return "", errors.New("query runner is nil")
	}
	query, args := getProfileThumbnailObjectKeyByUserIDQuery(strings.ToLower(strings.TrimSpace(dialect)), userID)
	var objectKey string
	if err := db.QueryRowContext(ctx, query, args...).Scan(&objectKey); err != nil {
		return "", err
	}
	return objectKey, nil
}

func GetProfileSettingsByID(
	ctx context.Context,
	db QueryRower,
	dialect string,
	profileID int,
) (*ProfileSettingsRecord, error) {
	if db == nil {
		return nil, errors.New("query runner is nil")
	}
	query, args := getProfileSettingsByIDQuery(strings.ToLower(strings.TrimSpace(dialect)), profileID)
	var row ProfileSettingsRecord
	if err := db.QueryRowContext(ctx, query, args...).Scan(
		&row.ID,
		&row.Bio,
		&row.Birthdate,
		&row.CountryCode,
		&row.PhoneNumberE164,
		&row.PhoneVerified,
		&row.FullyOnboarded,
	); err != nil {
		return nil, err
	}
	return &row, nil
}

func UpdateProfileBioByID(
	ctx context.Context,
	db Execer,
	dialect string,
	profileID int,
	bio string,
) error {
	if db == nil {
		return errors.New("exec runner is nil")
	}
	query, args := updateProfileBioByIDQuery(strings.ToLower(strings.TrimSpace(dialect)), profileID, bio)
	_, err := db.ExecContext(ctx, query, args...)
	return err
}

func UpdateProfilePhoneByID(
	ctx context.Context,
	db Execer,
	dialect string,
	profileID int,
	countryCode string,
	phoneE164 string,
) error {
	if db == nil {
		return errors.New("exec runner is nil")
	}
	query, args := updateProfilePhoneByIDQuery(strings.ToLower(strings.TrimSpace(dialect)), profileID, countryCode, phoneE164)
	_, err := db.ExecContext(ctx, query, args...)
	return err
}

func InsertProfile(
	ctx context.Context,
	db QueryExecRower,
	dialect string,
	userID int,
	bio string,
	birthdate time.Time,
	age int,
	countryCode *string,
	phoneE164 *string,
	createdAt time.Time,
	updatedAt time.Time,
) (int, error) {
	if db == nil {
		return 0, errors.New("query runner is nil")
	}
	query, args := insertProfileQuery(
		strings.ToLower(strings.TrimSpace(dialect)),
		createdAt,
		updatedAt,
		bio,
		birthdate,
		age,
		countryCode,
		phoneE164,
		userID,
	)
	var profileID int
	if err := db.QueryRowContext(ctx, query, args...).Scan(&profileID); err != nil {
		return 0, err
	}
	return profileID, nil
}

func UpdateProfileDetailsByID(
	ctx context.Context,
	db Execer,
	dialect string,
	profileID int,
	bio string,
	birthdate time.Time,
	age int,
	countryCode *string,
	phoneE164 *string,
) error {
	if db == nil {
		return errors.New("exec runner is nil")
	}
	query, args := updateProfileDetailsByIDQuery(
		strings.ToLower(strings.TrimSpace(dialect)),
		profileID,
		bio,
		birthdate,
		age,
		countryCode,
		phoneE164,
	)
	_, err := db.ExecContext(ctx, query, args...)
	return err
}

func MarkProfileFullyOnboardedByID(
	ctx context.Context,
	db Execer,
	dialect string,
	profileID int,
) error {
	if db == nil {
		return errors.New("exec runner is nil")
	}
	query, args := markProfileFullyOnboardedByIDQuery(strings.ToLower(strings.TrimSpace(dialect)), profileID)
	_, err := db.ExecContext(ctx, query, args...)
	return err
}

func MarkProfilePhoneVerifiedByID(
	ctx context.Context,
	db Execer,
	dialect string,
	profileID int,
) error {
	if db == nil {
		return errors.New("exec runner is nil")
	}
	query, args := markProfilePhoneVerifiedByIDQuery(strings.ToLower(strings.TrimSpace(dialect)), profileID)
	_, err := db.ExecContext(ctx, query, args...)
	return err
}

func GetFriendsByProfileID(
	ctx context.Context,
	db Queryer,
	dialect string,
	profileID int,
) ([]ProfileFriendRecord, error) {
	if db == nil {
		return nil, errors.New("query runner is nil")
	}
	query, args := getFriendsByProfileIDQuery(strings.ToLower(strings.TrimSpace(dialect)), profileID)
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []ProfileFriendRecord
	for rows.Next() {
		var rec ProfileFriendRecord
		if err := rows.Scan(
			&rec.ProfileID,
			&rec.UserID,
			&rec.Name,
			&rec.Age,
			&rec.Bio,
			&rec.PhoneNumberE164,
			&rec.CountryCode,
		); err != nil {
			return nil, err
		}
		out = append(out, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func AreProfilesFriends(
	ctx context.Context,
	db QueryRower,
	dialect string,
	profileID1 int,
	profileID2 int,
) (bool, error) {
	if db == nil {
		return false, errors.New("query runner is nil")
	}
	query, args := areProfilesFriendsQuery(strings.ToLower(strings.TrimSpace(dialect)), profileID1, profileID2)
	var exists bool
	if err := db.QueryRowContext(ctx, query, args...).Scan(&exists); err != nil {
		return false, err
	}
	return exists, nil
}

func GetProfilePhotosByProfileID(
	ctx context.Context,
	db Queryer,
	dialect string,
	profileID int,
) ([]ProfilePhotoSizeRecord, error) {
	if db == nil {
		return nil, errors.New("query runner is nil")
	}
	query, args := getProfilePhotosByProfileIDQuery(strings.ToLower(strings.TrimSpace(dialect)), profileID)
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []ProfilePhotoSizeRecord
	for rows.Next() {
		var rec ProfilePhotoSizeRecord
		if err := rows.Scan(&rec.ImageID, &rec.Size, &rec.Width, &rec.Height, &rec.ObjectKey); err != nil {
			return nil, err
		}
		out = append(out, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func GetProfileCoreByID(
	ctx context.Context,
	db QueryRower,
	dialect string,
	profileID int,
) (*ProfileCoreRecord, error) {
	if db == nil {
		return nil, errors.New("query runner is nil")
	}
	query, args := getProfileCoreByIDQuery(strings.ToLower(strings.TrimSpace(dialect)), profileID)
	var rec ProfileCoreRecord
	if err := db.QueryRowContext(ctx, query, args...).Scan(
		&rec.ProfileID,
		&rec.Name,
		&rec.Age,
		&rec.Bio,
		&rec.PhoneNumberE164,
		&rec.CountryCode,
	); err != nil {
		return nil, err
	}
	return &rec, nil
}

func CountUnseenNotificationsByProfile(
	ctx context.Context,
	db QueryRower,
	dialect string,
	profileID int,
) (int, error) {
	if db == nil {
		return 0, errors.New("query runner is nil")
	}
	query, args := countUnseenNotificationsByProfileQuery(strings.ToLower(strings.TrimSpace(dialect)), profileID)
	var count int
	if err := db.QueryRowContext(ctx, query, args...).Scan(&count); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, nil
		}
		return 0, err
	}
	return count, nil
}

func GetProfileImageByProfileID(
	ctx context.Context,
	db Queryer,
	dialect string,
	profileID int,
) ([]ProfilePhotoSizeRecord, error) {
	if db == nil {
		return nil, errors.New("query runner is nil")
	}
	query, args := getProfileImageByProfileIDQuery(strings.ToLower(strings.TrimSpace(dialect)), profileID)
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []ProfilePhotoSizeRecord
	for rows.Next() {
		var rec ProfilePhotoSizeRecord
		if err := rows.Scan(&rec.ImageID, &rec.Size, &rec.Width, &rec.Height, &rec.ObjectKey); err != nil {
			return nil, err
		}
		out = append(out, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func LinkProfilesAsFriends(
	ctx context.Context,
	db Execer,
	dialect string,
	profileID int,
	friendID int,
) error {
	if db == nil {
		return errors.New("exec runner is nil")
	}
	query, args := linkProfilesAsFriendsQuery(strings.ToLower(strings.TrimSpace(dialect)), profileID, friendID)
	_, err := db.ExecContext(ctx, query, args...)
	return err
}

func UnlinkProfilesAsFriends(
	ctx context.Context,
	db Execer,
	dialect string,
	profileID int,
	friendID int,
) error {
	if db == nil {
		return errors.New("exec runner is nil")
	}
	query, args := unlinkProfilesAsFriendsQuery(strings.ToLower(strings.TrimSpace(dialect)), profileID, friendID)
	_, err := db.ExecContext(ctx, query, args...)
	return err
}

func GetProfileImageIDByProfileID(
	ctx context.Context,
	db QueryRower,
	dialect string,
	profileID int,
) (sql.NullInt64, error) {
	if db == nil {
		return sql.NullInt64{}, errors.New("query runner is nil")
	}
	query, args := getProfileImageIDByProfileIDQuery(strings.ToLower(strings.TrimSpace(dialect)), profileID)
	var imageID sql.NullInt64
	if err := db.QueryRowContext(ctx, query, args...).Scan(&imageID); err != nil {
		return sql.NullInt64{}, err
	}
	return imageID, nil
}

func InsertImage(
	ctx context.Context,
	db QueryExecRower,
	dialect string,
	imageType string,
	createdAt time.Time,
	updatedAt time.Time,
) (int, error) {
	if db == nil {
		return 0, errors.New("query runner is nil")
	}
	d := strings.ToLower(strings.TrimSpace(dialect))
	query, args := insertImageQuery(d, createdAt, updatedAt, imageType)
	if dialectSuffix(d) == "postgres" {
		var imageID int
		if err := db.QueryRowContext(ctx, query, args...).Scan(&imageID); err != nil {
			return 0, err
		}
		return imageID, nil
	}
	res, err := db.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	return int(id), nil
}

func InsertImageSize(
	ctx context.Context,
	db Execer,
	dialect string,
	size string,
	width int,
	height int,
	imageID int,
	fileID int,
	createdAt time.Time,
	updatedAt time.Time,
) error {
	if db == nil {
		return errors.New("exec runner is nil")
	}
	query, args := insertImageSizeQuery(strings.ToLower(strings.TrimSpace(dialect)), createdAt, updatedAt, size, width, height, imageID, fileID)
	_, err := db.ExecContext(ctx, query, args...)
	return err
}

func SetProfileImageID(
	ctx context.Context,
	db Execer,
	dialect string,
	profileID int,
	imageID int,
) error {
	if db == nil {
		return errors.New("exec runner is nil")
	}
	query, args := setProfileImageIDQuery(strings.ToLower(strings.TrimSpace(dialect)), profileID, imageID)
	_, err := db.ExecContext(ctx, query, args...)
	return err
}

func AttachGalleryImageToProfile(
	ctx context.Context,
	db Execer,
	dialect string,
	imageID int,
	profileID int,
) error {
	if db == nil {
		return errors.New("exec runner is nil")
	}
	query, args := attachGalleryImageToProfileQuery(strings.ToLower(strings.TrimSpace(dialect)), imageID, profileID)
	_, err := db.ExecContext(ctx, query, args...)
	return err
}

func ImageBelongsToProfileGallery(
	ctx context.Context,
	db QueryRower,
	dialect string,
	imageID int,
	profileID int,
) (bool, error) {
	if db == nil {
		return false, errors.New("query runner is nil")
	}
	query, args := imageBelongsToProfileGalleryQuery(strings.ToLower(strings.TrimSpace(dialect)), imageID, profileID)
	var exists bool
	if err := db.QueryRowContext(ctx, query, args...).Scan(&exists); err != nil {
		return false, err
	}
	return exists, nil
}

func GetImageStorageObjectsByImageID(
	ctx context.Context,
	db Queryer,
	dialect string,
	imageID int,
) ([]ProfilePhotoSizeRecord, error) {
	if db == nil {
		return nil, errors.New("query runner is nil")
	}
	query, args := getImageStorageObjectsByImageIDQuery(strings.ToLower(strings.TrimSpace(dialect)), imageID)
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []ProfilePhotoSizeRecord
	for rows.Next() {
		var rec ProfilePhotoSizeRecord
		if err := rows.Scan(&rec.ImageID, &rec.Size, &rec.Width, &rec.Height, &rec.ObjectKey); err != nil {
			return nil, err
		}
		out = append(out, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func ClearProfileImageByImageID(
	ctx context.Context,
	db Execer,
	dialect string,
	imageID int,
) error {
	if db == nil {
		return errors.New("exec runner is nil")
	}
	query, args := clearProfileImageByImageIDQuery(strings.ToLower(strings.TrimSpace(dialect)), imageID)
	_, err := db.ExecContext(ctx, query, args...)
	return err
}

func DeleteImageByID(
	ctx context.Context,
	db Execer,
	dialect string,
	imageID int,
) error {
	if db == nil {
		return errors.New("exec runner is nil")
	}
	query, args := deleteImageByIDQuery(strings.ToLower(strings.TrimSpace(dialect)), imageID)
	_, err := db.ExecContext(ctx, query, args...)
	return err
}

func GetSubscriptionForBenefactorByProfileID(
	ctx context.Context,
	db QueryRower,
	dialect string,
	profileID int,
) (*SubscriptionBenefactorRecord, error) {
	if db == nil {
		return nil, errors.New("query runner is nil")
	}
	query, args := getSubscriptionForBenefactorByProfileIDQuery(strings.ToLower(strings.TrimSpace(dialect)), profileID)
	var out SubscriptionBenefactorRecord
	if err := db.QueryRowContext(ctx, query, args...).Scan(&out.SubscriptionID, &out.PayingProfile); err != nil {
		return nil, err
	}
	return &out, nil
}

func CountSubscriptionBenefactorsBySubscriptionID(
	ctx context.Context,
	db QueryRower,
	dialect string,
	subscriptionID int,
) (int, error) {
	if db == nil {
		return 0, errors.New("query runner is nil")
	}
	query, args := countSubscriptionBenefactorsBySubscriptionIDQuery(strings.ToLower(strings.TrimSpace(dialect)), subscriptionID)
	var count int
	if err := db.QueryRowContext(ctx, query, args...).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func RemoveSubscriptionBenefactorBySubscriptionAndProfile(
	ctx context.Context,
	db Execer,
	dialect string,
	subscriptionID int,
	profileID int,
) error {
	if db == nil {
		return errors.New("exec runner is nil")
	}
	query, args := removeSubscriptionBenefactorBySubscriptionAndProfileQuery(strings.ToLower(strings.TrimSpace(dialect)), subscriptionID, profileID)
	_, err := db.ExecContext(ctx, query, args...)
	return err
}

func DeleteSubscriptionByID(
	ctx context.Context,
	db Execer,
	dialect string,
	subscriptionID int,
) error {
	if db == nil {
		return errors.New("exec runner is nil")
	}
	query, args := deleteSubscriptionByIDQuery(strings.ToLower(strings.TrimSpace(dialect)), subscriptionID)
	_, err := db.ExecContext(ctx, query, args...)
	return err
}

func DeleteUserByProfileID(
	ctx context.Context,
	db Execer,
	dialect string,
	profileID int,
) error {
	if db == nil {
		return errors.New("exec runner is nil")
	}
	query, args := deleteUserByProfileIDQuery(strings.ToLower(strings.TrimSpace(dialect)), profileID)
	_, err := db.ExecContext(ctx, query, args...)
	return err
}

func getProfileFullyOnboardedByUserIDQuery(dialect string, userID int) (string, []any) {
	key := "get_profile_fully_onboarded_by_user_id_" + dialectSuffix(dialect)
	query, err := queries.Get(key)
	if err != nil {
		panic(err)
	}
	return query, []any{userID}
}

func getProfileThumbnailObjectKeyByUserIDQuery(dialect string, userID int) (string, []any) {
	key := "get_profile_thumbnail_object_key_by_user_id_" + dialectSuffix(dialect)
	query, err := queries.Get(key)
	if err != nil {
		panic(err)
	}
	return query, []any{userID}
}

func getProfileSettingsByIDQuery(dialect string, profileID int) (string, []any) {
	key := "get_profile_settings_by_id_" + dialectSuffix(dialect)
	query, err := queries.Get(key)
	if err != nil {
		panic(err)
	}
	return query, []any{profileID}
}

func updateProfileBioByIDQuery(dialect string, profileID int, bio string) (string, []any) {
	key := "update_profile_bio_by_id_" + dialectSuffix(dialect)
	query, err := queries.Get(key)
	if err != nil {
		panic(err)
	}
	if dialectSuffix(dialect) == "postgres" {
		return query, []any{profileID, bio}
	}
	return query, []any{bio, profileID}
}

func updateProfilePhoneByIDQuery(dialect string, profileID int, countryCode, phoneE164 string) (string, []any) {
	key := "update_profile_phone_by_id_" + dialectSuffix(dialect)
	query, err := queries.Get(key)
	if err != nil {
		panic(err)
	}
	if dialectSuffix(dialect) == "postgres" {
		return query, []any{profileID, countryCode, phoneE164}
	}
	return query, []any{countryCode, phoneE164, profileID}
}

func insertProfileQuery(
	dialect string,
	createdAt time.Time,
	updatedAt time.Time,
	bio string,
	birthdate time.Time,
	age int,
	countryCode *string,
	phoneE164 *string,
	userID int,
) (string, []any) {
	key := "insert_profile_" + dialectSuffix(dialect)
	query, err := queries.Get(key)
	if err != nil {
		panic(err)
	}
	return query, []any{createdAt, updatedAt, bio, birthdate, age, countryCode, phoneE164, userID}
}

func updateProfileDetailsByIDQuery(
	dialect string,
	profileID int,
	bio string,
	birthdate time.Time,
	age int,
	countryCode *string,
	phoneE164 *string,
) (string, []any) {
	key := "update_profile_details_by_id_" + dialectSuffix(dialect)
	query, err := queries.Get(key)
	if err != nil {
		panic(err)
	}
	if dialectSuffix(dialect) == "postgres" {
		return query, []any{profileID, bio, birthdate, age, countryCode, phoneE164}
	}
	return query, []any{bio, birthdate, age, countryCode, phoneE164, profileID}
}

func markProfileFullyOnboardedByIDQuery(dialect string, profileID int) (string, []any) {
	key := "mark_profile_fully_onboarded_by_id_" + dialectSuffix(dialect)
	query, err := queries.Get(key)
	if err != nil {
		panic(err)
	}
	return query, []any{profileID}
}

func markProfilePhoneVerifiedByIDQuery(dialect string, profileID int) (string, []any) {
	key := "mark_profile_phone_verified_by_id_" + dialectSuffix(dialect)
	query, err := queries.Get(key)
	if err != nil {
		panic(err)
	}
	return query, []any{profileID}
}

func getFriendsByProfileIDQuery(dialect string, profileID int) (string, []any) {
	key := "get_friends_by_profile_id_" + dialectSuffix(dialect)
	query, err := queries.Get(key)
	if err != nil {
		panic(err)
	}
	return query, []any{profileID}
}

func areProfilesFriendsQuery(dialect string, profileID1 int, profileID2 int) (string, []any) {
	key := "are_profiles_friends_" + dialectSuffix(dialect)
	query, err := queries.Get(key)
	if err != nil {
		panic(err)
	}
	return query, []any{profileID1, profileID2}
}

func getProfilePhotosByProfileIDQuery(dialect string, profileID int) (string, []any) {
	key := "get_profile_photos_by_profile_id_" + dialectSuffix(dialect)
	query, err := queries.Get(key)
	if err != nil {
		panic(err)
	}
	return query, []any{profileID}
}

func getProfileCoreByIDQuery(dialect string, profileID int) (string, []any) {
	key := "get_profile_core_by_id_" + dialectSuffix(dialect)
	query, err := queries.Get(key)
	if err != nil {
		panic(err)
	}
	return query, []any{profileID}
}

func countUnseenNotificationsByProfileQuery(dialect string, profileID int) (string, []any) {
	key := "count_unseen_notifications_by_profile_id_" + dialectSuffix(dialect)
	query, err := queries.Get(key)
	if err != nil {
		panic(err)
	}
	return query, []any{profileID, false}
}

func getProfileImageByProfileIDQuery(dialect string, profileID int) (string, []any) {
	key := "get_profile_image_by_profile_id_" + dialectSuffix(dialect)
	query, err := queries.Get(key)
	if err != nil {
		panic(err)
	}
	return query, []any{profileID}
}

func linkProfilesAsFriendsQuery(dialect string, profileID int, friendID int) (string, []any) {
	key := "link_profiles_as_friends_" + dialectSuffix(dialect)
	query, err := queries.Get(key)
	if err != nil {
		panic(err)
	}
	return query, []any{profileID, friendID}
}

func unlinkProfilesAsFriendsQuery(dialect string, profileID int, friendID int) (string, []any) {
	key := "unlink_profiles_as_friends_" + dialectSuffix(dialect)
	query, err := queries.Get(key)
	if err != nil {
		panic(err)
	}
	return query, []any{profileID, friendID}
}

func getProfileImageIDByProfileIDQuery(dialect string, profileID int) (string, []any) {
	key := "get_profile_image_id_by_profile_id_" + dialectSuffix(dialect)
	query, err := queries.Get(key)
	if err != nil {
		panic(err)
	}
	return query, []any{profileID}
}

func insertImageQuery(dialect string, createdAt, updatedAt time.Time, imageType string) (string, []any) {
	key := "insert_image_" + dialectSuffix(dialect)
	query, err := queries.Get(key)
	if err != nil {
		panic(err)
	}
	return query, []any{createdAt, updatedAt, imageType}
}

func insertImageSizeQuery(
	dialect string,
	createdAt time.Time,
	updatedAt time.Time,
	size string,
	width int,
	height int,
	imageID int,
	fileID int,
) (string, []any) {
	key := "insert_image_size_" + dialectSuffix(dialect)
	query, err := queries.Get(key)
	if err != nil {
		panic(err)
	}
	return query, []any{createdAt, updatedAt, size, width, height, imageID, fileID}
}

func setProfileImageIDQuery(dialect string, profileID int, imageID int) (string, []any) {
	key := "set_profile_image_id_" + dialectSuffix(dialect)
	query, err := queries.Get(key)
	if err != nil {
		panic(err)
	}
	if dialectSuffix(dialect) == "postgres" {
		return query, []any{profileID, imageID}
	}
	return query, []any{imageID, profileID}
}

func attachGalleryImageToProfileQuery(dialect string, imageID int, profileID int) (string, []any) {
	key := "attach_gallery_image_to_profile_" + dialectSuffix(dialect)
	query, err := queries.Get(key)
	if err != nil {
		panic(err)
	}
	if dialectSuffix(dialect) == "postgres" {
		return query, []any{imageID, profileID}
	}
	return query, []any{profileID, imageID}
}

func imageBelongsToProfileGalleryQuery(dialect string, imageID int, profileID int) (string, []any) {
	key := "image_belongs_to_profile_gallery_" + dialectSuffix(dialect)
	query, err := queries.Get(key)
	if err != nil {
		panic(err)
	}
	return query, []any{imageID, profileID}
}

func getImageStorageObjectsByImageIDQuery(dialect string, imageID int) (string, []any) {
	key := "get_image_storage_objects_by_image_id_" + dialectSuffix(dialect)
	query, err := queries.Get(key)
	if err != nil {
		panic(err)
	}
	return query, []any{imageID}
}

func clearProfileImageByImageIDQuery(dialect string, imageID int) (string, []any) {
	key := "clear_profile_image_by_image_id_" + dialectSuffix(dialect)
	query, err := queries.Get(key)
	if err != nil {
		panic(err)
	}
	return query, []any{imageID}
}

func deleteImageByIDQuery(dialect string, imageID int) (string, []any) {
	key := "delete_image_by_id_" + dialectSuffix(dialect)
	query, err := queries.Get(key)
	if err != nil {
		panic(err)
	}
	return query, []any{imageID}
}

func getSubscriptionForBenefactorByProfileIDQuery(dialect string, profileID int) (string, []any) {
	key := "get_subscription_for_benefactor_by_profile_id_" + dialectSuffix(dialect)
	query, err := queries.Get(key)
	if err != nil {
		panic(err)
	}
	return query, []any{profileID}
}

func countSubscriptionBenefactorsBySubscriptionIDQuery(dialect string, subscriptionID int) (string, []any) {
	key := "count_subscription_benefactors_by_subscription_id_" + dialectSuffix(dialect)
	query, err := queries.Get(key)
	if err != nil {
		panic(err)
	}
	return query, []any{subscriptionID}
}

func removeSubscriptionBenefactorBySubscriptionAndProfileQuery(dialect string, subscriptionID int, profileID int) (string, []any) {
	key := "remove_subscription_benefactor_by_subscription_and_profile_" + dialectSuffix(dialect)
	query, err := queries.Get(key)
	if err != nil {
		panic(err)
	}
	return query, []any{subscriptionID, profileID}
}

func deleteSubscriptionByIDQuery(dialect string, subscriptionID int) (string, []any) {
	key := "delete_subscription_by_id_" + dialectSuffix(dialect)
	query, err := queries.Get(key)
	if err != nil {
		panic(err)
	}
	return query, []any{subscriptionID}
}

func deleteUserByProfileIDQuery(dialect string, profileID int) (string, []any) {
	key := "delete_user_by_profile_id_" + dialectSuffix(dialect)
	query, err := queries.Get(key)
	if err != nil {
		panic(err)
	}
	return query, []any{profileID}
}
