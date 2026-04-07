# API Reference

Complete reference documentation for the `es` event sourcing library.

## Core Interfaces

### Aggregate

The main interface for event-sourced aggregates.

The default `aggregateBase` implementation is intentionally fail-fast for aggregate design-time mistakes. Invalid constructor inputs, duplicate handler registration, invalid handler type parameters, and invalid event-area mappings panic immediately instead of being treated as recoverable runtime errors.

```go
type Aggregate interface {
    // Metadata
    GetEntity() Entity
    GetAggregateID() uuid.UUID
    GetCorrelationID() uuid.UUID
    GetCausationID() uuid.UUID

    // Committed behavior
    AppendCommitted(DomainEvent)
    GetCommittedEvents() []DomainEvent
    GetCommittedSequence() uint64

    // Uncommitted behavior
    AppendUncommitted(DomainEvent)
    GetUncommittedEvents() []DomainEvent
    GetUncommittedSequence() uint64

    // Event behavior
    RegisterHandler(string, DomainEventHandler)
    Raise(DomainEvent) error
    Load([]DomainEvent) error
    Commit()
}
```

### DomainEvent

Interface that all domain events must implement.

```go
type DomainEvent interface {
    polymorphic.Polymorphic
    GetAggregateID() uuid.UUID
    GetArea() string
    GetSpaces() []string
    GetTenantID() uuid.UUID
    GetCausationID() uuid.UUID
    GetCorrelationID() uuid.UUID
    GetEntity() Entity
    GetEventID() uuid.UUID
    GetMetadata() EventMetadata
    GetSequence() uint64
    GetTimestamp() int64
    SetMetadata(metadata EventMetadata)
}
```

### Store

Interface for event persistence.

```go
type Store interface {
    SaveEvents(ctx context.Context, entity Entity, events []DomainEvent, expectedSequence uint64) error
    LoadEvents(ctx context.Context, entity Entity, minSequence uint64) ([]DomainEvent, error)
}
```

### Repository

High-level interface for aggregate operations.

Repository implementations emit OpenTelemetry spans for `Load` and `Save` operations. The spans include aggregate identity attributes, scope information, correlation and causation IDs when present in the incoming context, and event or sequence counts relevant to the operation.

```go
type Repository interface {
    Load(context.Context, Aggregate) error
    Save(context.Context, Aggregate) error
}
```

## Factory Functions

### NewAggregate

Creates a new global-scoped aggregate.

```go
func NewAggregate(ctx context.Context, area string, id uuid.UUID) Aggregate
```

This function panics when aggregate wiring is invalid.

Typical panic conditions are a nil aggregate ID or an empty area.

**Parameters:**
- `ctx`: Context for correlation and causation tracking
- `area`: Logical grouping for the aggregate type
- `id`: Unique identifier for the aggregate instance

### NewTenantAggregate

Creates a new tenant-scoped aggregate.

```go
func NewTenantAggregate(ctx context.Context, area string, tenantID, id uuid.UUID) Aggregate
```

This function panics when aggregate wiring is invalid.

Typical panic conditions are a nil tenant ID, a nil aggregate ID, or an empty area.

**Parameters:**
- `ctx`: Context for correlation and causation tracking
- `area`: Logical grouping for the aggregate type
- `tenantID`: Unique identifier for the tenant (must not be nil)
- `id`: Unique identifier for the aggregate instance

### NewRepository

Creates a new repository with the given event store.

```go
func NewRepository(store Store) Repository
```

### NewInMemoryEventStore

Creates a new in-memory event store for testing and development.

```go
func NewInMemoryEventStore() Store
```

## Utility Functions

### RegisterHandler

Registers a typed event handler for a specific event type.

```go
func RegisterHandler[T DomainEvent](a Aggregate, handler func(T))
```

This function panics on invalid aggregate design-time wiring such as duplicate handlers or invalid event type parameters.

Use `RegisterHandler` during aggregate construction so wiring mistakes fail immediately.

**Type Parameters:**
- `T`: The domain event type to handle

**Parameters:**
- `a`: The aggregate to register the handler on
- `handler`: The typed handler function

### Register

Registers a polymorphic type factory for JSON serialization.

```go
func Register[T polymorphic.Polymorphic](factory func() T)
```

### WithEventMetadata

Creates a new context with tracing information from a domain event.

```go
func WithEventMetadata(ctx context.Context, event DomainEvent) context.Context
```

## Types

### Entity

