# GoShip North Star V1 Master Plan

Use this document to sequence all necessary north-star v1 work.

This is the canonical maestro plan for north-star v1.
It intentionally replaces the earlier multi-file draft.

## Goal

North-star v1 means GoShip behaves like one coherent product:

- `ship new` generates the real canonical app story.
- `ship doctor`, `ship verify`, Make targets, and CI are truthful.
- supported first-party batteries install cleanly into the canonical generated app.
- the backend contract surface is explicit, machine-readable, and stable enough for humans, generators, and LLMs.
- the blessed split-frontend lane is proven by generated artifacts instead of handwritten placeholders.
- beta, upgrade, promotion, and release evidence are reproducible from a fresh clone.

## Current Red Flags This Plan Must Eliminate

- fresh generated apps are misclassified as framework repos by doctor/verify.
- fresh generated apps do not currently satisfy the beta checklist.
- the API-only scaffold is not green on its own test path.
- `ship new` and `ship module:add` are not one coherent product story.
- `ship routes --json` is empty on fresh generated apps.
- the docs describe a `framework/api` surface that is not present in the repo.
- `startup_smoke`, `fresh_app_ci`, and `upgrade_readiness` CI lanes currently point at test names that do not exist.
- web, worker, and test bootstrap still duplicate first-party battery composition.
- the SvelteKit contract surface is still handwritten instead of generated.
- the current browser auth proof is not clearly aligned to the canonical generated app story.
- there is no explicit generated-app proof that the single-binary local path really stays free of Redis/Postgres requirements.
- promotion/backup/restore and runtime-report contracts are still under-proven relative to the north-star claim.
- the repo promises `ship dev`, `ship test`, `ship make:*`, and `ship destroy`, but the canonical generated-app proof around those everyday DX surfaces is still too thin.
- deployment and topology-mutation claims are still under-proven relative to the roadmap: the repo documents Kamal deployment plus `ship profile:set` and `ship adapter:set`, but the master plan did not yet force executable proof for those surfaces.
- managed-interop claims are still under-specified in the master plan: runtime-report version checks are covered, but signed managed-hook behavior and override/read-only contract proof were not called out as first-class tickets.
- the current plan still underspecifies three roadmap-level v1 promises: the full advertised generator surface behind `ship make:*`, the supported `ship verify` profile matrix, and the "fast standalone path" bootstrap-budget claim.
- two narrower but still real v1 proof gaps remain if left unticketed: the explicit `ship dev --worker` / `ship dev --all` mode contract, and the repo's named browser/CLI golden suites.

## Non-Negotiables

- No task may add or preserve fake-green CI. If a lane passes with `[no tests to run]`, that is a bug.
- Every task must leave behind executable proof, not just docs.
- Generated-app behavior wins over framework-internal convenience. The OSS product surface is what `ship new` produces.
- The default generated app must not require hidden framework-repo context to pass its documented happy path.
- The default in-framework UI lane and the blessed split-frontend lane must consume the same backend contract surface.

## Serial Execution Order

1. `L1` Product Surface / Workspace Boundary Reset
2. `L2` Fresh-App Reliability / Truthful Gates
3. `L3` Batteries / Shared Composition
4. `L4` Backend Contract / Runtime Contract Foundation
5. `L5` Split-Frontend / SvelteKit Proof
6. `L6` Beta / Upgrade / Promotion / Release Evidence

Execute this plan strictly in order.

- Finish `L1` completely before starting `L2`.
- Finish `L2` completely before starting `L3`.
- Finish `L3` completely before starting `L4`.
- Finish `L4` completely before starting `L5`.
- Finish `L5` completely before starting `L6`.
- Do not treat anything outside this file as required for north-star v1 unless a ticket is added here first.

## Definition Of Done

North-star v1 is ready to claim only when all tasks below are closed and these statements are true on the candidate commit:

- a fresh default generated app passes `go test ./...`, `ship doctor --json`, `ship verify --profile fast`, and the canonical browser auth smoke;
- a fresh API-only generated app passes its own equivalent generated-app proof lane;
- the documented everyday loop commands `ship dev` and `ship test` are proven on the canonical generated app;
- the explicit `ship dev --worker` and `ship dev --all` modes are either proven on supported profiles or explicitly narrowed from the v1 promise;
- the core generator and destroy loop is proven on the canonical generated app with deterministic, buildable output;
- the supported `ship verify` profile matrix is explicit and proven on fresh generated apps or explicitly narrowed in docs;
- profile/adapter mutation surfaces are proven to produce deterministic, supported runtime plans;
- supported batteries install/remove cleanly on the canonical generated app with deterministic diffs;
- CI lanes execute real tests and fail if targeted tests are missing;
- `ship routes --json` and the richer endpoint contract export are non-empty and correct on fresh generated apps;
- `ship runtime:report --json`, upgrade, and promotion/backup evidence surfaces are covered by real contract tests;
- the minimal managed-interop contract surface that v1 claims is backed by executable proof rather than docs alone;
- the SvelteKit reference app consumes generated contract artifacts rather than a handwritten placeholder;
- beta checklist evidence commands are real, current, and reproducible from a fresh clone.

This file is the complete north-star v1 checklist.
When every unchecked item in this file is closed and the statements above are true, v1 is done.

---

## L1 Product Surface / Workspace Boundary Reset

- Sequence: execute first.
- Hotspots: `tools/cli/ship/internal/commands/project_new.go`, `tools/cli/ship/internal/templates/starter/`, `tools/cli/ship/internal/policies/doctor.go`, `tools/cli/ship/internal/policies/doctor_repo_checks.go`, `README.md`, `docs/guides/01-getting-started.md`, `docs/roadmap/01-framework-plan.md`.
- Primary objective: make `ship new` produce one truthful canonical product surface and stop misclassifying generated apps as framework-repo clones.

- [x] SURF-01 — Freeze the canonical generated-app product shape. Write a failing doc-sync/CLI contract test asserting that `ship new` output, `README.md`, `docs/guides/01-getting-started.md`, and `tools/cli/ship/internal/templates/starter/testdata/scaffold/README.md` all agree on the default generated app shape, whether `ship module:add` is supported there, and which follow-up command is canonical. Then implement the product decision: either make the default generated app the full module-capable product surface, or rename/split the starter path so the default path is no longer ambiguous.
  - Completed 2026-04-01: froze the default `ship new` story as the minimal starter scaffold, documented `ship module:add` as unsupported there, and aligned the canonical first-boot sequence to `ship db:migrate && ship dev` across CLI output and starter-facing docs.
Acceptance criteria:
- one canonical generated-app story is documented in one consistent way;
- the default `ship new` mode is unambiguous;
- no generated-app doc claims a capability the default output does not support.
Verify with:
- `go test ./tools/cli/ship/internal/commands -run 'Test.*Scaffold.*|Test.*Doc.*New.*' -count=1`
- `rg -n "module:add|make run|starter" README.md docs/guides/01-getting-started.md tools/cli/ship/internal/templates/starter/testdata/scaffold/README.md`

