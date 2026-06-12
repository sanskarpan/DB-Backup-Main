package tracing

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// SpanOption is a function that configures a span
type SpanOption func(span trace.Span)

// WithAttributes adds attributes to a span
func WithAttributes(attrs ...attribute.KeyValue) SpanOption {
	return func(span trace.Span) {
		span.SetAttributes(attrs...)
	}
}

// WithError marks a span as errored and records the error
func WithError(err error) SpanOption {
	return func(span trace.Span) {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
	}
}

// StartSpan starts a new span with the given name and options
func StartSpan(ctx context.Context, tracerName, spanName string, opts ...SpanOption) (context.Context, trace.Span) {
	tracer := otel.Tracer(tracerName)
	ctx, span := tracer.Start(ctx, spanName)

	for _, opt := range opts {
		opt(span)
	}

	return ctx, span
}

// EndSpan ends a span and optionally records an error
func EndSpan(span trace.Span, err error) {
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
	span.End()
}

// DatabaseSpan creates a span for database operations
func DatabaseSpan(ctx context.Context, operation, dbName, tableName string) (context.Context, trace.Span) {
	tracer := otel.Tracer("database")
	ctx, span := tracer.Start(ctx, "db."+operation)

	span.SetAttributes(
		attribute.String("db.system", "postgres"),
		attribute.String("db.operation", operation),
		attribute.String("db.name", dbName),
	)

	if tableName != "" {
		span.SetAttributes(attribute.String("db.table", tableName))
	}

	return ctx, span
}

// StorageSpan creates a span for storage operations
func StorageSpan(ctx context.Context, operation, provider, path string) (context.Context, trace.Span) {
	tracer := otel.Tracer("storage")
	ctx, span := tracer.Start(ctx, "storage."+operation)

	span.SetAttributes(
		attribute.String("storage.provider", provider),
		attribute.String("storage.operation", operation),
		attribute.String("storage.path", path),
	)

	return ctx, span
}

// BackupSpan creates a span for backup operations
func BackupSpan(ctx context.Context, backupType, dbName string) (context.Context, trace.Span) {
	tracer := otel.Tracer("backup")
	ctx, span := tracer.Start(ctx, "backup."+backupType)

	span.SetAttributes(
		attribute.String("backup.type", backupType),
		attribute.String("backup.database", dbName),
	)

	return ctx, span
}

// RestoreSpan creates a span for restore operations
func RestoreSpan(ctx context.Context, backupID, dbName string) (context.Context, trace.Span) {
	tracer := otel.Tracer("restore")
	ctx, span := tracer.Start(ctx, "restore")

	span.SetAttributes(
		attribute.String("restore.backup_id", backupID),
		attribute.String("restore.database", dbName),
	)

	return ctx, span
}

// CompressionSpan creates a span for compression operations
func CompressionSpan(ctx context.Context, operation, algorithm string, size int64) (context.Context, trace.Span) {
	tracer := otel.Tracer("compression")
	ctx, span := tracer.Start(ctx, "compression."+operation)

	span.SetAttributes(
		attribute.String("compression.algorithm", algorithm),
		attribute.String("compression.operation", operation),
		attribute.Int64("compression.size", size),
	)

	return ctx, span
}

// EncryptionSpan creates a span for encryption operations
func EncryptionSpan(ctx context.Context, operation, algorithm string) (context.Context, trace.Span) {
	tracer := otel.Tracer("encryption")
	ctx, span := tracer.Start(ctx, "encryption."+operation)

	span.SetAttributes(
		attribute.String("encryption.algorithm", algorithm),
		attribute.String("encryption.operation", operation),
	)

	return ctx, span
}

// HTTPSpan creates a span for HTTP operations
func HTTPSpan(ctx context.Context, method, url string) (context.Context, trace.Span) {
	tracer := otel.Tracer("http")
	ctx, span := tracer.Start(ctx, method+" "+url)

	span.SetAttributes(
		attribute.String("http.method", method),
		attribute.String("http.url", url),
	)

	return ctx, span
}

// AddEvent adds an event to the current span
func AddEvent(ctx context.Context, name string, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	span.AddEvent(name, trace.WithAttributes(attrs...))
}

// AddSpanAttributes adds attributes to the current span
func AddSpanAttributes(ctx context.Context, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(attrs...)
}

// RecordError records an error on the current span
func RecordError(ctx context.Context, err error) {
	if err == nil {
		return
	}
	span := trace.SpanFromContext(ctx)
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
}

// SetSpanStatus sets the status of the current span
func SetSpanStatus(ctx context.Context, code codes.Code, description string) {
	span := trace.SpanFromContext(ctx)
	span.SetStatus(code, description)
}

// TraceFunc wraps a function with a span
func TraceFunc(ctx context.Context, tracerName, spanName string, fn func(context.Context) error) error {
	ctx, span := StartSpan(ctx, tracerName, spanName)
	defer span.End()

	err := fn(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}

	return err
}

// TraceFuncWithResult wraps a function with a span and returns a result
func TraceFuncWithResult[T any](ctx context.Context, tracerName, spanName string, fn func(context.Context) (T, error)) (T, error) {
	ctx, span := StartSpan(ctx, tracerName, spanName)
	defer span.End()

	result, err := fn(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}

	return result, err
}

// MeasureLatency measures the latency of an operation and adds it as an attribute
func MeasureLatency(ctx context.Context, operation string, latencyMs int64) {
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(
		attribute.String("operation", operation),
		attribute.Int64("latency_ms", latencyMs),
	)
}

// RecordMetric records a metric value as a span attribute
func RecordMetric(ctx context.Context, name string, value interface{}) {
	span := trace.SpanFromContext(ctx)

	var attr attribute.KeyValue
	switch v := value.(type) {
	case int:
		attr = attribute.Int(name, v)
	case int64:
		attr = attribute.Int64(name, v)
	case float64:
		attr = attribute.Float64(name, v)
	case string:
		attr = attribute.String(name, v)
	case bool:
		attr = attribute.Bool(name, v)
	default:
		attr = attribute.String(name, fmt.Sprintf("%v", v))
	}

	span.SetAttributes(attr)
}
