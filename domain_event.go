package es

import (
	"github.com/fgrzl/json/polymorphic"
	"github.com/fgrzl/messaging"
	"github.com/google/uuid"
)

func Register[T polymorphic.Polymorphic](factory func() T) {
	polymorphic.Register(factory)
}

// DomainEvent interface for all events.
type DomainEvent interface {
	messaging.Event
	GetAggregateID() uuid.UUID
	GetCausationID() uuid.UUID
	GetCorrelationID() uuid.UUID
	GetEntity() Entity
	GetEventID() uuid.UUID
	GetMetadata() EventMetadata
	GetSequence() uint64
	GetTimestamp() int64
	SetMetadata(metadata EventMetadata)
}

// DomainEventBase provides common event fields.
type EventMetadata struct {
	Entity        Entity    `json:"entity"`
	EventID       uuid.UUID `json:"event_id"`
	CorrelationID uuid.UUID `json:"correlation_id"`
	CausationID   uuid.UUID `json:"causation_id"`
	Timestamp     int64     `json:"timestamp"`
	Sequence      uint64    `json:"sequence"`
}

type DomainEventBase struct {
	messaging.Event
	Metadata EventMetadata `json:"metadata"`
}

func (e *DomainEventBase) GetAggregateID() uuid.UUID   { return e.Metadata.Entity.ID }
func (e *DomainEventBase) GetAggregateType() string    { return e.Metadata.Entity.Type }
func (e *DomainEventBase) GetCausationID() uuid.UUID   { return e.Metadata.CausationID }
func (e *DomainEventBase) GetCorrelationID() uuid.UUID { return e.Metadata.CorrelationID }
func (e *DomainEventBase) GetEntity() Entity           { return e.Metadata.Entity }
func (e *DomainEventBase) GetEventID() uuid.UUID       { return e.Metadata.EventID }
func (e *DomainEventBase) GetMetadata() EventMetadata  { return e.Metadata }
func (e *DomainEventBase) GetSequence() uint64         { return e.Metadata.Sequence }
func (e *DomainEventBase) GetTimestamp() int64         { return e.Metadata.Timestamp }
func (e *DomainEventBase) SetMetadata(metadata EventMetadata) {
	var empty EventMetadata
	if e.Metadata == empty {
		e.Metadata = metadata
	}
}
