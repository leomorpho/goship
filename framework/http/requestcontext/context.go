package requestcontext

import (
	"context"
	"errors"
)

const (
	FormKey     = "form"
	IsFromIOSApp = "is_from_ios_app"
)

func IsCanceledError(err error) bool {
	return errors.Is(err, context.Canceled)
}
