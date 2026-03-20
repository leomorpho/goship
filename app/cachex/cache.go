package cachex

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/eko/gocache/v2/cache"
	"github.com/eko/gocache/v2/marshaler"
	"github.com/eko/gocache/v2/store"
	"github.com/go-redis/redis/v8"
	"github.com/leomorpho/goship/config"
	cacherepo "github.com/leomorpho/goship/framework/repos/cache"
)

type (
	// CacheClient is the client that allows you to interact with the cache
	CacheClient struct {
		// Client stores the client to the underlying cache service
		Client *redis.Client

		// cache stores the cache interface
		cache *cache.Cache

		// otter stores the in-memory cache backend for single-process mode.
		otter *cacherepo.OtterStore
	}

	// cacheSet handles chaining a set operation
	cacheSet struct {
		client     *CacheClient
		key        string
		group      string
		data       any
		expiration time.Duration
		tags       []string
	}

	// cacheGet handles chaining a get operation
	cacheGet struct {
		client   *CacheClient
		key      string
		group    string
		dataType any
	}

	// cacheFlush handles chaining a flush operation
	cacheFlush struct {
		client *CacheClient
		key    string
		group  string
		tags   []string
	}
)

// NewCacheClient creates a new cache client
func NewCacheClient(cfg *config.Config) (*CacheClient, error) {
	adapter := normalizeCacheAdapter(cfg.Adapters.Cache)
	if adapter == "otter" {
		store, err := cacherepo.NewOtterStore(10_000)
		if err != nil {
			return nil, err
		}
		return &CacheClient{otter: store}, nil
	}

	// Determine the database based on the environment
	db := cfg.Cache.Database
	if cfg.App.Environment == config.EnvTest {
		db = cfg.Cache.TestDatabase
	}

	// Connect to the cache
	c := &CacheClient{}
	c.Client = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Cache.Hostname, cfg.Cache.Port),
		Password: cfg.Cache.Password,
		DB:       db,
	})
	if _, err := c.Client.Ping(context.Background()).Result(); err != nil {
		return c, err
	}

	// Flush the database if this is the test environment
	if cfg.App.Environment == config.EnvTest {
		if err := c.Client.FlushDB(context.Background()).Err(); err != nil {
			return c, err
		}
	}

	cacheStore := store.NewRedis(c.Client, nil)
	c.cache = cache.New(cacheStore)
	return c, nil
}

// Close closes the connection to the cache
func (c *CacheClient) Close() error {
	if c == nil {
		return nil
	}
	if c.otter != nil {
		return c.otter.Close()
	}
	if c == nil || c.Client == nil {
		return nil
	}
	return c.Client.Close()
}

// Set creates a cache set operation
func (c *CacheClient) Set() *cacheSet {
	return &cacheSet{
		client: c,
	}
}

// Get creates a cache get operation
func (c *CacheClient) Get() *cacheGet {
	return &cacheGet{
		client: c,
	}
}

// Flush creates a cache flush operation
func (c *CacheClient) Flush() *cacheFlush {
	return &cacheFlush{
		client: c,
	}
}

// cacheKey formats a cache key with an optional group
func (c *CacheClient) cacheKey(group, key string) string {
	if group != "" {
		return fmt.Sprintf("%s::%s", group, key)
	}
	return key
}

// Key sets the cache key
func (c *cacheSet) Key(key string) *cacheSet {
	c.key = key
	return c
}

// Group sets the cache group
func (c *cacheSet) Group(group string) *cacheSet {
	c.group = group
	return c
}

// Data sets the data to cache
func (c *cacheSet) Data(data any) *cacheSet {
	c.data = data
	return c
}

// Expiration sets the expiration duration of the cached data
func (c *cacheSet) Expiration(expiration time.Duration) *cacheSet {
	c.expiration = expiration
	return c
}

// Tags sets the cache tags
func (c *cacheSet) Tags(tags ...string) *cacheSet {
	c.tags = tags
	return c
}

// Save saves the data in the cache
func (c *cacheSet) Save(ctx context.Context) error {
	if c.key == "" {
		return errors.New("no cache key specified")
	}
	cacheKey := c.client.cacheKey(c.group, c.key)
	expiration := normalizeCacheTTL(c.expiration)

	if c.client.otter != nil {
		payload, err := json.Marshal(c.data)
		if err != nil {
			return err
		}
		if err := c.client.otter.Set(cacheKey, payload, expiration); err != nil {
			return err
		}
		c.client.otter.SetTags(cacheKey, c.tags)
		return nil
	}

	opts := &store.Options{
		Expiration: expiration,
		Tags:       c.tags,
	}

	return marshaler.
		New(c.client.cache).
		Set(ctx, cacheKey, c.data, opts)
}

