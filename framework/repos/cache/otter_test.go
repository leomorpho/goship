package cache

import (
	"testing"
	"time"
)

func TestOtterStoreSetGetDelete(t *testing.T) {
	t.Parallel()

	store, err := NewOtterStore(32)
	if err != nil {
		t.Fatalf("new otter store: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	if err := store.Set("group::key", []byte("value"), time.Minute); err != nil {
		t.Fatalf("set: %v", err)
	}
	got, ok := store.Get("group::key")
	if !ok {
		t.Fatal("expected cached value")
	}
	if string(got) != "value" {
		t.Fatalf("unexpected value: %q", string(got))
	}

	if err := store.Delete("group::key"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, ok := store.Get("group::key"); ok {
		t.Fatal("expected value to be deleted")
	}
}

func TestOtterStoreInvalidateTagsAndPrefix(t *testing.T) {
	t.Parallel()

	store, err := NewOtterStore(32)
	if err != nil {
		t.Fatalf("new otter store: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	if err := store.Set("page::one", []byte("1"), time.Minute); err != nil {
		t.Fatalf("set page::one: %v", err)
	}
	store.SetTags("page::one", []string{"landing"})

	if err := store.Set("page::two", []byte("2"), time.Minute); err != nil {
		t.Fatalf("set page::two: %v", err)
	}
	store.SetTags("page::two", []string{"landing"})

	if err := store.Set("profile::one", []byte("3"), time.Minute); err != nil {
		t.Fatalf("set profile::one: %v", err)
	}

	if err := store.InvalidateTags([]string{"landing"}); err != nil {
		t.Fatalf("invalidate tags: %v", err)
	}
	if _, ok := store.Get("page::one"); ok {
		t.Fatal("expected tag invalidation to remove page::one")
	}
	if _, ok := store.Get("page::two"); ok {
		t.Fatal("expected tag invalidation to remove page::two")
	}
	if _, ok := store.Get("profile::one"); !ok {
		t.Fatal("expected untagged key to remain")
	}

	if err := store.Set("profile::two", []byte("4"), time.Minute); err != nil {
		t.Fatalf("set profile::two: %v", err)
	}
	if err := store.InvalidatePrefix("profile::"); err != nil {
		t.Fatalf("invalidate prefix: %v", err)
	}
	if _, ok := store.Get("profile::one"); ok {
		t.Fatal("expected prefix invalidation to remove profile::one")
	}
	if _, ok := store.Get("profile::two"); ok {
		t.Fatal("expected prefix invalidation to remove profile::two")
	}
}

func TestOtterStoreTTLExpiry(t *testing.T) {
	t.Parallel()

	store, err := NewOtterStore(32)
	if err != nil {
		t.Fatalf("new otter store: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	if err := store.Set("ttl::key", []byte("value"), time.Second); err != nil {
		t.Fatalf("set ttl key: %v", err)
	}
	time.Sleep(2 * time.Second)

	if _, ok := store.Get("ttl::key"); ok {
		t.Fatal("expected ttl key to expire")
	}
}
