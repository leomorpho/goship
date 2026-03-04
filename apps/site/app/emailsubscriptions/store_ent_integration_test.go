//go:build integration

package emailsubscriptions_test

import (
	"testing"

	"github.com/leomorpho/goship-modules/emailsubscriptions"
	appemailsubscriptions "github.com/leomorpho/goship/apps/site/app/emailsubscriptions"
	"github.com/leomorpho/goship/framework/domain"
	"github.com/leomorpho/goship/framework/tests"
	"github.com/stretchr/testify/assert"
)

func TestEntStoreAdapter(t *testing.T) {
	client, ctx := tests.CreateTestContainerPostgresEntClient(t)
	defer client.Close()

	service := emailsubscriptions.NewServiceWithVerifier(
		appemailsubscriptions.NewEntStore(client),
		func(string) error { return nil },
	)

	newsletter := emailsubscriptions.List(domain.EmailNewsletter.Value)
	announce := emailsubscriptions.List(domain.EmailInitialAnnoucement.Value)

	err := service.CreateList(ctx, newsletter)
	assert.NoError(t, err)
	err = service.CreateList(ctx, announce)
	assert.NoError(t, err)

	sub, err := service.Subscribe(ctx, "alice@example.com", newsletter, nil, nil)
	assert.NoError(t, err)
	assert.NotEmpty(t, sub.ConfirmationCode)

	err = service.Confirm(ctx, sub.ConfirmationCode)
	assert.NoError(t, err)

	err = service.Unsubscribe(ctx, "alice@example.com", sub.ConfirmationCode, newsletter)
	assert.NoError(t, err)
}
