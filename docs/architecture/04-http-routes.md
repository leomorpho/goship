# HTTP Route Map
<!-- FRONTEND_SYNC: Landing capability explorer in app/views/web/pages/landing_page.templ links here for Routing and Controllers. Keep both landing copy and this doc aligned. -->

Routes are wired through canonical `app/router.go`.

Ergonomic routing rule:

- URL declarations live in one place: `app/router.go`.
- Handler implementations live in `app/web/controllers/*.go`.

## Public/General Routes

- `GET /` starter home page
- `GET /demo/islands` framework islands demo (vanilla JS, React, Vue, Svelte)
- `GET /up` liveness healthcheck (legacy alias)
- `GET /health` liveness healthcheck
- `GET /health/ready` readiness healthcheck (returns `503` when a registered dependency check fails)
- `GET /clear-cookie`
- `GET /install-app`

Email subscription:

- `GET /emailSubscribe`
- `POST /emailSubscribe`
- `GET /email/subscription/:token`

Service worker / app-links:

- `GET /service-worker.js`
- `GET /.well-known/assetlinks.json`

## User-Not-Authenticated Group (`/user`)

- `GET /user/login`
- `POST /user/login`
- `GET /user/register`
- `POST /user/register`
- `GET /user/password`
- `POST /user/password`
- `GET /user/password/reset/token/:user/:password_token/:token`
- `POST /user/password/reset/token/:user/:password_token/:token`

## Authenticated Onboarding Group (`/welcome`)

- `GET /welcome/preferences`
- `GET /welcome/preferences/phone`
- `GET /welcome/preferences/phone/verification`
- `POST /welcome/preferences/phone/verification`
- `POST /welcome/preferences/phone/save`
- `GET /welcome/preferences/display-name/get`
- `POST /welcome/preferences/display-name/save`
- `GET /welcome/preferences/delete-account`
- `GET /welcome/preferences/delete-account/now`
- `GET /welcome/preferences/2fa/setup`
- `POST /welcome/preferences/2fa/setup`
- `GET /welcome/preferences/2fa/backup-codes`
- `GET /welcome/finish-onboarding`
- `GET /welcome/profileBio`
- `POST /welcome/profileBio/update`

Notification subscription management during onboarding:

- `GET /welcome/subscription/push`
- `POST /welcome/subscription/:platform`
- `DELETE /welcome/subscription/:platform`
- `GET /welcome/email-subscription/unsubscribe/:permission/:token`

## Authenticated Group (`/auth`)

- `GET /auth/logout`

Fully onboarded-only routes (`/auth` with onboarding guard):

- `GET /auth/homeFeed`
- `GET /auth/homeFeed/buttons`
- `GET /auth/profile`
- `GET /auth/uploadPhoto`
- `POST /auth/uploadPhoto`
- `DELETE /auth/uploadPhoto/:image_id`
- `GET /auth/currProfilePhoto`
- `POST /auth/currProfilePhoto`
- `GET /auth/notifications/normalNotificationsCount`
- `GET /auth/payments/get-public-key`
- `POST /auth/payments/create-checkout-session`
- `POST /auth/payments/create-portal-session`
- `GET /auth/payments/pricing`
- `GET /auth/payments/success`

Admin routes (`/auth/admin`, admin role required):

- `GET /auth/admin`
- `GET /auth/admin/queues`
- `GET /auth/admin/queues/*`
- `GET /auth/admin/managed-settings`
- `GET /auth/admin/audit-logs`
- `GET /auth/admin/trash`
- `GET /auth/admin/:resource`
- `GET /auth/admin/:resource/new`
- `POST /auth/admin/:resource`
- `GET /auth/admin/:resource/:id`
- `PUT /auth/admin/:resource/:id`
- `DELETE /auth/admin/:resource/:id`

## Auth-Adjacent Routes

- `GET /auth/2fa/verify`
- `POST /auth/2fa/verify`
- `GET /auth/oauth/:provider`
- `GET /auth/oauth/:provider/callback`
- `GET /email/verify/:token`

## External Integration Routes

- `POST /Q2HBfAY7iid59J1SUN8h1Y3WxJcPWA/payments/webhooks`
- `GET /managed/status` (managed mode only, signed)
- `POST /managed/backup` (managed mode only, signed)
- `POST /managed/restore` (managed mode only, signed)

## Development-Only Error Preview Routes

Registered only when not production:

- `GET /error/400`
- `GET /error/401`
- `GET /error/403`
- `GET /error/404`
- `GET /error/500`

## Development-Only Email Preview Routes

Registered only when `app.environment=dev`:

- `GET /dev/mail`
- `GET /dev/mail/welcome`
- `GET /dev/mail/password-reset`
- `GET /dev/mail/verify-email`

## Conditional Routes

Realtime is conditionally wired:

- `GET /auth/realtime` is registered only when runtime web features enable realtime (notifier + pubsub available).
- `GET /auth/ai-demo` and `GET /auth/ai-demo/stream` are registered only when the app environment is not production.
- `/managed/*` routes are registered only when `PAGODA_MANAGED_MODE=true` and require valid `X-GoShip-*` signature headers.

Notification center routes have implementations but are still not wired:

- list notifications
- mark all read
- delete notification
- mark read/unread endpoints
