package ratelimit

import (
	"testing"
	"time"
)

func TestOtterStoreAllow_EnforcesWindowLimit(t *testing.T) {
	t.Parallel()

	store, err := NewOtterStore(128)
	if err != nil {
		t.Fatalf("NewOtterStore error = %v", err)
	}
	t.Cleanup(store.Close)

	key := "POST:/user/login:ip:127.0.0.1"
	decision, err := store.Allow(key, 2, time.Minute)
	if err != nil {
		t.Fatalf("Allow #1 error = %v", err)
	}
	if !decision.Allowed {
		t.Fatal("Allow #1 denied, want allowed")
	}

	decision, err = store.Allow(key, 2, time.Minute)
	if err != nil {
		t.Fatalf("Allow #2 error = %v", err)
	}
	if !decision.Allowed {
		t.Fatal("Allow #2 denied, want allowed")
	}

	decision, err = store.Allow(key, 2, time.Minute)
	if err != nil {
		t.Fatalf("Allow #3 error = %v", err)
	}
	if decision.Allowed {
		t.Fatal("Allow #3 allowed, want denied")
	}
	if decision.RetryAfter <= 0 {
		t.Fatalf("RetryAfter = %v, want > 0", decision.RetryAfter)
	}
}

func TestOtterStoreAllow_ResetsAfterWindow(t *testing.T) {
	t.Parallel()

	store, err := NewOtterStore(128)
	if err != nil {
		t.Fatalf("NewOtterStore error = %v", err)
	}
	t.Cleanup(store.Close)

	key := "POST:/user/login:user:1"
	window := 30 * time.Millisecond

	decision, err := store.Allow(key, 1, window)
	if err != nil {
		t.Fatalf("Allow #1 error = %v", err)
	}
	if !decision.Allowed {
		t.Fatal("Allow #1 denied, want allowed")
	}

	decision, err = store.Allow(key, 1, window)
	if err != nil {
		t.Fatalf("Allow #2 error = %v", err)
	}
	if decision.Allowed {
		t.Fatal("Allow #2 allowed, want denied")
	}

	time.Sleep(window + 15*time.Millisecond)

	decision, err = store.Allow(key, 1, window)
	if err != nil {
		t.Fatalf("Allow #3 error = %v", err)
	}
	if !decision.Allowed {
		t.Fatal("Allow #3 denied after window, want allowed")
	}
}