Represents an aggregate's identity and scope.

```go
type Entity struct {
    ID       uuid.UUID `json:"id"`
    Area     string    `json:"area"`
    TenantID uuid.UUID `json:"tenant_id"`
    Scope    Scope     `json:"scope"`
}
```

**Methods:**
- `GetID() uuid.UUID`: Returns the entity ID
- `GetTenantID() uuid.UUID`: Returns the tenant ID
- `GetScope() Scope`: Returns the scope (global or tenant)
- `GetNamespace() uuid.UUID`: Returns a deterministic namespace UUID and panics when `Area` is empty
- `TryGetNamespace() (uuid.UUID, error)`: Returns a deterministic namespace UUID without panicking
- `IsEmpty() bool`: Checks if the entity is uninitialized

### EventMetadata

Contains metadata fields common to all domain events.

```go
type EventMetadata struct {
    Entity        Entity    `json:"entity"`
    EventID       uuid.UUID `json:"event_id"`
    CorrelationID uuid.UUID `json:"correlation_id"`
    CausationID   uuid.UUID `json:"causation_id"`
    Timestamp     int64     `json:"timestamp"`
    Sequence      uint64    `json:"sequence"`
}
```

### DomainEventBase

Base implementation of the DomainEvent interface.

```go
type DomainEventBase struct {
    Metadata EventMetadata `json:"metadata"`
}
```

### Scope

Enumeration for entity visibility scope.

```go
type Scope int

const (
    ScopeGlobal Scope = iota  // Global visibility
    ScopeTenant               // Tenant-specific visibility
)
```

## Error Types

### Standard Errors

The package exports sentinel errors that custom stores or aggregate workflows can wrap with `errors.Is`.
The built-in in-memory store returns errors matching `ErrConcurrency` for optimistic concurrency conflicts.
`Repository.Load` passes through store errors and does not synthesize `ErrNotFound` for empty streams.

The default aggregate implementation intentionally panics for invalid aggregate design-time setup such as duplicate handlers, missing IDs, or invalid event-area mappings. Those conditions are documented fail-fast aggregate wiring behavior, not recoverable runtime errors.

Use returned `error` values for store failures and business-rule validation in your own command methods. Do not treat aggregate wiring panics as part of the normal control flow.

```go
var (
    ErrAlreadyExists        error // Aggregate already exists
    ErrNotFound             error // Aggregate not found
    ErrConcurrency          error // Concurrency conflict detected
    ErrInvalidEventSpace    error // Compatibility-preserved sentinel for invalid event compatibility checks
    ErrEventHandlerNotFound error // Missing event handler
    ErrInvalidEntity        error // Entity validation failed
)
```

## Entity Factory Functions

### NewEntity

Creates a new global-scoped entity.

```go
func NewEntity(id uuid.UUID, area string) Entity
```

### NewEntityInArea

Creates a new global-scoped entity with a generated ID.

```go
func NewEntityInArea(area string) Entity
```

### NewTenantEntity

Creates a new tenant-scoped entity.

```go
func NewTenantEntity(tenantID, id uuid.UUID, area string) Entity
```

### NewTenantEntityInArea

Creates a new tenant-scoped entity with a generated ID.

```go
func NewTenantEntityInArea(tenantID, id uuid.UUID, area string) Entity
```

## Usage Patterns

### Event Handler Registration

```go
type MyAggregate struct {
    es.Aggregate
    // ... fields
}

func NewMyAggregate(ctx context.Context, id uuid.UUID) *MyAggregate {
    agg := &MyAggregate{Aggregate: es.NewAggregate(ctx, "my-area", id)}
    
    // Type-safe event handler registration
    es.RegisterHandler(agg, agg.OnMyEvent)
    
    return agg
}

func (a *MyAggregate) OnMyEvent(e *MyEvent) {
    // Handle event...
}
```

### Command Methods

```go
func (a *MyAggregate) DoSomething(param string) error {
    // Validate business rules
    if param == "" {
        return errors.New("param cannot be empty")
    }
    
    // Raise domain event
    return a.Raise(&SomethingDone{Param: param})
}
```

### Repository Usage

```go
// Save aggregate
if err := repo.Save(ctx, aggregate); err != nil {
    return fmt.Errorf("failed to save aggregate: %w", err)
}

// Load aggregate
if err := repo.Load(ctx, aggregate); err != nil {
    return fmt.Errorf("failed to load aggregate: %w", err)
}
```