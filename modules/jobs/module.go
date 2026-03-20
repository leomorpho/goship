package jobs

import (
	"fmt"

	backlitedriver "github.com/leomorpho/goship-modules/jobs/drivers/backlite"
	redisdriver "github.com/leomorpho/goship-modules/jobs/drivers/redis"
	sqldriver "github.com/leomorpho/goship-modules/jobs/drivers/sql"
)

type Module struct {
	backend   Backend
	jobs      Jobs
	inspector JobsInspector
}

func New(cfg Config) (*Module, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	mod := &Module{backend: cfg.Backend}
	switch cfg.Backend {
	case BackendBacklite:
		client, err := backlitedriver.New(backlitedriver.Config{
			SQLDB: cfg.SQLDB,
		})
		if err != nil {
			return nil, err
		}
		mod.jobs = newBackliteCoreJobs(client)
		mod.inspector = newNoopJobsInspector()
		return mod, nil
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
		client, err := sqldriver.New(sqldriver.Config{
			SQLDB: cfg.SQLDB,
		})
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

func (m *Module) Jobs() Jobs {
	if m == nil {
		return nil
	}
	return m.jobs
}

func (m *Module) Inspector() JobsInspector {
	if m == nil {
		return nil
	}
	return m.inspector
}
