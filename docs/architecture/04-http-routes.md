# HTTP Route Map
<!-- FRONTEND_SYNC: Landing capability explorer in app/goship/views/web/pages/landing_page.templ links here for Routing and Controllers. Keep both landing copy and this doc aligned. -->

Routes are wired through canonical `app/goship/router.go`.

Ergonomic routing rule:

- URL declarations live in one place: `app/goship/router.go`.
- Handler implementations live in `app/goship/web/routes/*.go`.

## Public/General Routes

- `GET /` landing page
- `GET /up` healthcheck
- `GET /clear-cookie`
- `GET /about`
- `GET /privacy-policy`
- `GET /install-app`

Docs pages (user-facing in-app docs):

- `GET /docs`
- `GET /docs/gettingStarted`
- `GET /docs/guidedTour`
- `GET /docs/architecture`

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

## Auth-Adjacent Routes

- `GET /email/verify/:token`

## External Integration Routes

- `POST /Q2HBfAY7iid59J1SUN8h1Y3WxJcPWA/payments/webhooks`

## Development-Only Error Preview Routes

Registered only when not production:

- `GET /error/400`
- `GET /error/401`
- `GET /error/403`
- `GET /error/404`
- `GET /error/500`

## Conditional Routes

Realtime is conditionally wired:

- `GET /auth/realtime` is registered only when runtime web features enable realtime (notifier + pubsub available).

Notification center routes have implementations but are still not wired:

- list notifications
- mark all read
- delete notification
- mark read/unread endpoints
