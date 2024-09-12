package seeder

import (
	"context"
	"errors"
	"math/rand"
	"os"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/mikestefanello/pagoda/config"
	"github.com/mikestefanello/pagoda/ent"
	"github.com/mikestefanello/pagoda/pkg/domain"
	"github.com/mikestefanello/pagoda/pkg/repos/emailsmanager"
	"github.com/mikestefanello/pagoda/pkg/repos/notifierrepo"
	"github.com/mikestefanello/pagoda/pkg/repos/profilerepo"
	storagerepo "github.com/mikestefanello/pagoda/pkg/repos/storage"
	"github.com/mikestefanello/pagoda/pkg/repos/subscriptions"
	"github.com/mikestefanello/pagoda/pkg/services"
	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/bcrypt"
)

var ErrObjectExists = errors.New("object already exists")

// RunIdempotentSeeder seeds all regular objects that the app needs to run smoothly
func RunIdempotentSeeder(cfg *config.Config, client *ent.Client) error {
	ctx := context.Background()
	emailRepo := emailsmanager.NewEmailSubscriptionRepo(client)

	err := emailRepo.CreateNewSubscriptionList(ctx, domain.EmailNewsletter)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create email list type")
	}

	err = emailRepo.CreateNewSubscriptionList(ctx, domain.EmailInitialAnnoucement)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create email list type")
	}
	// SeedUsers(cfg, client, true)

	runDataFixes(cfg, ctx, client)

	seedProdDemoData()

	return nil
}

func runDataFixes(cfg *config.Config, ctx context.Context, client *ent.Client) {
	// Place data fixes to run on startup here, make sure they're idempotent.

}

func seedProdDemoData() {
	// Put demo data required for prod here.
}

