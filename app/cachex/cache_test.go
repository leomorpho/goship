//go:build integration

package cachex

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/leomorpho/goship/app/foundation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCacheClient(t *testing.T) {
	c := foundation.NewContainer()
	t.Cleanup(func() {
		require.NoError(t, c.Shutdown())
	})
	if c.Cache == nil {
		t.Skip("cache dependency disabled in current runtime plan")
	}

	type cacheTest struct {
		Value string
	}
	// Cache some data
	data := cacheTest{Value: "abcdef"}
	group := "testgroup"
	key := "testkey"
	err := c.Cache.
		Set().
		Group(group).
		Key(key).
		Data(data).
		Save(context.Background())
	require.NoError(t, err)

	// Get the data
	fromCache, err := c.Cache.
		Get().
		Group(group).
		Key(key).
		Type(new(cacheTest)).
		Fetch(context.Background())
	require.NoError(t, err)
	cast, ok := fromCache.(*cacheTest)
	require.True(t, ok)
	assert.Equal(t, data, *cast)

	// The same key with the wrong group should fail
	_, err = c.Cache.
		Get().
		Key(key).
		Type(new(cacheTest)).
		Fetch(context.Background())
	assert.Error(t, err)

	// Flush the data
	err = c.Cache.
		Flush().
		Group(group).
		Key(key).
		Execute(context.Background())
	require.NoError(t, err)

	// The data should be gone
	assertFlushed := func() {
		require.Eventually(t, func() bool {
			_, err = c.Cache.
				Get().
				Group(group).
				Key(key).
				Type(new(cacheTest)).
				Fetch(context.Background())
			return errors.Is(err, redis.Nil)
		}, 2*time.Second, 20*time.Millisecond)
	}
	assertFlushed()

	if c.Cache.otter != nil {
		// Set with tags
		err = c.Cache.
			Set().
			Group(group).
			Key(key).
			Data(data).
			Tags("tag1").
			Save(context.Background())
		require.NoError(t, err)

		// Flush the tag
		err = c.Cache.
			Flush().
			Tags("tag1").
			Execute(context.Background())
		require.NoError(t, err)

		// The data should be gone
		assertFlushed()
	}

	// Set with expiration
	err = c.Cache.
		Set().
		Group(group).
		Key(key).
		Data(data).
		Expiration(time.Second).
		Save(context.Background())
	require.NoError(t, err)

	// Wait for expiration
	time.Sleep(1200 * time.Millisecond)

	// The data should be gone
	assertFlushed()
}
