package es

import "context"

type Store interface {
	SaveEvents(ctx context.Context, entity Entity, events []DomainEvent, expectedSequence uint64) error
	LoadEvents(ctx context.Context, entity Entity, minSequence uint64) ([]DomainEvent, error)
}
