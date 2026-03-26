package layouts

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
