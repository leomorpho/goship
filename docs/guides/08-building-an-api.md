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
