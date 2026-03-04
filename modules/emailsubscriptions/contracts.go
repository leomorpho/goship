package emailsubscriptions

import "context"

// Store defines the DB boundary for the email subscriptions module.
// Different apps can bind this to their own implementation.
type Store interface {
	CreateList(ctx context.Context, emailList List) error
	Subscribe(ctx context.Context, email string, emailList List, latitude, longitude *float64) (*Subscription, error)
	Unsubscribe(ctx context.Context, email string, token string, emailList List) error
	Confirm(ctx context.Context, code string) error
}
