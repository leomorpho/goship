package emailsubscriptions

import (
	"context"

	emailverifier "github.com/AfterShip/email-verifier"
)

// Service is the public API for the email subscriptions module.
type Service struct {
	store      Store
	verifyFunc func(email string) error
	catalog    ListCatalog
	strict     bool
}

func NewService(store Store) *Service {
	return NewServiceWithOptions(store, Options{
		VerifyFunc: verifyWithAfterShip(),
	})
}

func NewServiceWithVerifier(store Store, verifyFunc func(email string) error) *Service {
	return NewServiceWithOptions(store, Options{
		VerifyFunc: verifyFunc,
	})
}

type Options struct {
	VerifyFunc        func(email string) error
	ListCatalog       ListCatalog
	StrictListCatalog bool
}

func NewServiceWithOptions(store Store, opts Options) *Service {
	verifyFunc := opts.VerifyFunc
	if verifyFunc == nil {
		verifyFunc = verifyWithAfterShip()
	}
	return &Service{
		store:      store,
		verifyFunc: verifyFunc,
		catalog:    opts.ListCatalog,
		strict:     opts.StrictListCatalog,
	}
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
	list := NormalizeList(emailList)
	if list == "" {
		return ErrListNotAllowed
	}
	if err := s.validateList(list); err != nil {
		return err
	}
	return s.store.CreateList(ctx, list)
}

// Subscribe subscribes an email to a mailing list.
func (s *Service) Subscribe(
	ctx context.Context, email string, emailList List, latitude, longitude *float64,
) (*Subscription, error) {
	emailList = NormalizeList(emailList)
	if emailList == "" {
		return nil, ErrListNotAllowed
	}
	if err := s.validateList(emailList); err != nil {
		return nil, err
	}
	if err := s.VerifyEmailValidity(email); err != nil {
		return nil, ErrEmailAddressInvalidCatchAll
	}
	return s.store.Subscribe(ctx, email, emailList, latitude, longitude)
}

func (s *Service) Unsubscribe(ctx context.Context, email string, token string, emailList List) error {
	emailList = NormalizeList(emailList)
	if emailList == "" {
		return ErrListNotAllowed
	}
	if err := s.validateList(emailList); err != nil {
		return err
	}
	return s.store.Unsubscribe(ctx, email, token, emailList)
}

func (s *Service) Confirm(ctx context.Context, code string) error {
	return s.store.Confirm(ctx, code)
}

// EnsureCatalog bootstraps known lists into the store for installable module setup.
func (s *Service) EnsureCatalog(ctx context.Context) error {
	if s.catalog == nil {
		return nil
	}
	for _, spec := range s.catalog.Lists() {
		if err := s.store.CreateList(ctx, NormalizeList(spec.Key)); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) validateList(list List) error {
	if s.catalog == nil {
		return nil
	}
	spec, ok := s.catalog.ListByKey(list)
	if !ok {
		if s.strict {
			return ErrListNotAllowed
		}
		return nil
	}
	if !spec.Active {
		return ErrListInactive
	}
	return nil
}
