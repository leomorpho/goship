package types

import "time"

type UserRegistered struct {
	UserID int64
	Email  string
	At     time.Time
}

type UserLoggedIn struct {
	UserID int64
	IP     string
	At     time.Time
}

type UserLoggedOut struct {
	UserID int64
	At     time.Time
}

type PasswordChanged struct {
	UserID int64
	At     time.Time
}
