package cache

import (
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/maypok86/otter"
)

const (
	defaultOtterCapacity = 10_000
	persistentTTL        = 10 * 365 * 24 * time.Hour
)

type OtterStore struct {
	cache otter.CacheWithVariableTTL[string, []byte]

	mu      sync.RWMutex
	tags    map[string]map[string]struct{}
	keyTags map[string]map[string]struct{}
}

func NewOtterStore(capacity int) (*OtterStore, error) {
	if capacity < 1 {
		capacity = defaultOtterCapacity
	}

	store := &OtterStore{
		tags:    make(map[string]map[string]struct{}),
		keyTags: make(map[string]map[string]struct{}),
	}

	cache, err := otter.MustBuilder[string, []byte](capacity).
		CollectStats().
		DeletionListener(func(key string, _ []byte, _ otter.DeletionCause) {
			store.clearTagsForKey(key)
		}).
		WithVariableTTL().
		Build()
	if err != nil {
		return nil, err
	}

	store.cache = cache
	return store, nil
}

func (s *OtterStore) Get(key string) ([]byte, bool) {
	if s == nil {
		return nil, false
	}
	value, ok := s.cache.Get(key)
	if !ok {
		return nil, false
	}
	return append([]byte(nil), value...), true
}

func (s *OtterStore) Set(key string, value []byte, ttl time.Duration) error {
	if s == nil {
		return errors.New("otter cache is not initialized")
	}
	if ttl <= 0 {
		ttl = persistentTTL
	}
	if ok := s.cache.Set(key, append([]byte(nil), value...), ttl); !ok {
		return errors.New("otter cache rejected set")
	}
	return nil
}

func (s *OtterStore) Delete(key string) error {
	if s == nil {
		return errors.New("otter cache is not initialized")
	}
	s.cache.Delete(key)
	s.clearTagsForKey(key)
	return nil
}

func (s *OtterStore) InvalidatePrefix(prefix string) error {
	if s == nil {
		return errors.New("otter cache is not initialized")
	}
	keys := make([]string, 0)
	s.cache.Range(func(key string, _ []byte) bool {
		if strings.HasPrefix(key, prefix) {
			keys = append(keys, key)
		}
		return true
	})
	for _, key := range keys {
		s.cache.Delete(key)
		s.clearTagsForKey(key)
	}
	return nil
}

func (s *OtterStore) SetTags(key string, tags []string) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	s.clearTagsForKeyLocked(key)
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			continue
		}
		if _, ok := s.tags[tag]; !ok {
			s.tags[tag] = make(map[string]struct{})
		}
		s.tags[tag][key] = struct{}{}
		if _, ok := s.keyTags[key]; !ok {
			s.keyTags[key] = make(map[string]struct{})
		}
		s.keyTags[key][tag] = struct{}{}
	}
}

func (s *OtterStore) InvalidateTags(tags []string) error {
	if s == nil {
		return errors.New("otter cache is not initialized")
	}
	keys := make([]string, 0)
	seen := make(map[string]struct{})

	s.mu.RLock()
	for _, tag := range tags {
		for key := range s.tags[strings.TrimSpace(tag)] {
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			keys = append(keys, key)
		}
	}
	s.mu.RUnlock()

	for _, key := range keys {
		s.cache.Delete(key)
		s.clearTagsForKey(key)
	}
	return nil
}

func (s *OtterStore) Close() error {
	if s == nil {
		return nil
	}
	s.cache.Close()
	return nil
}

func (s *OtterStore) clearTagsForKey(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.clearTagsForKeyLocked(key)
}

func (s *OtterStore) clearTagsForKeyLocked(key string) {
	for tag := range s.keyTags[key] {
		if keys, ok := s.tags[tag]; ok {
			delete(keys, key)
			if len(keys) == 0 {
				delete(s.tags, tag)
			}
		}
	}
	delete(s.keyTags, key)
}
