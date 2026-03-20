package schedules

import (
	"context"
	"log/slog"

	"github.com/leomorpho/goship/framework/core"
	"github.com/robfig/cron/v3"
)

type JobsProvider func() core.Jobs

const (
	allDailyConversationJob = "notification.all_daily_conversation"
	deleteStaleJob          = "notification.recycling"
)

// Register wires periodic schedules that enqueue background jobs.
func Register(s *cron.Cron, jobsProvider JobsProvider) {
	if s == nil || jobsProvider == nil {
		return
	}

	addEnqueueSchedule(s, jobsProvider, "0 0 9 * * *", allDailyConversationJob)
	addEnqueueSchedule(s, jobsProvider, "0 0 * * * *", deleteStaleJob)

	// ship:schedules:start
	// ship:schedules:end
}

func addEnqueueSchedule(s *cron.Cron, jobsProvider JobsProvider, expr, name string) {
	_, err := s.AddFunc(expr, func() {
		jobs := jobsProvider()
		if jobs == nil {
			slog.Error("scheduled job enqueue skipped: jobs adapter unavailable", "job", name, "schedule", expr)
			return
		}
		if _, err := jobs.Enqueue(context.Background(), name, nil, core.EnqueueOptions{}); err != nil {
			slog.Error("scheduled job enqueue failed", "job", name, "schedule", expr, "error", err)
		}
	})
	if err != nil {
		slog.Error("failed to register schedule", "job", name, "schedule", expr, "error", err)
	}
}
