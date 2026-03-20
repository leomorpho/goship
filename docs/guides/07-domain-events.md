# Domain Events Guide

This guide defines the current framework contract for domain events in GoShip.

Last updated: 2026-03-13

## Goal

- Let modules react to cross-cutting behavior without direct imports.
- Keep event types in a shared framework package.
- Support synchronous in-process publish/subscribe first, with a jobs-backed async enqueue helper.

## Runtime Contract

Core package:

- [bus.go](/workspace/project/framework/events/bus.go)
- [async.go](/workspace/project/framework/events/async.go)

Shared event types:

- [auth.go](/workspace/project/framework/events/types/auth.go)
- [subscription.go](/workspace/project/framework/events/types/subscription.go)
- [profile.go](/workspace/project/framework/events/types/profile.go)

Container seam:

- [container.go](/workspace/project/app/foundation/container.go)
  - `EventBus *events.Bus`

## Publish / Subscribe

Use the generic subscribe helper:

```go
events.Subscribe(bus, func(ctx context.Context, event types.UserLoggedIn) error {
    return nil
})
```

Publish synchronously:

```go
err := bus.Publish(ctx, types.UserLoggedIn{UserID: 42, At: time.Now().UTC()})
```

Current behavior:

- handlers run synchronously in registration order
- publish returns the first handler error
- auth routes publish `UserRegistered`, `UserLoggedIn`, `UserLoggedOut`, and `PasswordChanged`

## Async Helper

`events.PublishAsync(...)` serializes an event envelope and enqueues it onto the jobs seam using:

- job name: `framework.events.publish`

This is the current enqueue contract only; no generic worker re-dispatcher is wired yet.

## Generator

Use the scaffold helper for new shared event types:

```text
ship make:event UserLoggedIn
```
