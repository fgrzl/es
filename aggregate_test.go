package es

import (
	"context"
	"testing"

	"github.com/fgrzl/timestamp"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestShouldApplyEventWhenCreated(t *testing.T) {
	// Arrange
	dummy := NewDummy()

	// Act
	err := dummy.Create("test")

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, "test", dummy.name)
}

func TestShouldRaiseEventAndAddToUncommitted(t *testing.T) {
	// Arrange
	dummy := NewDummy()

	// Act
	err := dummy.Create("test")

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, "test", dummy.name)
	assert.Len(t, dummy.GetUncommittedEvents(), 1)

	event := dummy.GetUncommittedEvents()[0].(*DummyCreated)
	assert.Equal(t, "test", event.Name)
}

func TestShouldCommitUncommittedEventsWhenCommitCalled(t *testing.T) {
	// Arrange
	dummy := NewDummy()
	_ = dummy.Create("test")

	// Act
	dummy.Commit()

	// Assert
	assert.Len(t, dummy.GetUncommittedEvents(), 0)
	assert.Len(t, dummy.GetCommittedEvents(), 1)

	event := dummy.GetCommittedEvents()[0].(*DummyCreated)
	assert.Equal(t, "test", event.Name)
}

func TestShouldLoadEventsAndApplyToAggregate(t *testing.T) {
	// Arrange
	dummy := NewDummy()
	event := &DummyCreated{Name: "loaded"}
	event.SetMetadata(EventMetadata{
		Entity:    dummy.GetEntity(),
		EventID:   uuid.New(),
		Sequence:  1,
		Timestamp: timestamp.GetTimestamp(),
	})

	// Act
	err := dummy.Load([]DomainEvent{event})

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, "loaded", dummy.name)
	assert.Len(t, dummy.GetCommittedEvents(), 1)
	assert.Equal(t, uint64(1), dummy.GetCommittedSequence())
}

func TestShouldPanicWhenRegisteringDuplicateHandler(t *testing.T) {
	// Arrange
	dummy := NewDummy()

	// Act & Assert
	assert.Panics(t, func() {
		RegisterHandler(dummy, dummy.OnDummyCreated)
	})
}

func TestShouldCreateTenantAggregateWithValidTenantID(t *testing.T) {
	// Arrange
	tenantID := uuid.New()
	aggregateID := uuid.New()
	ctx := context.Background()

	// Act
	agg := NewTenantAggregate(ctx, "test-area", tenantID, aggregateID)

	// Assert
	assert.NotNil(t, agg)
	assert.Equal(t, aggregateID, agg.GetAggregateID())
	assert.Equal(t, tenantID, agg.GetEntity().TenantID)
}

func TestShouldPanicWhenCreatingTenantAggregateWithNilTenantID(t *testing.T) {
	// Arrange
	aggregateID := uuid.New()
	ctx := context.Background()

	// Act & Assert
	assert.Panics(t, func() {
		NewTenantAggregate(ctx, "test-area", uuid.Nil, aggregateID)
	})
}

type DummyCreated struct {
	DomainEventBase
	Name string
}

func (e *DummyCreated) GetDiscriminator() string { return "dummy_created" }

func NewDummy() *Dummy {
	id := uuid.New()
	dummy := &Dummy{Aggregate: NewAggregate(context.Background(), "dummy", id)}
	RegisterHandler(dummy, dummy.OnDummyCreated)
	return dummy
}

type Dummy struct {
	Aggregate
	name string
}

func (a *Dummy) Create(name string) error {
	if name != a.name {
		return a.Raise(&DummyCreated{Name: name})
	}
	return nil
}

func (a *Dummy) OnDummyCreated(e *DummyCreated) {
	a.name = e.Name
}
