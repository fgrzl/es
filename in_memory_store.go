package es

import (
	"context"
	"fmt"
	"sync"
)

func NewInMemoryEventStore() Store {
	return &InMemoryEventStore{}
}

type InMemoryEventStore struct {
	data sync.Map
}

// LoadEvents implements Store.
func (s *InMemoryEventStore) LoadEvents(ctx context.Context, entity Entity, seequence uint64) ([]DomainEvent, error) {

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
		if event.GetSequence() >= seequence {
			result = append(result, event)
		}
	}
	return result, nil
}

// SaveEvents implements Store.
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
