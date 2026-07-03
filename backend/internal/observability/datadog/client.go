// Package datadog provides integration with Datadog APM and monitoring
package datadog

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

// Config holds Datadog configuration.
type Config struct {
	APIKey      string
	AppKey      string
	Site        string // datadoghq.com, datadoghq.eu, etc.
	ServiceName string
	Environment string
	Version     string
	Enabled     bool
}

// Client represents a Datadog APM client.
type Client struct {
	config *Config
}

// NewClient creates a new Datadog client.
func NewClient(config *Config) (*Client, error) {
	if config == nil {
		return nil, errors.New("config is required")
	}

	if !config.Enabled {
		return &Client{config: config}, nil
	}

	if config.APIKey == "" {
		return nil, errors.New("API key is required")
	}

	if config.ServiceName == "" {
		config.ServiceName = "db-backup-service"
	}

	if config.Environment == "" {
		config.Environment = "production"
	}

	if config.Site == "" {
		config.Site = "datadoghq.com"
	}

	return &Client{
		config: config,
	}, nil
}

// Start initializes the Datadog tracer.
func (c *Client) Start() error {
	if !c.config.Enabled {
		return nil
	}

	tracer.Start(
		tracer.WithService(c.config.ServiceName),
		tracer.WithEnv(c.config.Environment),
		tracer.WithServiceVersion(c.config.Version),
		tracer.WithAgentAddr("localhost:8126"),
		tracer.WithRuntimeMetrics(),
		tracer.WithDebugMode(false),
		tracer.WithSampler(tracer.NewAllSampler()),
	)

	return nil
}

// Stop gracefully shuts down the Datadog tracer.
func (c *Client) Stop() {
	if !c.config.Enabled {
		return
	}
	tracer.Stop()
}

// StartSpan creates a new span for tracing.
func (c *Client) StartSpan(operationName string, opts ...tracer.StartSpanOption) ddtrace.Span {
	if !c.config.Enabled {
		return &noopSpan{}
	}
	return tracer.StartSpan(operationName, opts...)
}

// StartSpanFromContext creates a new span from context.
func (c *Client) StartSpanFromContext(ctx context.Context, operationName string, opts ...tracer.StartSpanOption) (ddtrace.Span, context.Context) {
	if !c.config.Enabled {
		return &noopSpan{}, ctx
	}
	return tracer.StartSpanFromContext(ctx, operationName, opts...)
}

// TraceBackup traces a backup operation.
func (c *Client) TraceBackup(ctx context.Context, databaseName, backupType string, fn func(context.Context) error) error {
	span, ctx := c.StartSpanFromContext(
		ctx, "backup.create",
		tracer.ResourceName(fmt.Sprintf("backup:%s", databaseName)),
		tracer.Tag("database.name", databaseName),
		tracer.Tag("backup.type", backupType),
		tracer.Tag("service.name", c.config.ServiceName),
	)
	defer span.Finish()

	startTime := time.Now()
	err := fn(ctx)
	duration := time.Since(startTime)

	span.SetTag("backup.duration_ms", duration.Milliseconds())

	if err != nil {
		span.SetTag("error", true)
		span.SetTag("error.message", err.Error())
		span.SetTag("backup.status", "failed")
		return err
	}

	span.SetTag("backup.status", "success")
	return nil
}

// TraceRestore traces a restore operation.
func (c *Client) TraceRestore(ctx context.Context, databaseName string, fn func(context.Context) error) error {
	span, ctx := c.StartSpanFromContext(
		ctx, "restore.execute",
		tracer.ResourceName(fmt.Sprintf("restore:%s", databaseName)),
		tracer.Tag("database.name", databaseName),
		tracer.Tag("service.name", c.config.ServiceName),
	)
	defer span.Finish()

	startTime := time.Now()
	err := fn(ctx)
	duration := time.Since(startTime)

	span.SetTag("restore.duration_ms", duration.Milliseconds())

	if err != nil {
		span.SetTag("error", true)
		span.SetTag("error.message", err.Error())
		span.SetTag("restore.status", "failed")
		return err
	}

	span.SetTag("restore.status", "success")
	return nil
}

