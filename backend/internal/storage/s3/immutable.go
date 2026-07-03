// Package s3 provides AWS S3 storage with immutable backup support
package s3

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"

	pkgErrors "github.com/sanskarpan/db-backup/pkg/errors"
)

// ObjectLockMode defines the S3 Object Lock mode.
type ObjectLockMode string

const (
	// ObjectLockModeGovernance allows users with special permissions to alter retention.
	ObjectLockModeGovernance ObjectLockMode = "GOVERNANCE"
	// ObjectLockModeCompliance prevents anyone, including root, from altering retention.
	ObjectLockModeCompliance ObjectLockMode = "COMPLIANCE"
)

// ObjectLockConfig represents object lock configuration.
type ObjectLockConfig struct {
	// Mode specifies the object lock mode (GOVERNANCE or COMPLIANCE)
	Mode ObjectLockMode

	// RetentionDays specifies how many days the object is locked
	RetentionDays int

	// LegalHold if true, prevents deletion indefinitely
	LegalHold bool

	// RetainUntilDate specifies the exact date until which the object is locked
	// If set, this takes precedence over RetentionDays
	RetainUntilDate *time.Time
}

// ImmutableBackupConfig represents immutable backup configuration.
type ImmutableBackupConfig struct {
	// Enabled determines if immutable backups are enabled
	Enabled bool

	// DefaultMode is the default object lock mode for new backups
	DefaultMode ObjectLockMode

	// DefaultRetentionDays is the default retention period
	DefaultRetentionDays int

	// AllowBypass determines if governance mode can be bypassed (for GOVERNANCE mode only)
	AllowBypass bool
}

// EnableObjectLock enables Object Lock on the S3 bucket
// This operation can only be done when creating a new bucket or on a bucket without versioning.
func (p *S3Provider) EnableObjectLock(ctx context.Context) error {
	// Check if Object Lock is already enabled
	output, err := p.client.GetObjectLockConfiguration(ctx, &s3.GetObjectLockConfigurationInput{
		Bucket: aws.String(p.config.Bucket),
	})

	if err == nil && output.ObjectLockConfiguration != nil &&
		output.ObjectLockConfiguration.ObjectLockEnabled == types.ObjectLockEnabledEnabled {
		return nil // Already enabled
	}

	// Enable Object Lock
	_, err = p.client.PutObjectLockConfiguration(ctx, &s3.PutObjectLockConfigurationInput{
		Bucket: aws.String(p.config.Bucket),
		ObjectLockConfiguration: &types.ObjectLockConfiguration{
			ObjectLockEnabled: types.ObjectLockEnabledEnabled,
		},
	})
	if err != nil {
		return pkgErrors.Wrap(err, pkgErrors.ErrorTypeStorage,
			"failed to enable S3 Object Lock")
	}

	return nil
}

// SetDefaultRetention sets the default retention policy for the bucket.
func (p *S3Provider) SetDefaultRetention(ctx context.Context, mode ObjectLockMode, days int) error {
	s3Mode := types.ObjectLockRetentionModeGovernance
	if mode == ObjectLockModeCompliance {
		s3Mode = types.ObjectLockRetentionModeCompliance
	}

	_, err := p.client.PutObjectLockConfiguration(ctx, &s3.PutObjectLockConfigurationInput{
		Bucket: aws.String(p.config.Bucket),
		ObjectLockConfiguration: &types.ObjectLockConfiguration{
			ObjectLockEnabled: types.ObjectLockEnabledEnabled,
			Rule: &types.ObjectLockRule{
				DefaultRetention: &types.DefaultRetention{
					Mode: s3Mode,
					Days: aws.Int32(int32(days)),
				},
			},
		},
	})
	if err != nil {
		return pkgErrors.Wrap(err, pkgErrors.ErrorTypeStorage,
			"failed to set default retention policy")
	}

	return nil
}

