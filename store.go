package es

import "context"

// Store defines the interface for persisting and retrieving domain events.
// Implementations should handle concurrent access and ensure consistency.
type Store interface {
	// SaveEvents persists events for an entity with optimistic concurrency control.
	// expectedSequence is used to detect concurrent modifications.
	SaveEvents(ctx context.Context, entity Entity, events []DomainEvent, expectedSequence uint64) error

	// LoadEvents retrieves all events for an entity starting from minSequence.
	// Returns empty slice if no events are found.
	LoadEvents(ctx context.Context, entity Entity, minSequence uint64) ([]DomainEvent, error)
}
