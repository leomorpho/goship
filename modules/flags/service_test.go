package flags

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/leomorpho/goship/framework/core"
)

type fakeStore struct {
	flag      Flag
	findCalls int
}

func (f *fakeStore) Find(context.Context, string) (Flag, error) {
	f.findCalls++
	return f.flag, nil
}
func (f *fakeStore) List(context.Context) ([]Flag, error) { return []Flag{f.flag}, nil }
func (f *fakeStore) Create(context.Context, Flag) error   { return nil }
func (f *fakeStore) Update(context.Context, Flag) error   { return nil }
func (f *fakeStore) Delete(context.Context, string) error { return nil }

type fakeCache struct {
	values map[string][]byte
	gets   int
	sets   int
}

func (f *fakeCache) Get(_ context.Context, key string) ([]byte, bool, error) {
	f.gets++
	v, ok := f.values[key]
	return v, ok, nil
}
func (f *fakeCache) Set(_ context.Context, key string, value []byte, _ time.Duration) error {
	f.sets++
	if f.values == nil {
		f.values = map[string][]byte{}
	}
	f.values[key] = append([]byte(nil), value...)
	return nil
}
func (f *fakeCache) Delete(_ context.Context, key string) error {
	delete(f.values, key)
	return nil
}
func (f *fakeCache) InvalidatePrefix(context.Context, string) error { return nil }
func (f *fakeCache) Close() error                                   { return nil }

var _ core.Cache = (*fakeCache)(nil)

const (
	testMyFlagKey     FlagKey = "my_flag"
	testCachedFlagKey FlagKey = "cached_flag"
)

func TestServiceEnabled_UsesCacheAfterFirstLookup(t *testing.T) {
	store := &fakeStore{flag: Flag{Key: string(testMyFlagKey), Enabled: true, RolloutPct: 100}}
	cache := &fakeCache{values: map[string][]byte{}}
	service := NewService(store, cache)

	enabled, err := service.Enabled(context.Background(), testMyFlagKey)
	if err != nil {
		t.Fatalf("Enabled() error = %v", err)
	}
	if !enabled {
		t.Fatal("expected flag to be enabled")
	}

	enabled, err = service.Enabled(context.Background(), testMyFlagKey)
	if err != nil {
		t.Fatalf("Enabled() second call error = %v", err)
	}
	if !enabled {
		t.Fatal("expected cached flag to be enabled")
	}

	if store.findCalls != 1 {
		t.Fatalf("findCalls = %d, want 1", store.findCalls)
	}
	if cache.sets != 1 {
		t.Fatalf("cache sets = %d, want 1", cache.sets)
	}
}

func TestServiceEnabled_TargetingAndRolloutAreDeterministic(t *testing.T) {
	store := &fakeStore{flag: Flag{
		Key:        string(testMyFlagKey),
		Enabled:    true,
		RolloutPct: 25,
		UserIDs:    []int64{99},
	}}
	service := NewService(store, nil)

	targeted, err := service.Enabled(context.Background(), testMyFlagKey, 99)
	if err != nil {
		t.Fatalf("Enabled(targeted) error = %v", err)
	}
	if !targeted {
		t.Fatal("expected targeted user to be enabled")
	}

	first, err := service.Enabled(context.Background(), testMyFlagKey, 42)
	if err != nil {
		t.Fatalf("Enabled(first) error = %v", err)
	}
	second, err := service.Enabled(context.Background(), testMyFlagKey, 42)
	if err != nil {
		t.Fatalf("Enabled(second) error = %v", err)
	}
	if first != second {
		t.Fatalf("expected deterministic rollout, got %v then %v", first, second)
	}
}

func TestServiceEnabled_UsesCachedFlagPayload(t *testing.T) {
	flag := Flag{Key: string(testCachedFlagKey), Enabled: true, RolloutPct: 100}
	payload, err := json.Marshal(flag)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	store := &fakeStore{flag: Flag{Key: string(testCachedFlagKey), Enabled: false}}
	cache := &fakeCache{values: map[string][]byte{cacheKey(testCachedFlagKey): payload}}
	service := NewService(store, cache)

	enabled, err := service.Enabled(context.Background(), testCachedFlagKey)
	if err != nil {
		t.Fatalf("Enabled() error = %v", err)
	}
	if !enabled {
		t.Fatal("expected cached flag to be enabled")
	}
	if store.findCalls != 0 {
		t.Fatalf("findCalls = %d, want 0", store.findCalls)
	}
}