// UploadImmutable uploads a file with Object Lock protection.
func (p *S3Provider) UploadImmutable(ctx context.Context, localPath, remotePath string,
	lockConfig *ObjectLockConfig,
) error {
	// Calculate retention date
	var retainUntilDate time.Time
	if lockConfig.RetainUntilDate != nil {
		retainUntilDate = *lockConfig.RetainUntilDate
	} else {
		retainUntilDate = time.Now().AddDate(0, 0, lockConfig.RetentionDays)
	}

	// Open file
	file, err := p.openFile(localPath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Prepare S3 mode
	s3Mode := types.ObjectLockRetentionModeGovernance
	if lockConfig.Mode == ObjectLockModeCompliance {
		s3Mode = types.ObjectLockRetentionModeCompliance
	}

	// Upload with Object Lock
	input := &s3.PutObjectInput{
		Bucket:                    aws.String(p.config.Bucket),
		Key:                       aws.String(remotePath),
		Body:                      file,
		ObjectLockMode:            types.ObjectLockMode(s3Mode),
		ObjectLockRetainUntilDate: aws.Time(retainUntilDate),
	}

	// Set legal hold if requested
	if lockConfig.LegalHold {
		input.ObjectLockLegalHoldStatus = types.ObjectLockLegalHoldStatusOn
	}

	_, err = p.uploader.Upload(ctx, input)
	if err != nil {
		return pkgErrors.Wrap(err, pkgErrors.ErrorTypeStorage,
			"failed to upload immutable backup")
	}

	return nil
}

// GetObjectRetention retrieves the retention configuration for an object.
func (p *S3Provider) GetObjectRetention(ctx context.Context, remotePath string) (*ObjectLockConfig, error) {
	output, err := p.client.GetObjectRetention(ctx, &s3.GetObjectRetentionInput{
		Bucket: aws.String(p.config.Bucket),
		Key:    aws.String(remotePath),
	})
	if err != nil {
		return nil, pkgErrors.Wrap(err, pkgErrors.ErrorTypeStorage,
			"failed to get object retention")
	}

	if output.Retention == nil {
		return nil, nil
	}

	config := &ObjectLockConfig{
		Mode:            ObjectLockModeGovernance,
		RetainUntilDate: output.Retention.RetainUntilDate,
	}

	if output.Retention.Mode == types.ObjectLockRetentionModeCompliance {
		config.Mode = ObjectLockModeCompliance
	}

	// Calculate days until retention expires
	if config.RetainUntilDate != nil {
		duration := time.Until(*config.RetainUntilDate)
		config.RetentionDays = int(duration.Hours() / 24)
	}

	return config, nil
}

// GetLegalHoldStatus retrieves the legal hold status for an object.
func (p *S3Provider) GetLegalHoldStatus(ctx context.Context, remotePath string) (bool, error) {
	output, err := p.client.GetObjectLegalHold(ctx, &s3.GetObjectLegalHoldInput{
		Bucket: aws.String(p.config.Bucket),
		Key:    aws.String(remotePath),
	})
	if err != nil {
		return false, pkgErrors.Wrap(err, pkgErrors.ErrorTypeStorage,
			"failed to get legal hold status")
	}

	return output.LegalHold != nil &&
		output.LegalHold.Status == types.ObjectLockLegalHoldStatusOn, nil
}

// SetLegalHold sets or removes legal hold on an object.
func (p *S3Provider) SetLegalHold(ctx context.Context, remotePath string, enabled bool) error {
	status := types.ObjectLockLegalHoldStatusOff
	if enabled {
		status = types.ObjectLockLegalHoldStatusOn
	}

	_, err := p.client.PutObjectLegalHold(ctx, &s3.PutObjectLegalHoldInput{
		Bucket: aws.String(p.config.Bucket),
		Key:    aws.String(remotePath),
		LegalHold: &types.ObjectLockLegalHold{
			Status: status,
		},
	})
	if err != nil {
		return pkgErrors.Wrap(err, pkgErrors.ErrorTypeStorage,
			"failed to set legal hold")
	}

	return nil
}

// ExtendRetention extends the retention period for an object.
func (p *S3Provider) ExtendRetention(ctx context.Context, remotePath string,
	newRetentionDays int,
) error {
	// Get current retention
	current, err := p.GetObjectRetention(ctx, remotePath)
	if err != nil {
		return err
	}

	if current == nil {
		return pkgErrors.New(pkgErrors.ErrorTypeValidation,
			"object does not have retention set")
	}

	// Calculate new retention date
	newRetainUntil := time.Now().AddDate(0, 0, newRetentionDays)

	// Can only extend, not reduce
	if current.RetainUntilDate != nil && newRetainUntil.Before(*current.RetainUntilDate) {
		return pkgErrors.New(pkgErrors.ErrorTypeValidation,
			"cannot reduce retention period")
	}

	// Set new retention
	s3Mode := types.ObjectLockRetentionModeGovernance
	if current.Mode == ObjectLockModeCompliance {
		s3Mode = types.ObjectLockRetentionModeCompliance
	}

	_, err = p.client.PutObjectRetention(ctx, &s3.PutObjectRetentionInput{
		Bucket: aws.String(p.config.Bucket),
		Key:    aws.String(remotePath),
		Retention: &types.ObjectLockRetention{
			Mode:            s3Mode,
			RetainUntilDate: aws.Time(newRetainUntil),
		},
	})
	if err != nil {
		return pkgErrors.Wrap(err, pkgErrors.ErrorTypeStorage,
			"failed to extend retention")
	}

	return nil
}

// BypassGovernanceDelete deletes an object in GOVERNANCE mode with bypass permissions.
func (p *S3Provider) BypassGovernanceDelete(ctx context.Context, remotePath string) error {
	_, err := p.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket:                    aws.String(p.config.Bucket),
		Key:                       aws.String(remotePath),
		BypassGovernanceRetention: aws.Bool(true),
	})
	if err != nil {
		return pkgErrors.Wrap(err, pkgErrors.ErrorTypeStorage,
			"failed to bypass governance and delete object")
	}

	return nil
}

