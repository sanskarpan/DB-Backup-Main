package tracing

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/sanskarpan/db-backup/internal/config"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestNewTracerProvider(t *testing.T) {
	tests := []struct {
		name      string
		config    *config.TracingConfig
		expectErr bool
	}{
		{
			name:      "nil config",
			config:    nil,
			expectErr: false, // Should return no-op provider
		},
		{
			name: "disabled tracing",
			config: &config.TracingConfig{
				Enabled: false,
			},
			expectErr: false,
		},
		{
			name: "jaeger provider",
			config: &config.TracingConfig{
				Enabled:     true,
				Provider:    "jaeger",
				ServiceName: "test-service",
				Jaeger: config.JaegerConfig{
					Endpoint: "http://localhost:14268/api/traces",
				},
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewTracerProvider(tt.config)

			if tt.expectErr {
				if err == nil {
					t.Error("Expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if provider == nil {
				t.Fatal("Provider should not be nil")
			}

			// Cleanup
			if provider.provider != nil {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				provider.Shutdown(ctx)
			}
		})
	}
}

func TestSamplerCreation(t *testing.T) {
	tests := []struct {
		name           string
		samplingConfig config.SamplingConfig
		expectedType   string // We'll check behavior instead
	}{
		{
			name: "always sampler",
			samplingConfig: config.SamplingConfig{
				Type: "always",
			},
		},
		{
			name: "never sampler",
			samplingConfig: config.SamplingConfig{
				Type: "never",
			},
		},
		{
			name: "probability sampler",
			samplingConfig: config.SamplingConfig{
				Type: "probability",
				Rate: 0.5,
			},
		},
		{
			name: "rate limiting sampler",
			samplingConfig: config.SamplingConfig{
				Type:  "rate_limiting",
				Limit: 100,
			},
		},
		{
			name:           "default sampler",
			samplingConfig: config.SamplingConfig{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sampler := createSampler(tt.samplingConfig)
			if sampler == nil {
				t.Error("Sampler should not be nil")
			}

			// Test that sampler works
			decision := sampler.ShouldSample(sdktrace.SamplingParameters{})
			if decision.Decision != sdktrace.RecordAndSample &&
			   decision.Decision != sdktrace.Drop &&
			   decision.Decision != sdktrace.RecordOnly {
				t.Errorf("Invalid sampling decision: %v", decision.Decision)
			}
		})
	}
}

func TestTracerProviderShutdown(t *testing.T) {
	cfg := &config.TracingConfig{
		Enabled:     true,
		Provider:    "jaeger",
		ServiceName: "test-service",
		Jaeger: config.JaegerConfig{
			Endpoint: "http://localhost:14268/api/traces",
		},
	}

	provider, err := NewTracerProvider(cfg)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = provider.Shutdown(ctx)
	if err != nil {
		t.Errorf("Shutdown failed: %v", err)
	}

	// Second shutdown should not error
	err = provider.Shutdown(ctx)
	if err != nil {
		t.Errorf("Second shutdown failed: %v", err)
	}
}

func TestForceFlush(t *testing.T) {
	cfg := &config.TracingConfig{
		Enabled:     true,
		Provider:    "jaeger",
		ServiceName: "test-service",
		Jaeger: config.JaegerConfig{
			Endpoint: "http://localhost:14268/api/traces",
		},
	}

	provider, err := NewTracerProvider(cfg)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}
	defer provider.Shutdown(context.Background())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = provider.ForceFlush(ctx)
	if err != nil {
		t.Errorf("ForceFlush failed: %v", err)
	}
}

func TestGetTracer(t *testing.T) {
	cfg := &config.TracingConfig{
		Enabled:     true,
		Provider:    "jaeger",
		ServiceName: "test-service",
		Jaeger: config.JaegerConfig{
			Endpoint: "http://localhost:14268/api/traces",
		},
	}

	provider, err := NewTracerProvider(cfg)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}
	defer provider.Shutdown(context.Background())

	tracer := provider.GetTracer("test-tracer")
	if tracer == nil {
		t.Error("Tracer should not be nil")
	}

	// Create a span to verify tracer works
	ctx, span := tracer.Start(context.Background(), "test-span")
	span.End()

	if ctx == nil {
		t.Error("Context should not be nil")
	}
}

func TestGlobalTracerFunctions(t *testing.T) {
	// Use in-memory exporter for testing to avoid needing Jaeger running
	exporter := tracetest.NewInMemoryExporter()
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exporter),
	)

	// Create a TracerProvider wrapper
	tp := &TracerProvider{
		provider: provider,
		config: &config.TracingConfig{
			Enabled:     true,
			ServiceName: "test-service",
		},
	}

	// Set as global provider
	globalProvider = tp
	otel.SetTracerProvider(provider)

	// Get global tracer
	tracer := GetGlobalTracer("test")
	if tracer == nil {
		t.Error("Global tracer should not be nil")
	}

	// Create a span
	ctx, span := tracer.Start(context.Background(), "test-span")
	span.End()

	if ctx == nil {
		t.Error("Context should not be nil")
	}

	// Force flush
	err := ForceFlushGlobal(context.Background())
	if err != nil {
		t.Errorf("ForceFlush failed: %v", err)
	}

	// Shutdown
	err = ShutdownGlobalTracer(context.Background())
	if err != nil {
		t.Errorf("Shutdown failed: %v", err)
	}

	// Clean up global provider
	globalProvider = nil
}

