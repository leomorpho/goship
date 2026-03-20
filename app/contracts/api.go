package contracts

import frameworkapi "github.com/leomorpho/goship/framework/api"

type APIResponse[T any] = frameworkapi.Response[T]
type APIMeta = frameworkapi.Meta
type APIError = frameworkapi.APIError
