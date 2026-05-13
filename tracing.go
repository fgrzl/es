package es

import (
	"context"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type correlationContextKey struct{}

type causationContextKey struct{}

const tracerName = "github.com/fgrzl/es"

const (
	spanRepositoryLoad      = "es.repository.load"
	spanRepositorySave      = "es.repository.save"
	spanRepositorySaveAudit = "es.repository.save_audit"

	attributeEntityID          = "es.entity.id"
	attributeEntityArea        = "es.entity.area"
	attributeEntityScope       = "es.entity.scope"
	attributeEntityTenantID    = "es.entity.tenant_id"
	attributeCorrelationID     = "es.correlation_id"
	attributeCausationID       = "es.causation_id"
	attributeEventsCount       = "es.events.count"
	attributePendingAuditCount = "es.pending_audits.count"
	attributeSequenceExpected  = "es.sequence.expected"
	attributeSequenceCurrent   = "es.sequence.current"
)

// ContextWithTracing adds correlation and causation IDs to the context.
func ContextWithTracing(ctx context.Context, correlationID, causationID uuid.UUID) context.Context {
	ctx = context.WithValue(ctx, correlationContextKey{}, correlationID)
	ctx = context.WithValue(ctx, causationContextKey{}, causationID)
	return ctx
}

// GetCorrelationID retrieves the correlation ID from the context.
func GetCorrelationID(ctx context.Context) uuid.UUID {
	if correlationID, ok := ctx.Value(correlationContextKey{}).(uuid.UUID); ok {
		return correlationID
	}
	return uuid.Nil
}

// GetCausationID retrieves the causation ID from the context.
func GetCausationID(ctx context.Context) uuid.UUID {
	if causationID, ok := ctx.Value(causationContextKey{}).(uuid.UUID); ok {
		return causationID
	}
	return uuid.Nil
}

func startSpan(ctx context.Context, name string, entity Entity, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	if ctx == nil {
		ctx = context.Background()
	}

	spanAttrs := append(entityAttributes(entity), tracingAttributes(ctx)...)
	spanAttrs = append(spanAttrs, attrs...)

	return otel.Tracer(tracerName).Start(ctx, name, trace.WithAttributes(spanAttrs...))
}

func entityAttributes(entity Entity) []attribute.KeyValue {
	attrs := []attribute.KeyValue{
		attribute.String(attributeEntityID, entity.ID.String()),
		attribute.String(attributeEntityArea, entity.Area),
		attribute.String(attributeEntityScope, scopeAttributeValue(entity.Scope)),
	}

	if entity.Scope == ScopeTenant && entity.TenantID != uuid.Nil {
		attrs = append(attrs, attribute.String(attributeEntityTenantID, entity.TenantID.String()))
	}

	return attrs
}

func tracingAttributes(ctx context.Context) []attribute.KeyValue {
	attrs := make([]attribute.KeyValue, 0, 2)

	if correlationID := GetCorrelationID(ctx); correlationID != uuid.Nil {
		attrs = append(attrs, attribute.String(attributeCorrelationID, correlationID.String()))
	}

	if causationID := GetCausationID(ctx); causationID != uuid.Nil {
		attrs = append(attrs, attribute.String(attributeCausationID, causationID.String()))
	}

	return attrs
}

func scopeAttributeValue(scope Scope) string {
	if scope == ScopeTenant {
		return "tenant"
	}

	return "global"
}
