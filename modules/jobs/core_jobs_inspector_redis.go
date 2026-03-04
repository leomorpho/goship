package jobs

import (
	"context"
	"errors"

	redisdriver "github.com/leomorpho/goship-modules/jobs/drivers/redis"
	"github.com/leomorpho/goship/framework/core"
)

var errRedisInspectorNotImplemented = errors.New("redis jobs inspector is not implemented yet")

type redisJobsInspector struct {
	client *redisdriver.Client
}

func newRedisJobsInspector(client *redisdriver.Client) core.JobsInspector {
	return &redisJobsInspector{client: client}
}

func (r *redisJobsInspector) List(context.Context, core.JobListFilter) ([]core.JobRecord, error) {
	if r == nil || r.client == nil {
		return nil, nil
	}
	return nil, errRedisInspectorNotImplemented
}

func (r *redisJobsInspector) Get(context.Context, string) (core.JobRecord, bool, error) {
	if r == nil || r.client == nil {
		return core.JobRecord{}, false, nil
	}
	return core.JobRecord{}, false, errRedisInspectorNotImplemented
}
