package viewmodels

import "github.com/leomorpho/goship/db/ent"

type PageData struct {
	IsAuth   bool
	AuthUser *ent.User
	Data     any
	ToURL    func(name string, params ...any) string
}
