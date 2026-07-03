// Package datadog provides integration with Datadog APM and monitoring
package datadog

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

// ErrNotConfigured is returned by metric/event submission methods when the
// client is disabled or has no API key, i.e. no data can be sent to Datadog.
var ErrNotConfigured = errors.New("datadog client is not configured")

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
	config     *Config
	httpClient *http.Client
	// apiBaseURL is the Datadog API base (e.g. https://api.datadoghq.com).
	// It is derived from the configured Site and may be overridden in tests.
	apiBaseURL string
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
		config:     config,
		httpClient: &http.Client{Timeout: 10 * time.Second},
		apiBaseURL: "https://api." + config.Site,
	}, nil
}

// isConfigured reports whether the client can actually submit data to Datadog.
func (c *Client) isConfigured() bool {
	return c.config.Enabled && c.config.APIKey != "" && c.httpClient != nil
}

// buildTags renders a tag map into Datadog "key:value" tag strings, prefixed
// with the configured service and environment so they are always attached.
func (c *Client) buildTags(tags map[string]string) []string {
	result := make([]string, 0, len(tags)+2)
	if c.config.ServiceName != "" {
		result = append(result, "service:"+c.config.ServiceName)
	}
	if c.config.Environment != "" {
		result = append(result, "env:"+c.config.Environment)
	}
	for k, v := range tags {
		result = append(result, k+":"+v)
	}
	return result
}

// postJSON marshals payload and POSTs it to the given Datadog API path,
// authenticating with the configured API key.
func (c *Client) postJSON(ctx context.Context, path string, payload interface{}) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal datadog payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.apiBaseURL+path, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create datadog request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("DD-API-KEY", c.config.APIKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send datadog request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("datadog api %s returned status %d: %s", path, resp.StatusCode, bytes.TrimSpace(respBody))
	}
	return nil
}

// metricSeries is a single series entry for the Datadog v1 series API.
type metricSeries struct {
	Metric string       `json:"metric"`
	Points [][2]float64 `json:"points"`
	Type   string       `json:"type"`
	Tags   []string     `json:"tags,omitempty"`
}

// seriesPayload is the body of a POST to /api/v1/series.
type seriesPayload struct {
	Series []metricSeries `json:"series"`
}

// eventPayload is the body of a POST to /api/v1/events.
type eventPayload struct {
	Title     string   `json:"title"`
	Text      string   `json:"text"`
	Tags      []string `json:"tags,omitempty"`
	AlertType string   `json:"alert_type,omitempty"`
}

// submitSeries sends a single point of the given metric type to Datadog.
func (c *Client) submitSeries(metricType, name string, value float64, tags map[string]string) error {
	if !c.isConfigured() {
		return ErrNotConfigured
	}

	payload := seriesPayload{
		Series: []metricSeries{{
			Metric: name,
			Points: [][2]float64{{float64(time.Now().Unix()), value}},
			Type:   metricType,
			Tags:   c.buildTags(tags),
		}},
	}
	return c.postJSON(context.Background(), "/api/v1/series", payload)
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

// SendMetric sends a custom gauge metric to Datadog via the v1 series API.
// It returns ErrNotConfigured when the client is disabled or has no API key.
func (c *Client) SendMetric(name string, value float64, tags map[string]string) error {
	return c.submitSeries("gauge", name, value, tags)
}

// Gauge submits a gauge metric to Datadog (a snapshot value at a point in time).
func (c *Client) Gauge(name string, value float64, tags map[string]string) error {
	return c.submitSeries("gauge", name, value, tags)
}

// Count submits a count metric to Datadog (a value accumulated over the interval).
func (c *Client) Count(name string, value float64, tags map[string]string) error {
	return c.submitSeries("count", name, value, tags)
}

// Incr submits a count metric of 1 to Datadog, incrementing the named counter.
func (c *Client) Incr(name string, tags map[string]string) error {
	return c.submitSeries("count", name, 1, tags)
}

// SendEvent sends an event to Datadog via the v1 events API. alertType should be
// one of "info", "warning", "error" or "success". It returns ErrNotConfigured
// when the client is disabled or has no API key.
func (c *Client) SendEvent(title, text string, tags map[string]string, alertType string) error {
	if !c.isConfigured() {
		return ErrNotConfigured
	}

	payload := eventPayload{
		Title:     title,
		Text:      text,
		Tags:      c.buildTags(tags),
		AlertType: alertType,
	}
	return c.postJSON(context.Background(), "/api/v1/events", payload)
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
