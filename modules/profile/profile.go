package profiles

import (
	"context"
	"database/sql"
	"errors"
	"time"

	paidsubscriptions "github.com/leomorpho/goship-modules/paidsubscriptions"
	dbgen "github.com/leomorpho/goship/db/gen"
	"github.com/leomorpho/goship/framework/domain"
	storagerepo "github.com/leomorpho/goship/framework/repos/storage"
)

var ErrContactRequestAlreadyExists = errors.New("contact request already exists")
var ErrProfileDBNotConfigured = errors.New("profile database dependency not configured")

type ProfileService struct {
	db               *sql.DB
	dbDialect        string
	storageRepo      storagerepo.StorageClientInterface
	subscriptionRepo *paidsubscriptions.Service
	notificationRepo NotificationCountStore
}

func NewProfileServiceWithDBDeps(
	db *sql.DB,
	dbDialect string,
	storageRepo storagerepo.StorageClientInterface,
	subscriptionRepo *paidsubscriptions.Service,
	notificationRepo NotificationCountStore,
) *ProfileService {
	return &ProfileService{
		db:               db,
		dbDialect:        dbDialect,
		storageRepo:      storageRepo,
		subscriptionRepo: subscriptionRepo,
		notificationRepo: notificationRepo,
	}
}

// TODO: I originally did that to not have to dynamically derive the ages
// each time I calculated matches for the dating portion of the app, but
// this should be recalculated in a task, where every day we look at who's
// birthday is today, and update their age.
func CalculateAge(birthdate time.Time) int {
	now := time.Now()
	age := now.Year() - birthdate.Year()
	// If birthdate is not yet reached in the current year, subtract one year from age.
	if now.Month() < birthdate.Month() ||
		(now.Month() == birthdate.Month() && now.Day() < birthdate.Day()) {
		age--
	}
	return age
}

func (p *ProfileService) CreateProfile(
	ctx context.Context,
	userID int,
	bio string,
	birthdate time.Time,
	countryCode, e164PhoneNumber *string,
) (int, error) {
	if p.db == nil {
		return 0, ErrProfileDBNotConfigured
	}
	if userID <= 0 {
		return 0, errors.New("invalid user id")
	}
	now := time.Now().UTC()
	return dbgen.InsertProfile(
		ctx,
		p.db,
		p.dbDialect,
		userID,
		bio,
		birthdate,
		CalculateAge(birthdate),
		countryCode,
		e164PhoneNumber,
		now,
		now,
	)
}

func (p *ProfileService) UpdateProfile(
	ctx context.Context, profileID int, fullyOnboarded bool, bio string,
	birthdate time.Time, countryCode, e164PhoneNumber *string,
) error {
	if p.db == nil {
		return ErrProfileDBNotConfigured
	}
	if profileID <= 0 {
		return errors.New("invalid profile id")
	}
	if fullyOnboarded {
		return dbgen.UpdateProfileBioByID(ctx, p.db, p.dbDialect, profileID, bio)
	}
	return dbgen.UpdateProfileDetailsByID(
		ctx,
		p.db,
		p.dbDialect,
		profileID,
		bio,
		birthdate,
		CalculateAge(birthdate),
		countryCode,
		e164PhoneNumber,
	)
}

// GetFriends retrieves all friends
func (s *ProfileService) GetFriends(
	ctx context.Context, currentProfileID int,
) ([]domain.Profile, error) {
	if s.db == nil {
		return nil, ErrProfileDBNotConfigured
	}
	rows, err := dbgen.GetFriendsByProfileID(ctx, s.db, s.dbDialect, currentProfileID)
	if err != nil {
		return nil, err
	}
	profiles := make([]domain.Profile, 0, len(rows))
	for _, r := range rows {
		pf := domain.Profile{
			ID:              r.ProfileID,
			Name:            r.Name,
			Age:             int(r.Age.Int64),
			Bio:             r.Bio.String,
			PhoneNumberE164: r.PhoneNumberE164.String,
			CountryCode:     r.CountryCode.String,
		}
		thumbURL := s.GetProfilePhotoThumbnailURL(r.UserID)
		if thumbURL != "" {
			pf.ProfileImage = &domain.Photo{ThumbnailURL: thumbURL}
		}
		profiles = append(profiles, pf)
	}
	return profiles, nil
}

// AreProfilesFriends checks if two profiles are friends
func (p *ProfileService) AreProfilesFriends(ctx context.Context, profileID1, profileID2 int) (bool, error) {
	if p.db == nil {
		return false, ErrProfileDBNotConfigured
	}
	return dbgen.AreProfilesFriends(ctx, p.db, p.dbDialect, profileID1, profileID2)
}

