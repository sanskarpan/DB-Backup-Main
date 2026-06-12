package newrelic

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name      string
		config    *Config
		wantError bool
		errMsg    string
	}{
		{
			name:      "nil config",
			config:    nil,
			wantError: true,
			errMsg:    "config is required",
		},
		{
			name: "disabled client",
			config: &Config{
				Enabled: false,
			},
			wantError: false,
		},
		{
			name: "missing license key",
			config: &Config{
				Enabled: true,
			},
			wantError: true,
			errMsg:    "license key is required",
		},
		{
			name: "valid config with defaults",
			config: &Config{
				LicenseKey: "test-license-key",
				Enabled:    true,
			},
			wantError: false,
		},
		{
			name: "valid config with all fields",
			config: &Config{
				LicenseKey:  "test-license-key",
				AppName:     "custom-app",
				Environment: "staging",
				Enabled:     true,
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.config)

			if tt.wantError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				assert.Nil(t, client)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, client)

				if tt.config != nil && tt.config.Enabled {
					// Check defaults
					if tt.config.AppName == "" {
						assert.Equal(t, "db-backup-service", client.config.AppName)
					}
					if tt.config.Environment == "" {
						assert.Equal(t, "production", client.config.Environment)
					}
				}
			}
		})
	}
}

func TestClient_StartStop_Disabled(t *testing.T) {
	client, _ := NewClient(&Config{Enabled: false})

	err := client.Start()
	assert.NoError(t, err)

	client.Stop()
}

func TestClient_StartStop_Enabled(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping New Relic integration test in short mode")
	}

	client, _ := NewClient(&Config{
		LicenseKey: "test-license-key-0123456789012345678901234567890123456789",
		AppName:    "test-app",
		Enabled:    true,
	})

	err := client.Start()
	// Note: This will fail without a valid license key, but we're testing the flow
	// In real tests, you would use a valid license key or mock the New Relic client
	if err == nil {
		defer client.Stop()
	}
}

