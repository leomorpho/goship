package tasks

import (
	"context"

	"github.com/hibiken/asynq"
	"github.com/mikestefanello/pagoda/pkg/repos/subscriptions"
)

const TypeDeactivateExpiredSubscriptions = "subscription.deactivate_all_expired"

type (
	DeactivateExpiredSubscriptionsProcessor struct {
		subscriptionsRepo *subscriptions.SubscriptionsRepo
	}

	DeactivateExpiredSubscriptionsPayload struct {
	}
)

func NewDeactivateExpiredSubscriptionsProcessor(
	subscriptionsRepo *subscriptions.SubscriptionsRepo,
) *DeactivateExpiredSubscriptionsProcessor {

	return &DeactivateExpiredSubscriptionsProcessor{
		subscriptionsRepo: subscriptionsRepo,
	}
}
func (d *DeactivateExpiredSubscriptionsProcessor) ProcessTask(
	ctx context.Context, t *asynq.Task,
) error {

	return d.subscriptionsRepo.DeactivateExpiredSubscriptions(ctx)
}

// -------------------------------------------------------------
// TODO

const TypeSubscriptionPaymentFailed = "subscription.payment_failed"

type (
	SubscriptionPaymentFailedProcessor struct {
		subscriptionsRepo *subscriptions.SubscriptionsRepo
	}

	SubscriptionPaymentFailedPayload struct {
	}
)

func NewSubscriptionPaymentFailedProcessor(
	subscriptionsRepo *subscriptions.SubscriptionsRepo,
) *SubscriptionPaymentFailedProcessor {

	return &SubscriptionPaymentFailedProcessor{
		subscriptionsRepo: subscriptionsRepo,
	}
}
func (d *SubscriptionPaymentFailedProcessor) ProcessTask(
	ctx context.Context, t *asynq.Task,
) error {

	return d.subscriptionsRepo.DeactivateExpiredSubscriptions(ctx)
}

// -------------------------------------------------------------
// TODO

const TypeSubscriptionCreated = "subscription.created"

type (
	SubscriptionCreatedProcessor struct {
		subscriptionsRepo *subscriptions.SubscriptionsRepo
	}

	SubscriptionCreatedPayload struct {
	}
)

func NewSubscriptionCreatedProcessor(
	subscriptionsRepo *subscriptions.SubscriptionsRepo,
) *SubscriptionCreatedProcessor {

	return &SubscriptionCreatedProcessor{
		subscriptionsRepo: subscriptionsRepo,
	}
}
func (d *SubscriptionCreatedProcessor) ProcessTask(
	ctx context.Context, t *asynq.Task,
) error {

	return d.subscriptionsRepo.DeactivateExpiredSubscriptions(ctx)
}

// -------------------------------------------------------------
// TODO

const TypeSubscriptionUpdated = "subscription.updated"

type (
	SubscriptionUpdatedProcessor struct {
		subscriptionsRepo *subscriptions.SubscriptionsRepo
	}

	SubscriptionUpdatedPayload struct {
	}
)

func NewSubscriptionUpdatedProcessor(
	subscriptionsRepo *subscriptions.SubscriptionsRepo,
) *SubscriptionUpdatedProcessor {

	return &SubscriptionUpdatedProcessor{
		subscriptionsRepo: subscriptionsRepo,
	}
}
func (d *SubscriptionUpdatedProcessor) ProcessTask(
	ctx context.Context, t *asynq.Task,
) error {

	return d.subscriptionsRepo.DeactivateExpiredSubscriptions(ctx)
}

// -------------------------------------------------------------
// TODO

const TypeSubscriptionDeleted = "subscription.deleted"

type (
	SubscriptionDeletedProcessor struct {
		subscriptionsRepo *subscriptions.SubscriptionsRepo
	}

	SubscriptionDeletedPayload struct {
	}
)

func NewSubscriptionDeletedProcessor(
	subscriptionsRepo *subscriptions.SubscriptionsRepo,
) *SubscriptionDeletedProcessor {

	return &SubscriptionDeletedProcessor{
		subscriptionsRepo: subscriptionsRepo,
	}
}
func (d *SubscriptionDeletedProcessor) ProcessTask(
	ctx context.Context, t *asynq.Task,
) error {

	return d.subscriptionsRepo.DeactivateExpiredSubscriptions(ctx)
}
