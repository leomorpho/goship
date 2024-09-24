package emailsmanager_test

import (
	"testing"

	"github.com/mikestefanello/pagoda/pkg/domain"
	"github.com/mikestefanello/pagoda/pkg/repos/emailsmanager"
	"github.com/mikestefanello/pagoda/pkg/tests"
	"github.com/stretchr/testify/assert"
)

func TestEmailListSubscribtion(t *testing.T) {
	/*
		Test:
			- CreateNewSubscriptionList
			- SSESubscribe
			- ConfirmSubscription
	*/
	client, ctx := tests.CreateTestContainerPostgresEntClient(t)
	defer client.Close()

	emailRepo := emailsmanager.NewEmailSubscriptionRepo(client)

	err := emailRepo.CreateNewSubscriptionList(ctx, domain.EmailNewsletter)
	assert.Nil(t, err)

	err = emailRepo.CreateNewSubscriptionList(ctx, domain.EmailInitialAnnoucement)
	assert.Nil(t, err)

	subscriptionTypes, err := client.EmailSubscriptionType.Query().All(ctx)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(subscriptionTypes))
	assert.Equal(t, domain.EmailNewsletter.Value, subscriptionTypes[0].Name.String())
	assert.Equal(t, domain.EmailInitialAnnoucement.Value, subscriptionTypes[1].Name.String())
	assert.True(t, subscriptionTypes[0].Active)
	assert.True(t, subscriptionTypes[1].Active)

	subscriptionDomainObj, err := emailRepo.SSESubscribe(ctx, "", domain.EmailNewsletter, nil, nil)
	assert.NotNil(t, err)
	subscriptions, err := client.EmailSubscription.Query().WithSubscriptions().All(ctx)
	assert.Nil(t, subscriptionDomainObj)
	assert.Nil(t, err)
	assert.Equal(t, 0, len(subscriptions))

	email := "alice@gmail.com"
	lat := 0.4
	lon := 0.1
	subscriptionDomainObj, err = emailRepo.SSESubscribe(ctx, email, domain.EmailNewsletter, &lat, &lon)
	assert.Nil(t, err)
	assert.NotNil(t, subscriptionDomainObj)

	subscriptions, err = client.EmailSubscription.Query().WithSubscriptions().All(ctx)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(subscriptions))
	assert.Equal(t, email, subscriptions[0].Email)
	assert.Equal(t, lat, subscriptions[0].Latitude)
	assert.Equal(t, lon, subscriptions[0].Longitude)
	assert.False(t, subscriptions[0].Verified)
	assert.NotEmpty(t, subscriptions[0].ConfirmationCode)
	assert.Equal(t, 1, len(subscriptions[0].Edges.Subscriptions))
	confirmationCode := subscriptions[0].ConfirmationCode
	assert.Equal(t, confirmationCode, subscriptionDomainObj.ConfirmationCode)

	// Same person subscribes to the same list, no changes
	subscriptionDomainObj, err = emailRepo.SSESubscribe(ctx, email, domain.EmailNewsletter, nil, nil)
	assert.NotNil(t, err)
	subscriptions, err = client.EmailSubscription.Query().WithSubscriptions().All(ctx)
	assert.Nil(t, err)
	assert.Nil(t, subscriptionDomainObj)
	assert.Equal(t, 1, len(subscriptions))
	assert.Equal(t, email, subscriptions[0].Email)
	assert.False(t, subscriptions[0].Verified)
	assert.Equal(t, 1, len(subscriptions[0].Edges.Subscriptions))
	assert.Equal(t, confirmationCode, subscriptions[0].ConfirmationCode)

	// Same person subscribes to a new emailing list, which should be added to their subscriptions
	subscriptionDomainObj, err = emailRepo.SSESubscribe(ctx, email, domain.EmailInitialAnnoucement, nil, nil)
	assert.Nil(t, err)
	subscriptions, err = client.EmailSubscription.Query().WithSubscriptions().All(ctx)
	assert.Nil(t, err)
	assert.NotNil(t, subscriptionDomainObj)
	assert.Equal(t, 1, len(subscriptions))
	assert.Equal(t, domain.EmailNewsletter.Value, subscriptions[0].Edges.Subscriptions[0].Name.String())
	assert.Equal(t, domain.EmailInitialAnnoucement.Value, subscriptions[0].Edges.Subscriptions[1].Name.String())

	// Attempt to confirm subscription with invalid code
	err = emailRepo.ConfirmSubscription(ctx, "invalid code")
	assert.Equal(t, emailsmanager.ErrInvalidEmailConfirmationCode, err)

	// Confirm subscription
	err = emailRepo.ConfirmSubscription(ctx, confirmationCode)
	assert.Nil(t, err)
	subscription, err := client.EmailSubscription.Query().First(ctx)
	assert.Nil(t, err)
	assert.True(t, subscription.Verified)
	newConfirmationCode := subscription.ConfirmationCode
	assert.NotEqual(t, confirmationCode, newConfirmationCode)

	// Try to re-confirm with already confirmed subscription with stale confirmation code
	err = emailRepo.ConfirmSubscription(ctx, confirmationCode)
	assert.NotNil(t, err)

	// SSEUnsubscribe the subscriber
	err = emailRepo.SSEUnsubscribe(ctx, email, newConfirmationCode, domain.EmailInitialAnnoucement)
	assert.Nil(t, err)
	subscription, err = client.EmailSubscription.Query().WithSubscriptions().First(ctx)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(subscription.Edges.Subscriptions))

	err = emailRepo.SSEUnsubscribe(ctx, email, newConfirmationCode, domain.EmailNewsletter)
	assert.Nil(t, err)
	subscriptions, err = client.EmailSubscription.Query().WithSubscriptions().All(ctx)
	assert.Nil(t, err)
	assert.Equal(t, 0, len(subscriptions))

	// SSESubscribe two users to the same list
	subscriptionDomainObj, err = emailRepo.SSESubscribe(ctx, "test@gmail.com", domain.EmailNewsletter, nil, nil)
	assert.NotNil(t, subscriptionDomainObj)
	assert.Nil(t, err)
	subscriptionDomainObj, err = emailRepo.SSESubscribe(ctx, "test2@gmail.com", domain.EmailNewsletter, nil, nil)
	assert.NotNil(t, subscriptionDomainObj)
	assert.Nil(t, err)

	subscriptions, err = client.EmailSubscription.Query().WithSubscriptions().All(ctx)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(subscriptions))
}
