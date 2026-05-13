package es

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

// MockStore for testing repository behavior
type MockStore struct {
	mock.Mock
}

func (m *MockStore) LoadEvents(ctx context.Context, entity Entity, minSequence uint64) ([]DomainEvent, error) {
	args := m.Called(ctx, entity, minSequence)
	return args.Get(0).([]DomainEvent), args.Error(1)
}

func (m *MockStore) SaveEvents(ctx context.Context, entity Entity, events []DomainEvent, expectedSequence uint64) error {
	args := m.Called(ctx, entity, events, expectedSequence)
	return args.Error(0)
}

func TestShouldSaveAggregateSuccessfully(t *testing.T) {
	// Arrange
	mockStore := new(MockStore)
	repo := NewRepository(mockStore)
	dummy := NewDummy()
	_ = dummy.Create("test")

	mockStore.On("SaveEvents", mock.Anything, dummy.GetEntity(), dummy.GetUncommittedEvents(), uint64(0)).Return(nil)

	// Act
	err := repo.Save(context.Background(), dummy)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, uint64(1), dummy.GetCommittedSequence())
	assert.Len(t, dummy.GetUncommittedEvents(), 0)
	mockStore.AssertExpectations(t)
}

func TestShouldReturnErrorWhenSaveFails(t *testing.T) {
	// Arrange
	mockStore := new(MockStore)
	repo := NewRepository(mockStore)
	dummy := NewDummy()
	_ = dummy.Create("test")

	expectedError := errors.New("save failed")
	mockStore.On("SaveEvents", mock.Anything, dummy.GetEntity(), dummy.GetUncommittedEvents(), uint64(0)).Return(expectedError)

	// Act
	err := repo.Save(context.Background(), dummy)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, expectedError, err)
	mockStore.AssertExpectations(t)
}

func TestShouldSkipSaveWhenNoUncommittedEvents(t *testing.T) {
	// Arrange
	mockStore := new(MockStore)
	repo := NewRepository(mockStore)
	dummy := NewDummy()

	// Act
	err := repo.Save(context.Background(), dummy)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, uint64(0), dummy.GetCommittedSequence())
	mockStore.AssertNotCalled(t, "SaveEvents")
}

func TestShouldLoadAggregateSuccessfully(t *testing.T) {
	// Arrange
	mockStore := new(MockStore)
	repo := NewRepository(mockStore)
	dummy := NewDummy()

	// Create an event to load
	event := &DummyCreated{Name: "loaded"}
	events := []DomainEvent{event}

	mockStore.On("LoadEvents", mock.Anything, dummy.GetEntity(), uint64(0)).Return(events, nil)

	// Act
	err := repo.Load(context.Background(), dummy)

	// Assert
	assert.NoError(t, err)
	mockStore.AssertExpectations(t)
}

func TestShouldReturnErrorWhenLoadFails(t *testing.T) {
	// Arrange
	mockStore := new(MockStore)
	repo := NewRepository(mockStore)
	dummy := NewDummy()

	expectedError := errors.New("load failed")
	mockStore.On("LoadEvents", mock.Anything, dummy.GetEntity(), uint64(0)).Return([]DomainEvent(nil), expectedError)

	// Act
	err := repo.Load(context.Background(), dummy)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, expectedError, err)
	mockStore.AssertExpectations(t)
}

