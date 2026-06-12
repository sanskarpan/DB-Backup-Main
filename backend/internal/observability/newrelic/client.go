// Package newrelic provides integration with New Relic APM and monitoring
package newrelic

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/newrelic/go-agent/v3/newrelic"
)

// Config holds New Relic configuration
type Config struct {
	LicenseKey  string
	AppName     string
	Environment string
	Enabled     bool
}

// Client represents a New Relic APM client
type Client struct {
	config *Config
	app    *newrelic.Application
}

// NewClient creates a new New Relic client
func NewClient(config *Config) (*Client, error) {
	if config == nil {
		return nil, errors.New("config is required")
	}

	if !config.Enabled {
		return &Client{config: config}, nil
	}

	if config.LicenseKey == "" {
		return nil, errors.New("license key is required")
	}

	if config.AppName == "" {
		config.AppName = "db-backup-service"
	}

	if config.Environment == "" {
		config.Environment = "production"
	}

	return &Client{
		config: config,
	}, nil
}

// Start initializes the New Relic application
func (c *Client) Start() error {
	if !c.config.Enabled {
		return nil
	}

	app, err := newrelic.NewApplication(
		newrelic.ConfigAppName(c.config.AppName),
		newrelic.ConfigLicense(c.config.LicenseKey),
		newrelic.ConfigDistributedTracerEnabled(true),
		newrelic.ConfigAppLogForwardingEnabled(true),
		func(cfg *newrelic.Config) {
			cfg.Labels = map[string]string{
				"environment": c.config.Environment,
			}
		},
	)

	if err != nil {
		return fmt.Errorf("failed to create New Relic application: %w", err)
	}

	c.app = app
	return nil
}

// Stop gracefully shuts down the New Relic application
func (c *Client) Stop() {
	if !c.config.Enabled || c.app == nil {
		return
	}
	c.app.Shutdown(10 * time.Second)
}

// StartTransaction starts a new transaction
func (c *Client) StartTransaction(name string) *newrelic.Transaction {
	if !c.config.Enabled || c.app == nil {
		return nil
	}
	return c.app.StartTransaction(name)
}

// StartTransactionFromContext starts a new transaction from context
func (c *Client) StartTransactionFromContext(ctx context.Context, name string) (*newrelic.Transaction, context.Context) {
	if !c.config.Enabled || c.app == nil {
		return nil, ctx
	}

	txn := c.app.StartTransaction(name)
	return txn, newrelic.NewContext(ctx, txn)
}

// TraceBackup traces a backup operation
func (c *Client) TraceBackup(ctx context.Context, databaseName, backupType string, fn func(context.Context) error) error {
	if !c.config.Enabled || c.app == nil {
		return fn(ctx)
	}

	txn := c.app.StartTransaction(fmt.Sprintf("Backup/%s", databaseName))
	defer txn.End()

	txn.AddAttribute("database.name", databaseName)
	txn.AddAttribute("backup.type", backupType)

	ctx = newrelic.NewContext(ctx, txn)
	startTime := time.Now()

	err := fn(ctx)
	duration := time.Since(startTime)

	txn.AddAttribute("backup.duration_ms", duration.Milliseconds())

	if err != nil {
		txn.NoticeError(err)
		txn.AddAttribute("backup.status", "failed")
		return err
	}

	txn.AddAttribute("backup.status", "success")
	return nil
}

// TraceRestore traces a restore operation
func (c *Client) TraceRestore(ctx context.Context, databaseName string, fn func(context.Context) error) error {
	if !c.config.Enabled || c.app == nil {
		return fn(ctx)
	}

	txn := c.app.StartTransaction(fmt.Sprintf("Restore/%s", databaseName))
	defer txn.End()

	txn.AddAttribute("database.name", databaseName)

	ctx = newrelic.NewContext(ctx, txn)
	startTime := time.Now()

	err := fn(ctx)
	duration := time.Since(startTime)

	txn.AddAttribute("restore.duration_ms", duration.Milliseconds())

	if err != nil {
		txn.NoticeError(err)
		txn.AddAttribute("restore.status", "failed")
		return err
	}

	txn.AddAttribute("restore.status", "success")
	return nil
}

// TraceStorageOperation traces a storage operation
func (c *Client) TraceStorageOperation(ctx context.Context, operation, provider string, fn func(context.Context) error) error {
	if !c.config.Enabled || c.app == nil {
		return fn(ctx)
	}

	txn := c.app.StartTransaction(fmt.Sprintf("Storage/%s/%s", provider, operation))
	defer txn.End()

	txn.AddAttribute("storage.provider", provider)
	txn.AddAttribute("storage.operation", operation)

	ctx = newrelic.NewContext(ctx, txn)
	startTime := time.Now()

	err := fn(ctx)
	duration := time.Since(startTime)

	txn.AddAttribute("storage.duration_ms", duration.Milliseconds())

	if err != nil {
		txn.NoticeError(err)
		return err
	}

	return nil
}

