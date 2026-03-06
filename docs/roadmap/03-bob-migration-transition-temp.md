# Bob + Goose Transition Plan (Temporary Living Checklist)

Status: active temporary execution tracker. Delete this file when all checklist items are complete.

Last updated: 2026-03-06

## Scope

Transition GoShip from Ent-centric data ownership to Bob + Goose with module-owned database boundaries.

Non-goals for this plan:
- Building a custom Rails/Laravel/Django-style model DSL now.
- Supporting multi-database per mini-app now.

## Locked Decisions

- [x] Use **Bob** for DB code generation.
- [x] Use **Goose** as the default migration tool.
- [x] Keep one physical DB per app repo.
- [ ] Keep Ent only as temporary compatibility during migration.
- [ ] Remove Ent entirely when parity is complete.

## Architecture Target (Definition of Done)

- [ ] Core app owns only core DB artifacts.
- [ ] Each installable module owns its DB artifacts end-to-end.
- [x] `ship db:*` commands run Goose + Bob in a deterministic way.
- [x] No module DB package imports from app core DB package.
- [ ] No app feature logic depends directly on Ent.

Target ownership layout:

- Core:
  - `db/migrate/migrations/*` (transition baseline)
  - `db/schema/*.go` (Ent transition baseline)
  - `db/queries/*.sql` (Bob target)
  - `db/gen/*` (Bob target)

- Module:
  - `modules/<name>/db/migrations/*`
  - `modules/<name>/db/schema/*.sql` (optional)
  - `modules/<name>/db/queries/*.sql`
  - `modules/<name>/db/gen/*`

## Global Test Gate (Mandatory)

No checklist item is marked done unless all applicable tests pass.

Required gates for behavior changes:
- [x] Unit tests added/updated for changed command/runtime logic.
- [x] Integration test added/updated for end-to-end behavior.
- [x] `bash tools/scripts/precommit-tests.sh` passes.
- [x] If DB workflow changed: integration path validates real migration + generation flow.

Test design rules:
- [ ] Prefer table-driven unit tests for parser/flag/safety logic.
- [ ] Keep integration tests Docker-free where practical.
- [ ] Use deterministic test fixtures and temp dirs; no writes to live repo tree.
- [ ] Avoid hidden coupling between CLI package tests and app runtime tree.

## Current Baseline Snapshot (As of 2026-03-05)

- [x] Migration runtime is Atlas + Ent today (not Goose + Bob yet).
- [x] Current DB commands are `db:create`, `db:make`, `db:migrate`, `db:status`, `db:drop`, `db:reset`, `db:rollback`, `db:seed`.
- [x] Destructive DB safety contract exists (`--yes`, `--force`, `--dry-run`, production guards).
- [x] Installed-module registry file exists at `config/modules.yaml` (`modules: []` baseline).
- [x] `db:generate` for Bob does not exist yet.
- [ ] `make:migration` alias does not exist; `db:make` is current canonical command.
- [x] Goose runtime package/invocation path does not exist yet.

---

## Source-backed Constraints (Bob)

- Bob is database-first and expects schema to exist.
- Migrations are managed outside Bob.
- Query generation from `.sql` is first-class.

Sources:
- https://bob.stephenafamo.com/docs/code-generation/intro
- https://bob.stephenafamo.com/docs/code-generation/queries
- https://bob.stephenafamo.com/docs/code-generation/sql/
- https://bob.stephenafamo.com/vs/ent/

---

## Phase 0: Finalize Contracts and CLI Behavior

Goal: lock behavior before implementation drift starts.

### 0.1 DB command contract
- [x] Define current command set (Atlas/Ent baseline):
  - `ship db:migrate`
  - `ship db:status`
  - `ship db:create`
  - `ship db:drop`
  - `ship db:reset`
  - `ship db:make <migration_name>`
- [x] Decide transition contract for Bob/Goose:
  - keep `ship db:make <migration_name>` as canonical (no alias command for now).
  - add `ship db:generate` for Bob.
- [x] Define command help text and examples for current DB commands.
- [x] Define dry-run behavior (`--dry-run`) for destructive commands.

Acceptance criteria:
- [x] Commands and flags are documented in `docs/reference/01-cli.md`.
- [x] Commands and flags are covered by CLI unit tests.
- [x] Integration test coverage exists for DB flow and safety paths.
- [x] Add integration test coverage for new Bob/Goose commands when introduced.

### 0.2 Installed-module registry contract
- [x] Define one canonical installed-module registry location and format (`config/modules.yaml`).
- [x] Define deterministic execution order (core first, then modules sorted) for DB migrate/generate orchestration.
- [x] Define module enable/disable semantics for migration/generation participation.

Acceptance criteria:
- [x] `ship doctor` validates registry file format.
- [x] Integration test proves deterministic order.
- [x] Unit tests cover registry parsing and sort behavior.

### 0.3 Migration safety contract
- [x] Define local/dev/prod safety rules for `db:drop` and `db:reset`.
- [x] Define explicit confirmation semantics (`--yes`).
- [x] Define non-local DB protection behavior.

Acceptance criteria:
- [x] Safety behavior is unit-tested and integration-tested.
- [x] Negative tests cover non-local DB protection paths.

