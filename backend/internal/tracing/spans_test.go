package tracing

import (
	"context"
	"errors"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestStartSpan(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exporter),
	)
	otel.SetTracerProvider(provider)
	defer provider.Shutdown(context.Background())

	ctx := context.Background()

	// Start a span with attributes
	ctx, span := StartSpan(
		ctx, "test-tracer", "test-operation",
		WithAttributes(
			attribute.String("key1", "value1"),
			attribute.Int("key2", 42),
		),
	)
	span.End()

	// Verify span was created
	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("Expected 1 span, got %d", len(spans))
	}

	// Verify span name
	if spans[0].Name != "test-operation" {
		t.Errorf("Expected span name 'test-operation', got '%s'", spans[0].Name)
	}

	// Verify attributes
	attrs := spans[0].Attributes
	found := false
	for _, attr := range attrs {
		if string(attr.Key) == "key1" && attr.Value.AsString() == "value1" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected attribute 'key1=value1' not found")
	}
}

func TestEndSpan(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exporter),
	)
	otel.SetTracerProvider(provider)
	defer provider.Shutdown(context.Background())

	t.Run("without error", func(t *testing.T) {
		exporter.Reset()

		tracer := otel.Tracer("test")
		_, span := tracer.Start(context.Background(), "test-operation")

		EndSpan(span, nil)

		spans := exporter.GetSpans()
		if len(spans) != 1 {
			t.Fatalf("Expected 1 span, got %d", len(spans))
		}

		if spans[0].Status.Code != codes.Unset {
			t.Errorf("Expected unset status, got %v", spans[0].Status.Code)
		}
	})

	t.Run("with error", func(t *testing.T) {
		exporter.Reset()

		tracer := otel.Tracer("test")
		_, span := tracer.Start(context.Background(), "test-operation")

		testErr := errors.New("test error")
		EndSpan(span, testErr)

		spans := exporter.GetSpans()
		if len(spans) != 1 {
			t.Fatalf("Expected 1 span, got %d", len(spans))
		}

		if spans[0].Status.Code != codes.Error {
			t.Errorf("Expected error status, got %v", spans[0].Status.Code)
		}

		if spans[0].Status.Description != "test error" {
			t.Errorf("Expected error description 'test error', got '%s'", spans[0].Status.Description)
		}
	})
}

func TestDatabaseSpan(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exporter),
	)
	otel.SetTracerProvider(provider)
	defer provider.Shutdown(context.Background())

	_, span := DatabaseSpan(context.Background(), "SELECT", "mydb", "users")
	span.End()

	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("Expected 1 span, got %d", len(spans))
	}

	if spans[0].Name != "db.SELECT" {
		t.Errorf("Expected span name 'db.SELECT', got '%s'", spans[0].Name)
	}

	// Verify database attributes
	attrs := spans[0].Attributes
	expectedAttrs := map[string]string{
		"db.system":    "postgres",
		"db.operation": "SELECT",
		"db.name":      "mydb",
		"db.table":     "users",
	}

	for key, expectedValue := range expectedAttrs {
		found := false
		for _, attr := range attrs {
			if string(attr.Key) == key && attr.Value.AsString() == expectedValue {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected attribute '%s=%s' not found", key, expectedValue)
		}
	}
}

func TestStorageSpan(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exporter),
	)
	otel.SetTracerProvider(provider)
	defer provider.Shutdown(context.Background())

	_, span := StorageSpan(context.Background(), "upload", "s3", "/backups/file.sql")
	span.End()

	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("Expected 1 span, got %d", len(spans))
	}

	if spans[0].Name != "storage.upload" {
		t.Errorf("Expected span name 'storage.upload', got '%s'", spans[0].Name)
	}

	// Verify storage attributes
	attrs := spans[0].Attributes
	expectedAttrs := map[string]string{
		"storage.provider":  "s3",
		"storage.operation": "upload",
		"storage.path":      "/backups/file.sql",
	}

	for key, expectedValue := range expectedAttrs {
		found := false
		for _, attr := range attrs {
			if string(attr.Key) == key && attr.Value.AsString() == expectedValue {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected attribute '%s=%s' not found", key, expectedValue)
		}
	}
}

func TestBackupSpan(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exporter),
	)
	otel.SetTracerProvider(provider)
	defer provider.Shutdown(context.Background())

	_, span := BackupSpan(context.Background(), "full", "postgres-db")
	span.End()

	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("Expected 1 span, got %d", len(spans))
	}

	if spans[0].Name != "backup.full" {
		t.Errorf("Expected span name 'backup.full', got '%s'", spans[0].Name)
	}
}

