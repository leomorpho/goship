package gen

import (
	"context"
	"io"

	"github.com/a-h/templ"
)

func stubComponent() templ.Component {
	return templ.ComponentFunc(func(context.Context, io.Writer) error {
		return nil
	})
}

func PrevNavBarWithTitle(...any) templ.Component { return stubComponent() }
func Profile(...any) templ.Component             { return stubComponent() }
func FormCSRF(...any) templ.Component            { return stubComponent() }
func FormFieldErrors(...any) templ.Component     { return stubComponent() }
func AuthButtons(...any) templ.Component         { return stubComponent() }
