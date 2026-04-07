# Event Sourcing Documentation

This directory contains comprehensive documentation for the `es` event sourcing library.

## Architecture Overview

The library is built around several core concepts:

### 1. Aggregates
Aggregates are the main building blocks that represent business entities. They:
- Encapsulate business logic and maintain state
- Generate domain events when state changes occur
- Provide event handlers to reconstruct state from events
- Support both global and tenant-scoped instances

### 2. Domain Events
Events are immutable records that represent facts about what happened:
- Implement the `DomainEvent` interface
- Include rich metadata (correlation ID, causation ID, timestamps)
- Support polymorphic serialization for persistence
- Enable event replay and aggregate reconstruction

### 3. Event Store
The event store provides persistence for domain events:
- Interface-based design supports multiple implementations
- Optimistic concurrency control prevents conflicts
- Sequence-based ordering ensures event consistency
- Built-in filtering by sequence number

### 4. Repository Pattern
Repositories provide high-level aggregate operations:
- Load aggregates from event streams
- Save uncommitted events with conflict detection
- Automatic event application and state reconstruction
- Transaction-like semantics with commit operations

## Design Principles

### Type Safety
- Generic event handler registration prevents runtime errors
- Compile-time type checking for event handlers
- Strong typing throughout the API surface

### Fail-Fast Wiring
- Aggregate construction and handler registration are treated as design-time concerns
- Invalid aggregate IDs, tenant IDs, duplicate handlers, and invalid event-area mappings panic immediately
- Business-rule failures should still be returned from command methods as ordinary errors

### Extensibility
- Interface-based design allows custom implementations
- Pluggable event stores for different persistence needs
- Customizable aggregate behaviors

### Multi-tenancy
- Built-in support for tenant-scoped aggregates
- Tenant isolation at the entity level
- Consistent patterns for global and tenant operations

### Performance
- Minimal allocations in hot paths
- Efficient event filtering and loading
- Thread-safe concurrent operations

## Best Practices

### Event Design
- Keep events immutable and side-effect free
- Include all necessary data in the event payload
- Use descriptive discriminator names
- Version events for schema evolution

### Aggregate Design
- Keep aggregates focused on a single business concept
- Implement business invariants in command methods
- Use event handlers only for state application
- Avoid external dependencies in aggregate logic

### Error Handling
- Use standard error types provided by the library for store and integration failures
- Handle concurrency conflicts gracefully
- Validate business rules before raising events
- Return early on validation failures
- Treat aggregate wiring panics as programmer errors to fix rather than runtime conditions to recover from

### Testing
- Test business logic through aggregate methods
- Use behavioral test naming conventions
- Mock external dependencies
- Test error conditions and edge cases