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
	return &InMemoryEventStore{
		data: make(map[Entity][]DomainEvent),
	}
}

// InMemoryEventStore provides an in-memory implementation of the Store interface.
// It uses a mutex-protected map to store events keyed by entity.
// This implementation is thread-safe but data is not persisted across restarts.
type InMemoryEventStore struct {
	mu   sync.RWMutex
	data map[Entity][]DomainEvent
}

// LoadEvents implements Store.LoadEvents.
// It retrieves all events for the specified entity starting from the given `sequence` number.
func (s *InMemoryEventStore) LoadEvents(ctx context.Context, entity Entity, sequence uint64) ([]DomainEvent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	events, ok := s.data[entity]
	if !ok {
		return []DomainEvent{}, nil
	}

	result := make([]DomainEvent, 0, len(events))
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
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.data == nil {
		s.data = make(map[Entity][]DomainEvent)
	}

	existing := s.data[entity]
	currentSequence := uint64(len(existing))
	if expectedSequence != currentSequence {
		return concurrencyError{expectedSequence: expectedSequence, currentSequence: currentSequence}
	}

	newEvents := make([]DomainEvent, 0, len(existing)+len(events))
	newEvents = append(newEvents, existing...)
	newEvents = append(newEvents, events...)
	s.data[entity] = newEvents
	return nil
}

type concurrencyError struct {
	expectedSequence uint64
	currentSequence  uint64
}

func (e concurrencyError) Error() string {
	return fmt.Sprintf("version mismatch: expected %d, got %d", e.expectedSequence, e.currentSequence)
}

func (e concurrencyError) Unwrap() error {
	return ErrConcurrency
}
