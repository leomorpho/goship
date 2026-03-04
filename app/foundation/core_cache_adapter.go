package foundation

import (
	"context"
	"errors"
	"time"

	"github.com/go-redis/redis/v8"
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
	if a == nil || a.client == nil || a.client.Client == nil {
		return nil, false, errors.New("cache client is not initialized")
	}
	val, err := a.client.Client.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return val, true, nil
}

func (a *CoreCacheAdapter) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if a == nil || a.client == nil || a.client.Client == nil {
		return errors.New("cache client is not initialized")
	}
	return a.client.Client.Set(ctx, key, value, ttl).Err()
}

func (a *CoreCacheAdapter) Delete(ctx context.Context, key string) error {
	if a == nil || a.client == nil || a.client.Client == nil {
		return errors.New("cache client is not initialized")
	}
	return a.client.Client.Del(ctx, key).Err()
}

func (a *CoreCacheAdapter) InvalidatePrefix(ctx context.Context, prefix string) error {
	if a == nil || a.client == nil || a.client.Client == nil {
		return errors.New("cache client is not initialized")
	}

	keys, err := a.client.Client.Keys(ctx, prefix+"*").Result()
	if err != nil {
		return err
	}
	if len(keys) == 0 {
		return nil
	}
	return a.client.Client.Del(ctx, keys...).Err()
}

func (a *CoreCacheAdapter) Close() error {
	if a == nil || a.client == nil {
		return nil
	}
	return a.client.Close()
}
