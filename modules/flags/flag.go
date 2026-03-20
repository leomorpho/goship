package flags

import "slices"

type Flag struct {
	Key         string
	Enabled     bool
	RolloutPct  int
	UserIDs     []int64
	Description string
}

func (f Flag) IsUserTargeted(userID int64) bool {
	return slices.Contains(f.UserIDs, userID)
}
