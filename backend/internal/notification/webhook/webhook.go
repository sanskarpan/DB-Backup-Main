// Package webhook provides generic webhook notification integration
package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/sanskarpan/db-backup/internal/notification"
	pkgErrors "github.com/sanskarpan/db-backup/pkg/errors"
)

// WebhookNotifier implements generic webhook notifications.
type WebhookNotifier struct {
	url           string
	method        string
	headers       map[string]string
	timeout       time.Duration
	retries       int
	retryDelay    time.Duration
	client        *http.Client
	authenticated bool
	authType      string
	authToken     string
}

// Config holds webhook notifier configuration.
type Config struct {
	URL        string
	Method     string // GET, POST, PUT
	Headers    map[string]string
	Timeout    time.Duration
	Retries    int
	RetryDelay time.Duration
	AuthType   string // bearer, basic, custom
	AuthToken  string
	AuthHeader string
}

// NewWebhookNotifier creates a new webhook notifier.
func NewWebhookNotifier(cfg *Config) (*WebhookNotifier, error) {
	if cfg.URL == "" {
		return nil, pkgErrors.New(pkgErrors.ErrorTypeConfiguration, "webhook URL is required")
	}

	method := cfg.Method
	if method == "" {
		method = "POST"
	}

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	retries := cfg.Retries
	if retries == 0 {
		retries = 3
	}

	retryDelay := cfg.RetryDelay
	if retryDelay == 0 {
		retryDelay = 5 * time.Second
	}

	headers := cfg.Headers
	if headers == nil {
		headers = make(map[string]string)
	}

	// Set default content type if not specified
	if _, exists := headers["Content-Type"]; !exists {
		headers["Content-Type"] = "application/json"
	}

	// Setup authentication
	authenticated := false
	authType := cfg.AuthType
	authToken := cfg.AuthToken

	if authToken != "" {
		authenticated = true
		if authType == "" {
			authType = "bearer"
		}

		// Add auth header if not custom
		if authType == "bearer" {
			headers["Authorization"] = fmt.Sprintf("Bearer %s", authToken)
		} else if authType == "basic" {
			headers["Authorization"] = fmt.Sprintf("Basic %s", authToken)
		} else if cfg.AuthHeader != "" {
			headers[cfg.AuthHeader] = authToken
		}
	}

	return &WebhookNotifier{
		url:           cfg.URL,
		method:        method,
		headers:       headers,
		timeout:       timeout,
		retries:       retries,
		retryDelay:    retryDelay,
		authenticated: authenticated,
		authType:      authType,
		authToken:     authToken,
		client: &http.Client{
			Timeout: timeout,
		},
	}, nil
}

// Send sends a notification via webhook.
func (w *WebhookNotifier) Send(ctx context.Context, notif *notification.Notification) error {
	payload := w.buildPayload(notif)

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return pkgErrors.Wrap(err, pkgErrors.ErrorTypeOperation, "failed to marshal webhook payload")
	}

	var lastErr error
	for attempt := 0; attempt <= w.retries; attempt++ {
		if attempt > 0 {
			time.Sleep(w.retryDelay)
		}

		err := w.sendRequest(ctx, jsonData)
		if err == nil {
			return nil
		}

		lastErr = err
	}

	return pkgErrors.Wrap(lastErr, pkgErrors.ErrorTypeNetwork,
		fmt.Sprintf("webhook failed after %d retries", w.retries))
}

// sendRequest sends the HTTP request.
func (w *WebhookNotifier) sendRequest(ctx context.Context, data []byte) error {
	req, err := http.NewRequestWithContext(ctx, w.method, w.url, bytes.NewBuffer(data))
	if err != nil {
		return pkgErrors.Wrap(err, pkgErrors.ErrorTypeOperation, "failed to create request")
	}

	// Add headers
	for key, value := range w.headers {
		req.Header.Set(key, value)
	}

	resp, err := w.client.Do(req)
	if err != nil {
		return pkgErrors.Wrap(err, pkgErrors.ErrorTypeNetwork, "failed to send webhook request")
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return pkgErrors.New(pkgErrors.ErrorTypeOperation,
			fmt.Sprintf("webhook returned status %d", resp.StatusCode))
	}

	return nil
}

// GetType returns the provider type.
func (w *WebhookNotifier) GetType() notification.ProviderType {
	return notification.ProviderTypeWebhook
}

// ValidateConfig validates the configuration.
func (w *WebhookNotifier) ValidateConfig() error {
	if w.url == "" {
		return pkgErrors.New(pkgErrors.ErrorTypeConfiguration, "webhook URL is required")
	}
	return nil
}

// WebhookPayload represents the webhook JSON payload.
type WebhookPayload struct {
	Event     string                 `json:"event"`
	Level     string                 `json:"level"`
	Title     string                 `json:"title"`
	Message   string                 `json:"message"`
	Timestamp string                 `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Tags      []string               `json:"tags,omitempty"`
}

// buildPayload builds the webhook payload.
func (w *WebhookNotifier) buildPayload(notif *notification.Notification) *WebhookPayload {
	return &WebhookPayload{
		Event:     "backup.notification",
		Level:     string(notif.Level),
		Title:     notif.Title,
		Message:   notif.Message,
		Timestamp: notif.Timestamp.Format(time.RFC3339),
		Metadata:  notif.Metadata,
		Tags:      notif.Tags,
	}
}
