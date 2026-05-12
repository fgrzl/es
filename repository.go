package es

import (
	"context"
	"strconv"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

// Repository provides high-level operations for loading and saving aggregates.
// It coordinates between aggregates and the underlying event store.
type Repository interface {
	// Load reconstructs an aggregate from its stored events.
	Load(context.Context, Aggregate) error

	// Save persists uncommitted events from an aggregate to the store.
	Save(context.Context, Aggregate) error
}

type repository struct {
	store Store
}

// NewRepository creates a new repository with the given event store.
func NewRepository(store Store) Repository {
	return &repository{store: store}
}

func (r *repository) Load(ctx context.Context, a Aggregate) error {
	entity := a.GetEntity()
	ctx, span := startSpan(ctx, spanRepositoryLoad, entity)
	defer span.End()

	events, err := r.store.LoadEvents(ctx, entity, 0)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	span.SetAttributes(attribute.Int(attributeEventsCount, len(events)))

	err = a.Load(events)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	return nil
}

func (r *repository) Save(ctx context.Context, a Aggregate) error {
	entity := a.GetEntity()
	uncommitted := a.GetUncommittedEvents()
	expectedSequence := a.GetCommittedSequence()
	ctx, span := startSpan(
		ctx,
		spanRepositorySave,
		entity,
		attribute.Int(attributeEventsCount, len(uncommitted)),
		attribute.String(attributeSequenceExpected, strconv.FormatUint(expectedSequence, 10)),
		attribute.String(attributeSequenceCurrent, strconv.FormatUint(a.GetUncommittedSequence(), 10)),
	)
	defer span.End()

	if len(uncommitted) == 0 {
		return nil
	}

	err := r.store.SaveEvents(ctx, entity, uncommitted, expectedSequence)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	a.Commit()
	return nil
}
