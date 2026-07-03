package tracing

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func setupTestProvider() *sdktrace.TracerProvider {
	exporter := tracetest.NewInMemoryExporter()
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exporter),
	)
	otel.SetTracerProvider(provider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))
	return provider
}

func TestNewContextPropagator(t *testing.T) {
	propagator := NewContextPropagator()
	if propagator == nil {
		t.Fatal("Propagator should not be nil")
	}

	if propagator.propagator == nil {
		t.Error("Propagator should have a text map propagator")
	}
}

func TestInjectExtractHTTP(t *testing.T) {
	provider := setupTestProvider()
	defer provider.Shutdown(context.Background())

	propagator := NewContextPropagator()

	// Create a span
	tracer := otel.Tracer("test")
	ctx, span := tracer.Start(context.Background(), "test-operation")
	defer span.End()

	// Create HTTP request
	req, err := http.NewRequest("GET", "http://example.com", http.NoBody)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	// Inject context into headers
	propagator.InjectHTTP(ctx, req)

	// Verify headers were set
	if req.Header.Get("traceparent") == "" {
		t.Error("traceparent header should be set")
	}

	// Extract context from headers
	extractedCtx := propagator.ExtractHTTP(context.Background(), req)

	// Verify extracted context has span information
	extractedTraceID := GetTraceID(extractedCtx)
	originalTraceID := GetTraceID(ctx)

	if extractedTraceID != originalTraceID {
		t.Errorf("Trace IDs should match: got %s, want %s", extractedTraceID, originalTraceID)
	}
}

func TestInjectExtractMap(t *testing.T) {
	provider := setupTestProvider()
	defer provider.Shutdown(context.Background())

	propagator := NewContextPropagator()

	// Create a span
	tracer := otel.Tracer("test")
	ctx, span := tracer.Start(context.Background(), "test-operation")
	defer span.End()

	// Create a map carrier
	carrier := make(map[string]string)

	// Inject context into map
	propagator.InjectMap(ctx, carrier)

	// Verify map was populated
	if len(carrier) == 0 {
		t.Error("Carrier map should have entries")
	}

	if carrier["traceparent"] == "" {
		t.Error("traceparent should be in carrier map")
	}

	// Extract context from map
	extractedCtx := propagator.ExtractMap(context.Background(), carrier)

	// Verify extracted context has span information
	extractedTraceID := GetTraceID(extractedCtx)
	originalTraceID := GetTraceID(ctx)

	if extractedTraceID != originalTraceID {
		t.Errorf("Trace IDs should match: got %s, want %s", extractedTraceID, originalTraceID)
	}
}

func TestGetTraceID(t *testing.T) {
	provider := setupTestProvider()
	defer provider.Shutdown(context.Background())

	t.Run("with valid span", func(t *testing.T) {
		tracer := otel.Tracer("test")
		ctx, span := tracer.Start(context.Background(), "test-operation")
		defer span.End()

		traceID := GetTraceID(ctx)
		if traceID == "" {
			t.Error("Trace ID should not be empty")
		}

		if len(traceID) != 32 { // Trace ID is 16 bytes = 32 hex chars
			t.Errorf("Trace ID should be 32 characters, got %d", len(traceID))
		}
	})

	t.Run("without span", func(t *testing.T) {
		traceID := GetTraceID(context.Background())
		if traceID != "" {
			t.Error("Trace ID should be empty for context without span")
		}
	})
}

func TestGetSpanID(t *testing.T) {
	provider := setupTestProvider()
	defer provider.Shutdown(context.Background())

	t.Run("with valid span", func(t *testing.T) {
		tracer := otel.Tracer("test")
		ctx, span := tracer.Start(context.Background(), "test-operation")
		defer span.End()

		spanID := GetSpanID(ctx)
		if spanID == "" {
			t.Error("Span ID should not be empty")
		}

		if len(spanID) != 16 { // Span ID is 8 bytes = 16 hex chars
			t.Errorf("Span ID should be 16 characters, got %d", len(spanID))
		}
	})

	t.Run("without span", func(t *testing.T) {
		spanID := GetSpanID(context.Background())
		if spanID != "" {
			t.Error("Span ID should be empty for context without span")
		}
	})
}

func TestIsTraced(t *testing.T) {
	provider := setupTestProvider()
	defer provider.Shutdown(context.Background())

	t.Run("with valid span", func(t *testing.T) {
		tracer := otel.Tracer("test")
		ctx, span := tracer.Start(context.Background(), "test-operation")
		defer span.End()

		if !IsTraced(ctx) {
			t.Error("Context should be traced")
		}
	})

	t.Run("without span", func(t *testing.T) {
		if IsTraced(context.Background()) {
			t.Error("Context should not be traced")
		}
	})
}

