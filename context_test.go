package es

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestShouldCreateContextWithEventMetadata(t *testing.T) {
	// Arrange
	originalCtx := context.Background()
	entity := NewEntity(uuid.New(), "test-area")

	event := &mockDomainEvent{
		DomainEventBase: &DomainEventBase{
			Metadata: EventMetadata{
				Entity:        entity,
				EventID:       uuid.New(),
				CorrelationID: uuid.New(),
				CausationID:   uuid.New(),
				Timestamp:     123456789,
				Sequence:      1,
			},
		},
	}

	// Act
	newCtx := WithEventMetadata(originalCtx, event)

	// Assert
	assert.NotNil(t, newCtx)
	assert.NotEqual(t, originalCtx, newCtx)
}

func TestShouldPreserveExistingContextValues(t *testing.T) {
	// Arrange
	type contextKey string
	key := contextKey("test-key")
	value := "test-value"

	originalCtx := context.WithValue(context.Background(), key, value)
	entity := NewEntity(uuid.New(), "test-area")

	event := &mockDomainEvent{
		DomainEventBase: &DomainEventBase{
			Metadata: EventMetadata{
				Entity:        entity,
				EventID:       uuid.New(),
				CorrelationID: uuid.New(),
				CausationID:   uuid.New(),
				Timestamp:     123456789,
				Sequence:      1,
			},
		},
	}

	// Act
	newCtx := WithEventMetadata(originalCtx, event)

	// Assert
	assert.Equal(t, value, newCtx.Value(key))
}
