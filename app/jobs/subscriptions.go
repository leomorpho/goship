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