// TraceStorageOperation traces a storage operation.
func (c *Client) TraceStorageOperation(ctx context.Context, operation, provider string, fn func(context.Context) error) error {
	span, ctx := c.StartSpanFromContext(
		ctx, "storage."+operation,
		tracer.ResourceName(fmt.Sprintf("storage:%s:%s", provider, operation)),
		tracer.Tag("storage.provider", provider),
		tracer.Tag("storage.operation", operation),
		tracer.Tag("service.name", c.config.ServiceName),
	)
	defer span.Finish()

	startTime := time.Now()
	err := fn(ctx)
	duration := time.Since(startTime)

	span.SetTag("storage.duration_ms", duration.Milliseconds())

	if err != nil {
		span.SetTag("error", true)
		span.SetTag("error.message", err.Error())
		return err
	}

	return nil
}

// TraceEncryption traces an encryption operation.
func (c *Client) TraceEncryption(ctx context.Context, algorithm string, dataSize int64, fn func(context.Context) error) error {
	span, ctx := c.StartSpanFromContext(
		ctx, "encryption.encrypt",
		tracer.ResourceName("encryption:"+algorithm),
		tracer.Tag("encryption.algorithm", algorithm),
		tracer.Tag("encryption.data_size_bytes", dataSize),
		tracer.Tag("service.name", c.config.ServiceName),
	)
	defer span.Finish()

	startTime := time.Now()
	err := fn(ctx)
	duration := time.Since(startTime)

	span.SetTag("encryption.duration_ms", duration.Milliseconds())
	if dataSize > 0 && duration > 0 {
		throughputMBps := float64(dataSize) / (1024 * 1024) / duration.Seconds()
		span.SetTag("encryption.throughput_mbps", throughputMBps)
	}

	if err != nil {
		span.SetTag("error", true)
		span.SetTag("error.message", err.Error())
		return err
	}

	return nil
}

// SendMetric sends a custom metric to Datadog.
func (c *Client) SendMetric(name string, value float64, tags map[string]string) error {
	if !c.config.Enabled {
		return nil
	}

	// In production, use statsd client to send metrics
	// For now, this is a placeholder
	return nil
}

// SendEvent sends an event to Datadog.
func (c *Client) SendEvent(title, text string, tags map[string]string, alertType string) error {
	if !c.config.Enabled {
		return nil
	}

	// In production, use Datadog API to send events
	// For now, this is a placeholder
	return nil
}

// LogError logs an error to Datadog.
func (c *Client) LogError(ctx context.Context, err error, tags map[string]string) {
	if !c.config.Enabled {
		return
	}

	span, ok := tracer.SpanFromContext(ctx)
	if !ok {
		return
	}

	span.SetTag("error", true)
	span.SetTag("error.message", err.Error())
	for k, v := range tags {
		span.SetTag(k, v)
	}
}

// noopSpan is a no-op implementation of ddtrace.Span.
type noopSpan struct{}

func (n *noopSpan) SetTag(key string, value interface{})  {}
func (n *noopSpan) SetOperationName(operationName string) {}
func (n *noopSpan) BaggageItem(key string) string         { return "" }
func (n *noopSpan) SetBaggageItem(key, value string)      {}
func (n *noopSpan) Finish(opts ...ddtrace.FinishOption)   {}
func (n *noopSpan) Context() ddtrace.SpanContext          { return &noopSpanContext{} }

// noopSpanContext is a no-op implementation of ddtrace.SpanContext.
type noopSpanContext struct{}

func (n *noopSpanContext) TraceID() uint64 { return 0 }
func (n *noopSpanContext) SpanID() uint64  { return 0 }
func (n *noopSpanContext) ForeachBaggageItem(handler func(k, v string) bool) {
}