// TraceEncryption traces an encryption operation
func (c *Client) TraceEncryption(ctx context.Context, algorithm string, dataSize int64, fn func(context.Context) error) error {
	if !c.config.Enabled || c.app == nil {
		return fn(ctx)
	}

	txn := c.app.StartTransaction("Encryption/Encrypt")
	defer txn.End()

	txn.AddAttribute("encryption.algorithm", algorithm)
	txn.AddAttribute("encryption.data_size_bytes", dataSize)

	ctx = newrelic.NewContext(ctx, txn)
	startTime := time.Now()

	err := fn(ctx)
	duration := time.Since(startTime)

	txn.AddAttribute("encryption.duration_ms", duration.Milliseconds())
	if dataSize > 0 && duration > 0 {
		throughputMBps := float64(dataSize) / (1024 * 1024) / duration.Seconds()
		txn.AddAttribute("encryption.throughput_mbps", throughputMBps)
	}

	if err != nil {
		txn.NoticeError(err)
		return err
	}

	return nil
}

// RecordMetric records a custom metric
func (c *Client) RecordMetric(name string, value float64) error {
	if !c.config.Enabled || c.app == nil {
		return nil
	}

	c.app.RecordCustomMetric(name, value)
	return nil
}

// RecordEvent records a custom event
func (c *Client) RecordEvent(eventType string, attributes map[string]interface{}) error {
	if !c.config.Enabled || c.app == nil {
		return nil
	}

	c.app.RecordCustomEvent(eventType, attributes)
	return nil
}

// NoticeError reports an error to New Relic
func (c *Client) NoticeError(ctx context.Context, err error, attributes map[string]interface{}) {
	if !c.config.Enabled || c.app == nil {
		return
	}

	txn := newrelic.FromContext(ctx)
	if txn != nil {
		for k, v := range attributes {
			txn.AddAttribute(k, v)
		}
		txn.NoticeError(err)
	} else {
		// If no transaction in context, create a background transaction
		txn := c.app.StartTransaction("BackgroundError")
		defer txn.End()
		for k, v := range attributes {
			txn.AddAttribute(k, v)
		}
		txn.NoticeError(err)
	}
}

// StartSegment starts a new segment within a transaction
func (c *Client) StartSegment(ctx context.Context, name string) *newrelic.Segment {
	if !c.config.Enabled || c.app == nil {
		return nil
	}

	txn := newrelic.FromContext(ctx)
	if txn == nil {
		return nil
	}

	return &newrelic.Segment{
		Name:      name,
		StartTime: txn.StartSegmentNow(),
	}
}

// EndSegment ends a segment
func (c *Client) EndSegment(segment *newrelic.Segment) {
	if segment != nil {
		segment.End()
	}
}

// StartDatastoreSegment starts a new datastore segment
func (c *Client) StartDatastoreSegment(ctx context.Context, product, collection, operation string) *newrelic.DatastoreSegment {
	if !c.config.Enabled || c.app == nil {
		return nil
	}

	txn := newrelic.FromContext(ctx)
	if txn == nil {
		return nil
	}

	return &newrelic.DatastoreSegment{
		Product:    newrelic.DatastoreProduct(product),
		Collection: collection,
		Operation:  operation,
		StartTime:  txn.StartSegmentNow(),
	}
}

// EndDatastoreSegment ends a datastore segment
func (c *Client) EndDatastoreSegment(segment *newrelic.DatastoreSegment) {
	if segment != nil {
		segment.End()
	}
}

// StartExternalSegment starts a new external segment
func (c *Client) StartExternalSegment(ctx context.Context, url string) *newrelic.ExternalSegment {
	if !c.config.Enabled || c.app == nil {
		return nil
	}

	txn := newrelic.FromContext(ctx)
	if txn == nil {
		return nil
	}

	return &newrelic.ExternalSegment{
		URL:       url,
		StartTime: txn.StartSegmentNow(),
	}
}

// EndExternalSegment ends an external segment
func (c *Client) EndExternalSegment(segment *newrelic.ExternalSegment) {
	if segment != nil {
		segment.End()
	}
}

// Application returns the underlying New Relic application
func (c *Client) Application() *newrelic.Application {
	return c.app
}

// WaitForConnection waits for the New Relic agent to connect
func (c *Client) WaitForConnection(timeout time.Duration) error {
	if !c.config.Enabled || c.app == nil {
		return nil
	}

	return c.app.WaitForConnection(timeout)
}
