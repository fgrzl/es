package es

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShouldSaveEventsToInMemoryStore(t *testing.T) {
	// Arrange
	store := NewInMemoryEventStore()
	ctx := context.Background()

	dummy := NewDummy()
	dummy.Create("test entity")
	entity := dummy.GetEntity()
	events := dummy.GetUncommittedEvents()

	// Act
	err := store.SaveEvents(ctx, entity, events, 0)

	// Assert
	assert.NoError(t, err)
}

func TestShouldLoadSavedEventsFromInMemoryStore(t *testing.T) {
	// Arrange
	store := NewInMemoryEventStore()
	ctx := context.Background()

	dummy := NewDummy()
	dummy.Create("test entity")
	entity := dummy.GetEntity()
	events := dummy.GetUncommittedEvents()
	err := store.SaveEvents(ctx, entity, events, 0)
	assert.NoError(t, err)

	// Act
	loadedEvents, err := store.LoadEvents(ctx, entity, 0)

	// Assert
	assert.NoError(t, err, "expected no error when loading events")
	assert.Equal(t, len(loadedEvents), len(events), "expected same number of events loaded")

}

func TestShouldReturnErrorWhenVersionMismatchOnSave(t *testing.T) {

	// Arrange
	store := NewInMemoryEventStore()
	ctx := context.Background()

	dummy := NewDummy()
	dummy.Create("test entity")
	entity := dummy.GetEntity()
	events := dummy.GetUncommittedEvents()

	err := store.SaveEvents(ctx, entity, events, 0)
	require.NoError(t, err)
	// Act
	err = store.SaveEvents(ctx, entity, events, 0)

	// Assert
	assert.Error(t, err, "expected version mismatch error")
	assert.ErrorIs(t, err, ErrConcurrency)
}

func TestShouldReturnEmptySliceWhenNoEventsExist(t *testing.T) {

	// Arrange
	store := NewInMemoryEventStore()
	ctx := context.Background()

	dummy := NewDummy()
	entity := dummy.GetEntity()

	// Act
	loadedEvents, err := store.LoadEvents(ctx, entity, 0)

	// Assert
	assert.NoError(t, err, "expected no error when loading events")
	assert.NotNil(t, loadedEvents, "expected an empty slice when no events exist")
	assert.Equal(t, len(loadedEvents), 0, "expected no events loaded")
}

func TestShouldReturnConcurrencyErrorWhenCreatingMissingStreamWithUnexpectedSequence(t *testing.T) {
	// Arrange
	store := NewInMemoryEventStore()
	ctx := context.Background()

	dummy := NewDummy()
	err := dummy.Create("test entity")
	require.NoError(t, err)

	// Act
	err = store.SaveEvents(ctx, dummy.GetEntity(), dummy.GetUncommittedEvents(), 1)

	// Assert
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrConcurrency)
	assert.EqualError(t, err, "version mismatch: expected 1, got 0")
}

func TestShouldReturnConcurrencyErrorWhenConcurrentWritersAppendToSameStream(t *testing.T) {
	// Arrange
	store := NewInMemoryEventStore()
	ctx := context.Background()

	base := NewDummy()
	err := base.Create("base")
	require.NoError(t, err)
	require.NoError(t, store.SaveEvents(ctx, base.GetEntity(), base.GetUncommittedEvents(), 0))

	entity := base.GetEntity()
	writerOne := &Dummy{Aggregate: NewAggregate(context.Background(), entity.Area, entity.ID)}
	RegisterHandler(writerOne, writerOne.OnDummyCreated)
	require.NoError(t, writerOne.Load(base.GetUncommittedEvents()))
	require.NoError(t, writerOne.Create("writer-one"))

	writerTwo := &Dummy{Aggregate: NewAggregate(context.Background(), entity.Area, entity.ID)}
	RegisterHandler(writerTwo, writerTwo.OnDummyCreated)
	require.NoError(t, writerTwo.Load(base.GetUncommittedEvents()))
	require.NoError(t, writerTwo.Create("writer-two"))

	expectedSequence := writerOne.GetCommittedSequence()
	start := make(chan struct{})
	results := make(chan error, 2)

	// Act
	go func() {
		<-start
		results <- store.SaveEvents(ctx, entity, writerOne.GetUncommittedEvents(), expectedSequence)
	}()
	go func() {
		<-start
		results <- store.SaveEvents(ctx, entity, writerTwo.GetUncommittedEvents(), expectedSequence)
	}()
	close(start)

	errOne := <-results
	errTwo := <-results

	// Assert
	resultErrors := []error{errOne, errTwo}
	successes := 0
	conflicts := 0
	for _, resultErr := range resultErrors {
		switch {
		case resultErr == nil:
			successes++
		case errors.Is(resultErr, ErrConcurrency):
			conflicts++
		default:
			t.Fatalf("unexpected save error: %v", resultErr)
		}
	}

	assert.Equal(t, 1, successes)
	assert.Equal(t, 1, conflicts)

	loadedEvents, err := store.LoadEvents(ctx, entity, 0)
	require.NoError(t, err)
	assert.Len(t, loadedEvents, 2)
}