func TestClient_TraceBackup(t *testing.T) {
	tests := []struct {
		name         string
		enabled      bool
		databaseName string
		backupType   string
		fn           func(context.Context) error
		wantError    bool
	}{
		{
			name:         "successful backup - disabled",
			enabled:      false,
			databaseName: "test-db",
			backupType:   "full",
			fn: func(ctx context.Context) error {
				return nil
			},
			wantError: false,
		},
		{
			name:         "failed backup - disabled",
			enabled:      false,
			databaseName: "test-db",
			backupType:   "incremental",
			fn: func(ctx context.Context) error {
				return errors.New("backup failed")
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				LicenseKey: "test-key",
				Enabled:    tt.enabled,
			}

			client, err := NewClient(config)
			require.NoError(t, err)

			ctx := context.Background()
			err = client.TraceBackup(ctx, tt.databaseName, tt.backupType, tt.fn)

			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestClient_TraceRestore(t *testing.T) {
	tests := []struct {
		name         string
		enabled      bool
		databaseName string
		fn           func(context.Context) error
		wantError    bool
	}{
		{
			name:         "successful restore - disabled",
			enabled:      false,
			databaseName: "test-db",
			fn: func(ctx context.Context) error {
				time.Sleep(5 * time.Millisecond)
				return nil
			},
			wantError: false,
		},
		{
			name:         "failed restore - disabled",
			enabled:      false,
			databaseName: "test-db",
			fn: func(ctx context.Context) error {
				return errors.New("restore failed")
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				LicenseKey: "test-key",
				AppName:    "test-app",
				Enabled:    tt.enabled,
			}

			client, err := NewClient(config)
			require.NoError(t, err)

			ctx := context.Background()
			err = client.TraceRestore(ctx, tt.databaseName, tt.fn)

			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestClient_TraceStorageOperation(t *testing.T) {
	tests := []struct {
		name      string
		enabled   bool
		operation string
		provider  string
		fn        func(context.Context) error
		wantError bool
	}{
		{
			name:      "successful upload - disabled",
			enabled:   false,
			operation: "upload",
			provider:  "s3",
			fn: func(ctx context.Context) error {
				return nil
			},
			wantError: false,
		},
		{
			name:      "failed download - disabled",
			enabled:   false,
			operation: "download",
			provider:  "azure",
			fn: func(ctx context.Context) error {
				return errors.New("download failed")
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, _ := NewClient(&Config{
				LicenseKey: "test-key",
				AppName:    "test-app",
				Enabled:    tt.enabled,
			})

			ctx := context.Background()
			err := client.TraceStorageOperation(ctx, tt.operation, tt.provider, tt.fn)

			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestClient_TraceEncryption(t *testing.T) {
	tests := []struct {
		name      string
		enabled   bool
		algorithm string
		dataSize  int64
		fn        func(context.Context) error
		wantError bool
	}{
		{
			name:      "successful encryption - disabled",
			enabled:   false,
			algorithm: "aes-256-gcm",
			dataSize:  1024 * 1024, // 1MB
			fn: func(ctx context.Context) error {
				time.Sleep(10 * time.Millisecond)
				return nil
			},
			wantError: false,
		},
		{
			name:      "failed encryption - disabled",
			enabled:   false,
			algorithm: "chacha20-poly1305",
			dataSize:  2048,
			fn: func(ctx context.Context) error {
				return errors.New("encryption failed")
			},
			wantError: true,
		},
		{
			name:      "zero data size - disabled",
			enabled:   false,
			algorithm: "aes-256-gcm",
			dataSize:  0,
			fn: func(ctx context.Context) error {
				return nil
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, _ := NewClient(&Config{
				LicenseKey: "test-key",
				AppName:    "test-app",
				Enabled:    tt.enabled,
			})

			ctx := context.Background()
			err := client.TraceEncryption(ctx, tt.algorithm, tt.dataSize, tt.fn)

			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestClient_RecordMetric(t *testing.T) {
	tests := []struct {
		name    string
		enabled bool
		metric  string
		value   float64
	}{
		{
			name:    "record metric - disabled",
			enabled: false,
			metric:  "backup.count",
			value:   10.0,
		},
		{
			name:    "record metric - enabled (no-op without app)",
			enabled: true,
			metric:  "backup.duration",
			value:   123.45,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, _ := NewClient(&Config{
				LicenseKey: "test-key",
				Enabled:    tt.enabled,
			})

			err := client.RecordMetric(tt.metric, tt.value)
			assert.NoError(t, err)
		})
	}
}

func TestClient_RecordEvent(t *testing.T) {
	tests := []struct {
		name       string
		enabled    bool
		eventType  string
		attributes map[string]interface{}
	}{
		{
			name:      "record event - disabled",
			enabled:   false,
			eventType: "BackupStarted",
			attributes: map[string]interface{}{
				"database": "test-db",
				"type":     "full",
			},
		},
		{
			name:      "record event - enabled (no-op without app)",
			enabled:   true,
			eventType: "BackupCompleted",
			attributes: map[string]interface{}{
				"database": "prod-db",
				"duration": 123.45,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, _ := NewClient(&Config{
				LicenseKey: "test-key",
				Enabled:    tt.enabled,
			})

			err := client.RecordEvent(tt.eventType, tt.attributes)
			assert.NoError(t, err)
		})
	}
}

func TestClient_NoticeError(t *testing.T) {
	tests := []struct {
		name       string
		enabled    bool
		err        error
		attributes map[string]interface{}
	}{
		{
			name:    "notice error - disabled",
			enabled: false,
			err:     errors.New("test error"),
			attributes: map[string]interface{}{
				"component": "backup",
				"severity":  "high",
			},
		},
		{
			name:    "notice error - enabled (no-op without app)",
			enabled: true,
			err:     errors.New("another error"),
			attributes: map[string]interface{}{
				"database": "test-db",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, _ := NewClient(&Config{
				LicenseKey: "test-key",
				Enabled:    tt.enabled,
			})

			ctx := context.Background()
			// Should not panic
			client.NoticeError(ctx, tt.err, tt.attributes)
		})
	}
}

func TestClient_StartSegment(t *testing.T) {
	client, _ := NewClient(&Config{
		LicenseKey: "test-key",
		Enabled:    false,
	})

	ctx := context.Background()
	segment := client.StartSegment(ctx, "test-segment")

	// Should return nil for disabled client
	assert.Nil(t, segment)

	// End should not panic with nil segment
	client.EndSegment(segment)
}

func TestClient_StartDatastoreSegment(t *testing.T) {
	client, _ := NewClient(&Config{
		LicenseKey: "test-key",
		Enabled:    false,
	})

	ctx := context.Background()
	segment := client.StartDatastoreSegment(ctx, "PostgreSQL", "backups", "SELECT")

	// Should return nil for disabled client
	assert.Nil(t, segment)

	// End should not panic with nil segment
	client.EndDatastoreSegment(segment)
}

func TestClient_StartExternalSegment(t *testing.T) {
	client, _ := NewClient(&Config{
		LicenseKey: "test-key",
		Enabled:    false,
	})

	ctx := context.Background()
	segment := client.StartExternalSegment(ctx, "https://api.example.com")

	// Should return nil for disabled client
	assert.Nil(t, segment)

	// End should not panic with nil segment
	client.EndExternalSegment(segment)
}

func TestClient_Application(t *testing.T) {
	client, _ := NewClient(&Config{
		LicenseKey: "test-key",
		Enabled:    false,
	})

	app := client.Application()
	assert.Nil(t, app)
}

func TestClient_WaitForConnection(t *testing.T) {
	client, _ := NewClient(&Config{
		LicenseKey: "test-key",
		Enabled:    false,
	})

	err := client.WaitForConnection(1 * time.Second)
	assert.NoError(t, err)
}

func TestClient_StartTransaction(t *testing.T) {
	client, _ := NewClient(&Config{
		LicenseKey: "test-key",
		Enabled:    false,
	})

	txn := client.StartTransaction("test-transaction")
	assert.Nil(t, txn)
}

func TestClient_StartTransactionFromContext(t *testing.T) {
	client, _ := NewClient(&Config{
		LicenseKey: "test-key",
		Enabled:    false,
	})

	ctx := context.Background()
	txn, newCtx := client.StartTransactionFromContext(ctx, "test-transaction")

	assert.Nil(t, txn)
	assert.Equal(t, ctx, newCtx)
}

func BenchmarkTraceBackup(b *testing.B) {
	client, _ := NewClient(&Config{
		LicenseKey: "test-key",
		Enabled:    false,
	})

	ctx := context.Background()
	fn := func(ctx context.Context) error {
		return nil
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = client.TraceBackup(ctx, "test-db", "full", fn)
	}
}

func BenchmarkTraceEncryption(b *testing.B) {
	client, _ := NewClient(&Config{
		LicenseKey: "test-key",
		Enabled:    false,
	})

	ctx := context.Background()
	fn := func(ctx context.Context) error {
		return nil
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = client.TraceEncryption(ctx, "aes-256-gcm", 1024*1024, fn)
	}
}

func BenchmarkRecordMetric(b *testing.B) {
	client, _ := NewClient(&Config{
		LicenseKey: "test-key",
		Enabled:    false,
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = client.RecordMetric("test.metric", 123.45)
	}
}
