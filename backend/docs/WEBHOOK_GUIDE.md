# Webhook Integration Guide

## Table of Contents

- [Overview](#overview)
- [Quick Start](#quick-start)
- [Configuration](#configuration)
- [Webhook Payload](#webhook-payload)
- [Event Types](#event-types)
- [Authentication](#authentication)
- [Retry Logic](#retry-logic)
- [Security Best Practices](#security-best-practices)
- [Testing Webhooks](#testing-webhooks)
- [Examples](#examples)
- [Troubleshooting](#troubleshooting)

## Overview

The Database Backup Utility supports webhook notifications for real-time event delivery. Webhooks allow your application to receive HTTP callbacks when specific events occur, such as backup completions, failures, or schedule executions.

### Key Features

- **Real-time Notifications**: Receive instant notifications when events occur
- **Multiple Event Types**: Subscribe to specific event types or all events
- **Automatic Retries**: Built-in retry mechanism with exponential backoff
- **Signature Verification**: HMAC-SHA256 signatures for security
- **Custom Headers**: Support for custom authentication headers
- **Flexible Filtering**: Filter events by severity, tags, or other criteria
- **Rate Limiting**: Automatic rate limiting to prevent overload

## Quick Start

### 1. Configure Webhook in YAML

```yaml
notifications:
  enabled: true
  providers:
    - type: webhook
      enabled: true
      url: "https://your-app.com/webhooks/backup"
      method: POST
      headers:
        Authorization: "Bearer your-secret-token"
        X-Custom-Header: "custom-value"
      timeout: 10s
      retry:
        enabled: true
        max_attempts: 3
        initial_delay: 1s
        max_delay: 30s
      events:
        - backup.created
        - backup.completed
        - backup.failed
        - restore.completed
        - restore.failed
```

### 2. Set Up Webhook Endpoint

```go
package main

import (
    "encoding/json"
    "io"
    "net/http"
    "log"
)

type WebhookPayload struct {
    Event     string                 `json:"event"`
    Timestamp string                 `json:"timestamp"`
    Data      map[string]interface{} `json:"data"`
    Metadata  map[string]interface{} `json:"metadata"`
}

func handleWebhook(w http.ResponseWriter, r *http.Request) {
    // Verify signature (see Authentication section)
    if !verifySignature(r) {
        http.Error(w, "Invalid signature", http.StatusUnauthorized)
        return
    }

    // Parse payload
    body, _ := io.ReadAll(r.Body)
    var payload WebhookPayload
    if err := json.Unmarshal(body, &payload); err != nil {
        http.Error(w, "Invalid payload", http.StatusBadRequest)
        return
    }

    // Process event
    switch payload.Event {
    case "backup.completed":
        log.Printf("Backup completed: %v", payload.Data)
    case "backup.failed":
        log.Printf("Backup failed: %v", payload.Data)
    }

    // Return 200 to acknowledge receipt
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func main() {
    http.HandleFunc("/webhooks/backup", handleWebhook)
    log.Fatal(http.ListenAndServe(":8080", nil))
}
```

## Configuration

### Complete Configuration Options

```yaml
notifications:
  enabled: true

  # Webhook provider configuration
  providers:
    - type: webhook
      enabled: true

      # Endpoint configuration
      url: "https://your-app.com/webhooks/backup"
      method: POST  # GET, POST, PUT, PATCH

      # Custom headers
      headers:
        Authorization: "Bearer ${WEBHOOK_TOKEN}"
        Content-Type: "application/json"
        X-Custom-Header: "value"

      # Timeout settings
      timeout: 10s
      connect_timeout: 5s

      # TLS/SSL configuration
      tls:
        verify: true
        ca_cert: /path/to/ca.crt
        client_cert: /path/to/client.crt
        client_key: /path/to/client.key

      # Retry configuration
      retry:
        enabled: true
        max_attempts: 3
        initial_delay: 1s
        max_delay: 30s
        multiplier: 2.0
        retry_on_status: [429, 500, 502, 503, 504]

      # Rate limiting
      rate_limit:
        enabled: true
        requests_per_second: 10
        burst: 20

      # Event filtering
      events:
        - backup.created
        - backup.started
        - backup.completed
        - backup.failed
        - restore.started
        - restore.completed
        - restore.failed
        - schedule.created
        - schedule.executed
        - schedule.failed

      # Level filtering
      min_level: info  # info, warning, error

      # Tag filtering
      tags:
        - production
        - critical

      # Signature configuration
      signature:
        enabled: true
        secret: "${WEBHOOK_SECRET}"
        header: X-Backup-Signature
        algorithm: sha256  # sha256, sha512

      # Batching (optional)
      batching:
        enabled: false
        max_size: 10
        max_wait: 5s
```

### Environment Variables

Webhook configuration supports environment variable substitution:

```bash
export WEBHOOK_URL="https://your-app.com/webhooks"
export WEBHOOK_TOKEN="your-secret-token"
export WEBHOOK_SECRET="your-signing-secret"
```

Then in config:

```yaml
providers:
  - type: webhook
    url: "${WEBHOOK_URL}"
    headers:
      Authorization: "Bearer ${WEBHOOK_TOKEN}"
    signature:
      secret: "${WEBHOOK_SECRET}"
```

## Webhook Payload

### Standard Payload Structure

All webhook payloads follow this structure:

```json
{
  "event": "backup.completed",
  "event_id": "evt_1a2b3c4d5e6f",
  "timestamp": "2025-12-30T10:30:00Z",
  "api_version": "v1",
  "data": {
    // Event-specific data
  },
  "metadata": {
    "environment": "production",
    "source": "db-backup-utility",
    "version": "1.0.0"
  }
}
```

### Payload Fields

| Field | Type | Description |
|-------|------|-------------|
| `event` | string | Event type identifier (e.g., `backup.completed`) |
| `event_id` | string | Unique identifier for this event |
| `timestamp` | string | ISO 8601 timestamp when event occurred |
| `api_version` | string | API version for payload structure |
| `data` | object | Event-specific data (varies by event type) |
| `metadata` | object | Additional context and metadata |

## Event Types

### Backup Events

#### `backup.created`

Triggered when a backup is created and queued.

```json
{
  "event": "backup.created",
  "data": {
    "backup_id": "bkp_550e8400",
    "database": "production_db",
    "database_type": "postgres",
    "scheduled": false,
    "tags": {
      "environment": "production"
    }
  }
}
```

#### `backup.started`

Triggered when backup execution begins.

```json
{
  "event": "backup.started",
  "data": {
    "backup_id": "bkp_550e8400",
    "database": "production_db",
    "database_type": "postgres",
    "start_time": "2025-12-30T10:30:00Z",
    "estimated_duration": 300
  }
}
```

#### `backup.completed`

Triggered when backup completes successfully.

```json
{
  "event": "backup.completed",
  "data": {
    "backup_id": "bkp_550e8400",
    "database": "production_db",
    "database_type": "postgres",
    "size": 1073741824,
    "compressed_size": 268435456,
    "compression_ratio": 4.0,
    "duration": 287,
    "checksum": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
    "backup_path": "/backups/production_db_2025-12-30.sql.gz",
    "tables": [
      {
        "name": "users",
        "row_count": 100000,
        "data_size": 10485760
      }
    ]
  }
}
```

#### `backup.failed`

Triggered when backup fails.

```json
{
  "event": "backup.failed",
  "data": {
    "backup_id": "bkp_550e8400",
    "database": "production_db",
    "database_type": "postgres",
    "error": "Connection timeout",
    "error_code": "CONN_TIMEOUT",
    "duration": 30,
    "retry_count": 2
  }
}
```

### Restore Events

#### `restore.started`

```json
{
  "event": "restore.started",
  "data": {
    "restore_id": "rst_660f9500",
    "backup_id": "bkp_550e8400",
    "target_database": "production_db_restored",
    "start_time": "2025-12-30T11:00:00Z"
  }
}
```

#### `restore.completed`

```json
{
  "event": "restore.completed",
  "data": {
    "restore_id": "rst_660f9500",
    "backup_id": "bkp_550e8400",
    "target_database": "production_db_restored",
    "duration": 150,
    "rows_restored": 500000,
    "tables_restored": ["users", "orders", "products"]
  }
}
```

#### `restore.failed`

```json
{
  "event": "restore.failed",
  "data": {
    "restore_id": "rst_660f9500",
    "backup_id": "bkp_550e8400",
    "error": "Invalid backup file",
    "error_code": "INVALID_BACKUP"
  }
}
```

### Schedule Events

#### `schedule.executed`

```json
{
  "event": "schedule.executed",
  "data": {
    "schedule_id": "sch_770g0600",
    "schedule_name": "Daily Production Backup",
    "backup_id": "bkp_550e8400",
    "execution_time": "2025-12-30T02:00:00Z",
    "next_execution": "2025-12-31T02:00:00Z"
  }
}
```

#### `schedule.failed`

```json
{
  "event": "schedule.failed",
  "data": {
    "schedule_id": "sch_770g0600",
    "schedule_name": "Daily Production Backup",
    "error": "Backup failed",
    "error_code": "BACKUP_FAILED",
    "next_retry": "2025-12-30T03:00:00Z"
  }
}
```

## Authentication

### HMAC Signature Verification

The webhook sender includes an HMAC-SHA256 signature in the `X-Backup-Signature` header (configurable).

#### Go Example

```go
import (
    "crypto/hmac"
    "crypto/sha256"
    "encoding/hex"
    "io"
    "net/http"
)

func verifySignature(r *http.Request, secret string) bool {
    // Read body
    body, err := io.ReadAll(r.Body)
    if err != nil {
        return false
    }

    // Get signature from header
    signature := r.Header.Get("X-Backup-Signature")
    if signature == "" {
        return false
    }

    // Calculate expected signature
    mac := hmac.New(sha256.New, []byte(secret))
    mac.Write(body)
    expected := hex.EncodeToString(mac.Sum(nil))

    // Compare signatures
    return hmac.Equal([]byte(signature), []byte(expected))
}
```

#### Python Example

```python
import hmac
import hashlib

def verify_signature(request, secret):
    signature = request.headers.get('X-Backup-Signature')
    if not signature:
        return False

    body = request.get_data()
    expected = hmac.new(
        secret.encode('utf-8'),
        body,
        hashlib.sha256
    ).hexdigest()

    return hmac.compare_digest(signature, expected)
```

#### Node.js Example

```javascript
const crypto = require('crypto');

function verifySignature(req, secret) {
    const signature = req.headers['x-backup-signature'];
    if (!signature) {
        return false;
    }

    const body = JSON.stringify(req.body);
    const expected = crypto
        .createHmac('sha256', secret)
        .update(body)
        .digest('hex');

    return crypto.timingSafeEqual(
        Buffer.from(signature),
        Buffer.from(expected)
    );
}
```

### Bearer Token Authentication

For simpler use cases, use Bearer token authentication:

```yaml
providers:
  - type: webhook
    headers:
      Authorization: "Bearer ${WEBHOOK_TOKEN}"
```

Verify in your endpoint:

```go
func handleWebhook(w http.ResponseWriter, r *http.Request) {
    token := r.Header.Get("Authorization")
    expectedToken := "Bearer " + os.Getenv("WEBHOOK_TOKEN")

    if token != expectedToken {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }

    // Process webhook...
}
```

## Retry Logic

### Retry Behavior

The webhook system implements exponential backoff with the following default behavior:

1. **Initial Attempt**: Webhook is sent immediately
2. **First Retry**: Wait 1 second, then retry
3. **Second Retry**: Wait 2 seconds, then retry
4. **Third Retry**: Wait 4 seconds, then retry
5. **Final Retry**: Wait up to max_delay (30s default), then retry

### Retry Configuration

```yaml
retry:
  enabled: true
  max_attempts: 3          # Maximum retry attempts
  initial_delay: 1s        # First retry delay
  max_delay: 30s           # Maximum delay between retries
  multiplier: 2.0          # Exponential backoff multiplier
  retry_on_status:         # HTTP status codes that trigger retry
    - 429  # Too Many Requests
    - 500  # Internal Server Error
    - 502  # Bad Gateway
    - 503  # Service Unavailable
    - 504  # Gateway Timeout
```

### Retry Headers

Each retry includes additional headers:

```
X-Retry-Attempt: 2
X-Original-Event-Id: evt_1a2b3c4d5e6f
X-First-Attempt-Time: 2025-12-30T10:30:00Z
```

### Disabling Retries

To disable retries for specific endpoints:

```yaml
retry:
  enabled: false
```

## Security Best Practices

### 1. Use HTTPS

Always use HTTPS for webhook endpoints in production:

```yaml
url: "https://your-app.com/webhooks"  # ✓ Good
url: "http://your-app.com/webhooks"   # ✗ Bad
```

### 2. Verify Signatures

Always verify HMAC signatures:

```go
if !verifySignature(r, webhookSecret) {
    http.Error(w, "Invalid signature", http.StatusUnauthorized)
    return
}
```

### 3. Use Strong Secrets

Generate strong webhook secrets:

```bash
openssl rand -hex 32
```

### 4. Validate Payload

Validate all payload data before processing:

```go
var payload WebhookPayload
if err := json.Unmarshal(body, &payload); err != nil {
    return fmt.Errorf("invalid payload: %w", err)
}

if payload.Event == "" {
    return fmt.Errorf("missing event type")
}
```

### 5. Rate Limiting

Implement rate limiting on your webhook endpoint:

```go
limiter := rate.NewLimiter(rate.Limit(10), 20)

func handleWebhook(w http.ResponseWriter, r *http.Request) {
    if !limiter.Allow() {
        http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
        return
    }
    // Process webhook...
}
```

### 6. Timeout Protection

Set reasonable timeouts for webhook processing:

```go
ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
defer cancel()

// Process webhook with timeout
```

### 7. Idempotency

Handle duplicate deliveries gracefully using `event_id`:

```go
if alreadyProcessed(payload.EventID) {
    w.WriteHeader(http.StatusOK)
    return
}

processEvent(payload)
markAsProcessed(payload.EventID)
```

## Testing Webhooks

### 1. Using cURL

Test webhook delivery manually:

```bash
curl -X POST https://your-app.com/webhooks/backup \
  -H "Content-Type: application/json" \
  -H "X-Backup-Signature: $(echo -n '{"event":"backup.completed"}' | openssl dgst -sha256 -hmac 'your-secret' | cut -d' ' -f2)" \
  -d '{
    "event": "backup.completed",
    "event_id": "evt_test123",
    "timestamp": "2025-12-30T10:30:00Z",
    "data": {
      "backup_id": "bkp_test",
      "database": "test_db",
      "size": 1000000
    }
  }'
```

### 2. Using webhook.site

For development, use [webhook.site](https://webhook.site) to inspect webhooks:

```yaml
providers:
  - type: webhook
    url: "https://webhook.site/your-unique-url"
```

### 3. Local Testing with ngrok

Test webhooks locally using ngrok:

```bash
# Start your local server
go run main.go

# In another terminal, start ngrok
ngrok http 8080

# Use the ngrok URL in your config
providers:
  - type: webhook
    url: "https://abc123.ngrok.io/webhooks/backup"
```

### 4. Mock Webhook Server

Create a simple test server:

```go
package main

import (
    "encoding/json"
    "io"
    "log"
    "net/http"
)

func main() {
    http.HandleFunc("/webhooks/backup", func(w http.ResponseWriter, r *http.Request) {
        body, _ := io.ReadAll(r.Body)
        log.Printf("Received webhook: %s", string(body))

        var payload map[string]interface{}
        json.Unmarshal(body, &payload)
        log.Printf("Event: %s", payload["event"])

        w.WriteHeader(http.StatusOK)
        json.NewEncoder(w).Encode(map[string]bool{"success": true})
    })

    log.Println("Test webhook server running on :8080")
    log.Fatal(http.ListenAndServe(":8080", nil))
}
```

## Examples

### Slack Integration

Send webhook events to Slack:

```go
type SlackMessage struct {
    Text string `json:"text"`
}

func handleWebhook(w http.ResponseWriter, r *http.Request) {
    var payload WebhookPayload
    json.NewDecoder(r.Body).Decode(&payload)

    var message string
    switch payload.Event {
    case "backup.completed":
        message = fmt.Sprintf("✅ Backup completed: %s",
            payload.Data["database"])
    case "backup.failed":
        message = fmt.Sprintf("❌ Backup failed: %s - %s",
            payload.Data["database"], payload.Data["error"])
    }

    // Send to Slack
    slackMsg := SlackMessage{Text: message}
    sendToSlack(os.Getenv("SLACK_WEBHOOK_URL"), slackMsg)

    w.WriteHeader(http.StatusOK)
}
```

### PagerDuty Integration

Trigger PagerDuty incidents on backup failures:

```go
func handleWebhook(w http.ResponseWriter, r *http.Request) {
    var payload WebhookPayload
    json.NewDecoder(r.Body).Decode(&payload)

    if payload.Event == "backup.failed" {
        triggerPagerDuty(payload.Data)
    }

    w.WriteHeader(http.StatusOK)
}

func triggerPagerDuty(data map[string]interface{}) {
    incident := map[string]interface{}{
        "routing_key": os.Getenv("PAGERDUTY_KEY"),
        "event_action": "trigger",
        "payload": map[string]interface{}{
            "summary": fmt.Sprintf("Backup failed: %s", data["database"]),
            "severity": "critical",
            "source": "db-backup-utility",
        },
    }

    // Send to PagerDuty Events API
    // ...
}
```

### Database Logging

Store webhook events in a database:

```go
func handleWebhook(w http.ResponseWriter, r *http.Request) {
    var payload WebhookPayload
    json.NewDecoder(r.Body).Decode(&payload)

    // Store in database
    db.Exec(`
        INSERT INTO webhook_events (event_id, event_type, data, timestamp)
        VALUES ($1, $2, $3, $4)
    `, payload.EventID, payload.Event, payload.Data, payload.Timestamp)

    w.WriteHeader(http.StatusOK)
}
```

## Troubleshooting

### Common Issues

#### 1. Webhooks Not Being Delivered

**Check:**
- Webhook URL is accessible from the backup server
- Firewall rules allow outbound HTTPS traffic
- Webhook provider is enabled in configuration
- Events are not being filtered out

**Debug:**
```yaml
logging:
  level: debug  # Enable debug logging to see webhook attempts
```

#### 2. Signature Verification Failing

**Check:**
- Secret matches on both sides
- Body is being read correctly (not parsed before verification)
- Signature header name matches configuration

**Example Fix:**
```go
// Read body once and reuse
body, _ := io.ReadAll(r.Body)
r.Body = io.NopCloser(bytes.NewBuffer(body))

// Now verify signature using body
verifySignature(body, secret)

// Parse JSON from same body
json.Unmarshal(body, &payload)
```

#### 3. Timeouts

**Increase timeout:**
```yaml
providers:
  - type: webhook
    timeout: 30s  # Increase from default 10s
```

**Make webhook handler faster:**
```go
func handleWebhook(w http.ResponseWriter, r *http.Request) {
    // Acknowledge immediately
    w.WriteHeader(http.StatusOK)

    // Process asynchronously
    go processWebhookAsync(r.Body)
}
```

#### 4. Too Many Retries

**Reduce retry attempts:**
```yaml
retry:
  max_attempts: 2
  initial_delay: 2s
```

**Return 2xx faster:**
```go
// Bad - slow response
func handleWebhook(w http.ResponseWriter, r *http.Request) {
    processLongRunningTask()
    w.WriteHeader(http.StatusOK)
}

// Good - fast acknowledgment
func handleWebhook(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK)
    go processLongRunningTask()
}
```

### Debugging Tips

#### 1. Enable Debug Logging

```yaml
logging:
  level: debug
  output: /var/log/db-backup/debug.log
```

#### 2. Test with webhook.site

```yaml
providers:
  - type: webhook
    url: "https://webhook.site/unique-id"
```

#### 3. Check Webhook Delivery Logs

View webhook delivery attempts:

```bash
tail -f /var/log/db-backup/webhooks.log
```

#### 4. Verify Network Connectivity

```bash
curl -v https://your-webhook-endpoint.com/webhooks/backup
```

## Additional Resources

- [OpenAPI Specification](./openapi.yaml) - Full API documentation
- [Configuration Guide](../README.md#configuration) - Configuration options
- [Security Guide](./SECURITY.md) - Security best practices
- [Plugin Development Guide](./PLUGIN_DEVELOPMENT.md) - Extend functionality

## Support

For issues or questions:
- GitHub Issues: https://github.com/your-org/db-backup/issues
- Documentation: https://docs.backup.example.com
- Email: support@example.com
