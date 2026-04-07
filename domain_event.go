package es

import (
	"github.com/fgrzl/json/polymorphic"
	"github.com/google/uuid"
)

// DomainEvent defines the interface that all domain events must implement.
// It extends polymorphic identity with event sourcing specific metadata and functionality.
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
	Metadata EventMetadata `json:"metadata"`
}

// GetAggregateID returns the aggregate ID stored in the event metadata.
func (e *DomainEventBase) GetAggregateID() uuid.UUID { return e.Metadata.Entity.ID }

// GetArea returns the aggregate area stored in the event metadata.
func (e *DomainEventBase) GetArea() string { return e.Metadata.Entity.Area }

// GetCausationID returns the causation ID stored in the event metadata.
func (e *DomainEventBase) GetCausationID() uuid.UUID { return e.Metadata.CausationID }

// GetCorrelationID returns the correlation ID stored in the event metadata.
func (e *DomainEventBase) GetCorrelationID() uuid.UUID { return e.Metadata.CorrelationID }

// GetEntity returns the entity stored in the event metadata.
func (e *DomainEventBase) GetEntity() Entity { return e.Metadata.Entity }

// GetEventID returns the event ID stored in the event metadata.
func (e *DomainEventBase) GetEventID() uuid.UUID { return e.Metadata.EventID }

// GetMetadata returns the full event metadata.
func (e *DomainEventBase) GetMetadata() EventMetadata { return e.Metadata }

// GetSequence returns the event sequence number.
func (e *DomainEventBase) GetSequence() uint64 { return e.Metadata.Sequence }

// GetScope returns the entity scope stored in the event metadata.
func (e *DomainEventBase) GetScope() Scope { return e.Metadata.Entity.GetScope() }

// GetTenantID returns the tenant ID stored in the event metadata.
func (e *DomainEventBase) GetTenantID() uuid.UUID { return e.Metadata.Entity.GetTenantID() }

// GetTimestamp returns the event timestamp.
func (e *DomainEventBase) GetTimestamp() int64 { return e.Metadata.Timestamp }

// SetMetadata stores metadata on the event if it has not already been assigned.
func (e *DomainEventBase) SetMetadata(metadata EventMetadata) {
	var empty EventMetadata
	if e.Metadata == empty {
		e.Metadata = metadata
	}
}