---

## Phase 1: Goose Foundation in Ship CLI

Goal: deliver Goose-backed migration commands without changing app data access yet.

Transition note:
- Goose is now wired behind the existing `ship db:*` surface for a constrained DB subset (`postgres`, `mysql`, `sqlite/sqlite3`).
- Phase 1 implementation + test gates are complete.

### 1.1 Goose runtime wiring
- [x] Add Goose runner path in CLI runtime execution flow.
- [x] Support DB URL resolution from one canonical env var path.
- [x] Remove duplicate/ambiguous DB URL env resolution.

Acceptance criteria:
- [x] Unit tests for URL resolution and Goose invocation.
- [x] Integration smoke test verifies Goose runtime can execute with temp DB.

### 1.2 Core migration commands
- [x] Implement `ship db:migrate` (core app migrations only initially).
- [x] Implement `ship db:status`.
- [x] Implement `ship db:make <migration_name>` on Goose runtime (optionally add `ship make:migration <name>` alias if chosen in Phase 0.1).

Acceptance criteria:
- [x] Unit tests for command parsing and runtime calls.
- [x] Integration test: create migration -> migrate -> status reflects applied.
- [x] Regression test: rerun migrate is idempotent.

### 1.3 Destructive lifecycle commands
- [x] Implement `ship db:create`.
- [x] Implement `ship db:drop`.
- [x] Implement `ship db:reset` (`drop + create + migrate`, optional seed step).

Acceptance criteria:
- [x] Safety checks tested.
- [x] Integration test for local reset happy path.
- [x] Integration test for reset refusal on protected/non-local target.

---

## Phase 2: Bob Foundation (Core Only)

Goal: introduce Bob generation for core with minimal service-layer disruption.

### 2.1 Bob config and generation entrypoint
- [x] Add core Bob config file(s).
- [x] Add `ship db:generate` for core generation.
- [x] Add deterministic generation output locations.

Acceptance criteria:
- [x] `ship db:generate` idempotency test (no diff on second run).
- [x] Pre-commit check ensures generated code is current.
- [x] Integration test verifies generate succeeds from clean temp project.

### 2.2 Core DB adapter seam
- [x] Create app-internal data adapter interfaces independent of Ent/Bob. (first path: unseen notifications count in `app/profile`)
- [x] Implement Bob-backed adapter for first core path. (notifications unseen-count path via `db/gen` + `BobNotificationCountStore`)
- [x] Keep Ent adapter temporarily for fallback during migration. (first path now uses `EntNotificationCountStore`)

Progress notes (2026-03-06):
- `app/profile` runtime read/write paths are DB-first for friends/profile/media/delete-account operations.
- `DeleteUserData` now uses SQL cleanup for subscription and user removal (no Ent path in runtime flow).
- Composition roots (`cmd/web`, `cmd/worker`, web wiring/routes tests) no longer inject `c.ORM` into `ProfileService` runtime construction.
- `app/profile` non-test package no longer imports Ent; precommit now enforces this boundary.
- `ProfileService` constructors are DB-only (`NewProfileServiceWithDBDeps`), and `CreateProfile`/`UpdateProfile` now operate on SQL-native IDs/fields instead of Ent types.

Acceptance criteria:
- [x] Unit tests for adapter behavior parity. (first path: Ent + SQL/Bob-path stores covered)
- [x] No controller imports from Ent package.
- [x] Integration test covers one migrated read/write path through app API. (read path covered: notifications count endpoint)

---

## Phase 3: Module-owned DB Boundaries (First Module)

Goal: prove full module isolation end-to-end.

### 3.1 Select and migrate first module
- [x] Pick first module (recommended: `emailsubscriptions` for lower risk).
- [x] Create module-local Goose migrations.
- [x] Add module-local Bob config + generated package.
- [x] Refactor module services to use module-local generated DB package.

Acceptance criteria:
- [x] Module compiles standalone.
- [x] Module tests pass standalone.
- [x] Module no longer imports core DB package.
- [x] Integration test covers module install + migrate + functional happy path.

### 3.2 CLI orchestration for module migrations
- [x] Update `ship db:migrate` to run:
  - core migrations
  - installed modules in deterministic order
- [x] Update `ship db:status` to display core + module status.
- [x] Update `ship db:generate` for core + installed modules.

Acceptance criteria:
- [x] Integration test: fresh project + install module + migrate + generate succeeds.
- [x] Failure behavior tested (module migration fails -> aborts cleanly).
- [x] Regression test: ordering remains deterministic across repeated runs.
- [x] `ship doctor` validates enabled module DB artifacts (`db/migrate/migrations` + `db/bobgen.yaml`).

---

## Phase 4: Expand Module Coverage

Goal: move all installable modules to module-owned DB layers.

### 4.1 Migrate remaining modules
- [x] `jobs`
- [x] `notifications`
- [x] `paidsubscriptions`
- [x] any additional module added during transition

