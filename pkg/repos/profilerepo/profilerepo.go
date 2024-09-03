package profilerepo

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image/jpeg"
	"io"
	"path/filepath"
	"sort"
	"time"

	"github.com/disintegration/imaging"
	"github.com/google/uuid"
	"github.com/mikestefanello/pagoda/ent"
	"github.com/mikestefanello/pagoda/ent/filestorage"
	"github.com/mikestefanello/pagoda/ent/image"
	"github.com/mikestefanello/pagoda/ent/imagesize"
	"github.com/mikestefanello/pagoda/ent/monthlysubscription"
	"github.com/mikestefanello/pagoda/ent/notification"
	"github.com/mikestefanello/pagoda/ent/profile"
	"github.com/mikestefanello/pagoda/ent/user"
	"github.com/mikestefanello/pagoda/pkg/domain"
	storagerepo "github.com/mikestefanello/pagoda/pkg/repos/storage"
	"github.com/mikestefanello/pagoda/pkg/repos/subscriptions"
	"github.com/rs/zerolog/log"
)

var ErrContactRequestAlreadyExists = errors.New("contact request already exists")

type ProfileRepo struct {
	orm              *ent.Client
	storageRepo      storagerepo.StorageClientInterface
	subscriptionRepo *subscriptions.SubscriptionsRepo
}

