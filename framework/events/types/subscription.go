package types

import "time"

type SubscriptionCreated struct {
	UserID int64
	Plan   string
	At     time.Time
}

type SubscriptionCancelled struct {
	UserID int64
	At     time.Time
}
