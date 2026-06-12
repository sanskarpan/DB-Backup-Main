package datadog

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
			name: "missing API key",
			config: &Config{
				Enabled: true,
			},
			wantError: true,
			errMsg:    "API key is required",
		},
		{
			name: "valid config with defaults",
			config: &Config{
				APIKey:  "test-api-key",
				Enabled: true,
			},
			wantError: false,
		},
		{
			name: "valid config with all fields",
			config: &Config{
				APIKey:      "test-api-key",
				AppKey:      "test-app-key",
				Site:        "datadoghq.eu",
				ServiceName: "custom-service",
				Environment: "staging",
				Version:     "1.0.0",
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
					if tt.config.ServiceName == "" {
						assert.Equal(t, "db-backup-service", client.config.ServiceName)
					}
					if tt.config.Environment == "" {
						assert.Equal(t, "production", client.config.Environment)
					}
					if tt.config.Site == "" {
						assert.Equal(t, "datadoghq.com", client.config.Site)
					}
				}
			}
		})
	}
}

func TestClient_StartStop(t *testing.T) {
	t.Run("disabled client", func(t *testing.T) {
		client, _ := NewClient(&Config{Enabled: false})
		err := client.Start()
		assert.NoError(t, err)
		client.Stop()
	})

	t.Run("enabled client", func(t *testing.T) {
		client, _ := NewClient(&Config{
			APIKey:  "test-key",
			Enabled: true,
		})

		// Start and stop should not panic
		err := client.Start()
		assert.NoError(t, err)
		client.Stop()
	})
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
			name:         "successful backup - enabled",
			enabled:      true,
			databaseName: "test-db",
			backupType:   "full",
			fn: func(ctx context.Context) error {
				time.Sleep(10 * time.Millisecond)
				return nil
			},
			wantError: false,
		},
		{
			name:         "failed backup",
			enabled:      true,
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
				APIKey:  "test-key",
				Enabled: tt.enabled,
			}

			if tt.enabled {
				config.ServiceName = "test-service"
			}

			client, err := NewClient(config)
			require.NoError(t, err)

			if tt.enabled {
				err = client.Start()
				require.NoError(t, err)
				defer client.Stop()
			}

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
			name:         "successful restore",
			enabled:      true,
			databaseName: "test-db",
			fn: func(ctx context.Context) error {
				time.Sleep(5 * time.Millisecond)
				return nil
			},
			wantError: false,
		},
		{
			name:         "failed restore",
			enabled:      true,
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
				APIKey:      "test-key",
				ServiceName: "test-service",
				Enabled:     tt.enabled,
			}

			client, err := NewClient(config)
			require.NoError(t, err)

			if tt.enabled {
				err = client.Start()
				require.NoError(t, err)
				defer client.Stop()
			}

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
		operation string
		provider  string
		fn        func(context.Context) error
		wantError bool
	}{
		{
			name:      "successful upload",
			operation: "upload",
			provider:  "s3",
			fn: func(ctx context.Context) error {
				return nil
			},
			wantError: false,
		},
		{
			name:      "failed download",
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
				APIKey:      "test-key",
				ServiceName: "test-service",
				Enabled:     true,
			})

			err := client.Start()
			require.NoError(t, err)
			defer client.Stop()

			ctx := context.Background()
			err = client.TraceStorageOperation(ctx, tt.operation, tt.provider, tt.fn)

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
		algorithm string
		dataSize  int64
		fn        func(context.Context) error
		wantError bool
	}{
		{
			name:      "successful encryption",
			algorithm: "aes-256-gcm",
			dataSize:  1024 * 1024, // 1MB
			fn: func(ctx context.Context) error {
				time.Sleep(10 * time.Millisecond)
				return nil
			},
			wantError: false,
		},
		{
			name:      "failed encryption",
			algorithm: "chacha20-poly1305",
			dataSize:  2048,
			fn: func(ctx context.Context) error {
				return errors.New("encryption failed")
			},
			wantError: true,
		},
		{
			name:      "zero data size",
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
				APIKey:      "test-key",
				ServiceName: "test-service",
				Enabled:     true,
			})

			err := client.Start()
			require.NoError(t, err)
			defer client.Stop()

			ctx := context.Background()
			err = client.TraceEncryption(ctx, tt.algorithm, tt.dataSize, tt.fn)

			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestClient_SendMetric(t *testing.T) {
	tests := []struct {
		name    string
		enabled bool
		metric  string
		value   float64
		tags    map[string]string
	}{
		{
			name:    "send metric - disabled",
			enabled: false,
			metric:  "backup.count",
			value:   10.0,
			tags:    map[string]string{"env": "test"},
		},
		{
			name:    "send metric - enabled",
			enabled: true,
			metric:  "backup.duration",
			value:   123.45,
			tags:    map[string]string{"database": "postgres"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, _ := NewClient(&Config{
				APIKey:  "test-key",
				Enabled: tt.enabled,
			})

			err := client.SendMetric(tt.metric, tt.value, tt.tags)
			assert.NoError(t, err)
		})
	}
}

