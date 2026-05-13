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

	// Save persists uncommitted domain events and pending audit events.
	// Audit streams are written first, then the domain stream.
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
	pending := a.GetPendingAudits()
	expectedSequence := a.GetCommittedSequence()

	ctx, span := startSpan(
		ctx,
		spanRepositorySave,
		entity,
		attribute.Int(attributeEventsCount, len(uncommitted)),
		attribute.Int(attributePendingAuditCount, len(pending)),
		attribute.String(attributeSequenceExpected, strconv.FormatUint(expectedSequence, 10)),
		attribute.String(attributeSequenceCurrent, strconv.FormatUint(a.GetUncommittedSequence(), 10)),
	)
	defer span.End()

	if len(uncommitted) == 0 && len(pending) == 0 {
		return nil
	}

	batches := groupPendingAuditsByStream(pending)

	for _, batch := range batches {
		auditEntity := batch.entity
		ctxAudit, spanAudit := startSpan(ctx, spanRepositorySaveAudit, auditEntity,
			attribute.Int(attributeEventsCount, len(batch.items)),
		)

		events := make([]DomainEvent, 0, len(batch.items))
		for i, pa := range batch.items {
			pa.Event.SetMetadata(EventMetadata{
				Entity:        auditEntity,
				EventID:       pa.EventID,
				CorrelationID: a.GetCorrelationID(),
				CausationID:   a.GetCausationID(),
				Timestamp:     pa.Timestamp,
				Sequence:      uint64(i) + 1,
			})
			events = append(events, pa.Event)
		}

		err := r.store.SaveEvents(ctxAudit, auditEntity, events, 0)
		if err != nil {
			spanAudit.RecordError(err)
			spanAudit.SetStatus(codes.Error, err.Error())
			spanAudit.End()
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return err
		}
		a.TrimPendingAudits(len(batch.items))
		spanAudit.End()
	}

	if len(uncommitted) > 0 {
		err := r.store.SaveEvents(ctx, entity, uncommitted, expectedSequence)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return err
		}
	}

	a.Commit()
	a.DiscardPendingAudits()
	return nil
}

type auditStreamBatch struct {
	entity Entity
	items  []PendingAudit
}

func groupPendingAuditsByStream(pending []PendingAudit) []auditStreamBatch {
	if len(pending) == 0 {
		return nil
	}
	batchesByEntity := make(map[Entity]int)
	var batches []auditStreamBatch
	for _, pa := range pending {
		ent := pa.Entity
		if index, exists := batchesByEntity[ent]; exists {
			batches[index].items = append(batches[index].items, pa)
			continue
		}

		batchesByEntity[ent] = len(batches)
		batches = append(batches, auditStreamBatch{entity: ent, items: []PendingAudit{pa}})
	}
	return batches
}
