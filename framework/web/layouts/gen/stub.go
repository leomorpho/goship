package gen

import (
	"context"
	"io"

	"github.com/a-h/templ"
	"github.com/leomorpho/goship/framework/web/ui"
)

func stubComponent() templ.Component {
	return templ.ComponentFunc(func(context.Context, io.Writer) error {
		return nil
	})
}

func Main(content templ.Component, page *ui.Page) templ.Component {
	_ = content
	_ = page
	return stubComponent()
}

func Auth(content templ.Component, page *ui.Page) templ.Component {
	_ = content
	_ = page
	return stubComponent()
}

func Doctype(content templ.Component, page *ui.Page) templ.Component {
	_ = content
	_ = page
	return stubComponent()
}

func LandingPage(content templ.Component, page *ui.Page) templ.Component {
	_ = content
	_ = page
	return stubComponent()
}

func Email(content templ.Component) templ.Component {
	_ = content
	return stubComponent()
}