func TestShouldSaveAggregateWithCommittedEvents(t *testing.T) {
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

func TestShouldSkipSaveWhenAggregateHasNoUncommittedEvents(t *testing.T) {
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

func TestShouldCreateSpanWhenLoadingAggregate(t *testing.T) {
	// Arrange
	spanRecorder := setupSpanRecorder(t)
	mockStore := new(MockStore)
	repo := NewRepository(mockStore)
	dummy := NewDummy()
	correlationID := uuid.New()
	causationID := uuid.New()
	ctx := ContextWithTracing(context.Background(), correlationID, causationID)
	event := &DummyCreated{Name: "loaded"}
	events := []DomainEvent{event}

	mockStore.On(
		"LoadEvents",
		mock.MatchedBy(hasSpanContext),
		dummy.GetEntity(),
		uint64(0),
	).Return(events, nil)

	// Act
	err := repo.Load(ctx, dummy)

	// Assert
	assert.NoError(t, err)
	spans := spanRecorder.Ended()
	assert.Len(t, spans, 1)
	assertRepositorySpanAttributes(t, spans[0], spanRepositoryLoad, dummy.GetEntity(), correlationID, causationID)
	assertSpanInt64Attribute(t, spans[0], attributeEventsCount, int64(len(events)))
	assert.Equal(t, codes.Unset, spans[0].Status().Code)
	mockStore.AssertExpectations(t)
}

func TestShouldCreateSpanWhenSavingAggregate(t *testing.T) {
	// Arrange
	spanRecorder := setupSpanRecorder(t)
	mockStore := new(MockStore)
	repo := NewRepository(mockStore)
	dummy := NewDummy()
	correlationID := uuid.New()
	causationID := uuid.New()
	ctx := ContextWithTracing(context.Background(), correlationID, causationID)
	_ = dummy.Create("test")
	uncommitted := dummy.GetUncommittedEvents()

	mockStore.On(
		"SaveEvents",
		mock.MatchedBy(hasSpanContext),
		dummy.GetEntity(),
		uncommitted,
		uint64(0),
	).Return(nil)

	// Act
	err := repo.Save(ctx, dummy)

	// Assert
	assert.NoError(t, err)
	spans := spanRecorder.Ended()
	assert.Len(t, spans, 1)
	assertRepositorySpanAttributes(t, spans[0], spanRepositorySave, dummy.GetEntity(), correlationID, causationID)
	assertSpanInt64Attribute(t, spans[0], attributeEventsCount, int64(len(uncommitted)))
	assertSpanStringAttribute(t, spans[0], attributeSequenceExpected, "0")
	assertSpanStringAttribute(t, spans[0], attributeSequenceCurrent, "1")
	assert.Equal(t, codes.Unset, spans[0].Status().Code)
	mockStore.AssertExpectations(t)
}

func TestShouldRecordSpanErrorWhenLoadFails(t *testing.T) {
	// Arrange
	spanRecorder := setupSpanRecorder(t)
	mockStore := new(MockStore)
	repo := NewRepository(mockStore)
	dummy := NewDummy()
	correlationID := uuid.New()
	causationID := uuid.New()
	ctx := ContextWithTracing(context.Background(), correlationID, causationID)
	expectedError := errors.New("load failed")

	mockStore.On(
		"LoadEvents",
		mock.MatchedBy(hasSpanContext),
		dummy.GetEntity(),
		uint64(0),
	).Return([]DomainEvent(nil), expectedError)

	// Act
	err := repo.Load(ctx, dummy)

	// Assert
	assert.ErrorIs(t, err, expectedError)
	spans := spanRecorder.Ended()
	assert.Len(t, spans, 1)
	assert.Equal(t, spanRepositoryLoad, spans[0].Name())
	assert.Equal(t, codes.Error, spans[0].Status().Code)
	assert.Equal(t, expectedError.Error(), spans[0].Status().Description)
	mockStore.AssertExpectations(t)
}

func TestShouldRecordSpanErrorWhenSaveFails(t *testing.T) {
	// Arrange
	spanRecorder := setupSpanRecorder(t)
	mockStore := new(MockStore)
	repo := NewRepository(mockStore)
	dummy := NewDummy()
	correlationID := uuid.New()
	causationID := uuid.New()
	ctx := ContextWithTracing(context.Background(), correlationID, causationID)
	_ = dummy.Create("test")
	expectedError := errors.New("save failed")

	mockStore.On(
		"SaveEvents",
		mock.MatchedBy(hasSpanContext),
		dummy.GetEntity(),
		dummy.GetUncommittedEvents(),
		uint64(0),
	).Return(expectedError)

	// Act
	err := repo.Save(ctx, dummy)

	// Assert
	assert.ErrorIs(t, err, expectedError)
	spans := spanRecorder.Ended()
	assert.Len(t, spans, 1)
	assert.Equal(t, spanRepositorySave, spans[0].Name())
	assert.Equal(t, codes.Error, spans[0].Status().Code)
	assert.Equal(t, expectedError.Error(), spans[0].Status().Description)
	mockStore.AssertExpectations(t)
}

func TestShouldPersistAuditsBeforeDomainStream(t *testing.T) {
	ctx := context.Background()
	store := NewInMemoryEventStore()
	repo := NewRepository(store)
	dummy := NewDummy()
	_ = dummy.LogAudit("login")
	_ = dummy.Create("alice")
	auditEntity := dummy.GetPendingAudits()[0].Entity

	err := repo.Save(ctx, dummy)
	assert.NoError(t, err)

	auditEvents, err := store.LoadEvents(ctx, auditEntity, 0)
	assert.NoError(t, err)
	domainEvents, err := store.LoadEvents(ctx, dummy.GetEntity(), 0)
	assert.NoError(t, err)
	assert.Len(t, auditEvents, 1)
	assert.Len(t, domainEvents, 1)
	meta := auditEvents[0].GetMetadata()
	assert.Equal(t, auditEntity, meta.Entity)
	assert.Equal(t, uint64(1), meta.Sequence)
	assert.NotEqual(t, dummy.GetEntity().ID, meta.Entity.ID)
	assert.Equal(t, dummy.GetEntity().Area, meta.Entity.Area)
}

func TestShouldSavePendingAuditsWithoutDomainUncommitted(t *testing.T) {
	ctx := context.Background()
	store := NewInMemoryEventStore()
	repo := NewRepository(store)
	dummy := NewDummy()
	_ = dummy.LogAudit("peek")
	auditEntity := dummy.GetPendingAudits()[0].Entity

	err := repo.Save(ctx, dummy)
	assert.NoError(t, err)

	auditEvents, err := store.LoadEvents(ctx, auditEntity, 0)
	assert.NoError(t, err)
	assert.Len(t, auditEvents, 1)
	domainEvents, err := store.LoadEvents(ctx, dummy.GetEntity(), 0)
	assert.NoError(t, err)
	assert.Len(t, domainEvents, 0)
	assert.Len(t, dummy.GetPendingAudits(), 0)
}

func TestShouldTrimPendingAuditsWhenDomainSaveFailsAfterAuditPersisted(t *testing.T) {
	mockStore := new(MockStore)
	repo := NewRepository(mockStore)
	dummy := NewDummy()
	_ = dummy.LogAudit("a")
	_ = dummy.Create("b")

	auditEnt := dummy.GetPendingAudits()[0].Entity
	mockStore.On("SaveEvents", mock.Anything, auditEnt, mock.Anything, uint64(0)).Return(nil).Once()
	domainFail := errors.New("domain failed")
	mockStore.On("SaveEvents", mock.Anything, dummy.GetEntity(), mock.Anything, uint64(0)).Return(domainFail).Once()

	err := repo.Save(context.Background(), dummy)
	assert.ErrorIs(t, err, domainFail)
	assert.Len(t, dummy.GetPendingAudits(), 0)
	mockStore.AssertExpectations(t)
}

func TestShouldCreateSpanWhenSaveIsNoOp(t *testing.T) {
	// Arrange
	spanRecorder := setupSpanRecorder(t)
	mockStore := new(MockStore)
	repo := NewRepository(mockStore)
	dummy := NewDummy()
	correlationID := uuid.New()
	causationID := uuid.New()
	ctx := ContextWithTracing(context.Background(), correlationID, causationID)

	// Act
	err := repo.Save(ctx, dummy)

	// Assert
	assert.NoError(t, err)
	spans := spanRecorder.Ended()
	assert.Len(t, spans, 1)
	assertRepositorySpanAttributes(t, spans[0], spanRepositorySave, dummy.GetEntity(), correlationID, causationID)
	assertSpanInt64Attribute(t, spans[0], attributeEventsCount, 0)
	assertSpanInt64Attribute(t, spans[0], attributePendingAuditCount, 0)
	assertSpanStringAttribute(t, spans[0], attributeSequenceExpected, "0")
	assertSpanStringAttribute(t, spans[0], attributeSequenceCurrent, "0")
	assert.Equal(t, codes.Unset, spans[0].Status().Code)
	mockStore.AssertNotCalled(t, "SaveEvents")
}

func setupSpanRecorder(t *testing.T) *tracetest.SpanRecorder {
	t.Helper()

	spanRecorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(spanRecorder))
	previousProvider := otel.GetTracerProvider()
	otel.SetTracerProvider(provider)

	t.Cleanup(func() {
		assert.NoError(t, provider.Shutdown(context.Background()))
		otel.SetTracerProvider(previousProvider)
	})

	return spanRecorder
}

func hasSpanContext(ctx context.Context) bool {
	return trace.SpanContextFromContext(ctx).IsValid()
}

func assertRepositorySpanAttributes(t *testing.T, span sdktrace.ReadOnlySpan, expectedName string, entity Entity, correlationID, causationID uuid.UUID) {
	t.Helper()

	assert.Equal(t, expectedName, span.Name())
	assertSpanStringAttribute(t, span, attributeEntityID, entity.ID.String())
	assertSpanStringAttribute(t, span, attributeEntityArea, entity.Area)
	assertSpanStringAttribute(t, span, attributeEntityScope, scopeAttributeValue(entity.Scope))
	if entity.Scope == ScopeTenant {
		assertSpanStringAttribute(t, span, attributeEntityTenantID, entity.TenantID.String())
	}
	assertSpanStringAttribute(t, span, attributeCorrelationID, correlationID.String())
	assertSpanStringAttribute(t, span, attributeCausationID, causationID.String())
}

func assertSpanStringAttribute(t *testing.T, span sdktrace.ReadOnlySpan, key, expected string) {
	t.Helper()

	for _, attr := range span.Attributes() {
		if string(attr.Key) != key {
			continue
		}

		assert.Equal(t, expected, attr.Value.AsString())
		return
	}

	assert.Failf(t, "missing span attribute", "expected string attribute %s", key)
}

func assertSpanInt64Attribute(t *testing.T, span sdktrace.ReadOnlySpan, key string, expected int64) {
	t.Helper()

	for _, attr := range span.Attributes() {
		if string(attr.Key) != key {
			continue
		}

		assert.Equal(t, expected, attr.Value.AsInt64())
		return
	}

	assert.Failf(t, "missing span attribute", "expected int64 attribute %s", key)
}