// LinkProfilesAsFriends links two profiles as friends.
func (p *ProfileService) LinkProfilesAsFriends(
	ctx context.Context, inviterProfileID int, inviteeProfileID int,
) error {
	if p.db == nil {
		return ErrProfileDBNotConfigured
	}
	return dbgen.LinkProfilesAsFriends(ctx, p.db, p.dbDialect, inviterProfileID, inviteeProfileID)
}

// UnlinkProfilesAsFriends removes the friendship link between two profiles.
func (p *ProfileService) UnlinkProfilesAsFriends(
	ctx context.Context, profileID, friendID int,
) error {
	if p.db == nil {
		return ErrProfileDBNotConfigured
	}
	return dbgen.UnlinkProfilesAsFriends(ctx, p.db, p.dbDialect, profileID, friendID)
}

func (p *ProfileService) GetProfileByID(
	ctx context.Context, profileID int, selfProfileID *int,
) (*domain.Profile, error) {
	if p.db == nil {
		return nil, ErrProfileDBNotConfigured
	}

	core, err := dbgen.GetProfileCoreByID(ctx, p.db, p.dbDialect, profileID)
	if err != nil {
		return nil, err
	}

	photosRows, err := dbgen.GetProfilePhotosByProfileID(ctx, p.db, p.dbDialect, profileID)
	if err != nil {
		return nil, err
	}
	photos, err := p.storageRepo.GetImageObjectsFromFiles(mapProfilePhotoSizeRecordsToStorageImages(photosRows))
	if err != nil {
		if _, ok := err.(*storagerepo.NoImagesInFiles); !ok {
			return nil, err
		}
	}

	var currProfilePhoto *domain.Photo
	imageRows, err := dbgen.GetProfileImageByProfileID(ctx, p.db, p.dbDialect, profileID)
	if err != nil {
		return nil, err
	}
	imageFiles := mapProfilePhotoSizeRecordsToStorageImages(imageRows)
	if len(imageFiles) > 0 {
		currProfilePhoto, err = p.storageRepo.GetImageObjectFromFile(imageFiles[0])
		if err != nil {
			if _, ok := err.(*storagerepo.NoImagesInFiles); !ok {
				return nil, err
			}
		}
	}

	profile := &domain.Profile{
		ID:              core.ProfileID,
		Name:            core.Name,
		Age:             int(core.Age.Int64),
		Bio:             core.Bio.String,
		PhoneNumberE164: core.PhoneNumberE164.String,
		CountryCode:     core.CountryCode.String,
		ProfileImage:    currProfilePhoto,
		Photos:          photos,
	}

	return profile, nil
}

func (s *ProfileService) GetCountOfUnseenNotifications(ctx context.Context, profileID int) (int, error) {
	if s == nil || s.notificationRepo == nil {
		return 0, ErrNotificationCountStoreNotConfigured
	}
	return s.notificationRepo.CountUnseenNotifications(ctx, profileID)
}

func (p *ProfileService) IsProfileFullyOnboardedByUserID(ctx context.Context, userID int) (bool, error) {
	if p.db == nil {
		return false, ErrProfileDBNotConfigured
	}
	return dbgen.GetProfileFullyOnboardedByUserID(ctx, p.db, p.dbDialect, userID)
}

func (p *ProfileService) DeleteUserData(ctx context.Context, profileID int) error {
	if p.db == nil {
		return ErrProfileDBNotConfigured
	}
	if err := p.deleteProfileStorageFiles(ctx, profileID); err != nil {
		return err
	}

	sub, err := dbgen.GetSubscriptionForBenefactorByProfileID(ctx, p.db, p.dbDialect, profileID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	if err == nil && sub != nil {
		benefactorsCount, countErr := dbgen.CountSubscriptionBenefactorsBySubscriptionID(
			ctx, p.db, p.dbDialect, sub.SubscriptionID,
		)
		if countErr != nil {
			return countErr
		}

		deleteSub := sub.PayingProfile == profileID || benefactorsCount <= 1
		if !deleteSub && benefactorsCount > 1 {
			if err := dbgen.RemoveSubscriptionBenefactorBySubscriptionAndProfile(
				ctx, p.db, p.dbDialect, sub.SubscriptionID, profileID,
			); err != nil {
				return err
			}
		} else if deleteSub {
			if err := dbgen.DeleteSubscriptionByID(ctx, p.db, p.dbDialect, sub.SubscriptionID); err != nil {
				return err
			}
		}
	}

	return dbgen.DeleteUserByProfileID(ctx, p.db, p.dbDialect, profileID)
}
