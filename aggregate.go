package es

import (
	"context"
	"fmt"
	"reflect"

	"github.com/fgrzl/timestamp"
	"github.com/google/uuid"
)

const (
	errRegisterHandlerTypeMismatch   = "RegisterHandler: event %T does not match expected type %T"
	errRegisterHandlerAlreadyExists  = "RegisterHandler: handler for event %s already exists"
	errRegisterHandlerNilHandler     = "RegisterHandler: handler must not be nil"
	errRegisterHandlerNilType        = "RegisterHandler: event type must be concrete"
	errNewTenantAggregateNilTenantID = "NewTenantAggregate: tenantID must not be nil"
	errNewAggregateNilID             = "newAggregate: id cannot be nil"
	errNewAggregateEmptyArea         = "newAggregate: area cannot be empty"
	errRaiseInvalidAggregateArea     = "Raise: aggregate area '%s' is not valid for event %T"
	errAuditInvalidAggregateArea     = "Audit: aggregate area '%s' is not valid for event %T"
)

// DomainEventHandler defines a function that handles a domain event.
type DomainEventHandler func(DomainEvent)

// HandlerFactory creates a domain event handler for a specific aggregate.
type HandlerFactory func(Aggregate) DomainEventHandler

// RegisterHandler registers a typed event handler for a specific event type on an aggregate.
// The handler will be called when the event type is raised or loaded, but not for Audit.
// This function panics on invalid aggregate wiring such as nil handlers,
// duplicate handlers, or invalid event type parameters.
func RegisterHandler[T DomainEvent](a Aggregate, handler func(T)) {
	if handler == nil {
		panic(errRegisterHandlerNilHandler)
	}

	expectedEvent := newEventInstance[T]()

	eventDiscriminator := expectedEvent.GetDiscriminator()
	a.RegisterHandler(eventDiscriminator, func(event DomainEvent) {
		e, ok := event.(T)
		if !ok {
			panic(fmt.Sprintf(errRegisterHandlerTypeMismatch, event, expectedEvent))
		}
		handler(e)
	})
}

func newEventInstance[T DomainEvent]() T {
	var zero T
	eventType := reflect.TypeOf(zero)
	if eventType == nil {
		panic(errRegisterHandlerNilType)
	}

	if eventType.Kind() == reflect.Pointer {
		return reflect.New(eventType.Elem()).Interface().(T)
	}

	return reflect.New(eventType).Elem().Interface().(T)
}

// Aggregate defines the interface for event-sourced aggregates.
//
// Note: the audit methods on this interface are a deliberate breaking API change
// in this branch; external aggregate implementations must be updated together
// with the audit workflow changes.
// Aggregates are the primary building blocks of event sourcing, representing
// business entities that generate and respond to domain events.
//
// The default aggregate implementation in this package intentionally fails fast
// on invalid aggregate wiring. Constructor validation, duplicate handler
// registration, invalid handler type parameters, and invalid event-area
// mappings are treated as programmer errors and panic immediately.
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
	Audit(DomainEvent) error
	Load([]DomainEvent) error
	Commit()

	// GetPendingAudits returns a copy of staged audit events (metadata is applied on Repository.Save).
	GetPendingAudits() []PendingAudit
	// DiscardPendingAudits clears staged audits. The repository calls this after
	// those audits have been persisted successfully (or use TrimPendingAudits incrementally).
	DiscardPendingAudits()
	// TrimPendingAudits removes the first n staged audits. Repository.Save calls this
	// after each successful audit batch; application code should not use it.
	TrimPendingAudits(n int)
}

// PendingAudit is an audit event staged on the aggregate before Repository.Save.
// Event metadata is filled when the repository persists to the audit stream.
type PendingAudit struct {
	Event     DomainEvent
	Entity    Entity
	EventID   uuid.UUID
	Timestamp int64
}

// NewAggregate creates a new global-scoped aggregate with the specified area and ID.
// It panics when the aggregate definition is invalid, such as when the ID is nil
// or the area is empty.
func NewAggregate(ctx context.Context, area string, id uuid.UUID) Aggregate {
	return newAggregate(ctx, ScopeGlobal, area, uuid.Nil, id)
}

// NewTenantAggregate creates a new tenant-scoped aggregate with the specified area, tenant ID, and aggregate ID.
// It panics when the aggregate definition is invalid, such as when the tenant ID
// or aggregate ID is nil or the area is empty.
func NewTenantAggregate(ctx context.Context, area string, tenantID, id uuid.UUID) Aggregate {
	if tenantID == uuid.Nil {
		panic(errNewTenantAggregateNilTenantID)
	}

	return newAggregate(ctx, ScopeTenant, area, tenantID, id)
}

