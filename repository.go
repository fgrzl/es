package es

import (
	"context"
	"strconv"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

// AuditRouter resolves the event store entity for a staged audit event.
// When set on Repository via WithAuditRouter, it replaces the default AuditStreamEntity routing.
type AuditRouter func(ctx context.Context, agg Aggregate, event DomainEvent) (Entity, error)

// RepositoryOption configures Repository construction.
type RepositoryOption func(*repository)

// WithAuditRouter sets a custom audit stream resolver. When nil (default),
// audit events use the derived batch stream assigned by Aggregate.Audit.
func WithAuditRouter(router AuditRouter) RepositoryOption {
	return func(r *repository) {
		r.auditRouter = router
	}
}

// Repository provides high-level operations for loading and saving aggregates.
// It coordinates between aggregates and the underlying event store.
type Repository interface {
	// Load reconstructs an aggregate from its stored events.
	Load(context.Context, Aggregate) error

	// Save persists uncommitted domain events and pending audit events.
	// Audit streams are written first (derived batch entity or AuditRouter), then the domain stream.
	Save(context.Context, Aggregate) error
}

type repository struct {
	store       Store
	auditRouter AuditRouter
}

// NewRepository creates a new repository with the given event store and optional configuration.
func NewRepository(store Store, opts ...RepositoryOption) Repository {
	r := &repository{store: store}
	for _, o := range opts {
		o(r)
	}
	return r
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

	batches, err := groupPendingAuditsByStream(ctx, r, a, pending)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

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

		err = r.store.SaveEvents(ctxAudit, auditEntity, events, 0)
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

func groupPendingAuditsByStream(ctx context.Context, r *repository, a Aggregate, pending []PendingAudit) ([]auditStreamBatch, error) {
	if len(pending) == 0 {
		return nil, nil
	}
	batchesByEntity := make(map[Entity]int)
	var batches []auditStreamBatch
	for _, pa := range pending {
		ent, err := r.resolveAuditEntity(ctx, a, pa)
		if err != nil {
			return nil, err
		}
		if index, exists := batchesByEntity[ent]; exists {
			batches[index].items = append(batches[index].items, pa)
			continue
		}

		batchesByEntity[ent] = len(batches)
		batches = append(batches, auditStreamBatch{entity: ent, items: []PendingAudit{pa}})
	}
	return batches, nil
}

func (r *repository) resolveAuditEntity(ctx context.Context, a Aggregate, audit PendingAudit) (Entity, error) {
	if r.auditRouter != nil {
		return r.auditRouter(ctx, a, audit.Event)
	}
	return audit.Entity, nil
}
