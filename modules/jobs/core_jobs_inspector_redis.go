package jobs

import (
	"context"
	"errors"

	redisdriver "github.com/leomorpho/goship/v2-modules/jobs/drivers/redis"
)

var errRedisInspectorNotImplemented = errors.New("redis jobs inspector is not implemented yet")

type redisJobsInspector struct {
	client *redisdriver.Client
}

func newRedisJobsInspector(client *redisdriver.Client) JobsInspector {
	return &redisJobsInspector{client: client}
}

func (r *redisJobsInspector) List(context.Context, JobListFilter) ([]JobRecord, error) {
	if r == nil || r.client == nil {
		return nil, nil
	}
	return nil, errRedisInspectorNotImplemented
}

func (r *redisJobsInspector) Get(context.Context, string) (JobRecord, bool, error) {
	if r == nil || r.client == nil {
		return JobRecord{}, false, nil
	}
	return JobRecord{}, false, errRedisInspectorNotImplemented
}
