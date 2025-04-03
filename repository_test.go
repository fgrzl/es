package es

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRepository_Save_Success(t *testing.T) {
	// Arrange
	store := NewInMemoryEventStore()
	repo := NewRepository(store)
	dummy := NewDummy()
	_ = dummy.Create("test")

	// Act
	err := repo.Save(context.Background(), dummy)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, uint64(1), dummy.GetCommittedSequence())
}

func TestRepository_Save_NoUncommittedEvents(t *testing.T) {
	// Arrange
	store := NewInMemoryEventStore()
	repo := NewRepository(store)
	dummy := NewDummy()

	// Act
	err := repo.Save(context.Background(), dummy)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, uint64(0), dummy.GetCommittedSequence())
}
