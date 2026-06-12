package s3

import (
	"testing"
)

func TestValidateReplicationConfig(t *testing.T) {
	provider := &S3Provider{}

	tests := []struct {
		name      string
		config    *ReplicationConfig
		expectErr bool
	}{
		{
			name: "valid config",
			config: &ReplicationConfig{
				Role:              "arn:aws:iam::123456789012:role/replication-role",
				DestinationBucket: "arn:aws:s3:::destination-bucket",
				DestinationRegion: "us-west-2",
				Status:            "Enabled",
				StorageClass:      "STANDARD_IA",
			},
			expectErr: false,
		},
		{
			name: "missing role",
			config: &ReplicationConfig{
				DestinationBucket: "arn:aws:s3:::dest",
				DestinationRegion: "us-west-2",
				Status:            "Enabled",
			},
			expectErr: true,
		},
		{
			name: "missing destination bucket",
			config: &ReplicationConfig{
				Role:              "arn:aws:iam::123456789012:role/replication-role",
				DestinationRegion: "us-west-2",
				Status:            "Enabled",
			},
			expectErr: true,
		},
		{
			name: "missing destination region",
			config: &ReplicationConfig{
				Role:              "arn:aws:iam::123456789012:role/replication-role",
				DestinationBucket: "arn:aws:s3:::dest",
				Status:            "Enabled",
			},
			expectErr: true,
		},
		{
			name: "invalid status",
			config: &ReplicationConfig{
				Role:              "arn:aws:iam::123456789012:role/replication-role",
				DestinationBucket: "arn:aws:s3:::dest",
				DestinationRegion: "us-west-2",
				Status:            "Invalid",
			},
			expectErr: true,
		},
		{
			name: "invalid storage class",
			config: &ReplicationConfig{
				Role:              "arn:aws:iam::123456789012:role/replication-role",
				DestinationBucket: "arn:aws:s3:::dest",
				DestinationRegion: "us-west-2",
				Status:            "Enabled",
				StorageClass:      "INVALID_CLASS",
			},
			expectErr: true,
		},
		{
			name: "valid with all options",
			config: &ReplicationConfig{
				Role:                 "arn:aws:iam::123456789012:role/replication-role",
				DestinationBucket:    "arn:aws:s3:::dest",
				DestinationRegion:    "us-west-2",
				Priority:             1,
				Prefix:               "backups/",
				Status:               "Enabled",
				StorageClass:         "GLACIER",
				ReplicateDeletes:     true,
				ReplicateMetadata:    true,
				ReplicateTags:        true,
				ReplicationTimeValue: 15,
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := provider.validateReplicationConfig(tt.config)

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

func TestValidStorageClasses(t *testing.T) {
	provider := &S3Provider{}

	validClasses := []string{
		"STANDARD",
		"REDUCED_REDUNDANCY",
		"STANDARD_IA",
		"ONEZONE_IA",
		"INTELLIGENT_TIERING",
		"GLACIER",
		"DEEP_ARCHIVE",
	}

	for _, class := range validClasses {
		t.Run(class, func(t *testing.T) {
			config := &ReplicationConfig{
				Role:              "arn:aws:iam::123456789012:role/replication-role",
				DestinationBucket: "arn:aws:s3:::destination-bucket",
				DestinationRegion: "us-west-2",
				Status:            "Enabled",
				StorageClass:      class,
			}

			err := provider.validateReplicationConfig(config)
			if err != nil {
				t.Errorf("Valid storage class %s should not produce error: %v", class, err)
			}
		})
	}
}

func TestReplicationConfigDefaults(t *testing.T) {
	config := &ReplicationConfig{
		Role:              "arn:aws:iam::123456789012:role/replication-role",
		DestinationBucket: "arn:aws:s3:::destination-bucket",
		DestinationRegion: "us-west-2",
		Status:            "Enabled",
	}

	// Test that empty prefix is valid
	provider := &S3Provider{}
	err := provider.validateReplicationConfig(config)
	if err != nil {
		t.Errorf("Config with empty prefix should be valid: %v", err)
	}

	// Test that priority can be 0
	config.Priority = 0
	err = provider.validateReplicationConfig(config)
	if err != nil {
		t.Errorf("Config with priority 0 should be valid: %v", err)
	}

	// Test that replication time value can be 0 (disabled)
	config.ReplicationTimeValue = 0
	err = provider.validateReplicationConfig(config)
	if err != nil {
		t.Errorf("Config with replication time 0 should be valid: %v", err)
	}
}

func TestReplicationStatusValues(t *testing.T) {
	provider := &S3Provider{}

	tests := []struct {
		status    string
		expectErr bool
	}{
		{"Enabled", false},
		{"Disabled", false},
		{"enabled", true},   // Case sensitive
		{"disabled", true},  // Case sensitive
		{"ENABLED", true},   // Case sensitive
		{"DISABLED", true},  // Case sensitive
		{"Active", true},    // Invalid
		{"", true},          // Empty
		{"Paused", true},    // Invalid
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			config := &ReplicationConfig{
				Role:              "arn:aws:iam::123456789012:role/replication-role",
				DestinationBucket: "arn:aws:s3:::destination-bucket",
				DestinationRegion: "us-west-2",
				Status:            tt.status,
			}

			err := provider.validateReplicationConfig(config)

			if tt.expectErr && err == nil {
				t.Error("Expected error for invalid status but got nil")
			} else if !tt.expectErr && err != nil {
				t.Errorf("Unexpected error for valid status: %v", err)
			}
		})
	}
}

func TestReplicationMetricsStructure(t *testing.T) {
	metrics := &ReplicationMetrics{
		ObjectKey:         "test-object.txt",
		ReplicationStatus: "COMPLETED",
	}

	if metrics.ObjectKey != "test-object.txt" {
		t.Errorf("Expected object key 'test-object.txt', got '%s'", metrics.ObjectKey)
	}

	if metrics.ReplicationStatus != "COMPLETED" {
		t.Errorf("Expected status 'COMPLETED', got '%s'", metrics.ReplicationStatus)
	}
}

func TestReplicationStatusStructure(t *testing.T) {
	status := &ReplicationStatus{
		Enabled:           true,
		Role:              "arn:aws:iam::123456789012:role/replication-role",
		DestinationBucket: "arn:aws:s3:::destination-bucket",
		Priority:          1,
		StorageClass:      "STANDARD_IA",
		ReplicateDeletes:  true,
	}

	if !status.Enabled {
		t.Error("Expected status to be enabled")
	}

	if status.Role != "arn:aws:iam::123456789012:role/replication-role" {
		t.Errorf("Unexpected role: %s", status.Role)
	}

	if status.DestinationBucket != "arn:aws:s3:::destination-bucket" {
		t.Errorf("Unexpected destination bucket: %s", status.DestinationBucket)
	}

	if status.Priority != 1 {
		t.Errorf("Expected priority 1, got %d", status.Priority)
	}

	if status.StorageClass != "STANDARD_IA" {
		t.Errorf("Expected storage class 'STANDARD_IA', got '%s'", status.StorageClass)
	}

	if !status.ReplicateDeletes {
		t.Error("Expected ReplicateDeletes to be true")
	}
}

func TestReplicationConfigStructure(t *testing.T) {
	config := &ReplicationConfig{
		Role:                 "arn:aws:iam::123456789012:role/replication-role",
		DestinationBucket:    "arn:aws:s3:::destination-bucket",
		DestinationRegion:    "us-west-2",
		Priority:             1,
		Prefix:               "backups/",
		Status:               "Enabled",
		StorageClass:         "STANDARD_IA",
		ReplicateDeletes:     true,
		ReplicateMetadata:    true,
		ReplicateTags:        true,
		ReplicationTimeValue: 15,
	}

	// Verify all fields are set correctly
	if config.Role != "arn:aws:iam::123456789012:role/replication-role" {
		t.Errorf("Unexpected role: %s", config.Role)
	}

	if config.DestinationBucket != "arn:aws:s3:::destination-bucket" {
		t.Errorf("Unexpected destination bucket: %s", config.DestinationBucket)
	}

	if config.DestinationRegion != "us-west-2" {
		t.Errorf("Unexpected destination region: %s", config.DestinationRegion)
	}

	if config.Priority != 1 {
		t.Errorf("Expected priority 1, got %d", config.Priority)
	}

	if config.Prefix != "backups/" {
		t.Errorf("Unexpected prefix: %s", config.Prefix)
	}

	if config.Status != "Enabled" {
		t.Errorf("Unexpected status: %s", config.Status)
	}

	if config.StorageClass != "STANDARD_IA" {
		t.Errorf("Unexpected storage class: %s", config.StorageClass)
	}

	if !config.ReplicateDeletes {
		t.Error("Expected ReplicateDeletes to be true")
	}

	if !config.ReplicateMetadata {
		t.Error("Expected ReplicateMetadata to be true")
	}

	if !config.ReplicateTags {
		t.Error("Expected ReplicateTags to be true")
	}

	if config.ReplicationTimeValue != 15 {
		t.Errorf("Expected replication time value 15, got %d", config.ReplicationTimeValue)
	}
}

func TestReplicationConfigNilValidation(t *testing.T) {
	provider := &S3Provider{}

	err := provider.validateReplicationConfig(nil)
	if err == nil {
		t.Error("Expected error for nil config")
	}
}
