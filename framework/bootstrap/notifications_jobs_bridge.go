package bootstrap

import (
	"context"

	notifications "github.com/leomorpho/goship-modules/notifications"
	"github.com/leomorpho/goship/framework/core"
)

type notificationsJobsAdapter struct {
	inner core.Jobs
}

func AdaptNotificationsJobs(inner core.Jobs) notifications.Jobs {
	if inner == nil {
		return nil
	}
	return notificationsJobsAdapter{inner: inner}
}

func (a notificationsJobsAdapter) Enqueue(
	ctx context.Context,
	name string,
	payload []byte,
	options notifications.EnqueueOptions,
) (string, error) {
	return a.inner.Enqueue(ctx, name, payload, core.EnqueueOptions{
		Queue:      options.Queue,
		MaxRetries: options.MaxRetries,
	})
}
