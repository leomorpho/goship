package jobs

import (
	"fmt"

	redisdriver "github.com/leomorpho/goship-modules/jobs/drivers/redis"
	sqldriver "github.com/leomorpho/goship-modules/jobs/drivers/sql"
	"github.com/leomorpho/goship/framework/core"
)

type Module struct {
	backend   Backend
	jobs      core.Jobs
	inspector core.JobsInspector
}

func New(cfg Config) (*Module, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	mod := &Module{backend: cfg.Backend}
	switch cfg.Backend {
	case BackendRedis:
		client := redisdriver.New(redisdriver.Config{
			Addr:     cfg.Redis.Addr,
			Password: cfg.Redis.Password,
			DB:       cfg.Redis.DB,
		})
		mod.jobs = newRedisCoreJobs(client)
		mod.inspector = newRedisJobsInspector(client)
		return mod, nil
	case BackendSQL:
		client, err := sqldriver.New(sqldriver.Config{EntClient: cfg.EntClient})
		if err != nil {
			return nil, err
		}
		mod.jobs = newSQLCoreJobs(client)
		mod.inspector = newSQLJobsInspector(client)
		return mod, nil
	default:
		return nil, fmt.Errorf("unsupported jobs backend %q", cfg.Backend)
	}
}

func (m *Module) Backend() Backend {
	if m == nil {
		return ""
	}
	return m.backend
}

func (m *Module) Jobs() core.Jobs {
	if m == nil {
		return nil
	}
	return m.jobs
}

func (m *Module) Inspector() core.JobsInspector {
	if m == nil {
		return nil
	}
	return m.inspector
}
