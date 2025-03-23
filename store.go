package es

import "github.com/google/uuid"

type EventStore interface {
	SaveEvents(aggregateID uuid.UUID, events []DomainEvent, expectedVersion uint64) error
	LoadEvents(aggregateID uuid.UUID, minVersion uint64) ([]DomainEvent, error)
}