Progress notes (2026-03-05):
- `notifications`: added module-local SQL store + module migrations + `bobgen.yaml`; runtime now accepts module-local `PubSub` boundary (removed direct `framework/core` type dependency in notifier path).
- `notifications`: `NotificationPermissionService` now routes through storage interface with SQL + Ent implementations; module runtime uses SQL permissions store when DB is available.
- `notifications`: PWA/FCM "has permission" checks now use `NotificationPermissionService` instead of querying `notification_permissions` directly from push service code.
- `notifications`: `PlannedNotificationsService` now routes through storage interface with SQL + Ent implementations; module runtime uses SQL planned-notifications store when DB is available.
- `notifications`: `SMSSender` now routes through storage interface with SQL + Ent implementations; module runtime uses SQL SMS-code store when DB is available.
- `notifications`: PWA/FCM push subscription persistence now routes through storage interfaces with SQL + Ent implementations; module runtime uses SQL push-subscription stores when DB is available.
- `notifications`: module constructor now has explicit dependency guard (`requires either DB or ORM`), and SQL-mode constructor path is covered by tests.
- `notifications`: added SQL planned-notifications tests (candidate selection + same-day de-dupe) and restored tolerant `CreateNotificationTimeObjects` behavior that skips profiles with missing activity instead of failing the full batch.
- `notifications`: removed Docker/Ent-heavy integration tests for permissions/planned/FCM and replaced coverage with fast SQL/unit tests to align with Docker-free test goals.
- `jobs`: centralized `framework/core` dependency to `modules/jobs/contracts.go` using local aliases; internal jobs implementation files no longer import `framework/core` directly.
- `jobs`: promoted contracts to module-owned concrete types/interfaces (no framework aliases), and added app-level bridge adapters so container still consumes `core.Jobs`/`core.JobsInspector` without behavior changes.
- `jobs`: removed all jobs-specific entries from module-isolation allowlist; jobs now pass strict module-isolation checks without exceptions.
- `jobs`: SQL driver path now DB-first (`*sql.DB`) with module-local migrations and `bobgen.yaml`; Ent coupling removed from jobs SQL config/module entrypoint.

Acceptance criteria:
- [x] Every module has local migrations + generation.
- [x] No module imports core DB package.
- [x] Module-isolation checks enforce boundary automatically in precommit.

### 4.2 Module contract hardening
- [x] Standardize module DB package structure template.
- [x] Add `ship` scaffolder support for new module DB skeleton.

Acceptance criteria:
- [x] New module generation integration test covers DB scaffold.
- [x] Test verifies scaffold is generated in module-local DB paths only.

---

## Phase 5: Ent Decommission

Goal: remove Ent once runtime parity is complete.

### 5.1 Removal readiness checks
- [ ] Confirm no runtime code paths require Ent.
- [ ] Confirm migrations and generators run fully via Goose + Bob.

### 5.1a Runtime Ent cutover backlog (execution order)
- [ ] `app/profile`: extract SQL/Bob stores for profile reads/writes/media and remove direct Ent runtime dependency.
- [x] `app/jobs`: move notifications/mail/email-update DB access behind SQL/Bob stores.
- [x] `framework/repos/storage`: remove Ent-typed boundaries (`*ent.Image`) from runtime interfaces.
- [x] `app/foundation`: remove Ent-default auth/container runtime paths (keep temporary test-only seams only as needed).
- [x] `cmd/*` + app wiring: validate all module runtime deps run DB-first without Ent at composition root.

### 5.2 Remove Ent toolchain
- [x] Remove Ent generation commands from Ship and Make flow.
- [ ] Remove `db/ent` and Ent-specific docs.
- [ ] Remove Ent dependencies from `go.mod`/`go.work`.

Acceptance criteria:
- [ ] `ship doctor` has no Ent-related expectations.
- [ ] Full pre-commit and integration suite green without Ent.
- [ ] Clean-room build/test verifies no hidden Ent dependency remains.

---

## Phase 6: DX Hardening and Docs

Goal: ensure long-term maintainability and agent reliability.

### 6.1 Docs and runbooks
- [x] Update `docs/reference/01-cli.md` for final DB command behavior.
- [x] Add/refresh how-to guides for migration + generation workflows.
- [x] Keep `LLM.txt` generation aligned.

### 6.2 Guardrails
- [x] Add doctor checks for:
  - missing module migrations dir
  - missing module Bob config
  - forbidden cross-boundary imports
- [x] Add CI checks for generated-code drift.

Acceptance criteria:
- [ ] `ship doctor` enforces architecture contracts.
- [x] CI catches drift and boundary violations.
- [ ] Docs checks fail CI when CLI behavior and docs diverge.

---

## Cross-phase Risk Register

- [ ] Risk: command churn during transition confuses users.
  - Mitigation: lock command contract in Phase 0 and stick to it.

- [ ] Risk: mixed Ent/Bob period creates duplicate write paths.
  - Mitigation: adapter seam + strict feature-by-feature cutover.

- [ ] Risk: module migration order bugs.
  - Mitigation: deterministic ordering + integration tests + failure tests.

- [ ] Risk: generated code noise in PRs.
  - Mitigation: deterministic generation + pre-commit drift checks.

---

## Execution Log (append-only)

