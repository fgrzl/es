package es

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

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

func (s *streamStore) LoadEvents(ctx context.Context, entity Entity, minSequence uint64) ([]DomainEvent, error) {
	slog.Debug("Loading events", "space", entity.Space, "segment", entity.ID.String(), "minSequence", minSequence)

	args := &streams.ConsumeSegment{
		Space:       entity.Space,
		Segment:     entity.ID.String(),
		MinSequence: minSequence,
	}

	domainEvents := enumerators.Map(
		s.client.ConsumeSegment(ctx, args),
		func(entry *streams.Entry) (DomainEvent, error) {
			envelope := &polymorphic.Envelope{}
			if err := json.Unmarshal(entry.Payload, envelope); err != nil {
				slog.Error("Failed to unmarshal envelope", "error", err)
				return nil, err
			}
			domainEvent, ok := envelope.Content.(DomainEvent)
			if !ok {
				slog.Error("Invalid DomainEvent type", "actualSpace", fmt.Sprintf("%T", envelope.Content))
				return nil, fmt.Errorf("failed to cast to DomainEvent: %T", envelope.Content)
			}
			return domainEvent, nil
		})

	return enumerators.ToSlice(domainEvents)
}

func (s *streamStore) SaveEvents(ctx context.Context, entity Entity, events []DomainEvent, expectedSequence uint64) error {
	slog.Debug("Saving events", "space", entity.Space, "segment", entity.ID.String(), "expectedSequence", expectedSequence, "eventCount", len(events))

	space, segment := entity.Space, entity.ID.String()
	records := enumerators.Map(
		enumerators.Slice(events),
		func(event DomainEvent) (*streams.Record, error) {
			envelope := polymorphic.NewEnvelope(event)
			payload, err := json.Marshal(envelope)
			if err != nil {
				slog.Error("Failed to marshal event", "error", err)
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
		slog.Error("Failed to produce events", "error", err)
		return err
	}

	slog.Debug("Events saved successfully")
	return nil
}
