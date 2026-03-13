# AI Module Guide

This guide defines the current install and usage contract for `modules/ai`.

Last updated: 2026-03-13

## Goal

- Provide one app-facing AI seam through `container.AI`.
- Keep provider selection in config and app composition, not in feature code.
- Support plain completion, token streaming, and structured JSON decoding behind the same request shape.

## Runtime Contract

`modules/ai` exposes:

- `Provider.Complete(ctx, ai.Request)`
- `Provider.Stream(ctx, ai.Request)`
- `Service.Complete(ctx, ai.Request)`
- `Service.Stream(ctx, ai.Request)`

The app container wires the concrete provider in [container.go](/workspace/project/app/foundation/container.go).

## Provider Selection

Supported `AI_DRIVER` values today:

- `anthropic`
- `openai`
- `openrouter`

Credential requirements:

- `anthropic` requires `ANTHROPIC_API_KEY`
- `openai` requires `OPENAI_API_KEY`
- `openrouter` requires `OPENROUTER_API_KEY`

If the selected driver is missing credentials, `container.AI` stays non-nil but returns a clear provider-unavailable error when called.

## Request Shape

`ai.Request` supports:

- `Model`
- `System`
- `Messages`
- `MaxTokens`
- `Temperature`
- `Schema`
- `Tools`

Structured output:

- Set `Schema` to a pointer to the target Go value.
- The service will decode the model response JSON into that pointer.
- Anthropic uses a forced tool call for structured output.
- OpenAI and OpenRouter use JSON-schema response formatting.

## SSE Pattern

`modules/ai/stream_handler.go` provides `StreamCompletion(...)` for HTMX SSE flows.

Current example:

- `GET /auth/ai-demo`
- `GET /auth/ai-demo/stream?prompt=...`

The demo route is registered only outside production and shows the recommended HTMX pattern:

- render a page with `hx-ext="sse"`
- set `sse-connect` to the stream endpoint
- append token chunks with `sse-swap="message"`
- handle completion with `sse-swap="done"`