- 2026-03-04: locked Bob + Goose decisions and created this temporary checklist.
- 2026-03-05: aligned transition checklist with current Atlas/Ent baseline and clarified Bob/Goose delta tasks.
- 2026-03-05: switched `ship db:*` command execution to Goose with initial supported subset (`postgres`, `mysql`, `sqlite/sqlite3`) and updated CLI unit coverage.
- 2026-03-05: verified Goose DB flow integration (`make -> migrate -> status -> reset`) and added migrate idempotency assertion.
- 2026-03-05: added modules manifest parser/normalization + doctor format validation (`config/modules.yaml`) with runtime/policy unit tests.
- 2026-03-05: added module-aware Goose orchestration for `db:migrate/status/reset/drop` (core first, then sorted enabled modules from `config/modules.yaml`).
- 2026-03-05: added `ship db:generate` (`bobgen-sql`) and core Bob config scaffold at `db/bobgen.yaml` with deterministic output target (`db/gen`).
- 2026-03-05: added integration coverage for `ship db:generate` from a clean `ship new` project (including idempotency assertion).
- 2026-03-05: updated `ship new` scaffold with Bob defaults (`db/bobgen.yaml`, `db/queries/.gitkeep`, `db/gen/.gitkeep`).
- 2026-03-05: added pre-commit drift guard (`tools/scripts/check-bobgen-drift.sh`) to fail when Bob generated output is stale.
- 2026-03-05: added integration test proving module migration execution order is deterministic (`core -> sorted modules`).
- 2026-03-05: introduced first app-level data seam in `app/profile` for unseen notification counts (`NotificationCountStore`) with Ent adapter + unit tests.
- 2026-03-05: added SQL-backed unseen notification count store and wired web/worker runtime to use it for the first migrated read path.
- 2026-03-05: notifications-count controller path now reads authenticated profile ID from middleware context (`auth_profile_id`) instead of querying Ent user edges directly.
- 2026-03-05: added focused tests for migrated notifications-count path (`app/web/controllers/notifications_test.go`) and updated controller integration harness module wiring in `routes_test.go`.
- 2026-03-05: introduced first `db/gen` query wrapper and switched web/worker notifications-count runtime wiring to `BobNotificationCountStore`.
- 2026-03-05: added dedicated integration-tagged API tests for migrated notifications count path in `app/web/apitests` (success, error, and missing-profile cases).
- 2026-03-05: extracted shared auth-profile context helper for controllers and migrated additional handlers (`delete_account`, `realtime`, `profile_photo`, `upload_photo`, `push_notifs`) off direct `AuthenticatedUser -> QueryProfile()` reads.
- 2026-03-05: migrated `payments` and `profile` controllers to use middleware identity context (`auth_profile_id`, `auth_user_email`) and removed direct `QueryProfile()` reads there.
- 2026-03-05: added middleware context keys `auth_user_id` and `auth_user_email`, plus controller helper coverage for email access; login onboarding check now goes through profile service by user ID instead of controller-level `QueryProfile()`.
- 2026-03-05: added pre-commit boundary enforcement `tools/scripts/check-controller-auth-boundary.sh` to block reintroduction of direct `QueryProfile()` reads in controllers.
- 2026-03-05: removed direct `*ent.User` controller coupling in reset-password flow by propagating `auth_user_id`/`auth_user_email` from `LoadUser` middleware and updating password reset by user ID.
- 2026-03-05: simplified verify-email controller identity handling to use middleware auth keys (`auth_user_id`, `auth_user_email`) instead of controller-level authenticated-user entity assertions.
- 2026-03-05: introduced framework-level DB error helpers (`framework/dberrors`) and removed direct `db/ent` root imports from controllers (`login`, `forgot_password`, `register`, `push_notifs`) by switching to capability checks.
- 2026-03-05: eliminated remaining controller imports of `db/ent/*` by routing email lookups through `AuthClient.FindUserByEmail` and replacing profile field-select queries with ID-based entity loads.
- 2026-03-05: added controller DB boundary guardrail (`tools/scripts/check-controller-no-ent-imports.sh`) and wired it into precommit checks.
- 2026-03-05: shifted password-token middleware validation to use `auth_user_id` context key (instead of `context.UserKey` entity assertion), further reducing entity coupling in request middleware.
- 2026-03-05: removed legacy `context.UserKey` usage entirely; `LoadUser` now sets only explicit auth scalar keys (`auth_user_id`, `auth_user_email`) and middleware tests were updated accordingly.
- 2026-03-05: simplified password-reset auth API surface to avoid leaking Ent token entities to callers (`GeneratePasswordResetToken` now returns token + tokenID, `GetValidPasswordToken` now returns only error) and removed unused password token context storage.
- 2026-03-05: extracted auth persistence seam (`authStore`) with Ent-backed implementation in `app/foundation/auth_store.go`; `AuthClient` now delegates user lookup, last-seen writes, and password-token persistence through this store.
- 2026-03-05: removed `AuthenticatedUserKey` entity payload from request context; auth middleware/page assembly now rely on scalar keys (`auth_user_id`, `auth_user_name`, `auth_user_email`, `auth_profile_id`) and UI uses `AuthUserView` instead of `*ent.User`.
- 2026-03-05: replaced Ent-returning auth APIs with DTOs for callers (`GetAuthenticatedIdentity`, `FindUserRecordByEmail`), migrated middleware/controllers to those methods, and kept Ent usage contained to foundation store internals.
- 2026-03-05: converted `authStore` interface to DTO-based returns (`GetIdentityByUserID`, `GetUserRecordByEmail`) so Ent entity usage is isolated to the Ent store implementation only.
- 2026-03-05: added explicit auth-store selector seam (`selectAuthStore`) with `PAGODA_AUTH_STORE` env gate and Bob placeholder store wiring; default runtime remains Ent and selector behavior is covered by unit tests.
- 2026-03-05: implemented first real Bob auth read paths (`GetUserRecordByEmail`, `GetIdentityByUserID`) via new `db/gen/auth.go` queries and wired them through `bobAuthStore` while keeping writes on placeholder/not-implemented.
- 2026-03-05: implemented first Bob auth write path (`CreateLastSeenOnline`) via `db/gen/auth.go::InsertLastSeenOnline` and wired it through `bobAuthStore`, with sqlite coverage in both `db/gen` and foundation store tests.
- 2026-03-05: moved Bob auth SQL out of Go code into `db/queries/auth.sql` with embedded named-query registry (`db/queries/registry.go`) and updated `db/gen/auth.go` to resolve SQL by query name + dialect.
- 2026-03-05: added Bob implementations for user/password auth writes (`UpdateUserPasswordHashByUserID`, `MarkUserVerifiedByUserID`, password-token create/read/delete), and migrated `reset_password`/`verify_email` controllers to use `AuthClient` store-backed methods instead of direct ORM updates.
- 2026-03-05: normalized auth/middleware not-found handling across Ent and Bob (`ent.IsNotFound` + `sql.ErrNoRows`) through `framework/dberrors`, preventing Bob-mode no-row cases from surfacing as HTTP 500 in auth flows.
- 2026-03-05: extracted display-name read/write from preferences controller into auth store/API (`GetUserDisplayNameByUserID`, `UpdateUserDisplayNameByUserID`) with Bob-backed query implementations and sqlite test coverage.
- 2026-03-05: migrated `emailsubscriptions` primary store adapter off app Ent client to module-local SQL store (`NewSQLStore`), switched app web wiring to DB-handle injection, and removed the module isolation allowlist entry for the previous Ent store adapter.
- 2026-03-05: added module-local Bob scaffold for `emailsubscriptions` (`modules/emailsubscriptions/db/{bobgen.yaml,queries,gen,migrate/migrations}`) and updated `ship db:generate` to run Bob generation for core + enabled modules in deterministic manifest order.
- 2026-03-05: moved `UpdateEmailSender` out of `modules/emailsubscriptions` into `app/jobs` (app-specific concern), removing the module’s last direct import of app-core packages and tightening module isolation checks.
- 2026-03-05: added initial `emailsubscriptions` module migration file (`modules/emailsubscriptions/db/migrate/migrations/20260305170000_init_email_subscriptions.sql`) and added integration coverage that validates deterministic `ship db:generate` ordering across core + enabled modules.
- 2026-03-05: fixed CLI integration test root resolution (`ent_generate_integration_test`) to run Ent generation against the workspace `db/schema` path after repo layout changes.
- 2026-03-05: hardened `ent_generate_integration_test` with `GOFLAGS=-modcacherw` and temp-only `GOMODCACHE` to prevent read-only module cache cleanup failures; updated doctor line-budget walk to skip nested `.cache` directories.
- 2026-03-05: refactored `emailsubscriptions` SQL store to use module-local DB query package (`modules/emailsubscriptions/db/gen`) instead of embedding raw SQL in the service store implementation.
- 2026-03-05: added module integration coverage for `emailsubscriptions` that applies the module migration file and validates end-to-end subscribe/confirm/unsubscribe behavior against the migrated schema.
- 2026-03-05: added CLI integration tests for module-enabled fresh-project DB flow (`db:migrate` + `db:generate`) and explicit failure behavior when an enabled module is missing its migrations directory.
- 2026-03-05: enhanced `ship make:module` to scaffold module-local DB layout by default (`db/bobgen.yaml`, `db/migrate/migrations/.gitkeep`, `db/queries/.gitkeep`, `db/gen/.gitkeep`) and expanded generator integration assertions for this contract.
- 2026-03-05: fully isolated `modules/paidsubscriptions` from root app/framework imports by moving Ent adapter to `app/subscriptions`, introducing module-local plan type, and adding module-owned migrations including `subscription_customers` mapping for stripe customer IDs.
- 2026-03-05: added migration-backed integration coverage for `modules/paidsubscriptions` SQL store (`store_sql_integration_test.go`) to validate module-owned schema + lifecycle behavior end-to-end.
- 2026-03-05: improved `ship db:status` output to print explicit section headers per migration scope (`core` + each enabled module), with unit + integration coverage for deterministic multi-scope status behavior.
- 2026-03-05: scaffolded `modules/notifications` DB ownership baseline (`db/bobgen.yaml`, `db/migrate/migrations/20260305193000_init_notifications.sql`, `db/{queries,gen}`) as the first step toward notifications module Bob/Goose migration.
- 2026-03-05: added `notifications` SQL storage adapter (`NewSQLNotificationStore`) and unit coverage for notification lifecycle/read-state behavior to enable incremental cutover from Ent store logic.
- 2026-03-05: updated notifications runtime wiring to accept `DB` + `DBDialect` in `RuntimeDeps` and use `NewSQLNotificationStore` for notifier storage when DB deps are provided (fallback remains Ent store for compatibility).
- 2026-03-05: added migration-backed integration coverage for notifications SQL store (`notification_store_sql_integration_test.go`) to validate module-owned notification schema + storage lifecycle against module migration SQL.
- 2026-03-05: made jobs SQL backend DB-first (`SQLDB` support in jobs config + SQL driver, with Ent fallback) and switched web/worker `dbqueue` wiring to pass `*sql.DB` instead of Ent.
- 2026-03-05: scaffolded `modules/jobs` DB ownership baseline (`db/bobgen.yaml`, `db/migrate/migrations/20260305195000_init_jobs.sql`, `db/{queries,gen}`) and added migration-backed integration coverage for the SQL jobs driver.
- 2026-03-05: completed jobs SQL contract cleanup by removing `EntClient` from jobs module config/driver APIs (SQLDB-only), migrated module tests to SQLDB paths, and trimmed module-isolation allowlist entries no longer needed (`modules/jobs/config.go`, `modules/jobs/drivers/sql/client.go`).
- 2026-03-05: made jobs SQL schema provisioning migration-driven by embedding/parsing module goose migration SQL in `modules/jobs/db/migrate` and reusing it from the SQL driver (`ensureSchema`), removing duplicated inline DDL from driver code.
- 2026-03-05: made notifications SQL schema provisioning migration-driven by adding `modules/notifications/db/migrate` loader helpers and reusing module goose migration SQL in `NewSQLNotificationStoreWithSchema`/`ensureSchema`, removing duplicated test DDL and aligning schema source of truth.
- 2026-03-05: added jobs SQL boundary precommit guard (`tools/scripts/check-jobs-sql-boundary.sh`) to prevent reintroducing Ent coupling in jobs SQL config/driver paths after SQLDB-only migration.
- 2026-03-05: completed Phase 4 module DB-ownership acceptance criteria for current installable modules (`emailsubscriptions`, `jobs`, `notifications`, `paidsubscriptions`) and validated via precommit module-isolation guards.
- 2026-03-05: refreshed database workflow docs to Bob+Goose canonical behavior in `docs/guides/02-development-workflows.md` and `docs/guides/03-how-to-playbook.md`.
- 2026-03-05: expanded `ship doctor` with cross-boundary import guardrails (controller Ent + QueryProfile bans, jobs SQL Ent ban, notifications framework/core ban, module source-isolation scan with allowlist parity).
- 2026-03-05: aligned `LLM.txt` generation workflow/docs (`make llm-txt`, generator script, pre-commit auto-stage hook) and regenerated `LLM.txt` after docs updates.
- 2026-03-05: completed Phase 5 inventory pass for remaining runtime Ent usage; main active callsites are in `app/foundation` auth/container selection, `app/profile`, `app/jobs`, `app/subscriptions/store_ent.go`, `app/web/middleware/entity.go`, and `framework/repos/storage`.
- 2026-03-05: removed direct Ent dependency from `app/web/middleware/entity.go` (`LoadUser` now resolves via `AuthClient.GetIdentityByUserID` + `dberrors.IsNotFound`), and updated router/tests callsites to pass `c.Auth` instead of `c.ORM`.
- 2026-03-05: removed direct Ent import dependency from `framework/dberrors`; not-found/constraint detection now uses standard SQL checks + wrapped error type-name detection (`NotFoundError`/`ConstraintError`) to keep Ent optional at runtime seams.
- 2026-03-05: switched web/worker paid-subscriptions runtime wiring to module SQL store (`paidsubscriptions.NewSQLStore(c.Database, c.Config.Adapters.DB, ...)`) instead of app Ent store constructors.
- 2026-03-05: switched seed command paid-subscriptions wiring to module SQL store (`SeedUsers(..., db, dialect, ...)`), while keeping integration-route harness on Ent store until that test schema path is fully module-migration-backed.
- 2026-03-05: switched auth runtime selector to DB-first default (`PAGODA_AUTH_STORE` empty now resolves to Bob when DB is available; explicit `ent` still supported), with fallback to Ent only when DB is unavailable or explicitly requested.
- 2026-03-05: switched web/worker notifications runtime deps to DB-only mode (`ORM: nil`, `DB + DBDialect` provided), confirming the module path runs without Ent at composition root.
- 2026-03-05: added explicit Phase 5 runtime cutover backlog (`app/profile`, `app/jobs`, `framework/repos/storage`, `app/foundation`, and final cmd/wiring verification) to make remaining Ent decommission work trackable.
- 2026-03-05: started `app/profile` migration slice by replacing direct `ent.IsNotFound` checks with `framework/dberrors.IsNotFound` in `profile.go` and `profile_media.go` while preserving current Ent query behavior.
- 2026-03-05: started `app/jobs` migration slice by removing explicit Ent constructor threading in mail jobs (`NewEmailUpdateProcessor` and `NewUpdateEmailSender` now container-only), reducing composition-root Ent coupling while keeping behavior unchanged.
- 2026-03-05: continued `app/jobs` migration slice by removing Ent from `notifications` processors (`DeleteStaleNotificationsProcessor` now SQL-backed with dialect binding; `AllDailyConvo` processor no longer accepts Ent/ProfileService deps) and added SQLite unit coverage for stale-notification cleanup.
- 2026-03-05: continued `app/jobs` migration slice by removing Ent query/predicate dependencies from `email_update_sender` audience + sent-email persistence paths (now SQL-based with dialect binding against `notification_permissions`, `notifications`, and `sent_emails`).
- 2026-03-05: reduced auth-layer Ent coupling by changing `NewAuthClient` to accept an already-selected store, keeping Ent-specific selection logic in container/selector wiring instead of the core auth client surface.
- 2026-03-05: cleaned stale Ent-era commented query blocks from `app/jobs/email_update_sender.go`; `app/jobs` runtime sources now show no non-test `db/ent` imports in boundary scans.
- 2026-03-05: updated auth store selector fallback to be Bob-first for unknown `PAGODA_AUTH_STORE` values when DB is available (Ent fallback only when DB is unavailable), with updated selector unit coverage.
- 2026-03-05: switched router dependency wiring to Bob-backed profile notification count store (`app/web/wiring.go` now uses `NewProfileServiceWithDeps(..., NewBobNotificationCountStore(...))`), removing an Ent-default path from web route composition.
- 2026-03-05: added and enforced `app/jobs` Ent-boundary guard (`tools/scripts/check-app-jobs-no-ent-imports.sh`) in precommit, and marked the `app/jobs` Phase 5 cutover item complete.
- 2026-03-05: removed Ent types from storage runtime interfaces by introducing storage-owned image DTOs (`framework/repos/storage.ImageFile*`) and moving Ent-to-storage mapping into `app/profile`.
- 2026-03-05: removed Ent fallback as default auth runtime behavior when DB is missing; selector now returns an explicit unavailable auth store unless `PAGODA_AUTH_STORE=ent` is requested.
- 2026-03-05: hardened precommit script path resolution (`tools/scripts/precommit-tests.sh`) so checks run correctly from any working directory.
- 2026-03-05: added composition-root guardrail `tools/scripts/check-composition-no-ent-module-wiring.sh` and wired it into precommit to prevent Ent-backed module stores from re-entering runtime wiring in `cmd/*` and `app/web/wiring.go`.
- 2026-03-05: updated CI workflow (`.github/workflows/test.yml`) to run the stateless precommit quality gate, which now enforces Bob generated-code drift and boundary checks on push/PR.
- 2026-03-05: removed direct Atlas apply invocation from `make:scaffold --migrate`; scaffold now delegates migration execution through the existing `ship db:migrate` command path (Goose-native flow).
- 2026-03-05: removed dead Atlas execution/install fallback code paths from Ship CLI runtime (`runAtlasCmd` + `InstallAtlasBinary`), keeping migration command execution focused on Goose.
- 2026-03-05: fixed `ship upgrade` target file path after CLI refactor (`tools/cli/ship/internal/cli/cli.go`) and kept command tests aligned to the new location.
- 2026-03-05: removed hidden Ent-default profile notification-count fallback in `NewProfileServiceWithDeps`; profile now requires explicit notification store wiring for migrated paths instead of silently selecting Ent.
- 2026-03-05: added storage-interface boundary guard `tools/scripts/check-storage-interface-boundary.sh` and wired it into precommit to block Ent type leakage back into storage runtime interfaces.
- 2026-03-05: migrated login onboarding check to `AuthClient.GetIdentityByUserID` (store-backed/Bob path) and removed controller-level `ProfileService` instantiation for post-login onboarding routing.
- 2026-03-05: migrated profile read hotspots to DB/Bob path when DB is available: `IsProfileFullyOnboardedByUserID` and `GetProfilePhotoThumbnailURL` now use `db/gen/profile.go` queries first, with Ent fallback retained.
- 2026-03-05: added profile query registry entries (`db/queries/profile.sql`) and sqlite unit coverage in both `db/gen/profile_test.go` and `app/profile/profile_db_reads_test.go`.
- 2026-03-05: removed Ent selection path from auth store runtime selector; `selectAuthStore` is now Bob/DB-only with explicit unavailable-store behavior when DB is missing.
- 2026-03-05: removed dead Ent auth-store implementation (`entAuthStore` + selector wiring); foundation auth persistence path is now Bob-first with unavailable-store fail-fast when DB is absent.
- 2026-03-05: removed Ent fallback behavior from already-migrated profile read paths: `IsProfileFullyOnboardedByUserID` now requires DB (`ErrProfileDBNotConfigured` when missing), and `GetProfilePhotoThumbnailURL` is DB+Bobsql-only with default-image fallback when dependencies are unavailable.
- 2026-03-05: switched controller integration harness paid-subscriptions wiring from app Ent store to module SQL store (`paidsubscriptions.NewSQLStore`) to match runtime composition and reduce Ent test-path coupling.
- 2026-03-05: switched controller integration harness notifications runtime deps to DB-first mode (`ORM: nil`) to match runtime composition and reduce Ent coupling in integration wiring.
- 2026-03-05: removed `app/subscriptions` wrapper package and inlined product-plan domain mapping into web controllers (`subscription_plan.go`) to reduce app-layer indirection before full Ent decommission.
- 2026-03-05: fixed Stripe billing-portal source-of-truth mismatch by adding `GetStripeCustomerIDByProfileID` to `modules/paidsubscriptions` and switching portal-session creation to read customer IDs from module-owned `subscription_customers` storage instead of profile Ent fields.
- 2026-03-05: migrated profile settings read/write paths to DB query helpers (`db/gen/profile.go` + `db/queries/profile.sql`) and updated `ProfileService` settings methods (`GetProfileSettingsByID`, `UpdateProfileBio`, `UpdateProfilePhone`, `MarkProfileFullyOnboarded`) to SQL-first behavior.
- 2026-03-05: added sqlite coverage for new profile settings query helpers (`db/gen/profile_test.go::TestProfileSettingsQueries_SQLite`) and extended paidsubscriptions SQL tests to cover profile->stripe customer reverse lookup.
- 2026-03-05: made registration flow DB-first (`RegisterUserWithProfile` now uses SQL transaction path when `*sql.DB` is available, with Ent fallback retained only for transitional compatibility), reducing runtime dependence on Ent transaction APIs in signup path.
- 2026-03-05: migrated profile friendship read paths to SQL-first behavior (`GetFriends`, `AreProfilesFriends`) via new Bob query helpers and profile-friends SQL queries, with Ent fallback retained for transitional compatibility.
- 2026-03-05: added Bob query coverage for friendship lookup paths in `db/gen/profile_test.go` and validated full precommit suite after profile SQL-first cutover.
- 2026-03-05: migrated profile gallery/profile-page read paths to SQL-first behavior (`GetPhotosByProfileByID`, `GetProfileByID`) via new `db/gen/profile` helpers (`GetProfileCoreByID`, `GetProfileImageByProfileID`, `GetProfilePhotosByProfileID`) and storage DTO mapping from SQL rows.
- 2026-03-05: added SQL query/test coverage for profile core + profile-image joins in `db/gen/profile_test.go` and added mapper coverage for SQL photo-size row grouping in `app/profile/storage_image_mapper_test.go`.
- 2026-03-05: enforced profile package line-budget compliance by extracting `IsProfileFullyOnboarded` into `app/profile/profile_helpers.go` (keeping `app/profile/profile.go` <= 500 lines) so precommit doctor gates remain green.
- 2026-03-05: migrated friendship write paths to SQL-first behavior (`LinkProfilesAsFriends`, `UnlinkProfilesAsFriends`) via new `db/gen/profile` exec helpers and DB queries (`link/unlink_profiles_as_friends_*`), with Ent fallback retained for transitional compatibility.
- 2026-03-05: added sqlite coverage for friendship write helpers (`TestProfileFriendshipWriteQueries_SQLite`) and revalidated full precommit quality gate after line-budget and profile SQL-first updates.
- 2026-03-06: removed Ent schema bootstrap from server-DB runtime path in `app/foundation/container.go` by introducing SQL migration application from `db/migrate/migrations` with migration-version tracking; embedded DB mode temporarily keeps Ent schema fallback until SQLite migration parity is complete.
- 2026-03-06: removed legacy `make ent-gen` target from root Makefile and updated agent workflow docs to use `ship db:generate` for DB codegen.
- 2026-03-06: converted `app/profile/profile_test.go` integration assertions and fixture setup to DB-first SQL paths (no Ent imports in the test package) and added DB helper `tests.LinkFriendsDB` plus `tests.CreateTestContainerPostgresDB` for DB-only integration harness usage.
- 2026-03-06: removed Ent bootstrap from `framework/tests` helpers by making `CreateTestContainerPostgresDB` apply core SQL migrations directly (`db/migrate/migrations`) with deterministic version tracking; deleted unused Ent-only test helper functions from `framework/tests/tests.go`.
- 2026-03-06: refactored `ship make:model` to Bob/Goose-native scaffolding (writes `db/queries/<model>.sql` + next-step guidance) and removed Ent codegen invocation from the model generator path, with CLI/generator tests updated accordingly.
- 2026-03-06: deleted obsolete CLI integration smoke test `tools/cli/ship/tests/integration/ent_generate_integration_test.go` after `make:model` moved to Bob-first query scaffolding.
- 2026-03-06: removed Ent schema bootstrap artifacts from `ship new` scaffolding (`db/schema/user.go` + Ent go.mod pin) and made doctor enforce Bob-first core DB scaffold (`db/queries`, `db/bobgen.yaml`) instead.
- 2026-03-06: updated architecture/agent docs to describe Bob+Goose as the canonical data-model path (replacing stale Ent-first wording).

---

## Deletion Conditions

Delete this file when all are true:

- [ ] All phases complete.
- [ ] No active Ent runtime dependency.
- [ ] Permanent docs fully reflect Goose + Bob workflows.
- [ ] Temporary transition tasks are no longer referenced.


## Phase Completion Rule

A phase can be marked complete only when:
- [ ] All phase checkboxes are done.
- [ ] All phase acceptance criteria are done.
- [ ] Global Test Gate is satisfied for all changes in the phase.
- [ ] Execution log contains a dated completion note.