// ListImmutableBackups lists all backups with object lock configuration.
func (p *S3Provider) ListImmutableBackups(ctx context.Context, prefix string) ([]ImmutableBackupInfo, error) {
	var backups []ImmutableBackupInfo

	// List objects with the prefix
	paginator := s3.NewListObjectsV2Paginator(p.client, &s3.ListObjectsV2Input{
		Bucket: aws.String(p.config.Bucket),
		Prefix: aws.String(prefix),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, pkgErrors.Wrap(err, pkgErrors.ErrorTypeStorage,
				"failed to list objects")
		}

		for _, obj := range page.Contents {
			if obj.Key == nil {
				continue
			}

			info := ImmutableBackupInfo{
				Key:          *obj.Key,
				Size:         obj.Size,
				LastModified: obj.LastModified,
			}

			// Get retention info
			retention, err := p.GetObjectRetention(ctx, *obj.Key)
			if err == nil && retention != nil {
				info.RetentionMode = string(retention.Mode)
				info.RetainUntilDate = retention.RetainUntilDate
			}

			// Get legal hold status
			legalHold, err := p.GetLegalHoldStatus(ctx, *obj.Key)
			if err == nil {
				info.LegalHold = legalHold
			}

			backups = append(backups, info)
		}
	}

	return backups, nil
}

// ImmutableBackupInfo represents information about an immutable backup.
type ImmutableBackupInfo struct {
	Key             string
	Size            *int64
	LastModified    *time.Time
	RetentionMode   string
	RetainUntilDate *time.Time
	LegalHold       bool
}

// IsProtected returns true if the backup is currently protected.
func (i *ImmutableBackupInfo) IsProtected() bool {
	if i.LegalHold {
		return true
	}

	if i.RetainUntilDate != nil && time.Now().Before(*i.RetainUntilDate) {
		return true
	}

	return false
}

// DaysUntilUnlock returns the number of days until the backup can be deleted.
func (i *ImmutableBackupInfo) DaysUntilUnlock() int {
	if i.LegalHold {
		return -1 // Indefinite
	}

	if i.RetainUntilDate == nil {
		return 0 // Not locked
	}

	duration := time.Until(*i.RetainUntilDate)
	days := int(duration.Hours() / 24)

	if days < 0 {
		return 0
	}

	return days
}

// openFile is a helper function to open a file.
func (p *S3Provider) openFile(path string) (*os.File, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, pkgErrors.Wrap(err, pkgErrors.ErrorTypeStorage,
			fmt.Sprintf("failed to open file: %s", path))
	}
	return file, nil
}
