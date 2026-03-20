package tasks

import (
	"context"

	"github.com/hibiken/asynq"
	paidsubscriptions "github.com/leomorpho/goship-modules/paidsubscriptions"
)

const TypeDeactivateExpiredSubscriptions = "subscription.deactivate_all_expired"

type (
	DeactivateExpiredSubscriptionsProcessor struct {
		subscriptionsService *paidsubscriptions.Service
	}

	DeactivateExpiredSubscriptionsPayload struct {
	}
)

func NewDeactivateExpiredSubscriptionsProcessor(
	subscriptionsService *paidsubscriptions.Service,
) *DeactivateExpiredSubscriptionsProcessor {

	return &DeactivateExpiredSubscriptionsProcessor{
		subscriptionsService: subscriptionsService,
	}
}
func (d *DeactivateExpiredSubscriptionsProcessor) ProcessTask(
	ctx context.Context, t *asynq.Task,
) error {

	return d.subscriptionsService.DeactivateExpiredSubscriptions(ctx)
}

// -------------------------------------------------------------
// TODO

const TypeSubscriptionPaymentFailed = "subscription.payment_failed"

type (
	SubscriptionPaymentFailedProcessor struct {
		subscriptionsService *paidsubscriptions.Service
	}

	SubscriptionPaymentFailedPayload struct {
	}
)

func NewSubscriptionPaymentFailedProcessor(
	subscriptionsService *paidsubscriptions.Service,
) *SubscriptionPaymentFailedProcessor {

	return &SubscriptionPaymentFailedProcessor{
		subscriptionsService: subscriptionsService,
	}
}
func (d *SubscriptionPaymentFailedProcessor) ProcessTask(
	ctx context.Context, t *asynq.Task,
) error {

	return d.subscriptionsService.DeactivateExpiredSubscriptions(ctx)
}

// -------------------------------------------------------------
// TODO

const TypeSubscriptionCreated = "subscription.created"

type (
	SubscriptionCreatedProcessor struct {
		subscriptionsService *paidsubscriptions.Service
	}

	SubscriptionCreatedPayload struct {
	}
)

func NewSubscriptionCreatedProcessor(
	subscriptionsService *paidsubscriptions.Service,
) *SubscriptionCreatedProcessor {

	return &SubscriptionCreatedProcessor{
		subscriptionsService: subscriptionsService,
	}
}
func (d *SubscriptionCreatedProcessor) ProcessTask(
	ctx context.Context, t *asynq.Task,
) error {

	return d.subscriptionsService.DeactivateExpiredSubscriptions(ctx)
}

// -------------------------------------------------------------
// TODO

const TypeSubscriptionUpdated = "subscription.updated"

type (
	SubscriptionUpdatedProcessor struct {
		subscriptionsService *paidsubscriptions.Service
	}

	SubscriptionUpdatedPayload struct {
	}
)

func NewSubscriptionUpdatedProcessor(
	subscriptionsService *paidsubscriptions.Service,
) *SubscriptionUpdatedProcessor {

	return &SubscriptionUpdatedProcessor{
		subscriptionsService: subscriptionsService,
	}
}
func (d *SubscriptionUpdatedProcessor) ProcessTask(
	ctx context.Context, t *asynq.Task,
) error {

	return d.subscriptionsService.DeactivateExpiredSubscriptions(ctx)
}

// -------------------------------------------------------------
// TODO

const TypeSubscriptionDeleted = "subscription.deleted"

type (
	SubscriptionDeletedProcessor struct {
		subscriptionsService *paidsubscriptions.Service
	}

	SubscriptionDeletedPayload struct {
	}
)

func NewSubscriptionDeletedProcessor(
	subscriptionsService *paidsubscriptions.Service,
) *SubscriptionDeletedProcessor {

	return &SubscriptionDeletedProcessor{
		subscriptionsService: subscriptionsService,
	}
}
func (d *SubscriptionDeletedProcessor) ProcessTask(
	ctx context.Context, t *asynq.Task,
) error {

	return d.subscriptionsService.DeactivateExpiredSubscriptions(ctx)
}
