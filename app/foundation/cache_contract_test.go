package foundation

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/leomorpho/goship/config"
)

func TestCacheClient_GroupTagAndTTLContract_RedSpec(t *testing.T) {
	t.Skip("red spec: enable once the cache contract harness covers both memory and redis adapters")

	type cacheValue struct {
		Value string
	}

	for _, adapter := range []string{"memory", "redis"} {
		t.Run(adapter, func(t *testing.T) {
			client := newContractCacheClient(t, adapter)
			t.Cleanup(func() { _ = client.Close() })

			ctx := context.Background()
			group := "pages"
			key := "landing"
			want := cacheValue{Value: adapter}

			if err := client.Set().
				Group(group).
				Key(key).
				Data(want).
				Tags("marketing").
				Save(ctx); err != nil {
				t.Fatalf("save grouped value: %v", err)
			}

			got, err := client.Get().
				Group(group).
				Key(key).
				Type(new(cacheValue)).
				Fetch(ctx)
			if err != nil {
				t.Fatalf("fetch grouped value: %v", err)
			}

			cached, ok := got.(*cacheValue)
			if !ok {
				t.Fatalf("fetch type = %T, want *cacheValue", got)
			}
			if *cached != want {
				t.Fatalf("fetch value = %+v, want %+v", *cached, want)
			}

			if err := client.Flush().Tags("marketing").Execute(ctx); err != nil {
				t.Fatalf("flush tag: %v", err)
			}

			_, err = client.Get().
				Group(group).
				Key(key).
				Type(new(cacheValue)).
				Fetch(ctx)
			if !errors.Is(err, redis.Nil) {
				t.Fatalf("fetch after tag flush error = %v, want redis.Nil", err)
			}

			if err := client.Set().
				Group(group).
				Key(key).
				Data(want).
				Expiration(50 * time.Millisecond).
				Save(ctx); err != nil {
				t.Fatalf("save expiring value: %v", err)
			}

			time.Sleep(120 * time.Millisecond)

			_, err = client.Get().
				Group(group).
				Key(key).
				Type(new(cacheValue)).
				Fetch(ctx)
			if !errors.Is(err, redis.Nil) {
				t.Fatalf("fetch after ttl error = %v, want redis.Nil", err)
			}
		})
	}
}

func TestCacheClient_RawBytePrefixContract_RedSpec(t *testing.T) {
	t.Skip("red spec: enable once the cache contract harness covers both memory and redis adapters")

	for _, adapter := range []string{"memory", "redis"} {
		t.Run(adapter, func(t *testing.T) {
			client := newContractCacheClient(t, adapter)
			t.Cleanup(func() { _ = client.Close() })

			ctx := context.Background()

			if err := client.SetBytes(ctx, "pages::home", []byte("home"), time.Minute); err != nil {
				t.Fatalf("set pages::home: %v", err)
			}
			if err := client.SetBytes(ctx, "pages::about", []byte("about"), time.Minute); err != nil {
				t.Fatalf("set pages::about: %v", err)
			}
			if err := client.SetBytes(ctx, "profiles::leo", []byte("leo"), time.Minute); err != nil {
				t.Fatalf("set profiles::leo: %v", err)
			}

			got, found, err := client.GetBytes(ctx, "pages::home")
			if err != nil {
				t.Fatalf("get pages::home: %v", err)
			}
			if !found {
				t.Fatal("expected pages::home to exist")
			}
			if string(got) != "home" {
				t.Fatalf("get pages::home = %q, want home", string(got))
			}

			if err := client.InvalidatePrefix(ctx, "pages::"); err != nil {
				t.Fatalf("invalidate prefix: %v", err)
			}

			if _, found, err := client.GetBytes(ctx, "pages::home"); err != nil || found {
				t.Fatalf("pages::home after invalidate = (found=%v, err=%v), want false,nil", found, err)
			}
			if _, found, err := client.GetBytes(ctx, "pages::about"); err != nil || found {
				t.Fatalf("pages::about after invalidate = (found=%v, err=%v), want false,nil", found, err)
			}

			got, found, err = client.GetBytes(ctx, "profiles::leo")
			if err != nil {
				t.Fatalf("get profiles::leo: %v", err)
			}
			if !found {
				t.Fatal("expected profiles::leo to remain")
			}
			if string(got) != "leo" {
				t.Fatalf("get profiles::leo = %q, want leo", string(got))
			}
		})
	}
}

func newContractCacheClient(t *testing.T, adapter string) *CacheClient {
	t.Helper()

	cfg := &config.Config{}
	cfg.App.Environment = config.EnvLocal
	cfg.Adapters.Cache = adapter
	cfg.Cache.Hostname = "127.0.0.1"
	cfg.Cache.Port = 6379

	if adapter == "redis" {
		t.Fatal("TODO: wire redis contract harness for cache parity tests")
	}

	client, err := NewCacheClient(cfg)
	if err != nil {
		t.Fatalf("new cache client for adapter %q: %v", adapter, err)
	}
	return client
}
