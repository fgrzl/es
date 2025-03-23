package es

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestShouldApplyEvent(t *testing.T) {
	// Arrange
	dummy := NewDummy()

	// Act
	err := dummy.Create("test")

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, "test", dummy.name)
}

type DummyCreated struct {
	DomainEventBase
	Name string
}

func (e *DummyCreated) GetDiscriminator() string { return "dummy_created" }

func NewDummy() *Dummy {
	id := uuid.New()
	dummy := &Dummy{Aggregate: NewAggregate(context.Background(), id)}
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