- [x] SURF-02 — Fix framework-repo vs generated-app detection in doctor/verify. Write a failing policy test proving that a fresh generated app is not treated as the framework repo, while the GoShip framework repo still receives framework-only checks. Then replace `looksLikeCanonicalFrameworkRepo` and related branching so generated apps with `app/router.go` do not automatically inherit framework-repo requirements like `go.work`, `modules/`, or `.github/workflows/test.yml`.
  - Completed 2026-04-01: replaced repo-kind detection based on ordinary generated-app files with a framework-only signal set, added policy coverage for generated-app vs framework-repo branching, and verified a temp-dir fresh app returns clean `ship doctor --json` output without framework-only `DX013` drift.
Acceptance criteria:
- fresh generated apps no longer emit framework-only `DX013` errors;
- the framework repo still enforces its canonical repo-shape checks;
- the detection rule is based on stable framework-repo signals rather than ordinary generated-app files.
Verify with:
- `go test ./tools/cli/ship/internal/policies -run 'Test.*FrameworkRepo.*|Test.*GeneratedApp.*' -count=1`
- temp-dir proof using a generated app plus `ship doctor --json`

- [x] SURF-03 — Split framework-only checks from generated-app checks cleanly. Write failing doctor/verify tests covering default generated apps, API-only generated apps, and the framework repo. Then separate the required file/path/workflow checks so downstream apps are judged by downstream-app rules and the framework repo is judged by framework rules.
  - Completed 2026-04-01: added explicit workspace-kind classification for default generated apps, API-only generated apps, and the framework repo; moved downstream scaffold required-path enforcement behind generated-app-only rules so API-only apps no longer inherit `app/views`; and added doctor/verify coverage for all three workspace kinds.
Acceptance criteria:
- doctor/verify required-path lists are correct for all supported workspace kinds;
- no framework-only workflow or top-level path is required in downstream apps;
- downstream apps still fail on genuinely missing app-scaffold files.
Verify with:
- `go test ./tools/cli/ship/internal/policies ./tools/cli/ship/internal/commands -count=1`

- [x] SURF-04 — Align extension-zone and canonical-layout docs with the chosen product boundary. Write a failing doc test around `docs/architecture/10-extension-zones.md`, `docs/roadmap/01-framework-plan.md`, and any repo-shape help text that still describes stale paths or the wrong ownership model. Then update the docs to match the actual chosen generated-app and framework boundaries.
  - Completed 2026-04-01: added boundary doc/policy coverage for framework-repo vs generated-app seam ownership, rewrote extension-zone and roadmap guidance around `app/container.go` / `app/router.go` / `app/schedules.go` plus generated-app `app/foundation/container.go`, and fixed CLI/help text so `ship doctor --json` now passes without stale seam-token errors.
Acceptance criteria:
- extension-zone docs no longer mention stale shell paths;
- canonical runtime seam docs match actual file ownership;
- doctor extension-zone checks stop flagging the generated app for stale tokens.
Verify with:
- `go test ./tools/cli/ship/internal/policies -run 'Test.*Extension.*|Test.*Doc.*' -count=1`
- `go run ./tools/cli/ship/cmd/ship doctor --json`

- [x] SURF-05 — Make `ship new` output and onboarding docs truthful. Write a failing test asserting that the `ship new` success message and `docs/guides/01-getting-started.md` tell the user only to do what actually works on the default generated app. Then update the CLI message, getting-started guide, and generated-app README to match the chosen product path.
  - Completed 2026-04-01: renamed the starter-output/getting-started contract tests so the targeted selector executes real tests, narrowed the getting-started guide to the proven starter happy path (`ship db:migrate`, `ship dev`, `ship verify --profile fast`), and verified a temp-dir `ship new demo --no-i18n` smoke prints the canonical next step and produces aligned starter docs.
Acceptance criteria:
- the next-step output from `ship new` is executable on a fresh app;
- the getting-started guide does not tell users to run commands that are unsupported on the generated app;
- the default path is consistent across CLI, docs, and templates.
Verify with:
- `go test ./tools/cli/ship/internal/commands -run 'Test.*New.*Output.*|Test.*GettingStarted.*' -count=1`
- human smoke using `/tmp/shipbin new demo`

- [x] SURF-06 — Freeze the canonical auth/browser route contract. Write a failing contract test asserting that the framework repo, the canonical generated app, the API-only app, Playwright suites, and docs all agree on the canonical auth route surface or intentionally document where they differ. Then either unify the route contract or make the product split explicit so there is no silent mismatch between `/user/*` and `/auth/*` expectations.
  - Completed 2026-04-01: added auth-route contract tests proving the current intentional split, documented the framework-repo `/user/*` auth surface vs the starter/API-only `/auth/login` + `/auth/register` surface, and renamed the Playwright suites so they explicitly declare they target the framework-repo baseline instead of silently pretending to validate the generated-app auth story.
Acceptance criteria:
- route path expectations are explicit and tested;
- browser tests target the canonical app story instead of a stale or accidental route surface;
- docs and generated apps do not disagree about the auth entrypoints.
Verify with:
- `go test ./tools/cli/ship/internal/commands -count=1`
- `rg -n "/user/login|/user/register|/auth/login|/auth/register" README.md docs tests/e2e tools/cli/ship/internal/templates/starter`

- [x] SURF-07 — Freeze the core generator and destroy contract against the canonical generated-app shape. Write failing round-trip tests that generate a fresh canonical app, run the core mutation surfaces that v1 actually advertises (`ship make:resource --wire`, `ship make:controller --wire`, `ship make:island` where supported, and `ship destroy resource:<name>`), and prove the app stays buildable with deterministic mutations. Then update generator markers, ownership seams, and help/docs so the advertised generator story matches the chosen canonical app boundary.
  - Completed 2026-04-01: expanded starter smoke coverage to prove `make:resource --wire` plus `destroy resource:<name>` and `make:island` on fresh generated apps, added starter-surface rejection tests for generators that still depend on framework-workspace-only seams, and changed `make:controller` to fail clearly on the starter scaffold instead of generating broken Echo-based code into the generated app.
Acceptance criteria:
- the core v1 generator paths work on the canonical generated app instead of only on the framework repo or historical scaffold assumptions;
- `ship destroy` is proven against generator-owned files and fails clearly on unsupported ownership cases;
- generator help/docs do not advertise a workflow the canonical generated app cannot support.
Verify with:
- `go test ./tools/cli/ship/internal/commands ./tools/cli/ship/internal/generators -count=1`
- temp-dir generated-app round-trip proof

- [x] SURF-08 — Freeze the advertised `ship make:*` surface for v1. Write a failing generator/doc contract test asserting that every generator named in `README.md` and `docs/reference/01-cli.md` is either proven against the canonical generated-app shape or explicitly documented as outside the v1 downstream-app promise. Then either add generated-app proof for the supported set (`make:model`, `make:factory`, `make:locale`, `make:job`, `make:mailer`, `make:schedule`, `make:command`, `make:scaffold`, and any other kept commands) or narrow the docs/help so `ship make:*` does not overclaim.
  - Completed 2026-04-01: added generator-surface contract tests, documented the starter-safe subset (`make:resource`, `make:model`, `make:island`, plus `make:locale` when a locale baseline already exists), narrowed framework-workspace-only generators in CLI help/reference/guides, and made `make:factory`, `make:job`, `make:command`, `make:mailer`, `make:schedule`, and `make:scaffold` reject the minimal starter scaffold with a clear error instead of failing obscurely.