func newAggregate(ctx context.Context, scope Scope, area string, tenantID, id uuid.UUID) Aggregate {
	if id == uuid.Nil {
		panic(errNewAggregateNilID)
	}
	if area == "" {
		panic(errNewAggregateEmptyArea)
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

	correlationID := GetCorrelationID(ctx)
	if correlationID == uuid.Nil {
		correlationID = uuid.New()
	}
	causationID := GetCausationID(ctx)
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
	pendingAudits []PendingAudit
	handlers      map[string]DomainEventHandler
}

func (a *aggregateBase) GetEntity() Entity {
	return a.entity
}

func (a *aggregateBase) GetAggregateID() uuid.UUID {
	return a.entity.ID
}

func (a *aggregateBase) GetAggregateArea() string {
	return a.entity.Area
}

// GetAggregateSpace is deprecated. Use GetAggregateArea.
func (a *aggregateBase) GetAggregateSpace() string {
	return a.GetAggregateArea()
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

func (a *aggregateBase) GetPendingAudits() []PendingAudit {
	out := make([]PendingAudit, len(a.pendingAudits))
	copy(out, a.pendingAudits)
	return out
}

func (a *aggregateBase) DiscardPendingAudits() {
	a.pendingAudits = nil
}

func (a *aggregateBase) TrimPendingAudits(n int) {
	if n <= 0 {
		return
	}
	if n > len(a.pendingAudits) {
		n = len(a.pendingAudits)
	}
	a.pendingAudits = a.pendingAudits[n:]
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
		panic(fmt.Sprintf(errRegisterHandlerAlreadyExists, discriminator))
	}

	a.handlers[discriminator] = handler
}

// Raise applies an event to the aggregate, validates its area, and adds it to uncommitted events.
// Business-rule failures should be returned by aggregate command methods before
// calling Raise. The default aggregate implementation panics when the event
// definition is not valid for the aggregate because invalid event-area mappings
// are treated as design-time wiring errors.
func (a *aggregateBase) Raise(event DomainEvent) error {
	domainArea := a.entity.Area
	if !eventListsArea(event, domainArea) {
		panic(fmt.Sprintf(errRaiseInvalidAggregateArea, domainArea, event))
	}

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

// Audit stages an immutable audit fact for a derived batch stream (see AuditStreamEntity).
// It does not run domain handlers,
// does not append to uncommitted events, and is never replayed by Load.
//
// The event's GetAreas() must include the domain aggregate area.
func (a *aggregateBase) Audit(event DomainEvent) error {
	domainArea := a.entity.Area
	if !eventListsArea(event, domainArea) {
		panic(fmt.Sprintf(errAuditInvalidAggregateArea, domainArea, event))
	}
	if eventAlreadyStaged(a.pendingAudits, event) {
		panic("Audit: event instance must not be staged more than once")
	}

	auditEntity := AuditStreamEntity(a.entity)
	if len(a.pendingAudits) > 0 {
		auditEntity = a.pendingAudits[0].Entity
	}
	a.pendingAudits = append(a.pendingAudits, PendingAudit{
		Event:     event,
		Entity:    auditEntity,
		EventID:   uuid.New(),
		Timestamp: timestamp.GetTimestamp(),
	})
	return nil
}

func eventListsArea(event DomainEvent, area string) bool {
	for _, candidate := range eventAreas(event) {
		if candidate == area {
			return true
		}
	}
	return false
}

func eventAlreadyStaged(pending []PendingAudit, event DomainEvent) bool {
	if event == nil {
		return false
	}

	eventValue := reflect.ValueOf(event)
	if eventValue.Kind() != reflect.Pointer || eventValue.IsNil() {
		return false
	}

	eventPtr := eventValue.Pointer()
	for _, staged := range pending {
		if staged.Event == nil {
			continue
		}

		stagedValue := reflect.ValueOf(staged.Event)
		if stagedValue.Kind() != reflect.Pointer || stagedValue.IsNil() {
			continue
		}

		if stagedValue.Pointer() == eventPtr {
			return true
		}
	}

	return false
}

// Load replays committed events onto an aggregate
func (a *aggregateBase) Load(events []DomainEvent) error {
	for _, event := range events {
		a.applyEvent(event)
		a.AppendCommitted(event)
	}
	return nil
}