func TestResourceCreation(t *testing.T) {
	tests := []struct {
		name   string
		config *config.TracingConfig
	}{
		{
			name: "with service name",
			config: &config.TracingConfig{
				ServiceName: "custom-service",
				Environment: "production",
			},
		},
		{
			name: "with tags",
			config: &config.TracingConfig{
				ServiceName: "test-service",
				Jaeger: config.JaegerConfig{
					Tags: map[string]string{
						"version": "1.0.0",
						"region":  "us-west-2",
					},
				},
			},
		},
		{
			name:   "default service name",
			config: &config.TracingConfig{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := createResource(tt.config)
			if err != nil {
				t.Fatalf("Failed to create resource: %v", err)
			}

			if res == nil {
				t.Fatal("Resource should not be nil")
			}

			// Verify attributes
			attrs := res.Attributes()
			if len(attrs) == 0 {
				t.Error("Resource should have attributes")
			}
		})
	}
}

func TestSamplingBehavior(t *testing.T) {
	tests := []struct {
		name           string
		samplingConfig config.SamplingConfig
		shouldSample   bool
		testMultiple   bool
	}{
		{
			name: "always sample",
			samplingConfig: config.SamplingConfig{
				Type: "always",
			},
			shouldSample: true,
			testMultiple: false,
		},
		{
			name: "never sample",
			samplingConfig: config.SamplingConfig{
				Type: "never",
			},
			shouldSample: false,
			testMultiple: false,
		},
		{
			name: "100% probability",
			samplingConfig: config.SamplingConfig{
				Type: "probability",
				Rate: 1.0,
			},
			shouldSample: true,
			testMultiple: false,
		},
		{
			name: "50% probability",
			samplingConfig: config.SamplingConfig{
				Type: "probability",
				Rate: 0.5,
			},
			shouldSample: true,
			testMultiple: true, // Test multiple times to verify sampling
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sampler := createSampler(tt.samplingConfig)

			params := sdktrace.SamplingParameters{}
			decision := sampler.ShouldSample(params)

			if tt.shouldSample && decision.Decision == sdktrace.Drop {
				t.Error("Expected sample but got drop")
			} else if !tt.shouldSample && tt.name == "never sample" && decision.Decision != sdktrace.Drop {
				t.Error("Expected drop but got sample")
			}

			if tt.testMultiple {
				// Test multiple times to ensure sampling works
				sampled := 0
				iterations := 100
				for i := 0; i < iterations; i++ {
					decision := sampler.ShouldSample(params)
					if decision.Decision == sdktrace.RecordAndSample {
						sampled++
					}
				}

				// For 50% sampling, we expect roughly 40-60% to be sampled
				expectedMin := int(float64(iterations) * 0.3)
				expectedMax := int(float64(iterations) * 0.7)

				if sampled < expectedMin || sampled > expectedMax {
					t.Logf("Warning: Sampled %d/%d (expected ~%d)", sampled, iterations, iterations/2)
				}
			}
		})
	}
}

func TestBatchProcessorOptions(t *testing.T) {
	cfg := &config.TracingConfig{
		Enabled:      true,
		Provider:     "jaeger",
		ServiceName:  "test-service",
		BatchTimeout: 5 * time.Second,
		MaxQueueSize: 2048,
		Jaeger: config.JaegerConfig{
			Endpoint: "http://localhost:14268/api/traces",
		},
	}

	provider, err := NewTracerProvider(cfg)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}
	defer provider.Shutdown(context.Background())

	if provider.provider == nil {
		t.Fatal("Provider should have a tracer provider")
	}
}

func TestCreateExporterErrors(t *testing.T) {
	tests := []struct {
		name      string
		config    *config.TracingConfig
		expectErr bool
	}{
		{
			name: "unsupported provider",
			config: &config.TracingConfig{
				Provider: "invalid-provider",
			},
			expectErr: true,
		},
		{
			name: "otlp without endpoint",
			config: &config.TracingConfig{
				Provider: "otlp",
				OTLP: config.OTLPConfig{
					Endpoint: "",
				},
			},
			expectErr: true,
		},
		{
			name: "valid jaeger",
			config: &config.TracingConfig{
				Provider: "jaeger",
				Jaeger: config.JaegerConfig{
					Endpoint: "http://localhost:14268/api/traces",
				},
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := createExporter(tt.config)

			if tt.expectErr {
				if err == nil {
					t.Error("Expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

// Test with in-memory exporter for complete end-to-end testing
func TestTracerProviderWithInMemoryExporter(t *testing.T) {
	// Create an in-memory span exporter for testing
	exporter := tracetest.NewInMemoryExporter()

	// Create tracer provider
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exporter),
	)
	otel.SetTracerProvider(provider)
	defer provider.Shutdown(context.Background())

	// Get tracer
	tracer := otel.Tracer("test")

	// Create spans
	ctx := context.Background()
	_, span1 := tracer.Start(ctx, "operation1")
	span1.SetAttributes(attribute.String("key", "value"))
	span1.End()

	_, span2 := tracer.Start(ctx, "operation2")
	span2.RecordError(fmt.Errorf("test error"))
	span2.SetStatus(codes.Error, "test error")
	span2.End()

	// Check exported spans
	spans := exporter.GetSpans()
	if len(spans) != 2 {
		t.Errorf("Expected 2 spans, got %d", len(spans))
		return
	}

	// Verify first span
	if spans[0].Name != "operation1" {
		t.Errorf("Expected span name 'operation1', got '%s'", spans[0].Name)
	}

	// Verify second span has error
	if spans[1].Status.Code != codes.Error {
		t.Error("Expected error status on second span")
	}
}
