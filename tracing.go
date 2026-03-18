package es

import (
	"context"

	"github.com/fgrzl/telemetry"
	"github.com/google/uuid"
)

// ContextWithTracing adds correlation and causation IDs to the context.
func ContextWithTracing(ctx context.Context, correlationID, causationID uuid.UUID) context.Context {
	ctx = telemetry.WithCorrelationID(ctx, correlationID)
	ctx = telemetry.WithCausationID(ctx, causationID)
	return ctx
}

// GetCorrelationID retrieves the correlation ID from the context.
func GetCorrelationID(ctx context.Context) uuid.UUID {
	if correlationID, ok := telemetry.CorrelationIDFromContext(ctx); ok {
		return correlationID
	}
	return uuid.Nil
}

// GetCausationID retrieves the causation ID from the context.
func GetCausationID(ctx context.Context) uuid.UUID {
	if causationID, ok := telemetry.CausationIDFromContext(ctx); ok {
		return causationID
	}
	return uuid.Nil
}
