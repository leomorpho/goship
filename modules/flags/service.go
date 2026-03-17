package flags

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"time"

	"github.com/leomorpho/goship/framework/core"
)

const cacheTTL = 5 * time.Minute

type Service struct {
	store Store
	cache core.Cache
}

func NewService(store Store, cache core.Cache) *Service {
	return &Service{store: store, cache: cache}
}

func (s *Service) Enabled(ctx context.Context, key string, userID ...int64) (bool, error) {
	flag, err := s.lookup(ctx, key)
	if err != nil {
		return false, err
	}
	if !flag.Enabled {
		return false, nil
	}
	if len(userID) == 0 {
		return flag.RolloutPct >= 100, nil
	}
	if flag.IsUserTargeted(userID[0]) {
		return true, nil
	}
	return inRollout(key, userID[0], flag.RolloutPct), nil
}

func (s *Service) List(ctx context.Context) ([]Flag, error) {
	if s == nil || s.store == nil {
		return nil, fmt.Errorf("flag store unavailable")
	}
	return s.store.List(ctx)
}

func (s *Service) Toggle(ctx context.Context, key string) (Flag, error) {
	if s == nil || s.store == nil {
		return Flag{}, fmt.Errorf("flag store unavailable")
	}

	flag, err := s.lookup(ctx, key)
	if err != nil {
		return Flag{}, err
	}
	flag.Enabled = !flag.Enabled
	if err := s.store.Update(ctx, flag); err != nil {
		return Flag{}, err
	}
	if err := s.deleteCache(ctx, key); err != nil {
		return Flag{}, err
	}
	return flag, nil
}

func (s *Service) Create(ctx context.Context, flag Flag) error {
	if err := s.store.Create(ctx, flag); err != nil {
		return err
	}
	return s.deleteCache(ctx, flag.Key)
}

func (s *Service) Update(ctx context.Context, flag Flag) error {
	if err := s.store.Update(ctx, flag); err != nil {
		return err
	}
	return s.deleteCache(ctx, flag.Key)
}

func (s *Service) Delete(ctx context.Context, key string) error {
	if err := s.store.Delete(ctx, key); err != nil {
		return err
	}
	return s.deleteCache(ctx, key)
}

func (s *Service) lookup(ctx context.Context, key string) (Flag, error) {
	if s == nil || s.store == nil {
		return Flag{}, fmt.Errorf("flag store unavailable")
	}
	if s.cache != nil {
		if payload, found, err := s.cache.Get(ctx, cacheKey(key)); err == nil && found {
			var flag Flag
			if err := json.Unmarshal(payload, &flag); err == nil {
				return flag, nil
			}
		}
	}
	flag, err := s.store.Find(ctx, key)
	if err != nil {
		return Flag{}, err
	}
	if s.cache != nil {
		if payload, err := json.Marshal(flag); err == nil {
			_ = s.cache.Set(ctx, cacheKey(key), payload, cacheTTL)
		}
	}
	return flag, nil
}

func (s *Service) deleteCache(ctx context.Context, key string) error {
	if s.cache == nil {
		return nil
	}
	return s.cache.Delete(ctx, cacheKey(key))
}

func cacheKey(key string) string {
	return "flags::" + key
}

func inRollout(key string, userID int64, rolloutPct int) bool {
	if rolloutPct <= 0 {
		return false
	}
	if rolloutPct >= 100 {
		return true
	}
	h := fnv.New32a()
	_, _ = h.Write([]byte(fmt.Sprintf("%s:%d", key, userID)))
	return int(h.Sum32()%100) < rolloutPct
}
