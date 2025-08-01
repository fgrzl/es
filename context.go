package es

import (
	"context"

	"github.com/fgrzl/messaging"
)

// WithEventMetadata creates a new context with tracing information from a domain event.
// This enables correlation and causation tracking across service boundaries.
func WithEventMetadata(ctx context.Context, event DomainEvent) context.Context {
	metadata := event.GetMetadata()
	ctx = messaging.ContextWithTracing(ctx, metadata.CorrelationID, metadata.CausationID)
	return ctx
}