func NewProfileRepo(orm *ent.Client, storageRepo storagerepo.StorageClientInterface, subscriptionRepo *subscriptions.SubscriptionsRepo) *ProfileRepo {
	return &ProfileRepo{
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

func (p *ProfileRepo) CreateProfile(
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

func (p *ProfileRepo) UpdateProfile(
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
func (s *ProfileRepo) GetFriends(
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
func (p *ProfileRepo) AreProfilesFriends(ctx context.Context, profileID1, profileID2 int) (bool, error) {
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
func (p *ProfileRepo) LinkProfilesAsFriends(
	ctx context.Context, inviterProfileID int, inviteeProfileID int,
) error {
	return p.orm.Profile.
		UpdateOneID(inviterProfileID).
		AddFriendIDs(inviteeProfileID).
		Exec(ctx)
}

// UnlinkProfilesAsFriends removes the friendship link between two profiles.
func (p *ProfileRepo) UnlinkProfilesAsFriends(
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

func (p *ProfileRepo) GetProfileByID(
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

func (s *ProfileRepo) GetCountOfUnseenNotifications(ctx context.Context, profileID int) (int, error) {
	return s.orm.Notification.Query().
		Where(
			notification.HasProfileWith(profile.IDEQ(profileID)),
			notification.ReadEQ(false),
		).Count(ctx)
}

func (p *ProfileRepo) GetPhotosByProfileByID(
	ctx context.Context, profileID int,
) ([]domain.Photo, error) {
	photoObjects, err := p.orm.Profile.
		Query().
		Where(profile.IDEQ(profileID)).
		QueryPhotos().
		Order(ent.Desc(filestorage.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	photos, err := p.storageRepo.GetImageObjectsFromFiles(photoObjects)
	if err != nil {
		if customErr, ok := err.(*storagerepo.NoImagesInFiles); ok {
			log.Error().Str("Error", customErr.Message)
		} else {
			return nil, err
		}
	}

	return photos, nil
}

func (p *ProfileRepo) GetProfilePhotoThumbnailURL(userID int) string {
	defaultProfilePic := "https://www.gravatar.com/avatar/?d=mp&s=200"
	ctx := context.Background()
	image, err := p.orm.Profile.
		Query().
		Where(profile.HasUserWith(user.IDEQ(userID))).
		QueryProfileImage().
		WithSizes(func(s *ent.ImageSizeQuery) {
			s.WithFile()
		}).
		Only(ctx)
	if ent.IsNotFound(err) {
		return defaultProfilePic
	}
	photo, err := p.storageRepo.GetImageObjectFromFile(image)
	if err != nil {
		if customErr, ok := err.(*storagerepo.NoImagesInFiles); ok {
			log.Error().Str("Error", customErr.Message)
		} else {
			return defaultProfilePic
		}
	}
	return photo.ThumbnailURL
}

func (p *ProfileRepo) SetProfilePhoto(
	ctx context.Context,
	profileID int,
	fileStream io.Reader,
	fileName string,
) error {
	if p.storageRepo == nil {
		return errors.New("storage orm not initialized")
	}

	profileImage, err := p.orm.Profile.
		Query().
		Where(
			profile.IDEQ(profileID),
		).
		QueryProfileImage().
		First(ctx)
	if err != nil && !ent.IsNotFound(err) {
		return err
	}
	if err == nil {
		err = p.DeletePhoto(ctx, profileImage.ID, nil)
		if err != nil {
			log.Err(err).Str("err", "failed to delete old photo when uploading new profile photo")
		}
	}

	imageID, err := p.UploadImageSizes(
		ctx, profileID, fileStream, domain.ImageCategoryProfilePhoto, fileName,
		[]domain.ImageSize{domain.ImageSizeThumbnail, domain.ImageSizePreview})
	if err != nil {
		return err
	}

	_, err = p.orm.Profile.UpdateOneID(profileID).SetProfileImageID(*imageID).Save(ctx)
	if err != nil {
		return err
	}
	return nil
}

func (p *ProfileRepo) UploadPhoto(
	ctx context.Context,
	profileID int,
	fileStream io.Reader,
	fileName string,
) error {
	if p.storageRepo == nil {
		return errors.New("storage orm not initialized")
	}

	imageID, err := p.UploadImageSizes(
		ctx, profileID, fileStream, domain.ImageCategoryProfileGallery, fileName,
		[]domain.ImageSize{domain.ImageSizeFull, domain.ImageSizePreview})
	if err != nil {
		return err
	}
	_, err = p.orm.Profile.UpdateOneID(profileID).AddPhotoIDs(*imageID).Save(ctx)
	if err != nil {
		return err
	}
	return nil
}

func (p *ProfileRepo) UploadImageSizes(
	ctx context.Context,
	profileID int,
	fileStream io.Reader,
	imageCategory domain.ImageCategory,
	fileName string,
	sizes []domain.ImageSize,
) (*int, error) {

	// Before decoding, try to seek to the beginning of the stream
	if seeker, ok := fileStream.(io.Seeker); ok {
		_, err := seeker.Seek(0, io.SeekStart)
		if err != nil {
			return nil, fmt.Errorf("failed to seek file stream: %w", err)
		}
	}

	// Decode the image from the stream
	img, err := imaging.Decode(fileStream)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	var imageSizeIDs []int

	for _, sizeName := range sizes {

		maxSize, ok := domain.ImageSizeEnumToSizeMap[sizeName]
		if !ok {
			return nil, err
		}
		// Resize image maintaining the aspect ratio
		resizedImage := imaging.Fit(img, maxSize, maxSize, imaging.Lanczos)

		// Get actual dimensions of the resized image
		actualWidth := resizedImage.Bounds().Dx()
		actualHeight := resizedImage.Bounds().Dy()

		// Create a buffer to write the resized image for upload
		buf := new(bytes.Buffer)
		if err := jpeg.Encode(buf, resizedImage, &jpeg.Options{Quality: 90}); err != nil {
			return nil, fmt.Errorf("failed to encode resized image for %s: %w", sizeName, err)
		}

		// Generate a unique object name
		ext := filepath.Ext(fileName)
		if ext == "" {
			return nil, fmt.Errorf("invalid file extension for %s", fileName)
		}
		objectName := fmt.Sprintf("profile_%d_%s_%s_%s%s",
			profileID, imageCategory.Value, uuid.New().String(), sizeName.Value, ext)

		log.Info().
			Str("imageCategory", imageCategory.Value).
			Str("sizeName", sizeName.Value).
			Int("profileID", profileID).
			Msg("Uploading image for profile")

		// Upload the resized image
		filestorageEntryID, err := p.storageRepo.UploadFile(storagerepo.BucketMainApp, objectName, bytes.NewReader(buf.Bytes()))
		if err != nil {
			log.Err(err).
				Str("imageCategory", imageCategory.Value).
				Str("sizeName", sizeName.Value).
				Int("profileID", profileID).
				Msg("failed to upload image for profile")
			return nil, fmt.Errorf("failed to upload %s image: %w", sizeName, err)
		}

		// Create an image size record
		imageSize, err := p.orm.ImageSize.
			Create().
			SetSize(imagesize.Size(sizeName.Value)).
			SetWidth(actualWidth).
			SetHeight(actualHeight).
			SetFileID(*filestorageEntryID).
			Save(ctx)
		if err != nil {
			log.Err(err).
				Str("imageCategory", imageCategory.Value).
				Str("sizeName", sizeName.Value).
				Int("profileID", profileID).
				Msg("failed to save image size objects for profile")
			return nil, fmt.Errorf("failed to save image size for %s: %w", sizeName, err)
		}

		imageSizeIDs = append(imageSizeIDs, imageSize.ID)
	}

	image, err := p.orm.Image.
		Create().
		SetType(image.Type(imageCategory.Value)).
		AddSizeIDs(imageSizeIDs...).
		Save(ctx)
	if err != nil {
		log.Err(err).
			Str("imageCategory", imageCategory.Value).
			Int("profileID", profileID).
			Msg("failed to create image object for profile")
		return nil, err
	}
	log.Info().
		Str("imageCategory", imageCategory.Value).
		Int("profileID", profileID).
		Msg("Successfully added Image for profile")

	return &image.ID, nil
}

func (p *ProfileRepo) DeletePhoto(
	ctx context.Context,
	imageID int,
	profileID *int,

) error {
	if p.storageRepo == nil {
		return errors.New("storage orm not initialized")
	}

	if profileID != nil {
		// Check that the photo belongs to the profile
		exists, err := p.orm.Profile.
			Query().
			Where(
				profile.HasPhotosWith(image.IDEQ(imageID)),
				profile.IDEQ(*profileID),
			).Exist(ctx)
		if err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf("no photo with id %d belonging to user", imageID)
		}
	}

	im, err := p.orm.Image.
		Query().
		Where(
			image.IDEQ(imageID),
		).
		WithSizes(func(s *ent.ImageSizeQuery) {
			s.WithFile()
		}).
		Only(ctx)

	if err != nil {
		return err
	}

	for _, size := range im.Edges.Sizes {
		if size.Edges.File != nil {
			// Delete the file in S3
			err = p.storageRepo.DeleteFile(storagerepo.BucketMainApp, size.Edges.File.ObjectKey)
			if err != nil {
				return err
			}
			// NOTE: filestorage object gets deleted by storage repo
		}
	}

	// TODO: verify cascade delete works for image size and files linked to image
	return p.orm.Image.DeleteOneID(imageID).Exec(ctx)
}

func (p *ProfileRepo) EntProfileToDomainObject(e *ent.Profile) (*domain.Profile, error) {
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

func (p *ProfileRepo) DeleteUserData(ctx context.Context, profileID int) error {

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

// TODO: move to ProfileRepo
// IsProfileFullyOnboarded determines whether a profile is fully onboarded onto the app or not.
func IsProfileFullyOnboarded(d *ent.Profile) bool {
	return d.FullyOnboarded
}
