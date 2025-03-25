package es

import (
	"context"
	"fmt"

	"github.com/fgrzl/messaging"
	"github.com/fgrzl/timestamp"
	"github.com/google/uuid"
)

type DomainEventHandler func(DomainEvent)
type HandlerFactory func(Aggregate) DomainEventHandler

func RegisterHandler[T DomainEvent](a Aggregate, handler func(T)) {
	var zero T
	eventType := zero.GetDiscriminator()
	a.RegisterHandler(eventType, func(event DomainEvent) {
		e, ok := event.(T)
		if !ok {
			panic(fmt.Sprintf("RegisterHandler: event %T does not match expected type %T", event, zero))
		}
		handler(e)
	})
}

// Aggregate defines the interface for event-sourced aggregates
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

// NewAggregate creates a new base aggregate
func NewAggregate(ctx context.Context, entity Entity) Aggregate {
	if ctx == nil {
		ctx = context.Background()
	}
	return &aggregateBase{
		entity:        entity,
		correlationID: messaging.GetCorrelationID(ctx),
		causationID:   messaging.GetCausationID(ctx),
		handlers:      make(map[string]DomainEventHandler),
	}
}

// aggregateBase provides event-sourcing behavior
type aggregateBase struct {
	entity              Entity
	correlationID       uuid.UUID
	causationID         uuid.UUID
	committed           []DomainEvent
	committedSequence   uint64
	uncommitted         []DomainEvent
	uncommittedSequence uint64
	handlers            map[string]DomainEventHandler
}

func (a *aggregateBase) GetEntity() Entity           { return a.entity }
func (a *aggregateBase) GetAggregateID() uuid.UUID   { return a.entity.ID }
func (a *aggregateBase) GetAggregateType() string    { return a.entity.Type }
func (a *aggregateBase) GetCorrelationID() uuid.UUID { return a.correlationID }
func (a *aggregateBase) GetCausationID() uuid.UUID   { return a.causationID }
func (a *aggregateBase) AppendCommitted(event DomainEvent) {
	a.committed = append(a.committed, event)
	a.committedSequence = event.GetSequence()
}
func (a *aggregateBase) AppendUncommitted(event DomainEvent) {
	a.uncommitted = append(a.uncommitted, event)
	a.uncommittedSequence = event.GetSequence()
}
func (a *aggregateBase) GetCommittedEvents() []DomainEvent   { return a.committed }
func (a *aggregateBase) GetCommittedSequence() uint64        { return a.committedSequence }
func (a *aggregateBase) GetUncommittedEvents() []DomainEvent { return a.uncommitted }
func (a *aggregateBase) GetUncommittedSequence() uint64      { return a.uncommittedSequence }
func (a *aggregateBase) Commit() {
	a.committed = append(a.committed, a.uncommitted...)
	a.committedSequence = a.uncommittedSequence
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
