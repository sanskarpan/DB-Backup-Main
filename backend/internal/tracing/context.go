package tracing

import (
	"context"
	"net/http"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// ContextPropagator handles trace context propagation across service boundaries
type ContextPropagator struct {
	propagator propagation.TextMapPropagator
}

// NewContextPropagator creates a new context propagator
func NewContextPropagator() *ContextPropagator {
	return &ContextPropagator{
		propagator: otel.GetTextMapPropagator(),
	}
}

// InjectHTTP injects trace context into HTTP headers
func (cp *ContextPropagator) InjectHTTP(ctx context.Context, req *http.Request) {
	cp.propagator.Inject(ctx, propagation.HeaderCarrier(req.Header))
}

// ExtractHTTP extracts trace context from HTTP headers
func (cp *ContextPropagator) ExtractHTTP(ctx context.Context, req *http.Request) context.Context {
	return cp.propagator.Extract(ctx, propagation.HeaderCarrier(req.Header))
}

// InjectMap injects trace context into a map
func (cp *ContextPropagator) InjectMap(ctx context.Context, carrier map[string]string) {
	cp.propagator.Inject(ctx, propagation.MapCarrier(carrier))
}

// ExtractMap extracts trace context from a map
func (cp *ContextPropagator) ExtractMap(ctx context.Context, carrier map[string]string) context.Context {
	return cp.propagator.Extract(ctx, propagation.MapCarrier(carrier))
}

// GetTraceID returns the trace ID from the context
func GetTraceID(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		return span.SpanContext().TraceID().String()
	}
	return ""
}

// GetSpanID returns the span ID from the context
func GetSpanID(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		return span.SpanContext().SpanID().String()
	}
	return ""
}

// IsTraced returns true if the context has an active trace
func IsTraced(ctx context.Context) bool {
	span := trace.SpanFromContext(ctx)
	return span.SpanContext().IsValid()
}

// GetSpanContext returns the span context from the given context
func GetSpanContext(ctx context.Context) trace.SpanContext {
	span := trace.SpanFromContext(ctx)
	return span.SpanContext()
}

// HTTPMiddleware creates an HTTP middleware that extracts and injects trace context
func HTTPMiddleware(next http.Handler) http.Handler {
	propagator := NewContextPropagator()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract trace context from incoming request
		ctx := propagator.ExtractHTTP(r.Context(), r)

		// Start a new span for this request
		tracer := otel.Tracer("http")
		ctx, span := tracer.Start(ctx, r.Method+" "+r.URL.Path)
		defer span.End()

		// Inject trace context into response headers (for debugging)
		propagator.InjectHTTP(ctx, r)

		// Update request with new context
		r = r.WithContext(ctx)

		// Call next handler
		next.ServeHTTP(w, r)
	})
}

// PropagateContext propagates trace context from one context to another
func PropagateContext(from, to context.Context) context.Context {
	span := trace.SpanFromContext(from)
	if !span.SpanContext().IsValid() {
		return to
	}
	return trace.ContextWithSpanContext(to, span.SpanContext())
}
