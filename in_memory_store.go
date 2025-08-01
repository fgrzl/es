package es

import (
	"context"
	"fmt"
	"sync"
)

// NewInMemoryEventStore creates a new in-memory event store.
// This implementation is primarily intended for testing and development.
// For production use, consider a persistent store implementation.
func NewInMemoryEventStore() Store {
	return &InMemoryEventStore{}
}

// InMemoryEventStore provides an in-memory implementation of the Store interface.
// It uses a concurrent map to store events keyed by entity.
// This implementation is thread-safe but data is not persisted across restarts.
type InMemoryEventStore struct {
	data sync.Map
}

// LoadEvents implements Store.LoadEvents.
// It retrieves all events for the specified entity starting from the given sequence number.
func (s *InMemoryEventStore) LoadEvents(ctx context.Context, entity Entity, sequence uint64) ([]DomainEvent, error) {

	obj, ok := s.data.Load(entity)
	if !ok {
		return nil, nil
	}
	events, ok := obj.([]DomainEvent)
	if !ok {
		return nil, fmt.Errorf("invalid type for events: %T", obj)
	}

	var result []DomainEvent
	for _, event := range events {
		if event.GetSequence() >= sequence {
			result = append(result, event)
		}
	}
	return result, nil
}

// SaveEvents implements Store.SaveEvents.
// It appends new events to the entity's event stream with optimistic concurrency control.
func (s *InMemoryEventStore) SaveEvents(ctx context.Context, entity Entity, events []DomainEvent, expectedSequence uint64) error {
	existing, ok := s.data.Load(entity)
	if !ok {
		s.data.Store(entity, events)
		return nil
	}

	currentSequence := uint64(len(existing.([]DomainEvent)))
	if expectedSequence != currentSequence {
		return fmt.Errorf("version mismatch: expected %d, got %d", expectedSequence, currentSequence)
	}

	newEvents := append(existing.([]DomainEvent), events...)
	s.data.CompareAndSwap(entity, existing, newEvents)
	return nil
}