func TestRestoreSpan(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exporter),
	)
	otel.SetTracerProvider(provider)
	defer provider.Shutdown(context.Background())

	_, span := RestoreSpan(context.Background(), "backup-123", "postgres-db")
	span.End()

	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("Expected 1 span, got %d", len(spans))
	}

	if spans[0].Name != "restore" {
		t.Errorf("Expected span name 'restore', got '%s'", spans[0].Name)
	}
}

func TestCompressionSpan(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exporter),
	)
	otel.SetTracerProvider(provider)
	defer provider.Shutdown(context.Background())

	_, span := CompressionSpan(context.Background(), "compress", "gzip", 1024000)
	span.End()

	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("Expected 1 span, got %d", len(spans))
	}

	if spans[0].Name != "compression.compress" {
		t.Errorf("Expected span name 'compression.compress', got '%s'", spans[0].Name)
	}
}

func TestEncryptionSpan(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exporter),
	)
	otel.SetTracerProvider(provider)
	defer provider.Shutdown(context.Background())

	_, span := EncryptionSpan(context.Background(), "encrypt", "AES-256")
	span.End()

	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("Expected 1 span, got %d", len(spans))
	}

	if spans[0].Name != "encryption.encrypt" {
		t.Errorf("Expected span name 'encryption.encrypt', got '%s'", spans[0].Name)
	}
}

func TestHTTPSpan(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exporter),
	)
	otel.SetTracerProvider(provider)
	defer provider.Shutdown(context.Background())

	_, span := HTTPSpan(context.Background(), "GET", "http://api.example.com/users")
	span.End()

	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("Expected 1 span, got %d", len(spans))
	}

	if spans[0].Name != "GET http://api.example.com/users" {
		t.Errorf("Unexpected span name: %s", spans[0].Name)
	}
}

func TestAddEvent(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exporter),
	)
	otel.SetTracerProvider(provider)
	defer provider.Shutdown(context.Background())

	tracer := otel.Tracer("test")
	ctx, span := tracer.Start(context.Background(), "test-operation")

	AddEvent(
		ctx, "checkpoint",
		attribute.String("step", "validation"),
		attribute.Int("count", 5),
	)

	span.End()

	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("Expected 1 span, got %d", len(spans))
	}

	if len(spans[0].Events) != 1 {
		t.Fatalf("Expected 1 event, got %d", len(spans[0].Events))
	}

	if spans[0].Events[0].Name != "checkpoint" {
		t.Errorf("Expected event name 'checkpoint', got '%s'", spans[0].Events[0].Name)
	}
}

func TestAddSpanAttributes(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exporter),
	)
	otel.SetTracerProvider(provider)
	defer provider.Shutdown(context.Background())

	tracer := otel.Tracer("test")
	ctx, span := tracer.Start(context.Background(), "test-operation")

	AddSpanAttributes(
		ctx,
		attribute.String("user_id", "123"),
		attribute.Bool("premium", true),
	)

	span.End()

	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("Expected 1 span, got %d", len(spans))
	}

	// Verify attributes were added
	attrs := spans[0].Attributes
	found := false
	for _, attr := range attrs {
		if string(attr.Key) == "user_id" && attr.Value.AsString() == "123" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected attribute 'user_id=123' not found")
	}
}

func TestRecordError(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exporter),
	)
	otel.SetTracerProvider(provider)
	defer provider.Shutdown(context.Background())

	tracer := otel.Tracer("test")
	ctx, span := tracer.Start(context.Background(), "test-operation")

	testErr := errors.New("test error")
	RecordError(ctx, testErr)

	span.End()

	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("Expected 1 span, got %d", len(spans))
	}

	if spans[0].Status.Code != codes.Error {
		t.Errorf("Expected error status, got %v", spans[0].Status.Code)
	}
}

func TestSetSpanStatus(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exporter),
	)
	otel.SetTracerProvider(provider)
	defer provider.Shutdown(context.Background())

	t.Run("ok status", func(t *testing.T) {
		exporter.Reset()
		tracer := otel.Tracer("test")
		ctx, span := tracer.Start(context.Background(), "test-operation")

		SetSpanStatus(ctx, codes.Ok, "operation successful")

		span.End()

		spans := exporter.GetSpans()
		if len(spans) != 1 {
			t.Fatalf("Expected 1 span, got %d", len(spans))
		}

		if spans[0].Status.Code != codes.Ok {
			t.Errorf("Expected OK status, got %v", spans[0].Status.Code)
		}
	})

	t.Run("error status", func(t *testing.T) {
		exporter.Reset()
		tracer := otel.Tracer("test")
		ctx, span := tracer.Start(context.Background(), "test-operation")

		SetSpanStatus(ctx, codes.Error, "operation failed")

		span.End()

		spans := exporter.GetSpans()
		if len(spans) != 1 {
			t.Fatalf("Expected 1 span, got %d", len(spans))
		}

		if spans[0].Status.Code != codes.Error {
			t.Errorf("Expected Error status, got %v", spans[0].Status.Code)
		}

		if spans[0].Status.Description != "operation failed" {
			t.Errorf("Expected description 'operation failed', got '%s'", spans[0].Status.Description)
		}
	})
}

