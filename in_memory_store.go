package es

import (
	"fmt"
	"sync"

	"github.com/google/uuid"
)

var _ EventStore = (*InMemoryEventStore)(nil)

type InMemoryEventStore struct {
	events map[uuid.UUID][]DomainEvent
	mu     sync.RWMutex
}

func NewInMemoryEventStore() *InMemoryEventStore {
	return &InMemoryEventStore{
		events: make(map[uuid.UUID][]DomainEvent),
	}
}

func (s *InMemoryEventStore) SaveEvents(aggregateID uuid.UUID, events []DomainEvent, expectedVersion uint64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	existing := s.events[aggregateID]
	currentVersion := uint64(len(existing))
	if expectedVersion != currentVersion {
		return fmt.Errorf("version mismatch: expected %d, got %d", expectedVersion, len(existing))
	}

	s.events[aggregateID] = append(existing, events...)
	return nil
}

func (s *InMemoryEventStore) LoadEvents(aggregateID uuid.UUID, minVersion uint64) ([]DomainEvent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	events, ok := s.events[aggregateID]
	if !ok {
		return nil, fmt.Errorf("no events found for aggregate ID %s", aggregateID)
	}
	return events, nil
}
