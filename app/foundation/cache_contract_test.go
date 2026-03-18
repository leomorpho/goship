package foundation

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"github.com/leomorpho/goship/config"
)

func TestCacheClient_GroupTagAndTTLContract(t *testing.T) {
	type cacheValue struct {
		Value string
	}

	for _, adapter := range []string{"memory", "redis"} {
		t.Run(adapter, func(t *testing.T) {
			harness := newContractCacheHarness(t, adapter)
			t.Cleanup(func() { _ = harness.client.Close() })

			ctx := context.Background()
			group := "pages"
			key := "landing"
			want := cacheValue{Value: adapter}

			if err := harness.client.Set().
				Group(group).
				Key(key).
				Data(want).
				Tags("marketing").
				Save(ctx); err != nil {
				t.Fatalf("save grouped value: %v", err)
			}

			got, err := harness.client.Get().
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

			if err := harness.client.Flush().Tags("marketing").Execute(ctx); err != nil {
				t.Fatalf("flush tag: %v", err)
			}

			_, err = harness.client.Get().
				Group(group).
				Key(key).
				Type(new(cacheValue)).
				Fetch(ctx)
			if !errors.Is(err, redis.Nil) {
				t.Fatalf("fetch after tag flush error = %v, want redis.Nil", err)
			}

			ttl := 50 * time.Millisecond

			if err := harness.client.Set().
				Group(group).
				Key(key).
				Data(want).
				Expiration(ttl).
				Save(ctx); err != nil {
				t.Fatalf("save expiring value: %v", err)
			}

			// The shared cache seam normalizes positive TTLs to second precision so
			// memory and redis-backed adapters expire on the same schedule.
			harness.advanceTTL(250 * time.Millisecond)

			got, err = harness.client.Get().
				Group(group).
				Key(key).
				Type(new(cacheValue)).
				Fetch(ctx)
			if err != nil {
				t.Fatalf("fetch before normalized ttl expiry: %v", err)
			}
			cached, ok = got.(*cacheValue)
			if !ok {
				t.Fatalf("fetch type before normalized ttl expiry = %T, want *cacheValue", got)
			}
			if *cached != want {
				t.Fatalf("fetch value before normalized ttl expiry = %+v, want %+v", *cached, want)
			}

			harness.advanceTTL(950 * time.Millisecond)

			_, err = harness.client.Get().
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

func TestCacheClient_RawBytePrefixContract(t *testing.T) {
	for _, adapter := range []string{"memory", "redis"} {
		t.Run(adapter, func(t *testing.T) {
			harness := newContractCacheHarness(t, adapter)
			t.Cleanup(func() { _ = harness.client.Close() })

			ctx := context.Background()

			if err := harness.client.SetBytes(ctx, "pages::home", []byte("home"), time.Minute); err != nil {
				t.Fatalf("set pages::home: %v", err)
			}
			if err := harness.client.SetBytes(ctx, "pages::about", []byte("about"), time.Minute); err != nil {
				t.Fatalf("set pages::about: %v", err)
			}
			if err := harness.client.SetBytes(ctx, "profiles::leo", []byte("leo"), time.Minute); err != nil {
				t.Fatalf("set profiles::leo: %v", err)
			}

			got, found, err := harness.client.GetBytes(ctx, "pages::home")
			if err != nil {
				t.Fatalf("get pages::home: %v", err)
			}
			if !found {
				t.Fatal("expected pages::home to exist")
			}
			if string(got) != "home" {
				t.Fatalf("get pages::home = %q, want home", string(got))
			}

			if err := harness.client.InvalidatePrefix(ctx, "pages::"); err != nil {
				t.Fatalf("invalidate prefix: %v", err)
			}

			if _, found, err := harness.client.GetBytes(ctx, "pages::home"); err != nil || found {
				t.Fatalf("pages::home after invalidate = (found=%v, err=%v), want false,nil", found, err)
			}
			if _, found, err := harness.client.GetBytes(ctx, "pages::about"); err != nil || found {
				t.Fatalf("pages::about after invalidate = (found=%v, err=%v), want false,nil", found, err)
			}

			got, found, err = harness.client.GetBytes(ctx, "profiles::leo")
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

type contractCacheHarness struct {
	client     *CacheClient
	advanceTTL func(time.Duration)
}

func newContractCacheHarness(t *testing.T, adapter string) contractCacheHarness {
	t.Helper()

	cfg := &config.Config{}
	cfg.App.Environment = config.EnvTest
	cfg.Adapters.Cache = adapter
	cfg.Cache.TestDatabase = 1

	advanceTTL := time.Sleep
	if adapter == "redis" {
		server := miniredis.RunT(t)
		host, port := splitRedisAddr(t, server.Addr())
		cfg.Cache.Hostname = host
		cfg.Cache.Port = uint16(port)
		advanceTTL = server.FastForward
	} else {
		cfg.Cache.Hostname = "127.0.0.1"
		cfg.Cache.Port = 6379
	}

	client, err := NewCacheClient(cfg)
	if err != nil {
		t.Fatalf("new cache client for adapter %q: %v", adapter, err)
	}
	return contractCacheHarness{
		client:     client,
		advanceTTL: advanceTTL,
	}
}

func splitRedisAddr(t *testing.T, addr string) (string, int) {
	t.Helper()

	host, portText, ok := strings.Cut(addr, ":")
	if !ok {
		t.Fatalf("split redis addr %q", addr)
	}
	parsedPort, err := parseRedisPort(portText)
	if err != nil {
		t.Fatalf("parse redis port %q: %v", portText, err)
	}
	return host, parsedPort
}

func parseRedisPort(value string) (int, error) {
	port, err := strconv.Atoi(value)
	if err != nil {
		return 0, errors.New("invalid redis port")
	}
	return port, nil
}
