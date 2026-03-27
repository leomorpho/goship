package gen

import (
	"context"
	"io"

	"github.com/a-h/templ"
	"github.com/leomorpho/goship/framework/http/ui"
)

func stubComponent() templ.Component {
	return templ.ComponentFunc(func(context.Context, io.Writer) error {
		return nil
	})
}

func Main(templ.Component, *ui.Page) templ.Component {
	return stubComponent()
}
