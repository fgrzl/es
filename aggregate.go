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
	GetAggregateID() uuid.UUID
	GetCorrelationID() uuid.UUID
	GetCausationID() uuid.UUID

	// Committed behavior
	AppendCommitted(DomainEvent)
	GetCommittedEvents() []DomainEvent
	GetCommittedVersion() uint64

	// Uncommitted behavior
	AppendUncommitted(DomainEvent)
	GetUncommittedEvents() []DomainEvent
	GetUncommittedVersion() uint64

	// Event behavior
	RegisterHandler(string, DomainEventHandler)
	Raise(DomainEvent) error
	Load([]DomainEvent) error
	Commit()
}

// NewAggregate creates a new base aggregate
func NewAggregate(ctx context.Context, id uuid.UUID) Aggregate {
	if ctx == nil {
		ctx = context.Background()
	}
	return &aggregateBase{
		aggregateID:   id,
		correlationID: messaging.GetCorrelationID(ctx),
		causationID:   messaging.GetCausationID(ctx),
		handlers:      make(map[string]DomainEventHandler),
	}
}

// aggregateBase provides event-sourcing behavior
type aggregateBase struct {
	aggregateID        uuid.UUID
	correlationID      uuid.UUID
	causationID        uuid.UUID
	committed          []DomainEvent
	committedVersion   uint64
	uncommitted        []DomainEvent
	uncommittedVersion uint64
	handlers           map[string]DomainEventHandler
}

func (a *aggregateBase) GetAggregateID() uuid.UUID   { return a.aggregateID }
func (a *aggregateBase) GetCorrelationID() uuid.UUID { return a.correlationID }
func (a *aggregateBase) GetCausationID() uuid.UUID   { return a.causationID }
func (a *aggregateBase) AppendCommitted(event DomainEvent) {
	a.committed = append(a.committed, event)
	a.committedVersion = event.GetSequence()
}
func (a *aggregateBase) AppendUncommitted(event DomainEvent) {
	a.uncommitted = append(a.uncommitted, event)
	a.uncommittedVersion = event.GetSequence()
}
func (a *aggregateBase) GetCommittedEvents() []DomainEvent   { return a.committed }
func (a *aggregateBase) GetCommittedVersion() uint64         { return a.committedVersion }
func (a *aggregateBase) GetUncommittedEvents() []DomainEvent { return a.uncommitted }
func (a *aggregateBase) GetUncommittedVersion() uint64       { return a.uncommittedVersion }
func (a *aggregateBase) Commit() {
	a.committed = append(a.committed, a.uncommitted...)
	a.committedVersion = a.uncommittedVersion
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
		AggregateID:   a.GetAggregateID(),
		EventID:       uuid.New(),
		CorrelationID: a.GetCorrelationID(),
		CausationID:   a.GetCausationID(),
		Timestamp:     timestamp.GetTimestamp(),
		Sequence:      a.GetUncommittedVersion() + 1,
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
