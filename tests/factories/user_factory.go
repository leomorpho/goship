package factories

import (
	"time"

	"github.com/leomorpho/goship/v2/tests/factory"
)

type UserRecord struct {
	ID        int64     `db:"id"`
	Name      string    `db:"name"`
	Email     string    `db:"email"`
	Password  string    `db:"password"`
	Role      string    `db:"role"`
	CreatedAt time.Time `db:"created_at"`
}

func (UserRecord) TableName() string { return "users" }

var User = factory.New(func() UserRecord {
	return UserRecord{
		Name:      "Test User",
		Email:     factory.Sequence("user") + "@example.com",
		Password:  "$2a$10$CwTycUXWue0Thq9StjUM0uJ8cD1f5f6m7h8i9j0k1l2m3n4o5p6q",
		Role:      "member",
		CreatedAt: time.Now().UTC(),
	}
})

func WithAdminRole(u *UserRecord) { u.Role = "admin" }

func WithEmail(email string) func(*UserRecord) {
	return func(u *UserRecord) {
		u.Email = email
	}
}