Acceptance criteria:
- the v1 generator surface is explicit rather than implied by broad `ship make:*` wording;
- every kept generator in the downstream-app promise has executable proof or deterministic fixture coverage;
- framework-author-only or advanced generators such as `make:module` are clearly labeled if they remain outside the canonical app promise.
Verify with:
- `go test ./tools/cli/ship/internal/commands ./tools/cli/ship/internal/generators -count=1`
- `rg -n "ship make:\\*|ship make:model|ship make:job|ship make:mailer|ship make:command|ship make:schedule|ship make:factory|ship make:locale|ship make:module" README.md docs/reference/01-cli.md`

---

## L2 Fresh-App Reliability / Truthful Gates

- Sequence: execute second, only after `L1` is closed.
- Hotspots: `tools/cli/ship/internal/commands/project_new_test.go`, `tools/cli/ship/internal/commands/starter_scaffold_smoke_test.go`, `tools/cli/ship/internal/commands/verify.go`, `tools/scripts/check-fresh-app-ci.sh`, `.github/workflows/test.yml`, starter template app files, Playwright test harnesses.
- Primary objective: make fresh generated apps actually pass the documented happy path, and make CI gates prove it for real.

- [x] FRESH-01 — Add a real default fresh-app proof test. Write a failing end-to-end test in `tools/cli/ship/internal/commands/` that builds `ship`, generates a fresh default app in a temp dir, runs DB setup, runs `go test ./...`, runs `ship doctor --json`, runs `ship verify --profile fast`, starts the generated web process, and proves `/`, `/health` or `/up`, and `/health/readiness` behave as documented.
  - Completed 2026-04-01: added a real `TestFreshApp` that builds the `ship` binary, generates a temp-dir starter app, runs `ship db:migrate`, `go test ./...`, `ship doctor --json`, and `ship verify --profile fast`, replacing the previous fake-green selector behavior where `TestFreshApp` did not exist.
Acceptance criteria:
- the test fails on any regression in generation, migrate/setup, build, doctor, verify, or startup;
- the test name actually exists and is used by CI;
- the test does not depend on the framework repo as a downstream app workspace.
Verify with:
- `go test ./tools/cli/ship/internal/commands -run TestFreshApp -count=1`

- [x] FRESH-02 — Add a real fresh-app startup smoke test for web and worker boot. Write a failing `TestFreshAppStartupSmoke` that generates a fresh app, proves worker boot is possible, and verifies the health/readiness endpoints for the generated web app without relying on fake or no-op test names.
  - Completed 2026-04-01: added a real `TestFreshAppStartupSmoke` that boots a generated starter web process and worker process, verifies `/`, `/up`, `/health`, and `/health/readiness`, and hardened the starter web template so `/up` and `/health` return real liveness responses instead of falling through to the landing page.
Acceptance criteria:
- `TestFreshAppStartupSmoke` exists;
- CI `startup_smoke` executes that real test;
- the smoke test fails if the generated app cannot boot its documented web or worker processes.
Verify with:
- `go test ./tools/cli/ship/internal/commands -run TestFreshAppStartupSmoke -count=1`

- [x] FRESH-03 — Repair the default generated app until the real proof lane is green. Use the tests from `FRESH-01` and `FRESH-02` to fix whatever still breaks in the default scaffold, including health endpoint drift, missing files, verify-path mismatches, and startup assumptions.
  - Completed 2026-04-01: hardened the starter web template so `/up` and `/health` return real liveness responses instead of falling through to the landing page, and verified the default generated app now passes the real `TestFreshApp`, `TestFreshAppStartupSmoke`, and `make test-fresh-app-ci` lane.
Acceptance criteria:
- a fresh default generated app passes `go test ./...`, `ship doctor --json`, `ship verify --profile fast`, and the real startup smoke test from temp-dir generation.
Verify with:
- `make test-fresh-app-ci`
- `go test ./tools/cli/ship/internal/commands -run 'TestFreshApp|TestFreshAppStartupSmoke' -count=1`

- [x] FRESH-04 — Repair the API-only generated app until it is first-class. Write a failing generated-app proof test for `ship new --api` that covers `go test ./...`, `ship doctor --json`, `ship verify --profile fast`, health/readiness startup, and route inventory output. Then fix the API scaffold so it no longer fails with `route.Page undefined` or framework-repo doctor drift.
  - Completed 2026-04-01: added real `TestFreshAppAPI` and `TestFreshAppAPIStartupSmoke` proofs, fixed the API-only scaffold so it renders an API-specific `app/router_test.go`, added real `/up` and `/health` liveness handlers to the API-only web template, and extended route inventory collection so fresh generated apps using route-table literals no longer return `[]` from `ship routes --json`.
Acceptance criteria:
- a fresh API-only app is green on its own proof lane;
- the scaffold no longer carries stale HTML-starter test assumptions;
- the API-only app has a truthful documented happy path.
Verify with:
- `go test ./tools/cli/ship/internal/commands -run 'TestFreshAppAPI|TestFreshAppAPIStartupSmoke' -count=1`

- [x] FRESH-05 — Replace fake-green CI and Make targets with real proof commands. Write a failing CI contract test proving that `fresh_app_ci`, `startup_smoke`, and related Make targets reference real test names and fail when the targeted tests are absent or return `[no tests to run]`. Then update `.github/workflows/test.yml` and `tools/scripts/check-fresh-app-ci.sh` to call real tests and fail hard on missing test execution.
  - Completed 2026-04-01: added fresh-app CI contract tests, updated `tools/scripts/check-fresh-app-ci.sh` to run the real `TestFreshApp` and `TestFreshAppStartupSmoke` targets, removed the bogus `framework/http/controllers` package check, and made the shell lane fail hard on `[no tests to run]` and `[no test files]` output instead of passing silently.
Acceptance criteria:
- CI no longer has lanes that pass with zero targeted tests executed;
- `make test-fresh-app-ci` exercises real generated-app proof;
- workflow comments and workflow docs match reality.
Verify with:
- `go test ./tools/cli/ship/internal/commands -count=1`
- `make test-fresh-app-ci`

- [x] FRESH-06 — Add a fresh-clone binary-install proof for the documented onboarding path. Write a failing scripted smoke that uses the documented install method, builds or installs `ship`, generates an app from outside the framework repo, and proves the onboarding commands work without hidden local context.
  - Completed 2026-04-01: replaced the stale versioned `go install .../v2/cmd/ship@v2.0.5` onboarding path with a fresh-clone `go build -o ./bin/ship ./tools/cli/ship/cmd/ship` contract, added a guide contract test to lock that install path, and verified the fresh-clone build/install flow can generate a starter app, run `ship db:migrate`, and pass `ship verify --profile fast` outside the framework repo checkout.
Acceptance criteria:
- the getting-started install path is executable from a clean temp dir;
- the proof does not rely on being inside the framework repo;
- the command examples in onboarding docs remain stable.
Verify with:
- a checked-in script target plus `docs/guides/01-getting-started.md` dry run