func TestClient_SendEvent(t *testing.T) {
	tests := []struct {
		name      string
		enabled   bool
		title     string
		text      string
		tags      map[string]string
		alertType string
	}{
		{
			name:      "send event - disabled",
			enabled:   false,
			title:     "Backup Started",
			text:      "Backup started for database test-db",
			tags:      map[string]string{"env": "test"},
			alertType: "info",
		},
		{
			name:      "send event - enabled",
			enabled:   true,
			title:     "Backup Failed",
			text:      "Backup failed for database prod-db",
			tags:      map[string]string{"env": "production"},
			alertType: "error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, _ := NewClient(&Config{
				APIKey:  "test-key",
				Enabled: tt.enabled,
			})

			err := client.SendEvent(tt.title, tt.text, tt.tags, tt.alertType)
			assert.NoError(t, err)
		})
	}
}

func TestClient_LogError(t *testing.T) {
	client, _ := NewClient(&Config{
		APIKey:      "test-key",
		ServiceName: "test-service",
		Enabled:     true,
	})

	err := client.Start()
	require.NoError(t, err)
	defer client.Stop()

	ctx := context.Background()
	testErr := errors.New("test error")
	tags := map[string]string{
		"component": "backup",
		"severity":  "high",
	}

	// Should not panic
	client.LogError(ctx, testErr, tags)
}

func TestNoopSpan(t *testing.T) {
	span := &noopSpan{}

	// All methods should be no-op and not panic
	span.SetTag("key", "value")
	span.SetOperationName("test-op")
	span.SetBaggageItem("key", "value")
	assert.Equal(t, "", span.BaggageItem("key"))
	span.Finish()
	assert.NotNil(t, span.Context())
}

func TestNoopSpanContext(t *testing.T) {
	ctx := &noopSpanContext{}

	assert.Equal(t, uint64(0), ctx.TraceID())
	assert.Equal(t, uint64(0), ctx.SpanID())

	// Should not panic
	ctx.ForeachBaggageItem(func(k, v string) bool {
		return true
	})
}

func BenchmarkTraceBackup(b *testing.B) {
	client, _ := NewClient(&Config{
		APIKey:      "test-key",
		ServiceName: "test-service",
		Enabled:     true,
	})

	err := client.Start()
	require.NoError(b, err)
	defer client.Stop()

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
		APIKey:      "test-key",
		ServiceName: "test-service",
		Enabled:     true,
	})

	err := client.Start()
	require.NoError(b, err)
	defer client.Stop()

	ctx := context.Background()
	fn := func(ctx context.Context) error {
		return nil
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = client.TraceEncryption(ctx, "aes-256-gcm", 1024*1024, fn)
	}
}
