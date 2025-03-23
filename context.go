package es

import (
	"context"

	"github.com/fgrzl/messaging"
)

func WithEventMetadata(ctx context.Context, event DomainEvent) context.Context {
	metadata := event.GetMetadata()
	ctx = messaging.ContextWithTracing(ctx, metadata.CorrelationID, metadata.CausationID)
	return ctx
}