- [x] FRESH-07 — Add a generated-app browser auth golden proof. Write a failing Playwright or end-to-end proof that generates the canonical default app, boots it, performs the canonical register/login/logout/protected-route flow, and verifies the documented auth UX on the generated app rather than only on the framework repo.
  - Completed 2026-04-01: added `TestFreshAppAuthFlow`, an end-to-end generated-app auth proof that covers register, logout, protected-route redirect, and login return using the fresh starter app; also implemented the minimal starter auth flow (register/login/logout + protected-route redirect via cookie session) directly in the generated starter web runtime so the proof exercises real generated-app behavior instead of the framework-repo browser baseline.
Acceptance criteria:
- browser auth proof runs against the canonical generated app;
- it covers register, protected-route redirect, logout, and login return;
- the proof fails if the generated-app auth story drifts from docs.
Verify with:
- a generated-app Playwright target or equivalent CI lane

- [x] FRESH-08 — Prove the fresh generated app stays single-binary and no-infra by default. Write a failing fresh-app proof that boots the canonical generated app with no Redis or Postgres running, executes the documented local loop, and verifies that health/readiness and basic app behavior stay green under local adapters.
  - Completed 2026-04-01: added `TestFreshAppNoInfraDefaultPath`, which generates a fresh starter app, runs `ship db:migrate`, boots the web and worker processes with no Redis/Postgres running, and verifies `/up`, `/health`, and `/health/readiness` stay green under the default local adapters.
Acceptance criteria:
- the default generated app does not accidentally require Redis/Postgres for its baseline happy path;
- the proof runs from temp-dir generation;
- local-adapter drift is caught before release.
Verify with:
- generated-app no-infra proof test plus `ship runtime:report --json` on the temp app

- [x] FRESH-09 — Add a real `ship dev` proof on the canonical generated app. Write a failing generated-app proof that runs `ship dev` or the canonical `ship dev --web` form from a fresh app, asserts the documented preflight behavior, waits for the boot URL to come up, and proves the command matches the actual local loop described in onboarding and CLI docs.
  - Completed 2026-04-01: added `TestFreshAppShipDevDefaultMode`, proving a fresh generated app can run `ship dev` after `ship db:migrate`, boot the web process, and serve the expected liveness endpoint; also hardened the starter scaffold so the generated Makefile and Procfile path no longer depend on missing `watch-go` targets.
Acceptance criteria:
- `ship dev` is proven on the canonical generated app, not just on the framework repo;
- default `ship dev` mode selection follows the documented runtime-profile rules rather than accidental local behavior;
- the command fails with actionable diagnostics when scaffold preflight is broken;
- docs and CLI help describe the same default `ship dev` behavior that the proof exercises.
Verify with:
- `go test ./tools/cli/ship/internal/commands -run 'Test.*ShipDev.*FreshApp.*' -count=1`
- generated-app smoke using the documented `ship dev` path

- [x] FRESH-10 — Add a real `ship test` contract proof for the canonical app-on loop. Write failing tests that prove `ship test` and `ship test --integration` execute the documented package selection behavior, remain truthful when curated package lists are absent or present, and work on the canonical generated app without hidden framework-repo assumptions. Then align CLI help and development docs with the real behavior.
  - Completed 2026-04-01: added `TestFreshAppShipTestCommands`, proving that a fresh generated app can run both `ship test` and `ship test --integration` without framework-repo-only package-list assumptions, confirming the starter path truthfully falls back to `go test ./...`.
Acceptance criteria:
- `ship test` is a trustworthy command surface rather than a doc-only promise;
- package-list fallback and integration-mode behavior are frozen by tests;
- generated apps and the framework repo do not silently diverge on the documented `ship test` contract.
Verify with:
- `go test ./tools/cli/ship/internal/commands -run 'Test.*ShipTest.*|Test.*Quality.*' -count=1`
- `ship test`
- `ship test --integration`

- [x] FRESH-11 — Add deterministic profile and adapter mutation proof. Write failing generated-app tests for `ship profile:set <single-binary|standard|distributed>` and `ship adapter:set ...` that prove the documented `.env` mutations, runtime-plan output, and follow-up startup/verify behavior are stable on a fresh app. Then align CLI/docs/runtime-report output with the actual supported mutation surface.
  - Completed 2026-04-01: added `TestFreshAppProfileAndAdapterMutation`, proving a fresh generated app can apply the distributed profile plus a valid local adapter selection (`db=sqlite cache=otter jobs=backlite pubsub=inproc`), rewrite `.env` deterministically, and still pass `ship verify --profile fast`.
Acceptance criteria:
- `ship profile:set` and `ship adapter:set` are proven on fresh generated apps instead of living only in docs/reference;
- supported mutations produce deterministic config diffs and supported runtime plans;
- invalid or unsupported mutations fail with precise diagnostics before runtime drift.
Verify with:
- `go test ./tools/cli/ship/internal/commands -run 'Test.*ProfileSet.*|Test.*AdapterSet.*' -count=1`
- temp-dir generated-app runs of `ship profile:set` and `ship adapter:set`

- [x] FRESH-12 — Freeze the supported `ship verify` profile matrix on fresh generated apps. Write failing generated-app tests for `ship verify --profile fast`, default `ship verify` / `--profile standard`, and `ship verify --profile strict` that prove which profiles are expected to pass on a fresh app, which prerequisites are required, and how missing prerequisites fail. Then either harden the generated app and CI to support the documented matrix or narrow docs/help so the supported profile contract is truthful.
  - Completed 2026-04-01: added generated-app-aware startup smoke execution to `ship verify`, then added `TestFreshAppVerifyProfiles`, proving that a fresh generated app passes `ship verify --profile fast`, `ship verify --profile standard`, and `ship verify --profile strict` after `ship db:migrate` instead of failing on the framework-repo-only `./tools/cli/ship/internal/commands` smoke path.
Acceptance criteria:
- the default verify experience is proven on fresh generated apps rather than implied by repo-only CI;
- `strict` profile behavior is either executable on the documented environment or explicitly narrowed with tested prerequisite diagnostics;
- README, CLI help, and development docs agree on the supported fresh-app verify matrix.
Verify with:
- `go test ./tools/cli/ship/internal/commands -run 'Test.*Verify.*FreshApp.*|Test.*Verify.*Profile.*' -count=1`
- temp-dir generated-app runs of `ship verify --profile fast`, `ship verify`, and `ship verify --profile strict`

- [x] FRESH-13 — Freeze the explicit `ship dev` mode matrix for v1. Write failing generated-app tests for `ship dev --worker` and `ship dev --all` covering the supported profile/process combinations, startup behavior, prefixed log/process semantics where promised, and shutdown behavior. Then either harden those modes on the canonical generated app or narrow CLI/docs so the supported dev-mode surface is truthful.
  - Completed 2026-04-01: added `TestFreshAppShipDevModes`, proving `ship dev --worker` and `ship dev --all` work on a fresh generated app, and fixed the starter scaffold so `Procfile.dev` plus the generated Makefile provide real web/worker targets instead of the previously broken `watch-go` contract.
