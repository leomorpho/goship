package backlite

import (
	"context"
	"database/sql"
	"errors"
	"sync/atomic"
	"time"

	backlite "github.com/mikestefanello/backlite"
)

const (
	defaultNumWorkers      = 10
	defaultReleaseAfter    = 2 * time.Minute
	defaultCleanupInterval = 5 * time.Minute
)

type Config struct {
	SQLDB           *sql.DB
	NumWorkers      int
	ReleaseAfter    time.Duration
	CleanupInterval time.Duration
}

type Client struct {
	inner   *backlite.Client
	started atomic.Bool
}

func New(cfg Config) (*Client, error) {
	if cfg.SQLDB == nil {
		return nil, errors.New("missing database")
	}
	if cfg.NumWorkers < 1 {
		cfg.NumWorkers = defaultNumWorkers
	}
	if cfg.ReleaseAfter <= 0 {
		cfg.ReleaseAfter = defaultReleaseAfter
	}
	if cfg.CleanupInterval <= 0 {
		cfg.CleanupInterval = defaultCleanupInterval
	}

	inner, err := backlite.NewClient(backlite.ClientConfig{
		DB:              cfg.SQLDB,
		NumWorkers:      cfg.NumWorkers,
		ReleaseAfter:    cfg.ReleaseAfter,
		CleanupInterval: cfg.CleanupInterval,
	})
	if err != nil {
		return nil, err
	}
	if err := inner.Install(); err != nil {
		return nil, err
	}
	return &Client{inner: inner}, nil
}

func (c *Client) Register(queue backlite.Queue) {
	if c == nil || c.inner == nil {
		return
	}
	c.inner.Register(queue)
}

func (c *Client) Add(ctx context.Context, task backlite.Task, runAt time.Time) (string, error) {
	if c == nil || c.inner == nil {
		return "", errors.New("backlite client is not initialized")
	}
	op := c.inner.Add(task).Ctx(ctx)
	if !runAt.IsZero() {
		op = op.At(runAt)
	}
	ids, err := op.Save()
	if err != nil {
		return "", err
	}
	if len(ids) == 0 {
		return "", nil
	}
	return ids[0], nil
}

func (c *Client) Start(ctx context.Context) {
	if c == nil || c.inner == nil {
		return
	}
	if !c.started.CompareAndSwap(false, true) {
		return
	}
	c.inner.Start(ctx)
}

func (c *Client) Stop(ctx context.Context) bool {
	if c == nil || c.inner == nil {
		return true
	}
	if !c.started.Swap(false) {
		return true
	}
	return c.inner.Stop(ctx)
}
