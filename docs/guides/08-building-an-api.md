# Building an API

GoShip supports dual HTML and JSON responses without duplicating handler logic.
This guide also defines the one blessed split-frontend integration path.

## Blessed External Frontend Contract

Current blessed split-frontend contract identifier: `api-only-same-origin-sveltekit-v1`.

Canonical reference implementation:

- `examples/sveltekit-api-only/README.md`

Contract scope:

- one supported custom frontend story for now: `SvelteKit-first`
- `same-origin auth/session` is required for browser flows
- keep `cookie/CSRF` protections enabled; do not disable CSRF to support cross-origin browser writes
- CORS support is for controlled non-browser integrations, not for primary browser auth flows

## Canonical API-Only Scaffold

Start from the API-only starter mode:

```bash
ship new demo --module example.com/demo --api-only
```

This scaffold keeps route naming and auth endpoints while removing templ-first page assets.

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

## Local Development Topology (SvelteKit + GoShip API)

Use two local processes:

1. GoShip API app from the API-only scaffold.
2. SvelteKit frontend app.

Recommended local browser topology:

- Browser origin: `http://localhost:5173` (SvelteKit dev server).
- SvelteKit server-side handlers proxy API/auth requests to GoShip.
- Browser does not call GoShip cross-origin directly for session-authenticated writes.

Keep browser session semantics stable:

- login/logout and session endpoints remain on GoShip (`/auth/login`, `/auth/register`, `/auth/logout`)
- SvelteKit form actions and server endpoints forward cookies and CSRF headers
- write requests include the `X-CSRF-Token` header value from the same-origin session flow

## Deployment Topology (Same-Origin Requirement)

Production deployment must preserve same-origin browser behavior:

- serve SvelteKit and GoShip behind one public origin (same scheme + host + port)
- route `/api/*` and `/auth/*` to GoShip
- route page/UI requests to SvelteKit

Do not rely on cross-origin browser cookie sessions as the primary integration mode.
If you need third-party or server-to-server integration, use explicit API tokens and scoped CORS rules.
