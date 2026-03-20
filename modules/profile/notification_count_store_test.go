package profiles

import (
	"context"
	"errors"
	"testing"
)

type fakeNotificationCountStore struct {
	count int
	err   error
}

func (f fakeNotificationCountStore) CountUnseenNotifications(context.Context, int) (int, error) {
	if f.err != nil {
		return 0, f.err
	}
	return f.count, nil
}

func TestProfileService_GetCountOfUnseenNotifications_UsesStore(t *testing.T) {
	svc := NewProfileServiceWithDBDeps(nil, "", nil, nil, fakeNotificationCountStore{count: 7})
	got, err := svc.GetCountOfUnseenNotifications(context.Background(), 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != 7 {
		t.Fatalf("count = %d, want 7", got)
	}
}

func TestProfileService_GetCountOfUnseenNotifications_PropagatesStoreError(t *testing.T) {
	wantErr := errors.New("boom")
	svc := NewProfileServiceWithDBDeps(nil, "", nil, nil, fakeNotificationCountStore{err: wantErr})
	_, err := svc.GetCountOfUnseenNotifications(context.Background(), 42)
	if !errors.Is(err, wantErr) {
		t.Fatalf("error = %v, want %v", err, wantErr)
	}
}

func TestProfileService_GetCountOfUnseenNotifications_RequiresStore(t *testing.T) {
	var svc *ProfileService
	_, err := svc.GetCountOfUnseenNotifications(context.Background(), 1)
	if !errors.Is(err, ErrNotificationCountStoreNotConfigured) {
		t.Fatalf("error = %v, want %v", err, ErrNotificationCountStoreNotConfigured)
	}
}
