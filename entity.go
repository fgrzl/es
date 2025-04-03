package es

import (
	"github.com/google/uuid"
)

func NewEntity(id uuid.UUID, entityType string) Entity {
	return Entity{
		ID:   id,
		Type: entityType,
	}
}

func NewEntityOfType(entityType string) Entity {
	return Entity{
		ID:   uuid.New(),
		Type: entityType,
	}
}

// EmptyEntity represents an uninitialized entity.
var EmptyEntity = Entity{}

// Entity represents an entity with an ID and a Type. Use this as a value type.
type Entity struct {
	ID   uuid.UUID `json:"-"`
	Type string    `json:"-"`
}

// IsEmpty checks if an entity is empty.
func (e Entity) IsEmpty() bool {
	return e == EmptyEntity
}

func (e Entity) Namespace() uuid.UUID {
	return uuid.NewSHA1(e.ID, []byte(e.Type))
}
