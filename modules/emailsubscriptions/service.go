package emailsubscriptions

import (
	"context"

	emailverifier "github.com/AfterShip/email-verifier"
)

// Service is the public API for the email subscriptions module.
type Service struct {
	store      Store
	verifyFunc func(email string) error
}

func NewService(store Store) *Service {
	return NewServiceWithVerifier(store, verifyWithAfterShip())
}

func NewServiceWithVerifier(store Store, verifyFunc func(email string) error) *Service {
	return &Service{store: store, verifyFunc: verifyFunc}
}

func verifyWithAfterShip() func(email string) error {
	verifier := emailverifier.NewVerifier()
	return func(email string) error {
		ret, err := verifier.Verify(email)
		if err != nil {
			return &ErrEmailVerificationFailed{Err: err}
		}
		if !ret.Syntax.Valid {
			return ErrEmailSyntaxInvalid
		}
		return nil
	}
}

// VerifyEmailValidity confirms an email is valid according to our rules.
func (s *Service) VerifyEmailValidity(email string) error {
	if s.verifyFunc == nil {
		return nil
	}
	return s.verifyFunc(email)
}

// CreateList creates a new subscription list.
func (s *Service) CreateList(ctx context.Context, emailList List) error {
	return s.store.CreateList(ctx, emailList)
}

// Subscribe subscribes an email to a mailing list.
func (s *Service) Subscribe(
	ctx context.Context, email string, emailList List, latitude, longitude *float64,
) (*Subscription, error) {
	if err := s.VerifyEmailValidity(email); err != nil {
		return nil, ErrEmailAddressInvalidCatchAll
	}
	return s.store.Subscribe(ctx, email, emailList, latitude, longitude)
}

func (s *Service) Unsubscribe(ctx context.Context, email string, token string, emailList List) error {
	return s.store.Unsubscribe(ctx, email, token, emailList)
}

func (s *Service) Confirm(ctx context.Context, code string) error {
	return s.store.Confirm(ctx, code)
}
