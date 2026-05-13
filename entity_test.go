package es

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestShouldCreateEntityWithSpecifiedValues(t *testing.T) {
	// Arrange
	id := uuid.New()
	area := "test-area"

	// Act
	entity := NewEntity(id, area)

	// Assert
	assert.Equal(t, id, entity.ID)
	assert.Equal(t, area, entity.Area)
	assert.Equal(t, ScopeGlobal, entity.Scope)
	assert.Equal(t, uuid.Nil, entity.TenantID)
}

func TestShouldCreateEntityInAreaWithGeneratedID(t *testing.T) {
	// Arrange
	area := "test-area"

	// Act
	entity := NewEntityInArea(area)

	// Assert
	assert.NotEqual(t, uuid.Nil, entity.ID)
	assert.Equal(t, area, entity.Area)
	assert.Equal(t, ScopeGlobal, entity.Scope)
	assert.Equal(t, uuid.Nil, entity.TenantID)
}

func TestShouldCreateTenantEntityWithSpecifiedValues(t *testing.T) {
	// Arrange
	tenantID := uuid.New()
	id := uuid.New()
	area := "test-area"

	// Act
	entity := NewTenantEntity(tenantID, id, area)

	// Assert
	assert.Equal(t, id, entity.ID)
	assert.Equal(t, area, entity.Area)
	assert.Equal(t, ScopeTenant, entity.Scope)
	assert.Equal(t, tenantID, entity.TenantID)
}

func TestShouldCreateTenantEntityInAreaWithGeneratedID(t *testing.T) {
	// Arrange
	tenantID := uuid.New()
	id := uuid.New()
	area := "test-area"

	// Act
	entity := NewTenantEntityInArea(tenantID, id, area)

	// Assert
	assert.NotEqual(t, uuid.Nil, entity.ID)
	assert.Equal(t, area, entity.Area)
	assert.Equal(t, ScopeTenant, entity.Scope)
	assert.Equal(t, tenantID, entity.TenantID)
}

func TestShouldCreateAuditStreamEntityWithGeneratedIDAndSameArea(t *testing.T) {
	domain := NewTenantEntity(uuid.New(), uuid.New(), "users")

	audit := AuditStreamEntity(domain)

	assert.NotEqual(t, uuid.Nil, audit.ID)
	assert.NotEqual(t, domain.ID, audit.ID)
	assert.Equal(t, domain.Area, audit.Area)
	assert.Equal(t, domain.TenantID, audit.TenantID)
	assert.Equal(t, domain.Scope, audit.Scope)
}

func TestShouldReturnCorrectAreaForGlobalEntity(t *testing.T) {
	// Arrange
	entity := NewEntity(uuid.New(), "test-area")

	// Act
	area := entity.Area

	// Assert
	assert.Equal(t, "test-area", area)
}

func TestShouldReturnCorrectAreaForTenantEntity(t *testing.T) {
	// Arrange
	tenantID := uuid.New()
	entity := NewTenantEntity(tenantID, uuid.New(), "test-area")

	// Act
	area := entity.Area

	// Assert
	expected := "test-area"
	assert.Equal(t, expected, area)
}

func TestShouldDetectEmptyEntity(t *testing.T) {
	// Arrange
	empty := Entity{}
	nonEmpty := NewEntity(uuid.New(), "test")

	// Act & Assert
	assert.True(t, empty.IsEmpty())
	assert.False(t, nonEmpty.IsEmpty())
	assert.True(t, EmptyEntity.IsEmpty())
}

func TestShouldGenerateUniqueNamespace(t *testing.T) {
	// Arrange
	id := uuid.New()
	entity1 := NewEntity(id, "area1")
	entity2 := NewEntity(id, "area2")
	entity3 := NewTenantEntity(uuid.New(), id, "area1")

	// Act
	ns1 := entity1.GetNamespace()
	ns2 := entity2.GetNamespace()
	ns3 := entity3.GetNamespace()

	// Assert
	assert.NotEqual(t, ns1, ns2) // Different areas should have different namespaces
	assert.NotEqual(t, ns1, ns3) // Different tenants should have different namespaces
	assert.NotEqual(t, ns2, ns3)
}

func TestShouldReturnNamespaceWhenUsingTryGetNamespace(t *testing.T) {
	// Arrange
	entity := NewEntity(uuid.New(), "area1")

	// Act
	namespace, err := entity.TryGetNamespace()

	// Assert
	assert.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, namespace)
}

func TestShouldPanicWhenGettingNamespaceWithEmptyArea(t *testing.T) {
	// Arrange
	entity := Entity{ID: uuid.New()}

	// Act & Assert
	assert.Panics(t, func() {
		entity.GetNamespace()
	})
}

func TestShouldReturnErrorWhenGettingNamespaceWithEmptyAreaUsingTryGetNamespace(t *testing.T) {
	// Arrange
	entity := Entity{ID: uuid.New()}

	// Act
	namespace, err := entity.TryGetNamespace()

	// Assert
	assert.Equal(t, uuid.Nil, namespace)
	assert.ErrorIs(t, err, ErrInvalidEntity)
	assert.EqualError(t, err, "Area is required")
}

func TestShouldReturnCorrectEntityIDForGivenEntity(t *testing.T) {
	// Arrange
	id := uuid.New()
	entity := NewEntity(id, "test-area")

	// Act
	result := entity.GetID()

	// Assert
	assert.Equal(t, id, result)
}

func TestShouldReturnCorrectTenantIDForEntity(t *testing.T) {
	// Arrange
	tenantID := uuid.New()
	entity := NewTenantEntity(tenantID, uuid.New(), "test-area")

	// Act
	result := entity.GetTenantID()

	// Assert
	assert.Equal(t, tenantID, result)
}

func TestShouldReturnCorrectScopeForGlobalEntity(t *testing.T) {
	// Arrange
	entity := NewEntity(uuid.New(), "test-area")

	// Act
	scope := entity.GetScope()

	// Assert
	assert.Equal(t, ScopeGlobal, scope)
}

func TestShouldReturnCorrectScopeForTenantEntity(t *testing.T) {
	// Arrange
	entity := NewTenantEntity(uuid.New(), uuid.New(), "test-area")

	// Act
	scope := entity.GetScope()

	// Assert
	assert.Equal(t, ScopeTenant, scope)
}
