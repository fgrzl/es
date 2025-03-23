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
	GetMetadata() EventMetadata
	SetMetadata(metadata EventMetadata)
	GetEventID() uuid.UUID
	GetAggregateID() uuid.UUID
	GetTimestamp() int64
	GetSequence() uint64
}

// DomainEventBase provides common event fields.
type EventMetadata struct {
	AggregateID   uuid.UUID `json:"aggregate_id"`
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

func (e *DomainEventBase) GetMetadata() EventMetadata { return e.Metadata }
func (e *DomainEventBase) SetMetadata(metadata EventMetadata) {
	var empty EventMetadata
	if e.Metadata == empty {
		e.Metadata = metadata
	}
}
func (e *DomainEventBase) GetEventID() uuid.UUID     { return e.Metadata.EventID }
func (e *DomainEventBase) GetAggregateID() uuid.UUID { return e.Metadata.AggregateID }
func (e *DomainEventBase) GetTimestamp() int64       { return e.Metadata.Timestamp }
func (e *DomainEventBase) GetSequence() uint64       { return e.Metadata.Sequence }
