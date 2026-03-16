# Building an API

GoShip supports dual HTML and JSON responses without duplicating handler logic.

## Core Pattern

Use `framework/api` when a handler needs a JSON representation:

```go
func (pc *PostController) Show(c echo.Context) error {
	post, err := pc.store.Find(c.Request().Context(), id)
	if err != nil {
		return api.Fail(c, http.StatusNotFound, api.NotFound("post not found"))
	}

	if api.IsAPIRequest(c) {
		return api.OK(c, contracts.PostResponse{ID: post.ID, Title: post.Title})
	}

	return render(c, views.PostShow(toPostVM(post)))
}
```

`api.IsAPIRequest` returns `true` when:

- the `Accept` header prefers `application/json`
- the route path starts with `/api/`

## Localized API Errors

For API routes that return human-readable error messages, use localized helpers while keeping machine codes stable:

```go
return api.Fail(c, http.StatusUnauthorized, api.UnauthorizedLocalized(
	c.Request().Context(),
	ctr.Container.I18n,
	"api.errors.unauthorized",
	"Unauthorized",
))
```

Helpers:

- `api.NotFoundLocalized(...)` -> code `not_found`
- `api.UnauthorizedLocalized(...)` -> code `unauthorized`
- `api.ValidationLocalized(...)` -> code `validation_error`

Locale resolution for API requests is provided by the shared i18n middleware (`modules/i18n.DetectLanguage`) in this order:

1. `?lang=<code>` query parameter
2. profile preference (if authenticated)
3. `lang` cookie
4. `Accept-Language` header
5. i18n default language

## Response Envelope

Successful responses use:

```json
{
  "data": {},
  "meta": {
    "page": 1,
    "per_page": 20,
    "total": 42
  }
}
```

Errors use:

```json
{
  "data": {},
  "errors": [
    {
      "field": "title",
      "message": "is required",
      "code": "validation_error"
    }
  ]
}
```

## Router Convention

Reserve versioned JSON routes under:

```go
v1 := e.Group("/api/v1") // ship:routes:api:v1:start
// ship:routes:api:v1:end
```

Keep JSON-specific routes inside that marker block so generators and doctor checks have one canonical place to inspect.
