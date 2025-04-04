package es_test

import (
	"testing"
	"time"

	"github.com/fgrzl/es"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

type MockEntity struct {
	ID    uuid.UUID
	Space string
}

func TestDomainEventBase(t *testing.T) {
	entity := es.Entity{
		ID:    uuid.New(),
		Space: "TestEntity",
	}

	metadata := es.EventMetadata{
		Entity:        entity,
		EventID:       uuid.New(),
		CorrelationID: uuid.New(),
		CausationID:   uuid.New(),
		Timestamp:     time.Now().Unix(),
		Sequence:      1,
	}

	event := &es.DomainEventBase{
		Metadata: metadata,
	}

	t.Run("GetAggregateID", func(t *testing.T) {
		assert.Equal(t, metadata.Entity.ID, event.GetAggregateID())
	})

	t.Run("GetAggregateSpace", func(t *testing.T) {
		assert.Equal(t, metadata.Entity.Space, event.GetAggregateSpace())
	})

	t.Run("GetCausationID", func(t *testing.T) {
		assert.Equal(t, metadata.CausationID, event.GetCausationID())
	})

	t.Run("GetCorrelationID", func(t *testing.T) {
		assert.Equal(t, metadata.CorrelationID, event.GetCorrelationID())
	})

	t.Run("GetEntity", func(t *testing.T) {
		assert.Equal(t, metadata.Entity, event.GetEntity())
	})

	t.Run("GetEventID", func(t *testing.T) {
		assert.Equal(t, metadata.EventID, event.GetEventID())
	})

	t.Run("GetMetadata", func(t *testing.T) {
		assert.Equal(t, metadata, event.GetMetadata())
	})

	t.Run("GetSequence", func(t *testing.T) {
		assert.Equal(t, metadata.Sequence, event.GetSequence())
	})

	t.Run("GetTimestamp", func(t *testing.T) {
		assert.Equal(t, metadata.Timestamp, event.GetTimestamp())
	})

	t.Run("SetMetadata", func(t *testing.T) {
		newMetadata := es.EventMetadata{
			Entity:        entity,
			EventID:       uuid.New(),
			CorrelationID: uuid.New(),
			CausationID:   uuid.New(),
			Timestamp:     time.Now().Unix(),
			Sequence:      2,
		}
		event.SetMetadata(newMetadata)
		assert.Equal(t, metadata, event.GetMetadata()) // Metadata should not change if already set
	})
}