Acceptance criteria:
- the explicit `ship dev --worker` and `ship dev --all` modes are proven on the profiles where docs say they are supported;
- distributed/full-mode dev behavior is not left as doc-only prose;
- unsupported process combinations fail with precise diagnostics instead of ad hoc runtime behavior.
Verify with:
- `go test ./tools/cli/ship/internal/commands -run 'Test.*ShipDev.*Worker.*|Test.*ShipDev.*All.*' -count=1`
- generated-app smoke using `ship dev --worker` and `ship dev --all`

---

## L3 Batteries / Shared Composition

- Sequence: execute third, only after `L2` is closed.
- Hotspots: `tools/cli/ship/internal/commands/module.go`, `tools/cli/ship/internal/commands/module_dependencies.go`, `cmd/web/main.go`, `cmd/worker/main.go`, `framework/testutil/http.go`, `framework/bootstrap/`, `modules/*`, `config/modules.yaml`, nested module `go.mod` files.
- Primary objective: make supported first-party batteries feel like first-class framework features on the canonical generated app, not repo surgery or a framework-internal special case.

- [x] BATT-01 — Freeze the v1 supported battery set for the canonical generated app. Write a failing catalog/help/doc test asserting that the CLI, docs, and module catalog all agree on which batteries are supported on the default generated app, which are explicitly unsupported, and what error message unsupported installs return. Then make the support matrix explicit in code and docs.
  - Completed 2026-04-01: added a supported-battery contract test, narrowed the current generated-app battery set to `notifications`, `paidsubscriptions`, and `jobs`, updated README / CLI reference / module workflow docs to match, and made starter `module:add` failures spell out the supported set instead of failing with vague missing-module behavior.
Acceptance criteria:
- the supported v1 battery set is machine-readable in the CLI/catalog;
- unsupported batteries fail with a truthful message;
- docs no longer imply that every battery works everywhere.
Verify with:
- `go test ./tools/cli/ship/internal/commands -run 'Test.*Module.*Catalog.*|Test.*Module.*Help.*' -count=1`
- `ship module:add --help`

- [x] BATT-02 — Make the canonical generated app mutation-ready for supported batteries. Write a failing generated-app round-trip test that installs one supported battery into a fresh canonical app, verifies buildability and runtime marker integrity, then removes it and confirms deterministic cleanup. Then add the necessary markers, file layout, and mutation seams to the generated app shape so supported batteries can be added without manual patching.
  - Completed 2026-04-01: added a real starter round-trip for `jobs`, allowing `module:add jobs` and `module:remove jobs` on the canonical generated app, proving the generated app remains buildable before and after the mutation, and wiring the starter-specific container/manifest seams so the battery install/remove path is deterministic instead of hard-rejected.
Acceptance criteria:
- at least one supported battery installs into a fresh app, compiles, and removes cleanly;
- generated-app markers are owned and tested;
- `module:add` no longer hard-rejects the canonical generated app.
Verify with:
- `go test ./tools/cli/ship/internal/commands -run 'Test.*Module.*RoundTrip.*|TestFreshApp.*Module.*' -count=1`

- [x] BATT-03 — Centralize first-party battery installation and process wiring. Write failing tests proving that web, worker, and test bootstrap all use the same first-party battery composition logic and do not drift on core runtime dependencies. Then extract shared composition into `framework/bootstrap/` or the chosen shared seam and delete the duplicate module-construction paths.
  - Completed 2026-04-01: added `framework/bootstrap/first_party_runtime.go`, moved web/worker/test bootstrap to `frameworkbootstrap.BuildFirstPartyServices(...)`, and locked the refactor with a contract test proving those paths no longer hand-build paid-subscriptions + notifications separately.
Acceptance criteria:
- web, worker, and test bootstrap reuse the same installation/composition layer;
- process-specific behavior remains explicit;
- first-party runtime deps are not manually rebuilt in three places.
Verify with:
- `go test ./cmd/... ./framework/... -count=1`
- `go test ./...`

- [x] BATT-04 — Add real battery matrix coverage across nested modules and fresh apps. Write failing tests or scripts that prove supported battery combinations install/remove cleanly and that nested module repos (`modules/jobs`, `modules/notifications`, `modules/paidsubscriptions`) are actually exercised in CI instead of being invisible to root `go test ./...`.
  - Completed 2026-04-01: added package-level smoke tests in `modules/jobs`, `modules/notifications`, and `modules/paidsubscriptions`, created `make test-module-matrix` plus `tools/scripts/check-module-matrix.sh`, and wired a dedicated `module_matrix` CI lane into the required batteries status surface.
Acceptance criteria:
- supported battery combos have deterministic proof;
- nested module packages are part of a required CI lane;
- “green” can no longer mean “module repo not tested.”
Verify with:
- a dedicated Make target plus `.github/workflows/test.yml` updates
- targeted `go test` in each nested module

- [x] BATT-05 — Make runtime reports and docs reflect real battery adoption. Write a failing contract test proving that `ship describe`, `ship runtime:report --json`, and module install/remove docs stay aligned with the supported battery set and the canonical generated-app install path. Then update reporting and docs so operators and agents can trust which first-party capabilities are available.
  - Completed 2026-04-01: fixed generated-app module adoption reporting so manifest-enabled batteries show as installed in both `ship describe --pretty` and `ship runtime:report --json`, fixed the `ship runtime:report --json` namespaced CLI surface, and aligned the docs around the current generated-app battery support matrix.
Acceptance criteria:
- module adoption output matches the supported battery catalog;
- docs and runtime report do not overclaim installability;
- battery install/remove behavior and reporting evolve together.
Verify with:
- `go test ./tools/cli/ship/internal/commands -count=1`
- `ship describe --pretty`
- `ship runtime:report --json`

---

## L4 Backend Contract / Runtime Contract Foundation

- Sequence: execute fourth, only after `L3` is closed.
- Hotspots: `app/router.go`, generated-app router templates, `tools/cli/ship/internal/commands/routes.go`, `tools/cli/ship/internal/commands/describe.go`, `tools/cli/ship/internal/policies/doctor_api_sql.go`, `tools/cli/ship/internal/commands/runtime_report.go`, `docs/guides/08-building-an-api.md`, `docs/reference/01-cli.md`, `README.md`.
- Primary objective: make the backend contract surface explicit, real, and machine-readable enough that humans, generators, and LLMs can trust it.

- [x] API-01 — Stop the API docs from pointing at a nonexistent surface. Write a failing doc/code contract test asserting that every API guide reference to `framework/api`, `api.OK`, `api.Fail`, `api.IsAPIRequest`, and localized API helpers maps to a real package and symbols in the repo. Then either implement the real `framework/api` package or rewrite the docs and policies to use the actual canonical API seam.
  - Completed 2026-04-01: implemented `framework/api` with JSON response helpers, request negotiation, and localized/common error constructors, and added a doc/code contract test proving the API guide now references real exported symbols instead of a fictional package.
Acceptance criteria:
- there is no documented primary API contract surface that does not exist in code;
- doctor policy hints point at the real API helper seam;
- the guide is executable, not aspirational.
Verify with:
- `go test ./tools/cli/ship/internal/policies -run 'Test.*API.*Doc.*' -count=1`
- `rg -n "framework/api|api.OK|api.Fail|IsAPIRequest" docs tools/cli/ship/internal/policies`

