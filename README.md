[![ci](https://github.com/fgrzl/es/actions/workflows/ci.yaml/badge.svg)](https://github.com/fgrzl/es/actions/workflows/ci.yaml)
[![Dependabot Updates](https://github.com/fgrzl/es/actions/workflows/dependabot/dependabot-updates/badge.svg)](https://github.com/fgrzl/es/actions/workflows/dependabot/dependabot-updates)

# es

A comprehensive event sourcing library for Go providing clean, extensible interfaces for building event-driven applications.

## Features

- **Aggregate Roots**: Event-sourced aggregates with automatic event registration and replay
- **Event Handling**: Type-safe event handlers with generic registration
- **Event Storage**: Pluggable event store interface with in-memory implementation
- **Repository Pattern**: High-level aggregate persistence with optimistic concurrency control
- **Multi-tenancy**: Support for global and tenant-scoped aggregates
- **Context Propagation**: Built-in correlation and causation tracking
- **OpenTelemetry Spans**: Repository load and save operations emit OTEL spans with aggregate metadata

## Installation

```bash
go get github.com/fgrzl/es
```

## Quick Start

### 1. Define Your Aggregate

```go
package animals

import (
	"context"
	"github.com/fgrzl/es"
	"github.com/google/uuid"
)

type Cat struct {
	es.Aggregate
	Name    string
	Breed   string
	Age     int
	adopted bool
}

func NewCat(id uuid.UUID) *Cat {
	aggregate := &Cat{Aggregate: es.NewAggregate(context.Background(), "cats", id)}
	es.RegisterHandler(aggregate, aggregate.OnCatRenamed)
	es.RegisterHandler(aggregate, aggregate.OnCatAdopted)
	return aggregate
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

### 2. Define Your Events

```go
package animals

import "github.com/fgrzl/es"

func init(){
	// Register events for polymorphic serialization
	es.Register(func() *CatRenamed { return &CatRenamed{} })
	es.Register(func() *CatAdopted { return &CatAdopted{} })
}

type CatRenamed struct {
	es.DomainEventBase
	Name string `json:"name"`
}

func (e *CatRenamed) GetDiscriminator() string { return "cat.renamed" }

type CatAdopted struct{
	es.DomainEventBase
}

func (e *CatAdopted) GetDiscriminator() string { return "cat.adopted" }
```

### 3. Use the Repository

```go
package main

import (
	"context"
	"log"
	"github.com/fgrzl/es"
	"github.com/google/uuid"
)

func main() {
	// Create event store and repository
	store := es.NewInMemoryEventStore()
	repo := es.NewRepository(store)
	
	// Create a new cat
	catID := uuid.New()
	cat := NewCat(catID)
	
	// Perform business operations
	if err := cat.Rename("Whiskers"); err != nil {
		log.Fatal(err)
	}
	
	if err := cat.Adopt(); err != nil {
		log.Fatal(err)
	}
	
	// Save the aggregate
	if err := repo.Save(context.Background(), cat); err != nil {
		log.Fatal(err)
	}
	
	// Later, load the aggregate
	loadedCat := NewCat(catID)
	if err := repo.Load(context.Background(), loadedCat); err != nil {
		log.Fatal(err)
	}
	
	log.Printf("Cat name: %s, adopted: %v", loadedCat.Name, loadedCat.adopted)
}
```

## Core Concepts

### Aggregates

Aggregates are the primary building blocks that represent business entities. They:
- Maintain state through event sourcing
- Enforce business invariants
- Generate domain events when state changes
- Provide methods for business operations

### Domain Events

Events represent facts about what happened in your domain:
- Immutable records of state changes
- Include metadata (correlation ID, causation ID, timestamp, sequence)
- Support polymorphic serialization for storage

### Event Handlers

Type-safe event handlers that apply events to aggregate state:
- Registered using generics for compile-time type safety
- Automatically called when events are raised or loaded
- Keep aggregates in sync with their event stream

### Repository

High-level interface for aggregate persistence:
- Handles event loading and saving
- Provides optimistic concurrency control
- Automatically commits events after successful save

## Multi-tenancy

The library supports both global and tenant-scoped aggregates:

```go
// Global aggregate
globalCat := es.NewAggregate(ctx, "cats", catID)

// Tenant-specific aggregate  
tenantCat := es.NewTenantAggregate(ctx, "cats", tenantID, catID)
```

## Error Handling

The package exports standard sentinel errors for stores and aggregate workflows.
The built-in in-memory store returns errors matching `ErrConcurrency` for optimistic concurrency conflicts.
`Repository.Load` passes through store errors and does not synthesize `ErrNotFound` for empty streams.

## Contributing

1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Ensure all tests pass: `go test ./...`
5. Submit a pull request

## License

This project is licensed under the MIT License - see the LICENSE file for details.
