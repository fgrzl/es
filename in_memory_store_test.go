package es

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInMemoryEventStore_Save(t *testing.T) {
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

func TestInMemoryEventStore_Load(t *testing.T) {
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

func TestInMemoryEventStore_SaveEvents_VersionMismatch(t *testing.T) {

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
}

func TestInMemoryEventStore_LoadEvents_NoEvents(t *testing.T) {

	// Arrange
	store := NewInMemoryEventStore()
	ctx := context.Background()

	dummy := NewDummy()
	entity := dummy.GetEntity()

	// Act
	loadedEvents, err := store.LoadEvents(ctx, entity, 0)

	// Assert
	assert.NoError(t, err, "expected no error when loading events")
	assert.Equal(t, len(loadedEvents), 0, "expected no events loaded")
}
