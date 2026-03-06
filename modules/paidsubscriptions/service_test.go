package paidsubscriptions_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/leomorpho/goship-modules/paidsubscriptions"
	"github.com/stretchr/testify/assert"
)

type stubStore struct {
	updateToPaidProErr error
	updateToFreeErr    error
}

func (s *stubStore) CreateSubscription(context.Context, any, int) error {
	return nil
}

func (s *stubStore) DeactivateExpiredSubscriptions(context.Context) error {
	return nil
}

func (s *stubStore) UpdateToPaidPro(context.Context, int) error {
	return s.updateToPaidProErr
}

func (s *stubStore) GetCurrentlyActiveProduct(context.Context, int) (*paidsubscriptions.ProductType, *time.Time, bool, error) {
	return &paidsubscriptions.ProductTypeFree, nil, false, nil
}

func (s *stubStore) StoreStripeCustomerID(context.Context, int, string) error {
	return nil
}

func (s *stubStore) GetStripeCustomerIDByProfileID(context.Context, int) (string, error) {
	return "", nil
}

func (s *stubStore) GetProfileIDFromStripeCustomerID(context.Context, string) (int, error) {
	return 0, nil
}

func (s *stubStore) CancelWithGracePeriod(context.Context, int) error {
	return nil
}

func (s *stubStore) CancelOrRenew(context.Context, int, *time.Time) error {
	return nil
}

func (s *stubStore) UpdateToFree(context.Context, int) error {
	return s.updateToFreeErr
}

func TestServiceForwardsOperations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		store   *stubStore
		call    func(s *paidsubscriptions.Service) error
		wantErr error
	}{
		{
			name: "update paid pro error",
			store: &stubStore{
				updateToPaidProErr: errors.New("boom"),
			},
			call: func(s *paidsubscriptions.Service) error {
				return s.UpdateToPaidPro(context.Background(), 1)
			},
			wantErr: errors.New("boom"),
		},
		{
			name: "update free error",
			store: &stubStore{
				updateToFreeErr: errors.New("fail"),
			},
			call: func(s *paidsubscriptions.Service) error {
				return s.UpdateToFree(context.Background(), 1)
			},
			wantErr: errors.New("fail"),
		},
		{
			name:  "update paid pro success",
			store: &stubStore{},
			call: func(s *paidsubscriptions.Service) error {
				return s.UpdateToPaidPro(context.Background(), 1)
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc := paidsubscriptions.New(tt.store)
			err := tt.call(svc)
			if tt.wantErr == nil {
				assert.NoError(t, err)
				return
			}
			assert.EqualError(t, err, tt.wantErr.Error())
		})
	}
}
