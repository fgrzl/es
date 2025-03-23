package es

type Repository interface {
	Load(a Aggregate) error
	Save(a Aggregate) error
}

type repository struct {
	store EventStore
}

func NewRepository(store EventStore) Repository {
	return &repository{store: store}
}

func (r *repository) Load(a Aggregate) error {
	id := a.GetAggregateID()
	events, err := r.store.LoadEvents(id, 0)
	if err != nil {
		return err
	}

	return a.Load(events)
}

func (r *repository) Save(a Aggregate) error {
	uncommitted := a.GetUncommittedEvents()
	if len(uncommitted) == 0 {
		return nil
	}
	id := a.GetAggregateID()
	version := a.GetCommittedVersion()
	err := r.store.SaveEvents(id, uncommitted, version)
	if err != nil {
		return err
	}
	a.Commit()
	return nil
}
