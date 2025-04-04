[![ci](https://github.com/fgrzl/es/actions/workflows/ci.yml/badge.svg)](https://github.com/fgrzl/es/actions/workflows/ci.yml)
[![Dependabot Updates](https://github.com/fgrzl/es/actions/workflows/dependabot/dependabot-updates/badge.svg)](https://github.com/fgrzl/es/actions/workflows/dependabot/dependabot-updates)

# es

Basic event sourcing library for Go with clean, extensible interfaces.

## Features

- Aggregate roots with event registration
- Event raising and applying via handlers
- In-memory event store

## Example: Cat Aggregate

### Aggregate Definition

```go
package animals

import (
	"context"

	"github.com/fgrzl/es"
	"github.com/google/uuid"
)

func NewCat(id uuid.UUID) *Cat {
	aggregate := &Cat{Aggregate: es.NewAggregate(context.Background(), "cats", id)}
	es.RegisterHandler(aggregate, aggregate.OnCatRenamed)
	es.RegisterHandler(aggregate, aggregate.OnCatAdopted)
	return aggregate
}

type Cat struct {
	es.Aggregate
	Name    string
	Breed   string
	Age     int
	adopted bool
}

func (c *Cat) Rename(name string) error {
	if c.Name != name {
		return c.Raise(&CatRenamed{Name: name})
	}
	return nil
}

func (c *Cat) Adopt() error {
	if c.adopted {
		return nil
	}
	return c.Raise(&CatAdopted{})
}

func (c *Cat) OnCatRenamed(e *CatRenamed) {
	c.Name = e.Name
}

func (c *Cat) OnCatAdopted(e *CatAdopted) {
	c.adopted = true
}
```

### Events

```go
func init(){
	// Register the events as polymorphic types
	polymorphic.Register(func() *CatRenamed { return &CatRenamed{} })
	polymorphic.Register(func() *CatRenamed { return &CatRenamed{} })
}

type CatRenamed struct {
	es.DomainEventBase
	Name string
}

func (e *CatRenamed) GetDiscriminator() string { return "cat.renamed" }

type CatAdopted struct{
	es.DomainEventBase
}

func (e *CatRenamed) GetDiscriminator() string { return "cat.adopted" }
```
