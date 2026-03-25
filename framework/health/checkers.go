package health

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/leomorpho/goship/framework/core"
)

type DBChecker struct {
	db      *sql.DB
	timeout time.Duration
}

func NewDBChecker(db *sql.DB, timeout time.Duration) *DBChecker {
	return &DBChecker{db: db, timeout: timeout}
}

func (c *DBChecker) Name() string {
	return "db"
}

func (c *DBChecker) Check(ctx context.Context) CheckResult {
	if c == nil || c.db == nil {
		return CheckResult{Status: StatusError, Error: "database not configured"}
	}

	start := time.Now()
	pingCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	if err := c.db.PingContext(pingCtx); err != nil {
		return CheckResult{
			Status:    StatusError,
			LatencyMs: time.Since(start).Milliseconds(),
			Error:     err.Error(),
		}
	}

	return CheckResult{
		Status:    StatusOK,
		LatencyMs: time.Since(start).Milliseconds(),
	}
}

type CacheChecker struct {
	cache   core.Cache
	timeout time.Duration
}

func NewCacheChecker(cache core.Cache, timeout time.Duration) *CacheChecker {
	return &CacheChecker{cache: cache, timeout: timeout}
}

func (c *CacheChecker) Name() string {
	return "cache"
}

func (c *CacheChecker) Check(ctx context.Context) CheckResult {
	if c == nil || c.cache == nil {
		return CheckResult{Status: StatusError, Error: "cache not configured"}
	}

	start := time.Now()
	checkCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	key := fmt.Sprintf("health-check:%d", time.Now().UnixNano())
	if err := c.cache.Set(checkCtx, key, []byte("ok"), 5*time.Second); err != nil {
		return CheckResult{
			Status:    StatusError,
			LatencyMs: time.Since(start).Milliseconds(),
			Error:     err.Error(),
		}
	}
	value, found, err := c.cache.Get(checkCtx, key)
	if err != nil {
		return CheckResult{
			Status:    StatusError,
			LatencyMs: time.Since(start).Milliseconds(),
			Error:     err.Error(),
		}
	}
	if !found || string(value) != "ok" {
		return CheckResult{
			Status:    StatusError,
			LatencyMs: time.Since(start).Milliseconds(),
			Error:     "cache value mismatch",
		}
	}
	if err := c.cache.Delete(checkCtx, key); err != nil {
		return CheckResult{
			Status:    StatusError,
			LatencyMs: time.Since(start).Milliseconds(),
			Error:     err.Error(),
		}
	}

	return CheckResult{
		Status:    StatusOK,
		LatencyMs: time.Since(start).Milliseconds(),
	}
}

type JobsChecker struct {
	inspector core.JobsInspector
	timeout   time.Duration
}

type EnvRequirement struct {
	Name  string
	Value string
}

type EnvChecker struct {
	required []EnvRequirement
}

func NewEnvChecker(required ...EnvRequirement) *EnvChecker {
	return &EnvChecker{required: slices.Clone(required)}
}

func (c *EnvChecker) Name() string {
	return "env"
}

func (c *EnvChecker) ValidateStartup() error {
	if c == nil {
		return fmt.Errorf("required runtime environment variables are missing: [env checker not configured]")
	}

	missing := make([]string, 0)
	for _, requirement := range c.required {
		name := strings.TrimSpace(requirement.Name)
		if name == "" {
			continue
		}
		if strings.TrimSpace(requirement.Value) == "" {
			missing = append(missing, name)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("required runtime environment variables are missing: %v", missing)
	}

	return nil
}

func (c *EnvChecker) Check(context.Context) CheckResult {
	if err := c.ValidateStartup(); err != nil {
		return CheckResult{
			Status: StatusError,
			Error:  err.Error(),
		}
	}
	return CheckResult{Status: StatusOK}
}

func NewJobsChecker(inspector core.JobsInspector, timeout time.Duration) *JobsChecker {
	return &JobsChecker{inspector: inspector, timeout: timeout}
}

func (c *JobsChecker) Name() string {
	return "jobs"
}

func (c *JobsChecker) Check(ctx context.Context) CheckResult {
	if c == nil || c.inspector == nil {
		return CheckResult{Status: StatusError, Error: "jobs inspector not configured"}
	}

	start := time.Now()
	checkCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	rows, err := c.inspector.List(checkCtx, core.JobListFilter{
		Statuses: []core.JobStatus{core.JobStatusQueued},
		Limit:    1000,
	})
	if err != nil {
		// Some backends still expose a no-op inspector; treat that as non-fatal readiness signal.
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			return CheckResult{
				Status:    StatusError,
				LatencyMs: time.Since(start).Milliseconds(),
				Error:     err.Error(),
			}
		}
		return CheckResult{
			Status:    StatusOK,
			LatencyMs: time.Since(start).Milliseconds(),
			Extra: map[string]any{
				"inspector_error": err.Error(),
			},
		}
	}

	return CheckResult{
		Status:     StatusOK,
		LatencyMs:  time.Since(start).Milliseconds(),
		QueueDepth: len(rows),
	}
}
