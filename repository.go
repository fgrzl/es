package es

import "context"

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
	events, err := r.store.LoadEvents(ctx, a.GetEntity(), 0)
	if err != nil {
		return err
	}

	return a.Load(events)
}

func (r *repository) Save(ctx context.Context, a Aggregate) error {
	uncommitted := a.GetUncommittedEvents()
	if len(uncommitted) == 0 {
		return nil
	}
	entity := a.GetEntity()
	expectedSequence := a.GetCommittedSequence()
	err := r.store.SaveEvents(ctx, entity, uncommitted, expectedSequence)
	if err != nil {
		return err
	}
	a.Commit()
	return nil
}
