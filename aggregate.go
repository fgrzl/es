package es

import (
	"context"
	"fmt"

	"github.com/fgrzl/messaging"
	"github.com/fgrzl/timestamp"
	"github.com/google/uuid"
)

// DomainEventHandler defines a function that handles a domain event.
type DomainEventHandler func(DomainEvent)

// HandlerFactory creates a domain event handler for a specific aggregate.
type HandlerFactory func(Aggregate) DomainEventHandler

// RegisterHandler registers a typed event handler for a specific event type on an aggregate.
// The handler will be called when the event type is raised or loaded.
// This function uses generics to provide type safety for event handlers.
func RegisterHandler[T DomainEvent](a Aggregate, handler func(T)) {
	var zero T
	eventDiscriminator := zero.GetDiscriminator()
	a.RegisterHandler(eventDiscriminator, func(event DomainEvent) {
		e, ok := event.(T)
		if !ok {
			panic(fmt.Sprintf("RegisterHandler: event %T does not match expected type %T", event, zero))
		}
		handler(e)
	})
}

// Aggregate defines the interface for event-sourced aggregates.
// Aggregates are the primary building blocks of event sourcing, representing
// business entities that generate and respond to domain events.
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

// NewAggregate creates a new global-scoped aggregate with the specified area and ID.
func NewAggregate(ctx context.Context, area string, id uuid.UUID) Aggregate {
	return newAggregate(ctx, ScopeGlobal, area, uuid.Nil, id)
}

// NewTenantAggregate creates a new tenant-scoped aggregate with the specified area, tenant ID, and aggregate ID.
func NewTenantAggregate(ctx context.Context, area string, tenantID, id uuid.UUID) Aggregate {
	if tenantID == uuid.Nil {
		panic("NewTenantAggregate: tenantID must not be nil")
	}
	return newAggregate(ctx, ScopeTenant, area, tenantID, id)
}

func newAggregate(ctx context.Context, scope Scope, area string, tenantID, id uuid.UUID) Aggregate {
	if id == uuid.Nil {
		panic("newAggregate: id cannot be nil")
	}
	if area == "" {
		panic("newAggregate: space cannot be empty")
	}

	entity := Entity{
		ID:       id,
		Area:     area,
		TenantID: tenantID,
		Scope:    scope,
	}

	if ctx == nil {
		ctx = context.Background()
	}

	correlationID := messaging.GetCorrelationID(ctx)
	if correlationID == uuid.Nil {
		correlationID = uuid.New()
	}
	causationID := messaging.GetCausationID(ctx)
	if causationID == uuid.Nil {
		causationID = uuid.New()
	}

	return &aggregateBase{
		entity:        entity,
		correlationID: correlationID,
		causationID:   causationID,
		handlers:      make(map[string]DomainEventHandler),
	}
}

// aggregateBase provides event-sourcing behavior for aggregate implementations.
// It should be embedded in concrete aggregate types to inherit event sourcing capabilities.
type aggregateBase struct {
	entity        Entity
	correlationID uuid.UUID
	causationID   uuid.UUID
	committed     []DomainEvent
	uncommitted   []DomainEvent
	handlers      map[string]DomainEventHandler
}

func (a *aggregateBase) GetEntity() Entity {
	return a.entity
}

func (a *aggregateBase) GetAggregateID() uuid.UUID {
	return a.entity.ID
}

func (a *aggregateBase) GetAggregateSpace() string {
	return a.entity.Area
}

func (a *aggregateBase) GetCorrelationID() uuid.UUID {
	return a.correlationID
}

func (a *aggregateBase) GetCausationID() uuid.UUID {
	return a.causationID
}

func (a *aggregateBase) AppendCommitted(event DomainEvent) {
	a.committed = append(a.committed, event)
}

func (a *aggregateBase) AppendUncommitted(event DomainEvent) {
	a.uncommitted = append(a.uncommitted, event)
}

func (a *aggregateBase) GetCommittedEvents() []DomainEvent {
	return a.committed
}

func (a *aggregateBase) GetCommittedSequence() uint64 {
	return uint64(len(a.committed))
}

func (a *aggregateBase) GetUncommittedEvents() []DomainEvent {
	return a.uncommitted
}

func (a *aggregateBase) GetUncommittedSequence() uint64 {
	return uint64(len(a.committed) + len(a.uncommitted))
}

func (a *aggregateBase) Commit() {
	a.committed = append(a.committed, a.uncommitted...)
	a.uncommitted = make([]DomainEvent, 0)
}

// ApplyEvent provides default behavior - overridden by concrete types
func (a *aggregateBase) applyEvent(event DomainEvent) {
	eventName := event.GetDiscriminator()
	if handler, exists := a.handlers[eventName]; exists {
		handler(event)
	}
}

func (a *aggregateBase) RegisterHandler(discriminator string, handler DomainEventHandler) {

	if _, exists := a.handlers[discriminator]; exists {
		panic(fmt.Sprintf("RegisterHandler: handler for event %s already exists", discriminator))
	}

	a.handlers[discriminator] = handler
}

// Raise applies an event to an aggregate and adds it to uncommitted events
func (a *aggregateBase) Raise(event DomainEvent) error {

	event.SetMetadata(EventMetadata{
		Entity:        a.GetEntity(),
		EventID:       uuid.New(),
		CorrelationID: a.GetCorrelationID(),
		CausationID:   a.GetCausationID(),
		Timestamp:     timestamp.GetTimestamp(),
		Sequence:      a.GetUncommittedSequence() + 1,
	})

	a.applyEvent(event)
	a.AppendUncommitted(event)
	return nil
}

// Load replays committed events onto an aggregate
func (a *aggregateBase) Load(events []DomainEvent) error {
	for _, event := range events {
		a.applyEvent(event)
		a.AppendCommitted(event)
	}
	return nil
}
