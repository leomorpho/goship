package foundation

import (
	"context"
	"errors"
	"time"

	"github.com/leomorpho/goship/framework/core"
)

var _ core.Cache = (*CoreCacheAdapter)(nil)

// CoreCacheAdapter adapts CacheClient to the core.Cache interface.
type CoreCacheAdapter struct {
	client *CacheClient
}

func NewCoreCacheAdapter(client *CacheClient) *CoreCacheAdapter {
	return &CoreCacheAdapter{client: client}
}

func (a *CoreCacheAdapter) Get(ctx context.Context, key string) ([]byte, bool, error) {
	if a == nil || a.client == nil {
		return nil, false, errors.New("cache client is not initialized")
	}
	return a.client.GetBytes(ctx, key)
}

func (a *CoreCacheAdapter) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if a == nil || a.client == nil {
		return errors.New("cache client is not initialized")
	}
	return a.client.SetBytes(ctx, key, value, ttl)
}

func (a *CoreCacheAdapter) Delete(ctx context.Context, key string) error {
	if a == nil || a.client == nil {
		return errors.New("cache client is not initialized")
	}
	return a.client.DeleteKey(ctx, key)
}

func (a *CoreCacheAdapter) InvalidatePrefix(ctx context.Context, prefix string) error {
	if a == nil || a.client == nil {
		return errors.New("cache client is not initialized")
	}
	return a.client.InvalidatePrefix(ctx, prefix)
}

func (a *CoreCacheAdapter) Close() error {
	if a == nil || a.client == nil {
		return nil
	}
	return a.client.Close()
}
