package types

import "time"

type ProfileCompletedOnboarding struct {
	UserID    int64
	ProfileID int64
	At        time.Time
}

type ProfileUpdated struct {
	UserID    int64
	ProfileID int64
	At        time.Time
}
