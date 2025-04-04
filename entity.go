package es

import (
	"github.com/google/uuid"
)

func NewEntity(id uuid.UUID, space string) Entity {
	return Entity{
		ID:    id,
		Space: space,
	}
}

func NewEntityOfSpace(entitySpace string) Entity {
	return Entity{
		ID:    uuid.New(),
		Space: entitySpace,
	}
}

// EmptyEntity represents an uninitialized entity.
var EmptyEntity = Entity{}

// Entity represents an entity with an ID and a Space. Use this as a value type.
type Entity struct {
	ID    uuid.UUID `json:"id"`
	Space string    `json:"space"`
}

// IsEmpty checks if an entity is empty.
func (e Entity) IsEmpty() bool {
	return e == EmptyEntity
}
