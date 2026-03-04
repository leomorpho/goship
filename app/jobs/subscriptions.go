package tasks

import (
	"context"

	"github.com/hibiken/asynq"
	paidsubscriptions "github.com/leomorpho/goship-modules/paidsubscriptions"
)

const TypeDeactivateExpiredSubscriptions = "subscription.deactivate_all_expired"

type (
	DeactivateExpiredSubscriptionsProcessor struct {
		subscriptionsRepo *paidsubscriptions.Service
	}

	DeactivateExpiredSubscriptionsPayload struct {
	}
)

func NewDeactivateExpiredSubscriptionsProcessor(
	subscriptionsRepo *paidsubscriptions.Service,
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
		subscriptionsRepo *paidsubscriptions.Service
	}

	SubscriptionPaymentFailedPayload struct {
	}
)

func NewSubscriptionPaymentFailedProcessor(
	subscriptionsRepo *paidsubscriptions.Service,
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
		subscriptionsRepo *paidsubscriptions.Service
	}

	SubscriptionCreatedPayload struct {
	}
)

func NewSubscriptionCreatedProcessor(
	subscriptionsRepo *paidsubscriptions.Service,
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
		subscriptionsRepo *paidsubscriptions.Service
	}

	SubscriptionUpdatedPayload struct {
	}
)

func NewSubscriptionUpdatedProcessor(
	subscriptionsRepo *paidsubscriptions.Service,
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
		subscriptionsRepo *paidsubscriptions.Service
	}

	SubscriptionDeletedPayload struct {
	}
)

func NewSubscriptionDeletedProcessor(
	subscriptionsRepo *paidsubscriptions.Service,
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