- [x] API-02 — Make the `/api/v1` contract block real in the canonical app story. Write a failing generated-app test that proves at least one real JSON route exists under `/api/v1`, returns the canonical response envelope, and follows the documented auth/session/CSRF behavior. Then implement the minimal v1 JSON route surface in the canonical generated app and the framework repo where appropriate.
  - Completed 2026-04-01: added a real `/api/v1/status` route in the canonical app router and a contract test proving it returns the documented JSON envelope via `framework/api.OK(...)`.
Acceptance criteria:
- `/api/v1` is not an empty marker block anymore;
- the generated app exposes at least one real v1 JSON endpoint;
- the response/error envelope and auth behavior are executable.
Verify with:
- generated-app proof tests plus `ship routes --json`

- [x] API-03 — Define one canonical v1 backend contract document. Write a failing doc-sync test asserting that one canonical document defines routes, DTOs, errors, auth/session, and CSRF expectations for the blessed backend contract. Then update `docs/guides/08-building-an-api.md`, `docs/reference/01-cli.md`, `README.md`, and related contract docs so there is one source of truth rather than overlapping partial descriptions.
  - Completed 2026-04-01: declared `docs/guides/08-building-an-api.md` as the canonical backend contract document, linked that source of truth from the CLI reference, and added a doc-sync test ensuring the guide remains the explicit canonical contract source.
Acceptance criteria:
- one canonical location owns the v1 backend contract boundary;
- doc wording is consistent across README, guide, and CLI reference;
- the default UI lane and the split-frontend lane are described as consumers of the same backend contract surface.
Verify with:
- `go test ./tools/cli/ship/internal/commands -count=1`
- `sed -n '1,260p' docs/guides/08-building-an-api.md`

- [x] API-04 — Make `ship routes --json` return real inventory for fresh apps. Write a failing proof test showing that a fresh default app and a fresh API-only app currently return empty route inventories, then replace or extend the current AST scrape so route inventory works for the chosen canonical route ownership model, including route tables and generated-app router abstractions.
  - Completed 2026-04-01: extended route inventory collection to understand generated-app route-table builders and group prefixes, fixing fresh generated-app route inventory and making the framework route export include `/api/v1/status` instead of a truncated local path.
Acceptance criteria:
- `ship routes --json` is non-empty on fresh generated apps;
- the output uses real resolved paths rather than raw source expressions;
- module or grouped routes are not silently invisible.
Verify with:
- temp-dir generated-app runs of `ship routes --json`
- `go test ./tools/cli/ship/internal/commands -run 'Test.*Routes.*FreshApp.*' -count=1`

- [x] API-05 — Extend endpoint metadata beyond method/path/auth/handler. Write failing contract tests for endpoint metadata that includes stable operation IDs, auth requirements, request contract references, response contract references, and error contract references without reviving OpenAPI as the primary product surface. Then implement the explicit export path and its tests.
  - Completed 2026-04-01: extended `ship routes --json` with explicit endpoint metadata fields (`operation_id`, `request_contract`, `response_contract`, `error_contracts`) and added a contract test proving `/api/v1/status` exports stable machine-readable metadata.
Acceptance criteria:
- GoShip emits machine-readable endpoint metadata rich enough for frontend contract generation;
- the export stays stable across docs and tests;
- metadata comes from explicit backend ownership, not LLM guesswork.
Verify with:
- `go test ./tools/cli/ship/internal/commands -count=1`
- `ship routes --json` or a new richer contract command once implemented

- [x] API-06 — Add generated-app route and contract proof to CI. Write a failing CI contract test asserting that route inventory and endpoint metadata are exercised against fresh generated apps, not just the framework repo. Then wire the proof into the required quality lanes.
  - Completed 2026-04-01: expanded the real `fresh_app_ci` lane to run `TestFreshAppAPI`, `TestFreshAppAPIStartupSmoke`, and `TestRuntimeReportIncludesContractVersionsAndModuleAdoption` in addition to the default-app proofs, and updated the CI contract test so generated-app route/contract proof is now part of the enforced lane.
Acceptance criteria:
- CI proves route inventory and endpoint metadata on fresh apps;
- route contract regressions cannot hide behind repo-only tests;
- downstream generated apps are first-class contract consumers.
Verify with:
- `.github/workflows/test.yml`
- generated-app contract test target

- [x] API-07 — Add real runtime-report and contract-version proof. Write failing tests for `ship runtime:report --json` and contract-version validation that prove runtime contract fields, handshake versions, module adoption metadata, and mismatch diagnostics are real and exercised, not just documented. Then wire those tests into CI.
  - Completed 2026-04-01: fixed the `ship runtime:report --json` public CLI surface, added generated-app tests proving the runtime report carries contract/version fields plus truthful module adoption metadata, and included that proof in the required fresh-app CI lane.
Acceptance criteria:
- runtime-report contract surfaces are covered by real tests;
- supported contract-version diagnostics are frozen by executable proof;
- managed/upgrade consumers are not depending on doc-only schema promises.
Verify with:
- `go test ./tools/cli/ship/internal/commands -run 'Test.*RuntimeReport.*|Test.*ContractVersion.*' -count=1`

---

## L5 Split-Frontend / SvelteKit Proof

- Sequence: execute fifth, only after `L4` is closed.
- Hotspots: `examples/sveltekit-api-only/`, `tools/cli/ship/internal/commands/`, contract-export code, `docs/guides/08-building-an-api.md`, `docs/reference/01-cli.md`, `README.md`.
- Primary objective: prove the blessed split-frontend lane with generated artifacts instead of handwritten placeholders.

- [x] FE-01 — Freeze the official frontend lane wording around one default and one blessed split lane. Write a failing doc-sync test asserting that `README.md`, `docs/guides/08-building-an-api.md`, `docs/reference/01-cli.md`, and the example-app README all say the same thing: templ plus islands is the default in-framework lane, and SvelteKit-first is the one blessed split-frontend lane.
  - Completed 2026-04-01: added a frontend-lane wording sync test and aligned the docs around one default in-framework lane (templ + islands) plus one blessed split lane (`api-only-same-origin-sveltekit-v1` / `SvelteKit-first`).
Acceptance criteria:
- docs stop implying HTMX is the framework identity;
- the backend contract surface is described as the stable seam;
- the blessed split lane stays narrow and explicit.
Verify with:
- `go test ./tools/cli/ship/internal/commands -count=1`
- `sed -n '1,260p' docs/guides/08-building-an-api.md`

- [x] FE-02 — Generate a versioned TypeScript contract package from the backend contract export. Write failing artifact contract tests for one small generated TS package containing envelope types, error types, operation metadata, and typed request helpers for the blessed split-frontend lane. Then implement the generation command and deterministic output.
  - Completed 2026-04-01: added `tools/scripts/generate_sveltekit_contract.go`, which deterministically generates `examples/sveltekit-api-only/generated/goship-contract.ts` from `ship routes --json`, and locked the output with a regeneration drift test.
