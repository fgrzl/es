package es

import (
	"encoding/json"
	"fmt"
	"strings"

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

// MarshalJSON converts the entity to the format "type:id".
func (e *Entity) MarshalJSON() ([]byte, error) {
	if e.ID == uuid.Nil {
		return nil, fmt.Errorf("cannot marshal entity with nil ID")
	}
	return json.Marshal(fmt.Sprintf("%s:%s", e.Type, e.ID))
}

// UnmarshalJSON parses the entity from the format "type:id".
func (e *Entity) UnmarshalJSON(data []byte) error {
	var raw string
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	parts := strings.SplitN(raw, ":", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid entity format: %s", raw)
	}

	parsedID, err := uuid.Parse(parts[1])
	if err != nil {
		return fmt.Errorf("invalid UUID: %w", err)
	}

	e.Type = parts[0]
	e.ID = parsedID
	return nil
}

// IsEmpty checks if an entity is empty.
func (e Entity) IsEmpty() bool {
	return e == EmptyEntity
}

func (e Entity) Namespace() uuid.UUID {
	return uuid.NewSHA1(e.ID, []byte(e.Type))
}
