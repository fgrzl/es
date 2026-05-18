# Event sourcing (`es`) — documentation

This folder is the **canonical guide** for the [`github.com/fgrzl/es`](https://github.com/fgrzl/es) library: aggregates, domain events, audit batch streams, repository behavior, and how `Store` fits in.

## Contents

| Doc | Purpose |
|-----|---------|
| [Getting started](getting-started.md) | Walkthrough: events with `GetAreas`, aggregate, repository, tests |
| [API reference](api-reference.md) | Interfaces, factories, types, errors, usage snippets |
| [Audit events](audit_events.md) | `Audit` vs `Raise`, batch streams, save order, consumers, philosophy |

## What the library is

`es` is a **small, opinionated core** for event-sourced aggregates in Go:

- **`Store`** — append and read events by `Entity` (stream key), with optimistic concurrency on `SaveEvents`.
- **`Repository`** — `Load` / `Save` for one **domain** aggregate stream; `Save` also flushes **pending audits** to separate **audit batch streams** before appending domain events.
- **`Aggregate`** — replay (`Load`), `Raise` (domain handlers + uncommitted), `Audit` (stage only; no replay into aggregate).
- **`DomainEvent`** — polymorphic events + metadata; **`GetSpaces()`** is the compatibility contract, and new event types should also implement **`GetAreas()`** for wiring; the package prefers `GetAreas()` when present.

Production **`Store` implementations** (Postgres, EventStoreDB, Kafka-backed logs, etc.) **live in your repos**, not in `es`. This module defines the **`Store` interface** and ships **`NewInMemoryEventStore`** for tests and local development only.

## Architecture (mental model)

```text
                    ┌──────────────┐
                    │   Context    │  correlation / causation (optional)
                    └──────┬───────┘
                           │
    Command ──► Aggregate (domain Entity) ──► Raise ──► uncommitted ──┐
           │                         │                               │
           │                         └──► Audit ──► pendingAudits ─────┤
           │                                                           │
           └──────────────────────────────► Repository.Save ──────────┤
                                              │                         │
                         audit batch Entity(s) │                         │ domain Entity
                                              ▼                         ▼
                                         Store.SaveEvents         Store.SaveEvents
                                         (expectedSeq 0)          (expectedSeq = committed len)
```

`Repository.Load` reads **only** the domain `Entity` stream. Audit batch streams are **never** passed into `Load`.

## Design principles (summary)

- **Fail-fast wiring** — invalid aggregate ids, duplicate handlers, or events whose effective area list omits the aggregate’s `Area` surface as panics in the default implementation (design-time mistakes).
- **Metadata honesty** — persisted audit rows use `EventMetadata.Entity` for the **audit batch stream** (new stream id per batch), not the business root; link subjects via correlation and payload.
- **Replay purity** — audit volume does not affect aggregate reconstruction.
- **Tracing** — repository operations emit OpenTelemetry spans; pass correlation/causation via `ContextWithTracing` where needed.

## Where to go next

1. Read [Getting started](getting-started.md) if you are new to the API shape.
2. Use [API reference](api-reference.md) while coding against interfaces.
3. Read [Audit events](audit_events.md) before mixing `Audit` with domain `Save` semantics or building projections over audit streams.

## Best practices (short)

- Implement **`GetAreas()`** on every event type; keep **`GetSpaces() = GetAreas()`** until you drop the compatibility path in a major version.
- Put **subject / origin ids in audit payloads** when downstream needs them; do not overload `GetAggregateID()` on audits.
- Treat **`Repository.Save`** as **not** one atomic transaction across audit + domain unless your `Store` implements that.
- Test aggregates through **command methods**; use the in-memory store for unit tests.

## Testing

- Prefer **`es.NewInMemoryEventStore()`** in tests.
- Naming: behavioral test names (`TestShould…`) match the style used in this repository.

## Related

- [README](../README.md) — project entry point
- [CHANGELOG](../CHANGELOG.md) — release notes
- [CONTRIBUTING](../CONTRIBUTING.md) — contribution workflow
