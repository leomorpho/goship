package emailsubscriptions

import (
	"context"
	"errors"
	"testing"
)

type mockStore struct {
	createErr      error
	subscribeErr   error
	unsubscribeErr error
	confirmErr     error

	subscribeResult *Subscription
	createCalls     []List
}

func (m *mockStore) CreateList(_ context.Context, list List) error {
	m.createCalls = append(m.createCalls, list)
	return m.createErr
}

func (m *mockStore) Subscribe(context.Context, string, List, *float64, *float64) (*Subscription, error) {
	if m.subscribeResult == nil {
		m.subscribeResult = &Subscription{ID: 1, ConfirmationCode: "abc"}
	}
	return m.subscribeResult, m.subscribeErr
}

func (m *mockStore) Unsubscribe(context.Context, string, string, List) error { return m.unsubscribeErr }

func (m *mockStore) Confirm(context.Context, string) error { return m.confirmErr }

func TestServiceDelegatesStore(t *testing.T) {
	store := &mockStore{}
	svc := NewServiceWithVerifier(store, func(string) error { return nil })

	if err := svc.CreateList(context.Background(), List("newsletter")); err != nil {
		t.Fatalf("CreateList err: %v", err)
	}
	if _, err := svc.Subscribe(context.Background(), "a@b.com", List("newsletter"), nil, nil); err != nil {
		t.Fatalf("Subscribe err: %v", err)
	}
	if err := svc.Unsubscribe(context.Background(), "a@b.com", "tok", List("newsletter")); err != nil {
		t.Fatalf("Unsubscribe err: %v", err)
	}
	if err := svc.Confirm(context.Background(), "tok"); err != nil {
		t.Fatalf("Confirm err: %v", err)
	}
}

func TestServiceSubscribeVerificationFailure(t *testing.T) {
	svc := NewServiceWithVerifier(&mockStore{}, func(string) error { return errors.New("invalid") })
	if _, err := svc.Subscribe(context.Background(), "bad", List("newsletter"), nil, nil); !errors.Is(err, ErrEmailAddressInvalidCatchAll) {
		t.Fatalf("err = %v, want %v", err, ErrEmailAddressInvalidCatchAll)
	}
}

func TestService_StrictCatalogValidation(t *testing.T) {
	catalog, err := NewStaticListCatalog([]ListSpec{
		{Key: List("newsletter"), Active: true},
		{Key: List("legacy"), Active: false},
	})
	if err != nil {
		t.Fatalf("catalog err: %v", err)
	}

	store := &mockStore{}
	svc := NewServiceWithOptions(store, Options{
		VerifyFunc:        func(string) error { return nil },
		ListCatalog:       catalog,
		StrictListCatalog: true,
	})

	if _, err := svc.Subscribe(context.Background(), "a@b.com", List("unknown"), nil, nil); !errors.Is(err, ErrListNotAllowed) {
		t.Fatalf("err = %v, want %v", err, ErrListNotAllowed)
	}
	if _, err := svc.Subscribe(context.Background(), "a@b.com", List("legacy"), nil, nil); !errors.Is(err, ErrListInactive) {
		t.Fatalf("err = %v, want %v", err, ErrListInactive)
	}
	if _, err := svc.Subscribe(context.Background(), "a@b.com", List("newsletter"), nil, nil); err != nil {
		t.Fatalf("Subscribe should succeed, err: %v", err)
	}
}

func TestService_EnsureCatalog(t *testing.T) {
	catalog, err := NewStaticListCatalog([]ListSpec{
		{Key: List("newsletter"), Active: true},
		{Key: List("product-updates"), Active: true},
	})
	if err != nil {
		t.Fatalf("catalog err: %v", err)
	}
	store := &mockStore{}
	svc := NewServiceWithOptions(store, Options{
		VerifyFunc:  func(string) error { return nil },
		ListCatalog: catalog,
	})
	if err := svc.EnsureCatalog(context.Background()); err != nil {
		t.Fatalf("EnsureCatalog err: %v", err)
	}
	if len(store.createCalls) != 2 {
		t.Fatalf("expected 2 create calls, got %d", len(store.createCalls))
	}
}