// Key sets the cache key
func (c *cacheGet) Key(key string) *cacheGet {
	c.key = key
	return c
}

// Group sets the cache group
func (c *cacheGet) Group(group string) *cacheGet {
	c.group = group
	return c
}

// Type sets the expected Go type of the data being retrieved from the cache
func (c *cacheGet) Type(expectedType any) *cacheGet {
	c.dataType = expectedType
	return c
}

// Fetch fetches the data from the cache
func (c *cacheGet) Fetch(ctx context.Context) (any, error) {
	if c.key == "" {
		return nil, errors.New("no cache key specified")
	}
	cacheKey := c.client.cacheKey(c.group, c.key)

	if c.client.otter != nil {
		payload, ok := c.client.otter.Get(cacheKey)
		if !ok {
			return nil, redis.Nil
		}
		if c.dataType == nil {
			return payload, nil
		}
		if err := json.Unmarshal(payload, c.dataType); err != nil {
			return nil, err
		}
		return c.dataType, nil
	}

	return marshaler.New(c.client.cache).Get(
		ctx,
		cacheKey,
		c.dataType,
	)
}

// Key sets the cache key
func (c *cacheFlush) Key(key string) *cacheFlush {
	c.key = key
	return c
}

// Group sets the cache group
func (c *cacheFlush) Group(group string) *cacheFlush {
	c.group = group
	return c
}

// Tags sets the cache tags
func (c *cacheFlush) Tags(tags ...string) *cacheFlush {
	c.tags = tags
	return c
}

// Execute flushes the data from the cache
func (c *cacheFlush) Execute(ctx context.Context) error {
	if c.client.otter != nil {
		if len(c.tags) > 0 {
			if err := c.client.otter.InvalidateTags(c.tags); err != nil {
				return err
			}
		}
		if c.key != "" {
			return c.client.otter.Delete(c.client.cacheKey(c.group, c.key))
		}
		return nil
	}

	if len(c.tags) > 0 {
		if err := c.client.cache.Invalidate(ctx, store.InvalidateOptions{
			Tags: c.tags,
		}); err != nil {
			return err
		}
	}

	if c.key != "" {
		return c.client.cache.Delete(ctx, c.client.cacheKey(c.group, c.key))
	}

	return nil
}

func (c *CacheClient) GetBytes(ctx context.Context, key string) ([]byte, bool, error) {
	if c == nil {
		return nil, false, errors.New("cache client is not initialized")
	}
	if c.otter != nil {
		value, found := c.otter.Get(key)
		return value, found, nil
	}
	if c.Client == nil {
		return nil, false, errors.New("cache client is not initialized")
	}
	val, err := c.Client.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return val, true, nil
}

func (c *CacheClient) SetBytes(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if c == nil {
		return errors.New("cache client is not initialized")
	}
	ttl = normalizeCacheTTL(ttl)
	if c.otter != nil {
		return c.otter.Set(key, value, ttl)
	}
	if c.Client == nil {
		return errors.New("cache client is not initialized")
	}
	return c.Client.Set(ctx, key, value, ttl).Err()
}

func (c *CacheClient) DeleteKey(ctx context.Context, key string) error {
	if c == nil {
		return errors.New("cache client is not initialized")
	}
	if c.otter != nil {
		return c.otter.Delete(key)
	}
	if c.Client == nil {
		return errors.New("cache client is not initialized")
	}
	return c.Client.Del(ctx, key).Err()
}

func (c *CacheClient) InvalidatePrefix(ctx context.Context, prefix string) error {
	if c == nil {
		return errors.New("cache client is not initialized")
	}
	if c.otter != nil {
		return c.otter.InvalidatePrefix(prefix)
	}
	if c.Client == nil {
		return errors.New("cache client is not initialized")
	}
	keys, err := c.Client.Keys(ctx, prefix+"*").Result()
	if err != nil {
		return err
	}
	if len(keys) == 0 {
		return nil
	}
	return c.Client.Del(ctx, keys...).Err()
}

func normalizeCacheAdapter(v string) string {
	switch normalize := strings.ToLower(strings.TrimSpace(v)); normalize {
	case "", "memory", "otter":
		return "otter"
	default:
		return normalize
	}
}

func normalizeCacheTTL(ttl time.Duration) time.Duration {
	if ttl <= 0 {
		return ttl
	}
	if ttl%time.Second == 0 {
		return ttl
	}
	return ((ttl / time.Second) + 1) * time.Second
}
