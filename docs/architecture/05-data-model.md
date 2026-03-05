# Data Model
<!-- FRONTEND_SYNC: Landing capability explorer in app/views/web/pages/landing_page.templ links here for Models and ORM (Ent). Keep both landing copy and this doc aligned. -->

Primary schema is defined in `db/schema/*.go` and compiled into generated Ent code in `db/ent/`.

## Core Entities

Identity and profile:

- `User`
- `Profile`
- `PasswordToken`
- `LastSeenOnline`

Communication and notifications:

- `Notification`
- `NotificationPermission`
- `NotificationTime`
- `PwaPushSubscription`
- `FCMSubscriptions`
- `SentEmail`

Billing:

- `MonthlySubscription`

Media and file storage:

- `Image`
- `ImageSize`
- `FileStorage`

Other domain support:

- `EmailSubscription`
- `EmailSubscriptionType`
- `Invitation`
- `PhoneVerificationCode`
- `Emojis`

## Important Relationships

- `User` has one `Profile` (`User.profile` unique edge).
- `Profile` has many `notifications`, `photos`, `invitations`, push subscriptions, and notification permissions.
- `MonthlySubscription` has one `payer` profile and many `benefactors` profiles.
- `Notification` belongs to one `Profile`.
- `NotificationPermission` is unique per `(profile_id, permission, platform)`.
- `NotificationTime` is unique per `(profile_id, type)`.
- `Image` has many `ImageSize` records.
- `ImageSize` has one required `FileStorage` object.

## Domain Enums

Defined in `framework/domain/enum.go`:

- Notification types
- Notification permission types
- Notification delivery platforms
- Image sizes and categories
- Product type (free/pro)
- Bottom navbar item
- Email subscription list

## Subscription Model Notes

Current app logic treats free plan as absence of an active pro subscription.

- Creation path often creates trial pro subscription on onboarding.
- Active subscription uniqueness is enforced by unique index on `(paying_profile_id, is_active)`.

## Notification Model Notes

Notification records include:

- Type/title/text/link
- Read and read timestamp
- Optional actor/resource linkage
- Per-profile ownership

Permissions are separated from notifications and keyed by platform + permission type.

## Storage Model Notes

`FileStorage` holds object metadata and object key info for S3-compatible storage.
The storage repo generates presigned URLs and maps image sizes into frontend-friendly `domain.Photo` objects.
