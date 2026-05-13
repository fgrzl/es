package es

import (
	"context"
	"testing"

	"github.com/fgrzl/timestamp"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

const (
	AreaTest  string = "test-area"
	AreaDummy string = "dummy"
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

func TestShouldRegisterHandlerWithoutCallingNilEventReceiver(t *testing.T) {
	// Arrange
	aggregate := &Dummy{Aggregate: NewAggregate(context.Background(), AreaDummy, uuid.New())}

	// Act & Assert
	assert.NotPanics(t, func() {
		RegisterHandler(aggregate, func(event *panicOnNilDiscriminatorEvent) {})
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

func TestShouldPanicWhenCreatingAggregateWithNilID(t *testing.T) {
	// Arrange
	ctx := context.Background()

	// Act & Assert
	assert.Panics(t, func() {
		NewAggregate(ctx, "test-area", uuid.Nil)
	})
}

func TestShouldPanicWhenRaisingEventWithInvalidArea(t *testing.T) {
	// Arrange
	dummy := NewDummy()

	// Act & Assert
	assert.Panics(t, func() {
		_ = dummy.Raise(&WrongAreaDummyCreated{})
	})
	assert.Len(t, dummy.GetUncommittedEvents(), 0)
}

func TestShouldPanicWhenAuditingWithInvalidArea(t *testing.T) {
	// Arrange
	dummy := NewDummy()

	// Act & Assert
	assert.Panics(t, func() {
		_ = dummy.Audit(&WrongAreaDummyCreated{})
	})
	assert.Len(t, dummy.GetPendingAudits(), 0)
}

func TestShouldStageAuditWithoutRunningDomainHandlers(t *testing.T) {
	// Arrange
	dummy := NewDummy()

	// Act
	err := dummy.Audit(&DummyAuditLogged{Reason: "login"})

	// Assert
	assert.NoError(t, err)
	assert.Empty(t, dummy.GetUncommittedEvents())
	assert.Len(t, dummy.GetPendingAudits(), 1)
	assert.Equal(t, "", dummy.name)
}

func TestShouldReturnIndependentPendingAuditCopies(t *testing.T) {
	// Arrange
	dummy := NewDummy()
	_ = dummy.Audit(&DummyAuditLogged{Reason: "a"})

	// Act
	first := dummy.GetPendingAudits()
	assert.Len(t, first, 1)
	first = append(first, PendingAudit{EventID: uuid.New()})

	// Assert
	assert.Len(t, first, 2)
	assert.Len(t, dummy.GetPendingAudits(), 1)
}

func TestShouldReuseAuditBatchUntilDiscarded(t *testing.T) {
	// Arrange
	dummy := NewDummy()
	_ = dummy.LogAudit("one")
	_ = dummy.LogAudit("two")

	// Act
	pending := dummy.GetPendingAudits()

	// Assert
	assert.Len(t, pending, 2)
	assert.Equal(t, pending[0].Entity, pending[1].Entity)
	assert.NotEqual(t, uuid.Nil, pending[0].Entity.ID)
	assert.NotEqual(t, dummy.GetEntity().ID, pending[0].Entity.ID)
	assert.Equal(t, dummy.GetEntity().Area, pending[0].Entity.Area)

	firstBatchID := pending[0].Entity.ID
	dummy.DiscardPendingAudits()
	_ = dummy.LogAudit("three")

	next := dummy.GetPendingAudits()
	assert.Len(t, next, 1)
	assert.NotEqual(t, firstBatchID, next[0].Entity.ID)
	assert.Equal(t, dummy.GetEntity().Area, next[0].Entity.Area)
}

type DummyCreated struct {
	DomainEventBase
	Name string
}

type panicOnNilDiscriminatorEvent struct {
	DomainEventBase
}

type WrongAreaDummyCreated struct {
	DomainEventBase
}

func (e *DummyCreated) GetDiscriminator() string { return "dummy_created" }
func (e *DummyCreated) GetAreas() []string       { return []string{AreaTest, AreaDummy} }
func (e *DummyCreated) GetSpaces() []string      { return e.GetAreas() }

func (e *panicOnNilDiscriminatorEvent) GetDiscriminator() string {
	if e == nil {
		panic("nil event receiver")
	}

	return "panic_on_nil_discriminator"
}

func (e *panicOnNilDiscriminatorEvent) GetAreas() []string { return []string{AreaDummy} }
func (e *panicOnNilDiscriminatorEvent) GetSpaces() []string {
	return e.GetAreas()
}

func (e *WrongAreaDummyCreated) GetDiscriminator() string { return "wrong_area_dummy_created" }
func (e *WrongAreaDummyCreated) GetAreas() []string       { return []string{AreaTest} }
func (e *WrongAreaDummyCreated) GetSpaces() []string      { return e.GetAreas() }

type DummyAuditLogged struct {
	DomainEventBase
	Reason string
}

func (e *DummyAuditLogged) GetDiscriminator() string { return "dummy_audit_logged" }
func (e *DummyAuditLogged) GetAreas() []string       { return []string{AreaDummy} }
func (e *DummyAuditLogged) GetSpaces() []string      { return e.GetAreas() }

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

func (a *Dummy) LogAudit(reason string) error {
	return a.Audit(&DummyAuditLogged{Reason: reason})
}

func (a *Dummy) OnDummyCreated(e *DummyCreated) {
	a.name = e.Name
}
