// Package tracing provides distributed tracing support using OpenTelemetry
package tracing

import (
	"context"
	"fmt"
	"net"
	"strconv"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	//nolint:staticcheck // SA1019: Jaeger exporter is deprecated but still supported for existing deployments; migration to OTLP tracked separately.
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/sanskarpan/db-backup/internal/config"
)

// TracerProvider wraps the OpenTelemetry tracer provider.
type TracerProvider struct {
	provider *sdktrace.TracerProvider
	config   *config.TracingConfig
}

// NewTracerProvider creates and configures a new tracer provider.
func NewTracerProvider(cfg *config.TracingConfig) (*TracerProvider, error) {
	if cfg == nil || !cfg.Enabled {
		// Return a no-op provider
		return &TracerProvider{
			provider: sdktrace.NewTracerProvider(),
			config:   cfg,
		}, nil
	}

	// Create resource with service information
	res := createResource(cfg)

	// Create exporter based on provider type
	exporter, err := createExporter(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create exporter: %w", err)
	}

	// Create sampler
	sampler := createSampler(cfg.Sampling)

	// Configure batch span processor options
	batchOptions := []sdktrace.BatchSpanProcessorOption{}
	if cfg.BatchTimeout > 0 {
		batchOptions = append(batchOptions, sdktrace.WithBatchTimeout(cfg.BatchTimeout))
	}
	if cfg.MaxQueueSize > 0 {
		batchOptions = append(batchOptions, sdktrace.WithMaxQueueSize(cfg.MaxQueueSize))
	}

	// Create tracer provider
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sampler),
		sdktrace.WithBatcher(exporter, batchOptions...),
	)

	// Set global tracer provider
	otel.SetTracerProvider(provider)

	// Set global propagator for context propagation
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return &TracerProvider{
		provider: provider,
		config:   cfg,
	}, nil
}

// Shutdown gracefully shuts down the tracer provider.
func (tp *TracerProvider) Shutdown(ctx context.Context) error {
	if tp.provider == nil {
		return nil
	}
	return tp.provider.Shutdown(ctx)
}

// ForceFlush forces a flush of pending spans.
func (tp *TracerProvider) ForceFlush(ctx context.Context) error {
	if tp.provider == nil {
		return nil
	}
	return tp.provider.ForceFlush(ctx)
}

// GetTracer returns a tracer with the given name.
func (tp *TracerProvider) GetTracer(name string, options ...trace.TracerOption) trace.Tracer {
	if tp.provider == nil {
		return otel.Tracer(name, options...)
	}
	return tp.provider.Tracer(name, options...)
}

// Helper functions

func createResource(cfg *config.TracingConfig) *resource.Resource {
	serviceName := cfg.ServiceName
	if serviceName == "" {
		serviceName = "db-backup"
	}

	attrs := []attribute.KeyValue{
		semconv.ServiceName(serviceName),
		semconv.ServiceVersion("1.0.0"),
	}

	if cfg.Environment != "" {
		attrs = append(attrs, attribute.String("environment", cfg.Environment))
	}

	// Add Jaeger tags if configured
	if cfg.Jaeger.Tags != nil {
		for key, value := range cfg.Jaeger.Tags {
			attrs = append(attrs, attribute.String(key, value))
		}
	}

	return resource.NewWithAttributes(
		semconv.SchemaURL,
		attrs...,
	)
}

func createExporter(cfg *config.TracingConfig) (sdktrace.SpanExporter, error) {
	provider := cfg.Provider
	if provider == "" {
		provider = "jaeger"
	}

	switch provider {
	case "jaeger":
		return createJaegerExporter(cfg.Jaeger)
	case "otlp":
		return createOTLPExporter(cfg.OTLP)
	default:
		return nil, fmt.Errorf("unsupported tracing provider: %s", provider)
	}
}

func createJaegerExporter(cfg config.JaegerConfig) (sdktrace.SpanExporter, error) {
	endpoint := cfg.Endpoint
	if endpoint == "" && cfg.AgentHost != "" {
		// Use agent endpoint
		agentPort := cfg.AgentPort
		if agentPort == 0 {
			agentPort = 14268 // Default Jaeger collector port
		}
		endpoint = fmt.Sprintf("http://%s/api/traces", net.JoinHostPort(cfg.AgentHost, strconv.Itoa(agentPort)))
	} else if endpoint == "" {
		// Default to localhost collector
		endpoint = "http://localhost:14268/api/traces"
	}

	return jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(endpoint)))
}

func createOTLPExporter(cfg config.OTLPConfig) (sdktrace.SpanExporter, error) {
	if cfg.Endpoint == "" {
		return nil, fmt.Errorf("OTLP endpoint is required")
	}

	opts := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(cfg.Endpoint),
	}

	if cfg.Insecure {
		opts = append(opts, otlptracegrpc.WithInsecure())
	}

	if len(cfg.Headers) > 0 {
		opts = append(opts, otlptracegrpc.WithHeaders(cfg.Headers))
	}

	opts = append(opts, otlptracegrpc.WithDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())))

	client := otlptracegrpc.NewClient(opts...)
	return otlptrace.New(context.Background(), client)
}

func createSampler(cfg config.SamplingConfig) sdktrace.Sampler {
	samplerType := cfg.Type
	if samplerType == "" {
		samplerType = "always"
	}

	switch samplerType {
	case "always":
		return sdktrace.AlwaysSample()
	case "never":
		return sdktrace.NeverSample()
	case "probability":
		rate := cfg.Rate
		if rate <= 0 {
			rate = 0.1 // Default 10% sampling
		} else if rate > 1.0 {
			rate = 1.0
		}
		return sdktrace.TraceIDRatioBased(rate)
	case "rate_limiting":
		// Rate limiting sampler is not directly available in OpenTelemetry SDK
		// Fall back to probability-based sampling
		rate := 0.1
		if cfg.Limit > 0 {
			// Estimate rate based on expected traffic
			// This is a simplified approach
			rate = float64(cfg.Limit) / 1000.0
			if rate > 1.0 {
				rate = 1.0
			}
		}
		return sdktrace.TraceIDRatioBased(rate)
	default:
		return sdktrace.AlwaysSample()
	}
}

// Global tracer provider instance.
var globalProvider *TracerProvider

// InitGlobalTracer initializes the global tracer provider.
func InitGlobalTracer(cfg *config.TracingConfig) error {
	provider, err := NewTracerProvider(cfg)
	if err != nil {
		return err
	}
	globalProvider = provider
	return nil
}

// ShutdownGlobalTracer shuts down the global tracer provider.
func ShutdownGlobalTracer(ctx context.Context) error {
	if globalProvider == nil {
		return nil
	}
	return globalProvider.Shutdown(ctx)
}

// GetGlobalTracer returns the global tracer.
func GetGlobalTracer(name string) trace.Tracer {
	if globalProvider == nil {
		return otel.Tracer(name)
	}
	return globalProvider.GetTracer(name)
}

// ForceFlushGlobal forces a flush of the global tracer provider.
func ForceFlushGlobal(ctx context.Context) error {
	if globalProvider == nil {
		return nil
	}
	return globalProvider.ForceFlush(ctx)
}