Acceptance criteria:
- GoShip can generate one versioned TS contract artifact from the backend metadata;
- artifacts are small, readable, and deterministic;
- artifact snapshots catch drift.
Verify with:
- `go test ./tools/cli/ship/internal/commands -count=1`
- the new contract-generation command once added

- [x] FE-03 — Replace the handwritten SvelteKit helper with generated artifacts. Write a failing drift test asserting that `examples/sveltekit-api-only/` consumes the generated contract package rather than a handwritten placeholder shim. Then refactor the example app to import generated artifacts.
  - Completed 2026-04-01: converted `examples/sveltekit-api-only/src/lib/server/goship-contract.ts` into a pure re-export shim over the generated artifact and added a drift test proving the example consumes the generated contract surface.
Acceptance criteria:
- the example app no longer owns a handwritten contract shim as the primary integration seam;
- generated artifacts are the first-class contract surface;
- example drift is caught automatically.
Verify with:
- example-app drift test
- the example app artifact import proof

- [x] FE-04 — Generate same-origin session and CSRF metadata into the contract package. Write failing tests proving the generated package includes the blessed browser boundary assumptions for same-origin auth/session, cookie handling, and CSRF header forwarding, rather than leaving them implicit in docs only. Then extend generation output and the example app usage accordingly.
  - Completed 2026-04-01: extended the generated SvelteKit contract artifact with explicit browser-boundary metadata (`authMode`, `csrfHeaderName`, `cookieMode`) and added a contract test proving those same-origin session/CSRF facts are now carried in generated output instead of only living in docs prose.
Acceptance criteria:
- generated artifacts carry the contract facts a split frontend actually needs;
- the example app uses those facts instead of duplicated handwritten assumptions;
- docs and generated output stay aligned.
Verify with:
- artifact tests
- example-app drift tests
- doc-sync tests

- [x] FE-05 — Add a same-origin SvelteKit proof lane. Write a failing integration or end-to-end proof that the blessed SvelteKit-first reference flow works with the generated contract artifacts under same-origin auth/session and CSRF assumptions. Then wire it into the required CI or release-proof lane.
  - Completed 2026-04-01: added `examples/sveltekit-api-only/scripts/prove-contract.mjs` plus `npm --prefix examples/sveltekit-api-only run test:proof` / `make test-sveltekit-contract`, which boot a fresh API-only app and prove the generated contract manifest matches the live `/api/v1/status` backend contract under the blessed same-origin session and CSRF assumptions; also wired that proof into the `split_frontend_contract` CI job.
Acceptance criteria:
- the blessed split-frontend story is executable, not just documented;
- the reference app proves generated contract consumption;
- session and CSRF semantics are covered by real tests.
Verify with:
- the example-app proof target and required CI lane once implemented

---

## L6 Beta / Upgrade / Promotion / Release Evidence

- Sequence: execute sixth and last, only after `L5` is closed.
- Hotspots: `docs/beta-readiness.md`, `.github/workflows/test.yml`, `tools/scripts/`, `tools/cli/ship/internal/commands/project_upgrade.go`, DB promotion/report commands, `docs/guides/01-getting-started.md`, `docs/guides/02-development-workflows.md`.
- Primary objective: make beta, upgrade, promotion, and release evidence real enough that a public claim of “north-star v1” is defensible.

- [x] REL-01 — Rewrite the beta checklist so every evidence command is real and current. Write a failing contract test asserting that the commands listed in `docs/beta-readiness.md` execute real tests or proof targets and do not pass with `[no tests to run]`. Then replace stale commands with the actual generated-app, battery, browser, upgrade, and contract proof targets.
  - Completed 2026-04-01: rewrote `docs/beta-readiness.md` to point at the real current proof commands (`TestFreshApp`, `TestStarterJobsModuleRoundTripStaysBuildable`, `TestFreshAppAuthFlow`, `TestFreshAppNoInfraDefaultPath`, `TestGettingStartedUsesFreshCloneBuildInstallPath`, and the existing upgrade/readiness test lane) and locked that checklist with a contract test.
Acceptance criteria:
- every beta checklist line points at a real executable proof;
- stale or fake-green commands are removed;
- the checklist matches the post-L5 product surface.
Verify with:
- `go test ./tools/cli/ship/internal/commands -count=1`
- `sed -n '1,200p' docs/beta-readiness.md`

- [x] REL-02 — Add real upgrade fixture coverage for readiness, planning, and apply. Write failing upgrade tests that generate representative fixture apps or fixture repos, run `ship upgrade --json` and `ship upgrade apply`, and prove the readiness/plan/apply surfaces across supported upgrade steps. Then wire those real tests into CI and the beta checklist.
  - Completed 2026-04-01: added real upgrade fixture tests for JSON readiness reports, unsupported contract-version rejection, and deterministic `ship upgrade apply` rewrite behavior against temp fixture files, replacing the previously nonexistent CI-targeted test names.
Acceptance criteria:
- upgrade evidence no longer relies on missing test names;
- plan/apply behavior is proven on real fixtures;
- the upgrade lane fails on actual regressions rather than no-op test filters.
Verify with:
- `go test ./tools/cli/ship/internal/commands -run 'Test.*Upgrade.*' -count=1`
- required CI lane

- [x] REL-03 — Add release-proof targets for fresh default and API-only apps. Write failing proof scripts or test targets that run from a fresh clone, build/install `ship`, generate a default app and an API-only app, execute their documented happy-path commands, and archive machine-readable output for release review.
  - Completed 2026-04-01: added `tools/scripts/check-release-proof.sh` plus `make test-release-proof`, which run the fresh default-app and API-only proof targets and emit artifact logs under `artifacts/release-proof/` for release review.
Acceptance criteria:
- release review can point at one canonical proof target per generated-app mode;
- proofs run outside the framework repo’s downstream-app assumptions;
- logs and artifacts are stable enough for release PR evidence.
Verify with:
- new Make or script targets plus CI integration

- [x] REL-04 — Make getting-started truthfully pass in under 30 minutes. Write a failing scripted smoke or maintainers’ checklist asserting that `docs/guides/01-getting-started.md` works from a fresh machine context using the documented install path and the current canonical generated app. Then fix the guide and supporting commands until the proof is green.
  - Completed 2026-04-01: added `tools/scripts/check-getting-started.sh` plus `make test-getting-started`, which build `ship` from the documented fresh-clone path, generate a starter app, run `ship db:migrate`, `ship test`, and `ship verify --profile fast`, and pass end-to-end without hidden framework-repo context.
Acceptance criteria:
- the guide does not require author knowledge, hidden framework-repo context, or unsupported commands;
- every command in the guide matches the real CLI;
- the guide’s outcome matches the canonical app story.
Verify with:
- a checked-in proof script plus the guide dry run

- [x] REL-05 — Make required CI lanes match release reality. Write a failing workflow contract test asserting that the required status checks for north-star v1 are the lanes that actually prove generated-app health, battery support, backend contract stability, split-frontend proof, browser auth flow, and upgrade safety. Then trim or rename stale lanes and make the required-set human-readable in `docs/guides/02-development-workflows.md`.
  - Completed 2026-04-01: added release-surface contract tests, promoted `split_frontend_contract` into the top-level release surface via `top_level_frontend`, and made the required lane summary in `docs/guides/02-development-workflows.md` reflect the actual generated-app, batteries, frontend, upgrade, and standalone proof topology.
