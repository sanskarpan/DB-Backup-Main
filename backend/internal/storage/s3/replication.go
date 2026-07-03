// Package s3 provides cross-region replication for AWS S3
package s3

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"

	pkgErrors "github.com/sanskarpan/db-backup/pkg/errors"
)

// ReplicationConfig holds cross-region replication configuration.
type ReplicationConfig struct {
	Role                 string // IAM role ARN for replication
	DestinationBucket    string // Destination bucket ARN
	DestinationRegion    string // Destination region
	Priority             int32  // Rule priority
	Prefix               string // Object prefix filter
	Status               string // "Enabled" or "Disabled"
	StorageClass         string // Destination storage class
	ReplicateDeletes     bool   // Whether to replicate delete markers
	ReplicateMetadata    bool   // Whether to replicate object metadata
	ReplicateTags        bool   // Whether to replicate object tags
	ReplicationTimeValue int32  // Replication time in minutes (0 = disabled)
}

// SetupReplication configures cross-region replication for the bucket.
func (p *S3Provider) SetupReplication(ctx context.Context, config *ReplicationConfig) error {
	if config == nil {
		return pkgErrors.New(pkgErrors.ErrorTypeConfiguration, "replication config is required")
	}

	// Validate configuration
	if err := p.validateReplicationConfig(config); err != nil {
		return err
	}

	// Enable versioning on source bucket (required for replication)
	if err := p.enableVersioning(ctx); err != nil {
		return fmt.Errorf("failed to enable versioning: %w", err)
	}

	// Build replication rule
	rule := types.ReplicationRule{
		ID:       aws.String("cross-region-replication"),
		Priority: aws.Int32(config.Priority),
		Status:   types.ReplicationRuleStatus(config.Status),
		Filter: &types.ReplicationRuleFilterMemberPrefix{
			Value: config.Prefix,
		},
		Destination: &types.Destination{
			Bucket:       aws.String(config.DestinationBucket),
			StorageClass: types.StorageClass(config.StorageClass),
		},
	}

	// Add delete marker replication if enabled
	if config.ReplicateDeletes {
		rule.DeleteMarkerReplication = &types.DeleteMarkerReplication{
			Status: types.DeleteMarkerReplicationStatusEnabled,
		}
	}

	// Add metadata replication if enabled
	if config.ReplicateMetadata || config.ReplicateTags {
		rule.SourceSelectionCriteria = &types.SourceSelectionCriteria{
			SseKmsEncryptedObjects: &types.SseKmsEncryptedObjects{
				Status: types.SseKmsEncryptedObjectsStatusEnabled,
			},
		}
	}

	// Add replication time control if specified
	if config.ReplicationTimeValue > 0 {
		rule.Destination.ReplicationTime = &types.ReplicationTime{
			Status: types.ReplicationTimeStatusEnabled,
			Time: &types.ReplicationTimeValue{
				Minutes: aws.Int32(config.ReplicationTimeValue),
			},
		}
		rule.Destination.Metrics = &types.Metrics{
			Status: types.MetricsStatusEnabled,
			EventThreshold: &types.ReplicationTimeValue{
				Minutes: aws.Int32(config.ReplicationTimeValue),
			},
		}
	}

	// Create or update replication configuration
	replicationConfig := &types.ReplicationConfiguration{
		Role:  aws.String(config.Role),
		Rules: []types.ReplicationRule{rule},
	}

	_, err := p.client.PutBucketReplication(ctx, &s3.PutBucketReplicationInput{
		Bucket:                   aws.String(p.config.Bucket),
		ReplicationConfiguration: replicationConfig,
	})
	if err != nil {
		return pkgErrors.New(pkgErrors.ErrorTypeStorage, fmt.Sprintf("failed to setup replication: %v", err))
	}

	return nil
}

// GetReplicationStatus retrieves the current replication configuration.
func (p *S3Provider) GetReplicationStatus(ctx context.Context) (*ReplicationStatus, error) {
	result, err := p.client.GetBucketReplication(ctx, &s3.GetBucketReplicationInput{
		Bucket: aws.String(p.config.Bucket),
	})
	if err != nil {
		return nil, pkgErrors.New(pkgErrors.ErrorTypeStorage, fmt.Sprintf("failed to get replication status: %v", err))
	}

	if result.ReplicationConfiguration == nil || len(result.ReplicationConfiguration.Rules) == 0 {
		return &ReplicationStatus{
			Enabled: false,
		}, nil
	}

	// Parse first rule (assuming single rule setup)
	rule := result.ReplicationConfiguration.Rules[0]

	status := &ReplicationStatus{
		Enabled:           rule.Status == types.ReplicationRuleStatusEnabled,
		Role:              aws.ToString(result.ReplicationConfiguration.Role),
		DestinationBucket: aws.ToString(rule.Destination.Bucket),
		Priority:          aws.ToInt32(rule.Priority),
	}

	if rule.Destination.StorageClass != "" {
		status.StorageClass = string(rule.Destination.StorageClass)
	}

	if rule.DeleteMarkerReplication != nil {
		status.ReplicateDeletes = rule.DeleteMarkerReplication.Status == types.DeleteMarkerReplicationStatusEnabled
	}

	return status, nil
}

