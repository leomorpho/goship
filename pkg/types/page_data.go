package types

import "github.com/mikestefanello/pagoda/ent"

type PageData struct {
	IsAuth   bool
	AuthUser *ent.User
	Data     any
	ToURL    func(name string, params ...any) string
}