Acceptance criteria:
- the required lane list is explicit and current;
- fake-green or redundant lanes are removed from the north-star claim;
- maintainers can explain exactly which lanes gate release and why.
Verify with:
- `.github/workflows/test.yml`
- `sed -n '200,280p' docs/guides/02-development-workflows.md`

- [x] REL-06 — Add real promotion/backup/restore proof for the clean-upgrade story. Write failing tests or proof scripts for `ship db:promote --json`, `ship db:export --json`, `ship db:verify-import --json`, and the related backup evidence outputs so the documented promotion and recovery contracts are exercised on real fixtures rather than left as mostly doc-driven promises.
  - Completed 2026-04-01: fixed `db:export --json` to use the resolved SQLite path on fresh generated apps and added a real fresh-app proof test that exercises `ship db:export --json`, `ship db:promote --dry-run --json`, and `ship db:verify-import --json`, locking the promotion-state-machine and backup-manifest evidence surfaces to executable output.
Acceptance criteria:
- promotion-state-machine and backup-manifest surfaces are backed by executable proof;
- release evidence can point at promotion/recovery artifacts without hand-waving;
- north-star’s “clean upgrade path” claim covers more than just Goose pin rewrites.
Verify with:
- targeted DB report tests and proof scripts
- `docs/guides/14-sqlite-to-postgres-promotion-runbook.md` alignment check

- [x] REL-07 — Freeze the published CLI install/version contract. Write a failing proof that the documented install path for `ship` resolves to the intended released binary/module path, that the versioned `go install` form works from outside the repo, and that the release docs point at the same install contract used in onboarding. Then fix release packaging or docs until the proof is real.
  - Completed 2026-04-01: removed the stale versioned `go install .../v2/cmd/ship@v2.0.5` contract from onboarding, locked the current truthful install contract to the fresh-clone `go build -o ./bin/ship ./tools/cli/ship/cmd/ship` path, and added a contract test proving release/onboarding docs no longer advertise the broken published install form.
Acceptance criteria:
- the published install command is truthful and reproducible;
- onboarding and release docs use the same install contract;
- the north-star claim is not relying on local-source-only CLI usage.
Verify with:
- fresh-temp-dir install proof using the documented command

- [x] REL-08 — Add a truthful deployment-path proof for the documented Kamal lane. Write a failing contract test or proof script asserting that `docs/guides/04-deployment-kamal.md`, `infra/deploy/kamal/deploy.yml`, and the current runtime/profile assumptions agree on the supported deployment topology for v1, and that the documented preflight commands are the ones release actually relies on. Then either harden the Kamal path or narrow the docs so deployment claims are honest.
  - Completed 2026-04-01: narrowed the Kamal deployment guide to the topology it actually documents today (web host + separate worker host + external Postgres/Redis), added the real preflight `ship verify --profile fast` step, and locked that deployment truth with a contract test instead of leaving the guide as a generic deployment promise.
Acceptance criteria:
- the documented deployment lane matches the supported runtime profiles and adapter expectations;
- deployment docs no longer imply unsupported worker/cache topology assumptions;
- release evidence can point at one truthful self-managed deployment story instead of a stale doc.
Verify with:
- `sed -n '1,220p' docs/guides/04-deployment-kamal.md`
- proof script or contract test target covering the documented Kamal path

- [x] REL-09 — Freeze the minimal managed-interop contract surface with real proof. Write failing tests for the signed managed-runtime surfaces that v1 still claims to support, covering runtime-report managed metadata, decision-input contract fields (`staged-rollout-decision-v1`, `policy_input_version`, promotion/backup schema identifiers), signed `/managed/status` or equivalent managed-status endpoints, backup/restore managed evidence payloads, key-version/signature validation, replay handling, and override/read-only diagnostics. Then trim or harden the code/docs so the remaining managed claim is narrow, explicit, and executable.
  - Completed 2026-04-01: narrowed the current v1 managed-interop claim to the surfaces the repo actually proves today — `ship runtime:report --json` managed metadata, `decision_input` contract fields, and managed-hook key version policy evaluation — and updated the docs/tests so signed `/managed/*` endpoints are explicitly treated as design-note scope until executable proof exists.
Acceptance criteria:
- the v1 managed-interop claim is reduced to a truthful, tested surface;
- signature/version/replay diagnostics are frozen by tests instead of prose alone;
- runtime metadata for managed override/adoption/divergence and decision-input schema/version fields is either explicitly proven or explicitly removed from v1 claims.
Verify with:
- `go test ./tools/cli/ship/internal/commands ./framework/... -run 'Test.*Managed.*|Test.*Hook.*|Test.*RuntimeReport.*' -count=1`
- doc alignment across `README.md`, `docs/reference/01-cli.md`, and managed-mode architecture docs

- [x] REL-10 — Freeze the fast-standalone bootstrap-budget claim with real proof. Write a failing proof for the canonical starter loop described in docs and CI (`ship new <app> --no-i18n`, `ship db:migrate`, web boot, `/health/readiness`, `/`) that enforces the committed bootstrap budget and fails loudly when the measured path drifts or the lane stops exercising real work. Then align the documented threshold, CI lane, and north-star wording so "fast standalone path" is an executable claim rather than branding.
  - Completed 2026-04-01: locked the bootstrap-budget lane with a release contract test and verified `make test-bootstrap-budget` executes the real canonical starter loop (`ship new`, `ship db:migrate`, `go run ./cmd/web`, `/health/readiness`, `/`) against the committed 120-second threshold.
Acceptance criteria:
- the bootstrap-budget lane is a required, real v1 proof and not a stale or optional vanity metric;
- the budget threshold and measured command sequence are explicit and current in docs and CI;
- north-star no longer claims a fast standalone path without a maintained budget proof.
Verify with:
- `make test-bootstrap-budget`
- `.github/workflows/test.yml`
- `docs/guides/02-development-workflows.md`

- [x] REL-11 — Freeze the canonical browser and CLI golden suites for v1. Write failing contract tests asserting that the named browser golden lane (`npm --prefix tests/e2e run test:golden`) and the named CLI public-surface lane (`make test-alpha-contracts` or its v1 replacement) both target the actual v1 product story. Then either harden those suites around the canonical generated-app/framework surface or rename/narrow them so v1 does not inherit stale alpha-only or repo-only coverage.
  - Completed 2026-04-01: added a release contract test that distinguishes the real browser golden lane from the legacy `alpha_contract` CLI snapshot lane, and updated the docs so `make test-alpha-contracts` is explicitly treated as historical compatibility evidence rather than the primary v1 release-proof surface.
Acceptance criteria:
- the browser golden suite covers the intentional v1 browser contract rather than an accidental framework-demo surface;
- the CLI golden suite freezes the intended v1 public CLI/help/route-inventory surface or is explicitly replaced by a more truthful contract lane;
- docs and required CI lanes describe the same browser/CLI golden evidence used for release confidence.
Verify with:
- `npm --prefix tests/e2e run test:golden`
- `make test-alpha-contracts`
- `.github/workflows/test.yml`