// SeedUsers add the initial users to the database.
func SeedUsers(cfg *config.Config, client *ent.Client, useS3 bool) error {
	// NOTE: DO NOT RUN THIS SEEDER AS AN IDEMPOTENT SEEDER AS IT IS NOT IDEMPOTENT
	// RUN WITH: `make seed` in CLI
	log.Printf("Start seeding database...")
	ctx := context.Background()
	// fake := faker.New()

	// Wrapper function to create a user and ignore if exists
	createUser := func(name, email, password string) *ent.User {

		user, err := client.User.
			Create().
			SetName(name).
			SetEmail(email).
			SetPassword(password).
			SetVerified(true).
			Save(ctx)

		if err != nil {
			log.Printf("User with email %s already exists. Skipping.", email)
			return nil
		}
		return user
	}

	var storageClient *storagerepo.StorageClient
	if useS3 {
		storageClient = storagerepo.NewStorageClient(cfg, client)
	}
	profileRepo := profilerepo.NewProfileRepo(client, storageClient, nil)
	subscriptionsRepo := subscriptions.NewSubscriptionsRepo(client, 3, 3)
	notificationSendPermissionRepo := notifierrepo.NewNotificationSendPermissionRepo(client)

	createProfile := func(
		user *ent.User,
		bio string,
		age time.Time,
		countryCode, e164PhoneNumber string,
	) *ent.Profile {
		profile, err := profileRepo.CreateProfile(
			ctx, user, bio, age,
			&countryCode, &e164PhoneNumber)

		if err != nil {
			log.Printf("Error creating profile for user %s: %v\n", user.Name, err)
			return nil
		}

		for _, perm := range domain.NotificationPermissions.Members() {
			err := notificationSendPermissionRepo.CreatePermission(
				ctx, profile.ID, perm, &domain.NotificationPlatformEmail)
			if err != nil {
				log.Fatal().Err(err).Msg("failed to create notification permission")
			}
		}
		return profile
	}

	hashPassword := func(password string) string {
		hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			panic("failed to hash password")
		}
		return string(hash)
	}

	// TODO: for prod, use passwords set in env var
	// Create users
	alice := createUser("Alice Bonjovi", "alice@test.com", hashPassword("password"))
	spew.Dump(alice)
	bob := createUser("Bob Lupin", "bob@test.com", hashPassword("password"))
	sandrine := createUser("Sandrine Bonnaire", "sandrine@test.com", hashPassword("password"))
	luca := createUser("Luca George", "luca@test.com", hashPassword("password"))
	lucy := createUser("Lucy Liu", "lucy@test.com", hashPassword("password"))
	elliot := createUser("Elliot Ness", "elliot@test.com", hashPassword("password"))

	aliceProfile := createProfile(alice, "", time.Now().AddDate(-23, 0, 0), "US", "+15551234567")
	subscriptionsRepo.CreateSubscription(ctx, nil, aliceProfile.ID)

	bobProfile := createProfile(bob, "", time.Now().AddDate(-24, 0, 0), "GB", "+442012345678")
	subscriptionsRepo.CreateSubscription(ctx, nil, bobProfile.ID)

	sandrineProfile := createProfile(sandrine, "", time.Now().AddDate(-25, 0, 0), "CA", "+14165551234")
	subscriptionsRepo.UpdateToPaidPro(ctx, sandrineProfile.ID)

	lucaProfile := createProfile(luca, "", time.Now().AddDate(-26, 0, 0), "AU", "+61212345678")

	lucyProfile := createProfile(lucy, "", time.Now().AddDate(-25, 0, 0), "", "")

	elliotProfile := createProfile(elliot, "", time.Now().AddDate(-26, 0, 0), "FR", "+33123456789")

	// if useS3 {
	if false {

		//////////////////////////////////////////
		// Photos
		//////////////////////////////////////////
		log.Printf("Uploading some photos...")

		photo1, err := os.Open("testdata/photos/1.jpg")
		if err != nil {
			log.Fatal().Err(err).Msg("failed to open file")
		}
		defer photo1.Close()

		photo2, err := os.Open("testdata/photos/2.jpg")
		if err != nil {
			log.Fatal().Err(err).Msg("failed to open file")
		}
		defer photo2.Close()

		// Upload photos for alice, bob and Sandrine
		err = profileRepo.UploadPhoto(ctx, aliceProfile.ID, photo1, "image.jpg")
		if err != nil {
			log.Fatal().Err(err).Msg("failed to open file")
		}
		// TODO: not sure why, but when setting  profile pic for bob and alice, I get:
		// 2023/11/25 08:53:43 failed to upload photo: ent: constraint failed: ERROR: update or delete on table "file_storages" violates foreign key constraint "profiles_file_storages_profile_image" on table "profiles" (SQLSTATE 23503)
		err = profileRepo.UploadPhoto(ctx, aliceProfile.ID, photo2, "profile.jpg")
		if err != nil {
			log.Fatal().Err(err).Msg("failed to upload image")
		}

		err = profileRepo.UploadPhoto(ctx, bobProfile.ID, photo1, "image.jpg")
		if err != nil {
			log.Fatal().Err(err).Msg("failed to upload image")
		}
		err = profileRepo.SetProfilePhoto(ctx, bobProfile.ID, photo2, "profile.jpg")
		if err != nil {
			log.Fatal().Err(err).Msg("failed to upload image")
		}

		err = profileRepo.UploadPhoto(ctx, sandrineProfile.ID, photo1, "image.jpg")
		if err != nil {
			log.Fatal().Err(err).Msg("failed to upload image")
		}
		err = profileRepo.SetProfilePhoto(ctx, sandrineProfile.ID, photo2, "profile.jpg")
		if err != nil {
			log.Fatal().Err(err).Msg("failed to upload image")
		}

		err = profileRepo.UploadPhoto(ctx, lucaProfile.ID, photo1, "image.jpg")
		if err != nil {
			log.Fatal().Err(err).Msg("failed to upload image")
		}
		err = profileRepo.SetProfilePhoto(ctx, lucaProfile.ID, photo2, "profile.jpg")
		if err != nil {
			log.Fatal().Err(err).Msg("failed to upload image")
		}
	}
	//////////////////////////////////////////
	// Friends
	//////////////////////////////////////////
	// Make Alice and Bob friends
	client.Profile.
		UpdateOneID(aliceProfile.ID).
		AddFriendIDs(bobProfile.ID).
		ExecX(ctx)

	// Make Alice and Sandrine friends
	client.Profile.
		UpdateOneID(aliceProfile.ID).
		AddFriendIDs(sandrineProfile.ID).
		ExecX(ctx)

	client.Profile.
		UpdateOneID(lucyProfile.ID).
		AddFriendIDs(elliotProfile.ID).
		ExecX(ctx)

	// Create a lot of notifs for one person (to test infinite load)
	c := services.NewContainer()
	for i := 0; i < 500; i++ {
		c.Notifier.PublishNotification(ctx, domain.Notification{
			Type:      domain.NotificationTypePlatformUpdate,
			Title:     "Yay! You're a user who'll get a ton of notifications for later visual testing!",
			Text:      "Now, go get some therapy",
			ProfileID: aliceProfile.ID,
		}, true, false)
	}

	return nil
}

func randomTimeInRange(startDate, endDate time.Time) (time.Time, error) {
	if endDate.Before(startDate) {
		return time.Time{}, errors.New("end date must be after start date")
	}

	duration := endDate.Sub(startDate)
	randomDuration := time.Duration(rand.Int63n(int64(duration)))
	randomTime := startDate.Add(randomDuration)
	return randomTime, nil
}

// RandomNumber generates a random number between 0 and max
func RandomNumber(max int) int {
	// Seed the random number generator to ensure different results each time
	// Generate a random number between 0 and 30000
	return rand.Intn(max) // rand.Intn generates a number in [0, n)
}