func TestTraceFunc(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exporter),
	)
	otel.SetTracerProvider(provider)
	defer provider.Shutdown(context.Background())

	t.Run("success", func(t *testing.T) {
		exporter.Reset()

		err := TraceFunc(context.Background(), "test-tracer", "test-operation", func(ctx context.Context) error {
			// Verify context has trace
			if !IsTraced(ctx) {
				t.Error("Context should be traced")
			}
			return nil
		})
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		spans := exporter.GetSpans()
		if len(spans) != 1 {
			t.Fatalf("Expected 1 span, got %d", len(spans))
		}

		if spans[0].Status.Code == codes.Error {
			t.Error("Span should not have error status")
		}
	})

	t.Run("with error", func(t *testing.T) {
		exporter.Reset()

		testErr := errors.New("test error")
		err := TraceFunc(context.Background(), "test-tracer", "test-operation", func(ctx context.Context) error {
			return testErr
		})

		if err != testErr {
			t.Errorf("Expected error %v, got %v", testErr, err)
		}

		spans := exporter.GetSpans()
		if len(spans) != 1 {
			t.Fatalf("Expected 1 span, got %d", len(spans))
		}

		if spans[0].Status.Code != codes.Error {
			t.Error("Span should have error status")
		}
	})
}

func TestTraceFuncWithResult(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exporter),
	)
	otel.SetTracerProvider(provider)
	defer provider.Shutdown(context.Background())

	t.Run("success", func(t *testing.T) {
		exporter.Reset()

		result, err := TraceFuncWithResult(context.Background(), "test-tracer", "test-operation",
			func(ctx context.Context) (string, error) {
				return "success", nil
			})
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		if result != "success" {
			t.Errorf("Expected result 'success', got '%s'", result)
		}

		spans := exporter.GetSpans()
		if len(spans) != 1 {
			t.Fatalf("Expected 1 span, got %d", len(spans))
		}
	})

	t.Run("with error", func(t *testing.T) {
		exporter.Reset()

		testErr := errors.New("test error")
		result, err := TraceFuncWithResult(context.Background(), "test-tracer", "test-operation",
			func(ctx context.Context) (int, error) {
				return 0, testErr
			})

		if err != testErr {
			t.Errorf("Expected error %v, got %v", testErr, err)
		}

		if result != 0 {
			t.Errorf("Expected result 0, got %d", result)
		}

		spans := exporter.GetSpans()
		if len(spans) != 1 {
			t.Fatalf("Expected 1 span, got %d", len(spans))
		}

		if spans[0].Status.Code != codes.Error {
			t.Error("Span should have error status")
		}
	})
}

func TestMeasureLatency(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exporter),
	)
	otel.SetTracerProvider(provider)
	defer provider.Shutdown(context.Background())

	tracer := otel.Tracer("test")
	ctx, span := tracer.Start(context.Background(), "test-operation")

	MeasureLatency(ctx, "database-query", 150)

	span.End()

	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("Expected 1 span, got %d", len(spans))
	}

	// Verify latency attribute was added
	attrs := spans[0].Attributes
	found := false
	for _, attr := range attrs {
		if string(attr.Key) == "latency_ms" && attr.Value.AsInt64() == 150 {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected latency_ms attribute not found")
	}
}

func TestRecordMetric(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exporter),
	)
	otel.SetTracerProvider(provider)
	defer provider.Shutdown(context.Background())

	tracer := otel.Tracer("test")
	ctx, span := tracer.Start(context.Background(), "test-operation")

	// Test different types
	RecordMetric(ctx, "count", 42)
	RecordMetric(ctx, "size", int64(1024))
	RecordMetric(ctx, "ratio", 0.75)
	RecordMetric(ctx, "name", "test")
	RecordMetric(ctx, "enabled", true)

	span.End()

	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("Expected 1 span, got %d", len(spans))
	}

	attrs := spans[0].Attributes
	if len(attrs) < 5 {
		t.Errorf("Expected at least 5 attributes, got %d", len(attrs))
	}
}