func TestGetSpanContext(t *testing.T) {
	provider := setupTestProvider()
	defer provider.Shutdown(context.Background())

	tracer := otel.Tracer("test")
	ctx, span := tracer.Start(context.Background(), "test-operation")
	defer span.End()

	spanContext := GetSpanContext(ctx)
	if !spanContext.IsValid() {
		t.Error("Span context should be valid")
	}

	if !spanContext.TraceID().IsValid() {
		t.Error("Trace ID should be valid")
	}

	if !spanContext.SpanID().IsValid() {
		t.Error("Span ID should be valid")
	}
}

func TestHTTPMiddleware(t *testing.T) {
	provider := setupTestProvider()
	defer provider.Shutdown(context.Background())

	// Create a test handler
	called := false
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true

		// Verify context has trace information
		if !IsTraced(r.Context()) {
			t.Error("Request context should be traced")
		}

		w.WriteHeader(http.StatusOK)
	})

	// Wrap with middleware
	handler := HTTPMiddleware(testHandler)

	// Create test request
	req := httptest.NewRequest("GET", "/test", http.NoBody)
	w := httptest.NewRecorder()

	// Execute request
	handler.ServeHTTP(w, req)

	if !called {
		t.Error("Handler should have been called")
	}

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestHTTPMiddlewareWithPropagation(t *testing.T) {
	provider := setupTestProvider()
	defer provider.Shutdown(context.Background())

	// Create a parent span and inject into request
	tracer := otel.Tracer("test")
	ctx, parentSpan := tracer.Start(context.Background(), "parent-operation")
	defer parentSpan.End()

	propagator := NewContextPropagator()

	req := httptest.NewRequest("GET", "/test", http.NoBody)
	propagator.InjectHTTP(ctx, req)

	originalTraceID := GetTraceID(ctx)

	// Create handler that verifies trace ID matches
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestTraceID := GetTraceID(r.Context())
		if requestTraceID != originalTraceID {
			t.Errorf("Trace IDs should match: got %s, want %s", requestTraceID, originalTraceID)
		}
		w.WriteHeader(http.StatusOK)
	})

	// Wrap with middleware
	handler := HTTPMiddleware(testHandler)
	w := httptest.NewRecorder()

	// Execute request
	handler.ServeHTTP(w, req)
}

func TestPropagateContext(t *testing.T) {
	provider := setupTestProvider()
	defer provider.Shutdown(context.Background())

	t.Run("with valid trace", func(t *testing.T) {
		tracer := otel.Tracer("test")
		ctx, span := tracer.Start(context.Background(), "test-operation")
		defer span.End()

		originalTraceID := GetTraceID(ctx)

		// Propagate to new context
		newCtx := context.Background()
		propagatedCtx := PropagateContext(ctx, newCtx)

		propagatedTraceID := GetTraceID(propagatedCtx)
		if propagatedTraceID != originalTraceID {
			t.Errorf("Trace IDs should match: got %s, want %s", propagatedTraceID, originalTraceID)
		}
	})

	t.Run("without trace", func(t *testing.T) {
		fromCtx := context.Background()
		toCtx := context.Background()

		propagatedCtx := PropagateContext(fromCtx, toCtx)
		if IsTraced(propagatedCtx) {
			t.Error("Propagated context should not be traced")
		}
	})
}

func TestContextPropagationEndToEnd(t *testing.T) {
	provider := setupTestProvider()
	defer provider.Shutdown(context.Background())

	// Simulate service A creating a span
	tracerA := otel.Tracer("service-a")
	ctxA, spanA := tracerA.Start(context.Background(), "service-a-operation")
	defer spanA.End()

	traceID := GetTraceID(ctxA)

	// Service A makes HTTP call to Service B
	propagator := NewContextPropagator()
	reqToB, _ := http.NewRequest("GET", "http://service-b/endpoint", http.NoBody)
	propagator.InjectHTTP(ctxA, reqToB)

	// Service B receives request and extracts context
	ctxB := propagator.ExtractHTTP(context.Background(), reqToB)

	// Service B creates a child span
	tracerB := otel.Tracer("service-b")
	ctxB, spanB := tracerB.Start(ctxB, "service-b-operation")
	defer spanB.End()

	// Verify trace IDs match (same trace)
	traceBTraceID := GetTraceID(ctxB)
	if traceBTraceID != traceID {
		t.Errorf("Trace IDs should match across services: got %s, want %s", traceBTraceID, traceID)
	}

	// Verify span IDs are different (different spans)
	spanAID := GetSpanID(ctxA)
	spanBID := GetSpanID(ctxB)
	if spanAID == spanBID {
		t.Error("Span IDs should be different for different operations")
	}
}
