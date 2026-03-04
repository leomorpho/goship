package emailsubscriptions

// List identifies a subscription list (e.g. newsletter).
type List string

// Subscription is the module-owned subscription model.
type Subscription struct {
	ID               int
	Email            string
	Verified         bool
	ConfirmationCode string
	Lat              float64
	Lon              float64
}
