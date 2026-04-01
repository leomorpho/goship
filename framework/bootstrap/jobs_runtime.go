package bootstrap

import (
	"database/sql"
	"fmt"

	jobsmodule "github.com/leomorpho/goship/v2-modules/jobs"
	"github.com/leomorpho/goship/v2/config"
	"github.com/leomorpho/goship/v2/framework/core"
)

type JobsProcess string

const (
	JobsProcessWeb    JobsProcess = "web"
	JobsProcessWorker JobsProcess = "worker"
)

type JobsRuntime struct {
	Jobs      core.Jobs
	Inspector core.JobsInspector
}

// WireJobsRuntime creates runtime jobs + inspector bridges from adapter config.
func WireJobsRuntime(cfg *config.Config, db *sql.DB, process JobsProcess) (JobsRuntime, error) {
	if cfg == nil {
		return JobsRuntime{}, fmt.Errorf("missing runtime config")
	}

	switch cfg.Adapters.Jobs {
	case "asynq":
		mod, err := jobsmodule.New(jobsmodule.Config{
			Backend: jobsmodule.BackendRedis,
			Redis: jobsmodule.RedisConfig{
				Addr:     fmt.Sprintf("%s:%d", cfg.Cache.Hostname, cfg.Cache.Port),
				Password: cfg.Cache.Password,
				DB:       cfg.Cache.Database,
			},
		})
		if err != nil {
			return JobsRuntime{}, err
		}
		return JobsRuntime{
			Jobs:      AdaptModuleJobs(mod.Jobs()),
			Inspector: AdaptModuleJobsInspector(mod.Inspector()),
		}, nil
	case "dbqueue":
		mod, err := jobsmodule.New(jobsmodule.Config{
			Backend: jobsmodule.BackendSQL,
			SQLDB:   db,
		})
		if err != nil {
			return JobsRuntime{}, err
		}
		return JobsRuntime{
			Jobs:      AdaptModuleJobs(mod.Jobs()),
			Inspector: AdaptModuleJobsInspector(mod.Inspector()),
		}, nil
	case "backlite":
		if process == JobsProcessWorker {
			return JobsRuntime{}, fmt.Errorf("jobs adapter %q runs in cmd/web and cannot be started in cmd/worker", cfg.Adapters.Jobs)
		}
		mod, err := jobsmodule.New(jobsmodule.Config{
			Backend: jobsmodule.BackendBacklite,
			SQLDB:   db,
		})
		if err != nil {
			return JobsRuntime{}, err
		}
		return JobsRuntime{
			Jobs:      AdaptModuleJobs(mod.Jobs()),
			Inspector: AdaptModuleJobsInspector(mod.Inspector()),
		}, nil
	case "inproc":
		return JobsRuntime{}, nil
	default:
		return JobsRuntime{}, fmt.Errorf("unsupported jobs adapter %q", cfg.Adapters.Jobs)
	}
}
