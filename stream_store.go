package es

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/fgrzl/enumerators"
	"github.com/fgrzl/json/polymorphic"
	"github.com/fgrzl/streams"
)

func NewStreamStore(client streams.Client) Store {
	return &streamStore{
		client: client,
	}
}

type streamStore struct {
	client streams.Client
}

// LoadEvents implements Store.
func (s *streamStore) LoadEvents(ctx context.Context, entity Entity, minSequence uint64) ([]DomainEvent, error) {
	args := &streams.ConsumeSegment{
		Space:       entity.Type,
		Segment:     entity.ID.String(),
		MinSequence: minSequence,
	}
	entries := s.client.ConsumeSegment(ctx, args)
	domainEvents := enumerators.Map(
		entries,
		func(entry *streams.Entry) (DomainEvent, error) {

			envelope := &polymorphic.Envelope{}
			if err := json.Unmarshal(entry.Payload, envelope); err != nil {
				return nil, err
			}
			domainEvent, ok := envelope.Content.(DomainEvent)
			if !ok {
				return nil, fmt.Errorf("failed to cast to DomainEvent: %T", envelope.Content)
			}
			return domainEvent, nil
		})

	return enumerators.ToSlice(domainEvents)
}

// SaveEvents implements Store.
func (s *streamStore) SaveEvents(ctx context.Context, entity Entity, events []DomainEvent, expectedSequence uint64) error {

	space, segment := entity.Type, entity.ID.String()
	records := enumerators.Map(
		enumerators.Slice(events),
		func(event DomainEvent) (*streams.Record, error) {
			envelope := polymorphic.NewEnvelope(event)
			payload, err := json.Marshal(envelope)
			if err != nil {
				return nil, err
			}
			entry := &streams.Record{
				Sequence: event.GetSequence(),
				Payload:  payload,
			}
			return entry, nil
		})
	results := s.client.Produce(ctx, space, segment, records)
	if err := enumerators.Consume(results); err != nil {
		return err
	}

	return nil
}
