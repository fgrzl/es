# Audit events

This document describes how **audit** works in `es` today: staging on the aggregate, persistence, stream identity, and how that differs from **domain** events raised with `Raise`.

## Goals

- Record immutable facts (`DomainEvent`) from a command without growing the **domain** aggregate’s replay stream.
- Allow **concurrent** audit writers without contending on one long-lived audit stream per business aggregate.
- Keep audit payloads as normal **`DomainEvent`** types (same interface, polymorphic serialization, consumer pipelines).

## Staging: `Audit` vs `Raise`

| | `Raise` | `Audit` |
|---|---------|---------|
| Validated against aggregate area | Yes (`GetAreas()` must include `Entity.Area`) | Yes (same rule) |
| Runs registered handlers | Yes | **No** |
| Goes into uncommitted / committed | Yes | **No** |
| Replayed by `Load` | Yes (domain stream only) | **No** |
| Buffered | `uncommitted` | `pendingAudits` (`[]PendingAudit`) |

Each `PendingAudit` holds the event pointer, a pre-assigned **`EventID`** and **`Timestamp`** (assigned at `Audit()` time, not at save time), and the **`Entity` of the audit batch stream** (see below). Full `EventMetadata` is applied in `Repository.Save`, not at `Audit()` time.

Pre-assigning identity fields preserves **causal ordering**, **stable references** across retries, and avoids retry-time identity drift when the same staged event is persisted after a transient failure.

## Audit batch stream (not a second “domain aggregate”)

Audits are **not** appended to the originating aggregate’s `Entity` (`ID` = business root). Each **pending audit batch** is persisted as its own **short-lived stream** (append partition):

- **`Area`**, **`TenantID`**, and **`Scope`** match the source aggregate (same area and tenancy as the command).
- **`ID`** is a **new UUID** (`uuid.New()`), created when the first `Audit()` runs after the previous batch was flushed (see `AuditStreamEntity` in `entity.go` and `aggregateBase.Audit` in `aggregate.go`).
- Multiple `Audit()` calls before the next successful `Save()` share **one** batch stream `Entity` (one stream id).

That model gives **append locality**, **no long-lived OCC hotspot** on a single audit tail, **bounded stream size per batch** (typically one `Save` worth of events), and **natural parallelism** across concurrent commands (each batch gets its own stream id).

Terminology: prefer **audit batch stream** or **audit append stream** over “derived audit aggregate” — audits are not replayed into aggregate state; the name is stream/partition oriented, not DDD aggregate semantics.

On a persisted audit row, **`GetAggregateID()` / `GetEntity()` identify that audit batch stream**, not the business root. That is **metadata honesty**: the store partition and the event envelope agree. Tie an audit fact back to the business entity using **correlation / causation** and **payload fields** (e.g. subject id), not by overloading `GetAggregateID()`.

## Philosophy: what an audit row means

**`Audit()` is valid even if the command later fails** — including failures before any domain event is raised, or after audits are already staged. That supports high-value records such as denied access, failed validation, rejected workflows, suspicious behavior, or partial external failures.

**Default save order is audits before domain.** Roughly:

- An **audit append may exist without** a successful domain append (e.g. domain `SaveEvents` fails after audit batches succeeded).
- A **domain append does not run until** pending audit batches for that `Save` have been written successfully (for that call).

Treat persisted audits as **observational facts about what was attempted or observed**, not as a guarantee that a **committed business transition** occurred. If you need “domain definitely committed,” correlate audit rows with successful domain stream positions, outbox completion, or application-level markers — do not infer it from audit presence alone.

## Persistence: `Repository.Save`

The library does not ship database or cloud **`Store`** implementations—only the interface and an in-memory implementation for tests. Your adapter is responsible for append semantics and concurrency per stream. See the **Implementing `Store` outside this module** section in [api-reference.md](api-reference.md).

1. **Snapshot** pending audits (`GetPendingAudits()`).
2. **Group** staged items by stream `Entity`.
3. For each batch, in order:
   - Stamp `EventMetadata` with `Entity =` that batch’s stream `Entity`, sequences `1..n` within the batch, plus correlation/causation and the staged `EventID` / `Timestamp`.
   - **`SaveEvents(ctx, auditEntity, events, expectedSequence=0)`** — no prior `LoadEvents` on the audit stream; each batch stream starts empty.
   - On success, **`TrimPendingAudits(n)`** removes the persisted prefix from the in-memory buffer.
4. Then persist **domain** uncommitted events to `agg.GetEntity()` with normal optimistic concurrency (`expectedSequence = committed length`).
5. On full success: **`Commit()`** (domain) and **`DiscardPendingAudits()`** (clears any remainder).

There is **no** cross-stream transaction in the default `Store` API unless your implementation provides one.

## Consumers: “like a domain event”

Audit rows are still **`DomainEvent`**:

- Same metadata envelope shape, discriminators, JSON polymorphism, buses, etc.
- **`GetAggregateID()` is the audit batch stream id**, not the originating business aggregate id — do not assume they match.
- **`Sequence`** is **within the audit batch stream** (1..n for that `Save`), not the domain aggregate’s global position.

Linking strategies:

- **Tracing:** `CorrelationID` / `CausationID` on the event match the aggregate at save time.
- **Payload:** include subject ids (e.g. originating aggregate id) in the audit event type when projections need them.

## `Load` and replay (invariant)

**Audit batch streams are never hydrated into the aggregate.** `Repository.Load` only reads **`agg.GetEntity()`** (the domain stream). That invariant is what keeps **replay purity** and prevents audit volume from affecting aggregate reconstruction.

Rebuilding read models from audit data is separate: subscribe to or query the audit batch stream keys your application emits.

## Operational notes

- **`DiscardPendingAudits` / `TrimPendingAudits`** are on the public `Aggregate` interface because `Repository.Save` must trim after partial multi-batch audit writes; application code should not call them except in advanced tooling.
- **`Commit()`** does not clear pending audits; only `Save` + discard/trim does.

### `SetMetadata` and retries

**`DomainEventBase.SetMetadata` only applies once** when metadata is still empty. That means:

- Successful identity fields on a staged event are **stable** across retries (good for dedupe and references).
- A `Save` failure **after** metadata was stamped on in-memory pointers but **before** the store acknowledged persistence needs **disciplined** retry behavior from the **store**: ambiguous partial writes, timeouts after success, or non-idempotent retries can desynchronize process memory and storage.

Stricter stores (append-is-atomic, clear OCC rules) keep this manageable; weaker ones may eventually need **idempotent append keys**, **append tokens**, or **dedupe** at the storage layer. The library does not prescribe that today.

## Future consideration: explicit event classification

For indexing, retention, warehousing, subscriptions, and export pipelines, some teams add an explicit **kind** or **classification** on metadata (e.g. domain vs audit), instead of inferring intent only from stream layout or discriminators. That is **not** part of `EventMetadata` in `es` yet; if you need it, encode it in payload conventions or extend your own envelope until a first-class field exists.

## Related code

- `aggregate.go` — `Audit`, `PendingAudit`, `GetPendingAudits`, `TrimPendingAudits`, `DiscardPendingAudits`
- `repository.go` — `Save`, batch grouping
- `entity.go` — `AuditStreamEntity`
- `tracing.go` — `es.repository.save_audit` span name
