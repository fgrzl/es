package es

import (
	"encoding/json"

	"github.com/google/uuid"
)

// NewEntity creates a new global-scoped entity with the specified ID and area.
func NewEntity(id uuid.UUID, area string) Entity {
	return Entity{
		ID:    id,
		Area:  area,
		Scope: ScopeGlobal,
	}
}

// NewEntityInArea creates a new global-scoped entity with a generated ID in the specified area.
func NewEntityInArea(area string) Entity {
	return Entity{
		ID:    uuid.New(),
		Area:  area,
		Scope: ScopeGlobal,
	}
}

// NewTenantEntity creates a new tenant-scoped entity with the specified tenant ID, entity ID, and area.
func NewTenantEntity(tenantID, id uuid.UUID, area string) Entity {
	return Entity{
		ID:       id,
		TenantID: tenantID,
		Area:     area,
		Scope:    ScopeTenant,
	}
}

// NewTenantEntityInArea creates a new tenant-scoped entity with a generated ID in the specified area.
func NewTenantEntityInArea(tenantID, id uuid.UUID, area string) Entity {
	return Entity{
		ID:       uuid.New(),
		TenantID: tenantID,
		Area:     area,
		Scope:    ScopeTenant,
	}
}

// AuditStreamEntity returns a new default audit stream identity derived from a domain aggregate.
// It shares Area, TenantID, and Scope with the domain entity, but uses a fresh ID so
// each audit batch is an independent stream.
func AuditStreamEntity(domain Entity) Entity {
	return Entity{
		ID:       uuid.New(),
		Area:     domain.Area,
		TenantID: domain.TenantID,
		Scope:    domain.Scope,
	}
}

// EmptyEntity represents an uninitialized entity.
var EmptyEntity = Entity{}

// Entity represents an aggregate's identity and scope within the event store.
// It uniquely identifies an aggregate within its scope and area.
type Entity struct {
	ID       uuid.UUID `json:"id"`
	Area     string    `json:"area"`
	TenantID uuid.UUID `json:"tenant_id"`
	Scope    Scope     `json:"scope"`
}

// UnmarshalJSON unmarshals an entity from JSON while preserving backward compatibility with legacy field names.
func (e *Entity) UnmarshalJSON(data []byte) error {
	// Create a shadow type to avoid recursion
	type Alias Entity

	// First try to unmarshal into the new format
	var alias Alias
	if err := json.Unmarshal(data, &alias); err != nil {
		return err
	}

	// Parse raw map for fallback fields
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	// Handle "area" fallback from "space"
	if alias.Area == "" {
		if spaceRaw, ok := raw["space"]; ok {
			var space string
			if err := json.Unmarshal(spaceRaw, &space); err == nil {
				alias.Area = space
			}
		}
	}

	// Handle "id" fallback from "ID"
	if alias.ID == uuid.Nil {
		if idRaw, ok := raw["ID"]; ok {
			var id uuid.UUID
			if err := json.Unmarshal(idRaw, &id); err == nil {
				alias.ID = id
			}
		}
	}

	// Copy fields back
	*e = Entity(alias)
	return nil
}

// GetID returns the entity ID.
func (e *Entity) GetID() uuid.UUID {
	return e.ID
}

// GetTenantID returns the tenant ID.
func (e *Entity) GetTenantID() uuid.UUID {
	return e.TenantID
}

// GetScope returns the entity scope.
func (e *Entity) GetScope() Scope {
	return e.Scope
}

// GetNamespace returns a deterministic namespace UUID for the entity.
// It panics when the entity does not have an area. Use TryGetNamespace to avoid panics.
func (e *Entity) GetNamespace() uuid.UUID {
	namespace, err := e.TryGetNamespace()
	if err != nil {
		panic(err.Error())
	}

	return namespace
}

// TryGetNamespace returns a deterministic namespace UUID for the entity.
func (e *Entity) TryGetNamespace() (uuid.UUID, error) {
	if e.Area == "" {
		return uuid.Nil, wrapSentinelError("Area is required", ErrInvalidEntity)
	}

	b := make([]byte, len(e.Area)+16)
	copy(b, e.Area)
	copy(b[len(e.Area):], e.TenantID[:])

	return uuid.NewSHA1(e.ID, b), nil
}

// IsEmpty checks if an entity is empty.
func (e Entity) IsEmpty() bool {
	return e == EmptyEntity
}
