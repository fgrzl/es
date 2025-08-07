package es

import (
	"github.com/fgrzl/json/polymorphic"
	"github.com/fgrzl/messaging"
	"github.com/google/uuid"
)

// Register registers a polymorphic type factory for JSON serialization/deserialization.
// This should be called during package initialization for all domain event types.
func Register[T polymorphic.Polymorphic](factory func() T) {
	polymorphic.Register(factory)
}

// DomainEvent defines the interface that all domain events must implement.
// It extends messaging.Message with event sourcing specific metadata and functionality.
type DomainEvent interface {
	messaging.Message
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

// EventMetadata contains the metadata fields common to all domain events.
type EventMetadata struct {
	Entity        Entity    `json:"entity"`
	EventID       uuid.UUID `json:"event_id"`
	CorrelationID uuid.UUID `json:"correlation_id"`
	CausationID   uuid.UUID `json:"causation_id"`
	Timestamp     int64     `json:"timestamp"`
	Sequence      uint64    `json:"sequence"`
}

// DomainEventBase provides a base implementation of the DomainEvent interface.
// Event types should embed this struct to inherit standard event behavior.
type DomainEventBase struct {
	messaging.Message
	Metadata EventMetadata `json:"metadata"`
}

func (e *DomainEventBase) GetAggregateID() uuid.UUID   { return e.Metadata.Entity.ID }
func (e *DomainEventBase) GetArea() string             { return e.Metadata.Entity.Area }
func (e *DomainEventBase) GetCausationID() uuid.UUID   { return e.Metadata.CausationID }
func (e *DomainEventBase) GetCorrelationID() uuid.UUID { return e.Metadata.CorrelationID }
func (e *DomainEventBase) GetEntity() Entity           { return e.Metadata.Entity }
func (e *DomainEventBase) GetEventID() uuid.UUID       { return e.Metadata.EventID }
func (e *DomainEventBase) GetMetadata() EventMetadata  { return e.Metadata }
func (e *DomainEventBase) GetSequence() uint64         { return e.Metadata.Sequence }
func (e *DomainEventBase) GetScope() Scope             { return e.Metadata.Entity.GetScope() }
func (e *DomainEventBase) GetTenantID() uuid.UUID      { return e.Metadata.Entity.GetTenantID() }
func (e *DomainEventBase) GetTimestamp() int64         { return e.Metadata.Timestamp }
func (e *DomainEventBase) SetMetadata(metadata EventMetadata) {
	var empty EventMetadata
	if e.Metadata == empty {
		e.Metadata = metadata
	}
}