// DisableReplication disables cross-region replication.
func (p *S3Provider) DisableReplication(ctx context.Context) error {
	_, err := p.client.DeleteBucketReplication(ctx, &s3.DeleteBucketReplicationInput{
		Bucket: aws.String(p.config.Bucket),
	})
	if err != nil {
		return pkgErrors.New(pkgErrors.ErrorTypeStorage, fmt.Sprintf("failed to disable replication: %v", err))
	}

	return nil
}

// ValidateReplicationSetup validates that replication is properly configured.
func (p *S3Provider) ValidateReplicationSetup(ctx context.Context) error {
	// Check if versioning is enabled
	versioningResult, err := p.client.GetBucketVersioning(ctx, &s3.GetBucketVersioningInput{
		Bucket: aws.String(p.config.Bucket),
	})
	if err != nil {
		return fmt.Errorf("failed to check versioning: %w", err)
	}

	if versioningResult.Status != types.BucketVersioningStatusEnabled {
		return pkgErrors.New(pkgErrors.ErrorTypeValidation, "bucket versioning must be enabled for replication")
	}

	// Check replication configuration
	status, err := p.GetReplicationStatus(ctx)
	if err != nil {
		return err
	}

	if !status.Enabled {
		return pkgErrors.New(pkgErrors.ErrorTypeValidation, "replication is not enabled")
	}

	return nil
}

// ReplicationStatus represents the current replication status.
type ReplicationStatus struct {
	Enabled           bool
	Role              string
	DestinationBucket string
	Priority          int32
	StorageClass      string
	ReplicateDeletes  bool
}

// Helper methods

func (p *S3Provider) validateReplicationConfig(config *ReplicationConfig) error {
	if config == nil {
		return pkgErrors.New(pkgErrors.ErrorTypeValidation, "replication config cannot be nil")
	}

	if config.Role == "" {
		return pkgErrors.New(pkgErrors.ErrorTypeValidation, "IAM role ARN is required for replication")
	}

	if config.DestinationBucket == "" {
		return pkgErrors.New(pkgErrors.ErrorTypeValidation, "destination bucket ARN is required")
	}

	if config.DestinationRegion == "" {
		return pkgErrors.New(pkgErrors.ErrorTypeValidation, "destination region is required")
	}

	if config.Status != "Enabled" && config.Status != "Disabled" {
		return pkgErrors.New(pkgErrors.ErrorTypeValidation, "status must be 'Enabled' or 'Disabled'")
	}

	// Validate storage class
	validClasses := map[string]bool{
		"STANDARD":            true,
		"REDUCED_REDUNDANCY":  true,
		"STANDARD_IA":         true,
		"ONEZONE_IA":          true,
		"INTELLIGENT_TIERING": true,
		"GLACIER":             true,
		"DEEP_ARCHIVE":        true,
	}

	if config.StorageClass != "" && !validClasses[config.StorageClass] {
		return pkgErrors.New(pkgErrors.ErrorTypeValidation, fmt.Sprintf("invalid storage class: %s", config.StorageClass))
	}

	return nil
}

func (p *S3Provider) enableVersioning(ctx context.Context) error {
	_, err := p.client.PutBucketVersioning(ctx, &s3.PutBucketVersioningInput{
		Bucket: aws.String(p.config.Bucket),
		VersioningConfiguration: &types.VersioningConfiguration{
			Status: types.BucketVersioningStatusEnabled,
		},
	})
	if err != nil {
		return pkgErrors.New(pkgErrors.ErrorTypeStorage, fmt.Sprintf("failed to enable versioning: %v", err))
	}

	return nil
}

// MonitorReplication provides metrics about replication progress.
func (p *S3Provider) MonitorReplication(ctx context.Context, objectKey string) (*ReplicationMetrics, error) {
	// Get object replication status
	result, err := p.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(p.config.Bucket),
		Key:    aws.String(objectKey),
	})
	if err != nil {
		return nil, pkgErrors.New(pkgErrors.ErrorTypeStorage, fmt.Sprintf("failed to get object metadata: %v", err))
	}

	metrics := &ReplicationMetrics{
		ObjectKey:         objectKey,
		ReplicationStatus: string(result.ReplicationStatus),
	}

	return metrics, nil
}

// ReplicationMetrics contains replication metrics for an object.
type ReplicationMetrics struct {
	ObjectKey         string
	ReplicationStatus string
}
