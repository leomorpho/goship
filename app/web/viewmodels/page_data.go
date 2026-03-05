package viewmodels

import "github.com/leomorpho/goship/app/web/ui"

type PageData struct {
	IsAuth   bool
	AuthUser *ui.AuthUserView
	Data     any
	ToURL    func(name string, params ...any) string
}
