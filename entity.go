package es

import (
	"encoding/json"

	"github.com/google/uuid"
)

func NewEntity(id uuid.UUID, area string) Entity {
	return Entity{
		ID:    id,
		Area:  area,
		Scope: ScopeGlobal,
	}
}

func NewEnityInArea(area string) Entity {
	return Entity{
		ID:    uuid.New(),
		Area:  area,
		Scope: ScopeGlobal,
	}
}

func NewTenantEntity(tenantID, id uuid.UUID, area string) Entity {
	return Entity{
		ID:       id,
		TenantID: tenantID,
		Area:     area,
		Scope:    ScopeTenant,
	}
}

func NewTenantEnityInArea(tenantID, id uuid.UUID, area string) Entity {
	return Entity{
		ID:       uuid.New(),
		TenantID: tenantID,
		Area:     area,
		Scope:    ScopeTenant,
	}
}

// EmptyEntity represents an uninitialized entity.
var EmptyEntity = Entity{}

type Entity struct {
	ID       uuid.UUID `json:"id"`
	Area     string    `json:"area"`
	TenantID uuid.UUID `json:"tenant_id"`
	Scope    Scope     `json:"scope"`
}

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

func (e *Entity) GetID() uuid.UUID {
	return e.ID
}

func (e *Entity) GetSpace() string {

	if e.TenantID != uuid.Nil {
		return e.TenantID.String() + "." + e.Area
	}

	return e.Area
}

func (e *Entity) GetTenantID() uuid.UUID {
	return e.TenantID
}

func (e *Entity) GetScope() Scope {
	return e.Scope
}

func (e *Entity) GetNamespace() uuid.UUID {
	if e.Area == "" {
		panic("Area is required")
	}

	b := make([]byte, len(e.Area)+16)
	copy(b, e.Area)
	copy(b[len(e.Area):], e.TenantID[:])

	return uuid.NewSHA1(e.ID, b)
}

// IsEmpty checks if an entity is empty.
func (e Entity) IsEmpty() bool {
	return e == EmptyEntity
}
