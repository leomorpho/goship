package profiles

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	paidsubscriptions "github.com/leomorpho/goship-modules/paidsubscriptions"
	"github.com/leomorpho/goship/db/ent"
	"github.com/leomorpho/goship/db/ent/monthlysubscription"
	"github.com/leomorpho/goship/db/ent/notification"
	"github.com/leomorpho/goship/db/ent/profile"
	"github.com/leomorpho/goship/db/ent/user"
	"github.com/leomorpho/goship/framework/domain"
	storagerepo "github.com/leomorpho/goship/framework/repos/storage"
	"github.com/rs/zerolog/log"
)

var ErrContactRequestAlreadyExists = errors.New("contact request already exists")

type ProfileService struct {
	orm              *ent.Client
	storageRepo      storagerepo.StorageClientInterface
	subscriptionRepo *paidsubscriptions.Service
}

func NewProfileService(orm *ent.Client, storageRepo storagerepo.StorageClientInterface, subscriptionRepo *paidsubscriptions.Service) *ProfileService {
	return &ProfileService{
		orm:              orm,
		storageRepo:      storageRepo,
		subscriptionRepo: subscriptionRepo,
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
	user *ent.User,
	bio string,
	birthdate time.Time,
	countryCode, e164PhoneNumber *string,
) (*ent.Profile, error) {
	if user == nil {
		return nil, errors.New("user is nil")
	}
	// Create profile
	profile, err := p.orm.Profile.
		Create().
		SetUser(user).
		SetBirthdate(birthdate).
		SetAge(CalculateAge(birthdate)).
		SetNillableCountryCode(countryCode).
		SetNillablePhoneNumberE164(e164PhoneNumber).
		Save(ctx)
	if err != nil {
		return nil, err
	}
	return profile, nil
}

func (p *ProfileService) UpdateProfile(
	ctx context.Context, profile *ent.Profile, bio string,
	birthdate time.Time, countryCode, e164PhoneNumber *string,
) error {
	// Begin a transaction
	tx, err := p.orm.Tx(ctx)
	if err != nil {
		return err
	}

	query := tx.Profile.
		UpdateOneID(profile.ID).
		SetBio(string(bio))

	if !IsProfileFullyOnboarded(profile) {
		// There's no reason an onboarded profile would need to change their age.
		query.SetBirthdate(birthdate)
		query.SetAge(CalculateAge(birthdate))
	}

	if countryCode != nil {
		query.SetCountryCode(*countryCode)
	}

	if e164PhoneNumber != nil {
		query.SetPhoneNumberE164(*e164PhoneNumber)
	}

	err = query.Exec(ctx)

	if err != nil {
		// handle error and possibly rollback the transaction
		tx.Rollback()
		return err
	}

	// Commit the transaction
	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

// GetFriends retrieves all friends
func (s *ProfileService) GetFriends(
	ctx context.Context, currentProfileID int,
) ([]domain.Profile, error) {
	entProfiles, err := s.orm.Profile.Query().
		Where(profile.IDEQ(currentProfileID)).
		QueryFriends().
		WithProfileImage(func(pi *ent.ImageQuery) {
			pi.WithSizes(func(s *ent.ImageSizeQuery) {
				s.WithFile()
			})
		}).
		WithUser(func(uq *ent.UserQuery) {
			uq.Select(user.FieldName) // Only select the 'Name' field of the user.
		}).
		All(ctx)
	if err != nil {
		return nil, err
	}

	var profiles []domain.Profile
	for _, entProfile := range entProfiles {
		newProfile, err := s.EntProfileToDomainObject(entProfile)
		if err != nil {
			return nil, err
		}
		profiles = append(profiles, *newProfile)
	}
	return profiles, nil
}

// AreProfilesFriends checks if two profiles are friends
func (p *ProfileService) AreProfilesFriends(ctx context.Context, profileID1, profileID2 int) (bool, error) {
	// Check if there is a friendship link between the two profiles
	exists, err := p.orm.Profile.
		Query().
		Where(profile.IDEQ(profileID1)).
		QueryFriends().
		Where(profile.IDEQ(profileID2)).
		Exist(ctx)

	if err != nil {
		log.Error().
			Err(err).
			Int("ProfileID1", profileID1).
			Int("ProfileID2", profileID2).
			Msg("Error checking friendship status between profiles")
		return false, fmt.Errorf("error checking if profiles %d and %d are friends: %w", profileID1, profileID2, err)
	}

	return exists, nil
}

// LinkProfilesAsFriends links two profiles as friends.
func (p *ProfileService) LinkProfilesAsFriends(
	ctx context.Context, inviterProfileID int, inviteeProfileID int,
) error {
	return p.orm.Profile.
		UpdateOneID(inviterProfileID).
		AddFriendIDs(inviteeProfileID).
		Exec(ctx)
}

// UnlinkProfilesAsFriends removes the friendship link between two profiles.
func (p *ProfileService) UnlinkProfilesAsFriends(
	ctx context.Context, profileID, friendID int,
) error {
	// Remove friendID from profileID's friends list
	err := p.orm.Profile.
		UpdateOneID(profileID).
		RemoveFriendIDs(friendID).
		Exec(ctx)

	if err != nil {
		return err
	}

	return nil
}

func (p *ProfileService) GetProfileByID(
	ctx context.Context, profileID int, selfProfileID *int,
) (*domain.Profile, error) {
	query := p.orm.Profile.
		Query().
		Where(profile.IDEQ(profileID)).
		WithUser().
		WithProfileImage(func(pi *ent.ImageQuery) {
			pi.WithSizes(func(s *ent.ImageSizeQuery) {
				s.WithFile()
			})
		}).
		WithPhotos(func(pi *ent.ImageQuery) {
			pi.WithSizes(func(s *ent.ImageSizeQuery) {
				s.WithFile()
			})
		})

	entProfile, err := query.First(ctx)

	if err != nil {
		return nil, err
	}

	// Sort the photos manually in Go
	sort.Slice(entProfile.Edges.Photos, func(i, j int) bool {
		return entProfile.Edges.Photos[i].CreatedAt.After(entProfile.Edges.Photos[j].CreatedAt)
	})

	photos, err := p.storageRepo.GetImageObjectsFromFiles(entProfile.Edges.Photos)
	if err != nil {
		if customErr, ok := err.(*storagerepo.NoImagesInFiles); ok {
			log.Error().Str("Error", customErr.Message)
		} else {
			return nil, err
		}
	}

	var currProfilePhoto *domain.Photo
	if entProfile.Edges.ProfileImage != nil {
		currProfilePhoto, err = p.storageRepo.GetImageObjectFromFile(entProfile.Edges.ProfileImage)
		if err != nil {
			if customErr, ok := err.(*storagerepo.NoImagesInFiles); ok {
				log.Error().Str("Error", customErr.Message)
			} else {
				return nil, err
			}
		}
	}

	profile := &domain.Profile{
		ID:              entProfile.ID,
		Name:            entProfile.Edges.User.Name,
		Age:             entProfile.Age,
		Bio:             entProfile.Bio,
		PhoneNumberE164: entProfile.PhoneNumberE164,
		CountryCode:     entProfile.CountryCode,
		ProfileImage:    currProfilePhoto,
		Photos:          photos,
	}

	return profile, nil
}

func (s *ProfileService) GetCountOfUnseenNotifications(ctx context.Context, profileID int) (int, error) {
	return s.orm.Notification.Query().
		Where(
			notification.HasProfileWith(profile.IDEQ(profileID)),
			notification.ReadEQ(false),
		).Count(ctx)
}

func (p *ProfileService) EntProfileToDomainObject(e *ent.Profile) (*domain.Profile, error) {
	var name string
	if e.Edges.User != nil {
		name = e.Edges.User.Name
	}
	var profileImage *domain.Photo
	if e.Edges.ProfileImage != nil {
		image, err := p.storageRepo.GetImageObjectFromFile(e.Edges.ProfileImage)
		if err != nil {
			return nil, err
		}
		profileImage = image // Linting error if we don't do that
	}

	return &domain.Profile{
		ID:           e.ID,
		Name:         name,
		Age:          e.Age,
		Bio:          e.Bio,
		ProfileImage: profileImage,
	}, nil
}

func (p *ProfileService) DeleteUserData(ctx context.Context, profileID int) error {

	profileImage, err := p.orm.Profile.
		Query().
		Where(
			profile.IDEQ(profileID),
		).
		QueryProfileImage().
		WithSizes(func(isq *ent.ImageSizeQuery) {
			isq.WithFile()
		}).
		All(ctx)
	if err != nil {
		return err
	}

	photos, err := p.orm.Profile.
		Query().
		Where(
			profile.IDEQ(profileID),
		).
		QueryPhotos().
		WithSizes(func(isq *ent.ImageSizeQuery) {
			isq.WithFile()
		}).
		All(ctx)
	if err != nil {
		return err
	}

	if profileImage != nil {
		photos = append(photos, profileImage...)
	}
	for _, img := range photos {
		for _, imgSize := range img.Edges.Sizes {
			err = p.storageRepo.DeleteFile(storagerepo.BucketMainApp, imgSize.Edges.File.ObjectKey)
			if err != nil {
				log.Error().Err(err).
					Str("objectKey", imgSize.Edges.File.ObjectKey).
					Int("imageID", img.ID).
					Int("sizeID", imgSize.ID).
					Int("fileID", imgSize.Edges.File.ID).
					Msg("failed to delete file while deleting user data")
			}

		}
	}

	sub, err := p.orm.MonthlySubscription.Query().Where(
		monthlysubscription.HasBenefactorsWith(profile.IDEQ(profileID)),
	).
		WithPayer().
		WithBenefactors().
		Only(ctx)
	if err != nil && !ent.IsNotFound(err) {
		return err
	}
	if !ent.IsNotFound(err) {

		delete := false

		if sub.Edges.Payer.ID == profileID {
			delete = true
		}

		if len(sub.Edges.Benefactors) > 1 {
			// One person payed for both in the relationship
			err = p.orm.MonthlySubscription.
				UpdateOne(sub).
				RemoveBenefactorIDs(profileID).
				Exec(ctx)
			if err != nil {
				return err
			}
		} else {
			delete = true
		}

		if delete {
			err = p.orm.MonthlySubscription.
				DeleteOne(sub).
				Exec(ctx)
			if err != nil {
				return err
			}
		}
	}

	_, err = p.orm.User.Delete().Where(
		user.HasProfileWith(profile.IDEQ(profileID)),
	).Exec(ctx)

	return err
}

// TODO: move to ProfileService
// IsProfileFullyOnboarded determines whether a profile is fully onboarded onto the app or not.
func IsProfileFullyOnboarded(d *ent.Profile) bool {
	return d.FullyOnboarded
}
