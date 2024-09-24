package emailsmanager_test

import (
	"database/sql"
	"testing"

	"github.com/jackc/pgx/stdlib"
)

func init() {
	// Register "pgx" as "postgres" explicitly for database/sql
	sql.Register("postgres", stdlib.GetDefaultDriver())
}

// // We can generate a dummy email for udpates with TestSendUpdateEmail,
// // go to http://localhost:8026/ to see the emails in Mailpit.
// func TestSendUpdateEmailCommitted(t *testing.T) {
// 	client, ctx := tests.CreateTestContainerPostgresEntClient(t)
// 	defer client.Close()

// 	qaRepo, err := repo.NewQARepo(client, tests.NewMockStorageClient(), nil, &config.AppConfig{})
// 	assert.NoError(t, err)

// 	c := services.NewContainer()

// 	emailUpdateProcessor := tasks.NewEmailUpdateProcessor(c, client, qaRepo)

// 	err = emailUpdateProcessor.SendUpdateEmail(
// 		ctx,
// 		"jo@ben.com",
// 		"Jack",
// 		"Christa",
// 		[]types.QuestionInEmail{
// 			{
// 				Question: "What is the most difficult event you have recently dealt with?",
// 			},
// 			{
// 				Question: "How do you deal with the needs of your partner?",
// 			},
// 			{
// 				Question: "What is your biggest pet peeve?",
// 			},
// 		},
// 		[]types.QuestionInEmail{
// 			{
// 				Question: "If you were in a movie, which one would it be?",
// 			},
// 			{
// 				Question: "What is the most pressing concern in your relationship?",
// 			},
// 			{
// 				Question: "How do you deal with joy?",
// 			},
// 		},
// 		"", "",
// 	)
// 	assert.NoError(t, err)
// }

// func TestSendUpdateEmailDating(t *testing.T) {
// 	client, ctx := tests.CreateTestContainerPostgresEntClient(t)
// 	defer client.Close()
// 	assert.Equal(t, 2, 2)
// 	assert.Equal(t, 1, 1)

// 	qaRepo, err := repo.NewQARepo(client, tests.NewMockStorageClient(), nil, &config.AppConfig{})
// 	assert.NoError(t, err)

// 	c := services.NewContainer()
// 	emailUpdateProcessor := tasks.NewEmailUpdateProcessor(c, client, qaRepo)

// 	err = emailUpdateProcessor.SendUpdateEmail(
// 		ctx,
// 		"jo@ben.com",
// 		"Jack",
// 		"", // dating mode, no partner name
// 		[]types.QuestionInEmail{
// 			{
// 				Question: "What is the most difficult event you have recently dealt with?",
// 			},
// 			{
// 				Question: "How do you deal with the needs of your partner?",
// 			},
// 			{
// 				Question: "What is your biggest pet peeve?",
// 			},
// 		},
// 		[]types.QuestionInEmail{
// 			{
// 				Question: "If you were in a movie, which one would it be?",
// 			},
// 			{
// 				Question: "What is the most pressing concern in your relationship?",
// 			},
// 			{
// 				Question: "How do you deal with joy?",
// 			},
// 		},
// 		"", "",
// 	)
// 	assert.NoError(t, err)
// }

// NOTE: below test relies on seeding code so changing it could break this test.

func TestSendUpdateNotifsForSeededProfiles(t *testing.T) {
	// // NOTE: this test is really just a convenience test to trigger sending the emails
	// // and is not meant to be used in a CI
	// t.Skip("Skipping test in automated testing environment")

	// const NUM_PEOPLE_TO_GENERATE_FOR = 3
	// client, ctx := tests.CreateTestContainerPostgresEntClient(t)
	// defer client.Close()

	// config := config.Config{}

	// c := services.NewContainer()
	// routes.BuildRouter(c)

	// emailUpdateSender := emailsmanager.NewUpdateEmailSender(c.ORM, c)

	// err := seeder.SeedUsers(&config, client, false)
	// assert.NoError(t, err)

	// // NOTE: the below queries are copied from the tested function
	// notifFilter := func(nq *ent.NotificationQuery) {
	// 	nq.Where(
	// 		notification.ReadEQ(false),
	// 		notification.TypeIn(
	// 			notification.Type(domain.NotificationTypeCommittedRelationshipRequest.Value),
	// 			notification.Type(domain.NotificationTypeConnectionReactedToAnswer.Value),
	// 			notification.Type(domain.NotificationTypeConnectionRequestAccepted.Value),
	// 			notification.Type(domain.NotificationTypeConnectionEngagedWithQuestion.Value),
	// 			notification.Type(domain.NotificationTypeMutualQuestionAnswered.Value),
	// 		),
	// 	)
	// }
	// entProfiles, err := client.Profile.Query().
	// 	WithNotifications(notifFilter).
	// 	WithNotificationPermissions(func(npq *ent.NotificationPermissionQuery) {
	// 		npq.Where(notificationpermission.PlatformEQ(notificationpermission.Platform(domain.NotificationPlatformEmail.Value)))
	// 	}).
	// 	WithUser(func(uq *ent.UserQuery) {
	// 		uq.Select(user.FieldEmail, user.FieldName)
	// 	}).
	// 	Select(profile.FieldID).
	// 	Limit(NUM_PEOPLE_TO_GENERATE_FOR).
	// 	All(ctx)
	// assert.NoError(t, err)

	// for _, entProfile := range entProfiles {
	// 	err = emailUpdateSender.PrepareAndSendUpdateEmailForProfile(ctx, entProfile)
	// 	assert.NoError(t, err)
	// }
}
