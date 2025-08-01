package es

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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
