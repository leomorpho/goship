package tasks

import (
	"context"

	"github.com/hibiken/asynq"
	paidsubscriptions "github.com/leomorpho/goship-modules/paidsubscriptions"
)

const TypeDeactivateExpiredSubscriptions = "subscription.deactivate_all_expired"

type (
	DeactivateExpiredSubscriptionsProcessor struct {
		subscriptionsRepo *paidsubscriptions.SubscriptionsRepo
	}

	DeactivateExpiredSubscriptionsPayload struct {
	}
)

func NewDeactivateExpiredSubscriptionsProcessor(
	subscriptionsRepo *paidsubscriptions.SubscriptionsRepo,
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
		subscriptionsRepo *paidsubscriptions.SubscriptionsRepo
	}

	SubscriptionPaymentFailedPayload struct {
	}
)

func NewSubscriptionPaymentFailedProcessor(
	subscriptionsRepo *paidsubscriptions.SubscriptionsRepo,
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
		subscriptionsRepo *paidsubscriptions.SubscriptionsRepo
	}

	SubscriptionCreatedPayload struct {
	}
)

func NewSubscriptionCreatedProcessor(
	subscriptionsRepo *paidsubscriptions.SubscriptionsRepo,
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
		subscriptionsRepo *paidsubscriptions.SubscriptionsRepo
	}

	SubscriptionUpdatedPayload struct {
	}
)

func NewSubscriptionUpdatedProcessor(
	subscriptionsRepo *paidsubscriptions.SubscriptionsRepo,
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
		subscriptionsRepo *paidsubscriptions.SubscriptionsRepo
	}

	SubscriptionDeletedPayload struct {
	}
)

func NewSubscriptionDeletedProcessor(
	subscriptionsRepo *paidsubscriptions.SubscriptionsRepo,
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
