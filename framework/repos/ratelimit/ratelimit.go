package ratelimit

import (
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/maypok86/otter"
)

const defaultOtterCapacity = 10_000

type Decision struct {
	Allowed    bool
	RetryAfter time.Duration
}

type Store interface {
	Allow(key string, max int, window time.Duration) (Decision, error)
}

type otterEntry struct {
	Count   int
	ResetAt time.Time
}

type OtterStore struct {
	mu    sync.Mutex
	cache otter.CacheWithVariableTTL[string, otterEntry]
}

func NewOtterStore(capacity int) (*OtterStore, error) {
	if capacity < 1 {
		capacity = defaultOtterCapacity
	}
	cache, err := otter.MustBuilder[string, otterEntry](capacity).
		WithVariableTTL().
		Build()
	if err != nil {
		return nil, err
	}
	return &OtterStore{cache: cache}, nil
}

func (s *OtterStore) Allow(key string, max int, window time.Duration) (Decision, error) {
	if s == nil {
		return Decision{}, errors.New("rate limit store is not initialized")
	}
	if strings.TrimSpace(key) == "" {
		return Decision{}, errors.New("rate limit key is empty")
	}
	if max <= 0 || window <= 0 {
		return Decision{Allowed: true}, nil
	}

	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()

	if current, ok := s.cache.Get(key); ok && now.Before(current.ResetAt) {
		if current.Count >= max {
			return Decision{Allowed: false, RetryAfter: current.ResetAt.Sub(now)}, nil
		}
		current.Count++
		ttl := time.Until(current.ResetAt)
		if ttl <= 0 {
			ttl = time.Millisecond
		}
		if ok := s.cache.Set(key, current, ttl); !ok {
			return Decision{}, errors.New("rate limit store rejected set")
		}
		return Decision{Allowed: true}, nil
	}

	next := otterEntry{
		Count:   1,
		ResetAt: now.Add(window),
	}
	if ok := s.cache.Set(key, next, window); !ok {
		return Decision{}, errors.New("rate limit store rejected set")
	}
	return Decision{Allowed: true}, nil
}

func (s *OtterStore) Close() {
	if s == nil {
		return
	}
	s.cache.Close()
}
