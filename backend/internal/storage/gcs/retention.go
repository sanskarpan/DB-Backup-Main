// Package gcs provides Google Cloud Storage with retention policy support
package gcs

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/storage"
	pkgErrors "github.com/sanskarpan/db-backup/pkg/errors"
)

// RetentionMode defines the retention mode for objects
type RetentionMode string

const (
	// RetentionModeUnlocked allows policy modification
	RetentionModeUnlocked RetentionMode = "Unlocked"
	// RetentionModeLocked prevents policy modification
	RetentionModeLocked RetentionMode = "Locked"
)

// RetentionPolicyConfig represents GCS retention policy configuration
type RetentionPolicyConfig struct {
	// RetentionPeriod is the retention period in seconds
	RetentionPeriod time.Duration

	// EffectiveTime is when the policy becomes effective
	EffectiveTime time.Time

	// IsLocked indicates if the policy is locked (cannot be removed/reduced)
	IsLocked bool
}

// ObjectRetentionConfig represents object-level retention configuration
type ObjectRetentionConfig struct {
	// Mode is the retention mode (Unlocked/Locked)
	Mode RetentionMode

	// RetainUntilTime is the time until which the object is retained
	RetainUntilTime time.Time
}

// ObjectHoldConfig represents object hold configuration
type ObjectHoldConfig struct {
	// EventBasedHold prevents deletion until explicitly removed
	EventBasedHold bool

	// TemporaryHold prevents deletion until explicitly removed
	TemporaryHold bool
}

// SetBucketRetentionPolicy sets a retention policy on the bucket
func (p *GCSProvider) SetBucketRetentionPolicy(ctx context.Context, retentionPeriod time.Duration) error {
	// Get current bucket attributes
	attrs, err := p.bucket.Attrs(ctx)
	if err != nil {
		return pkgErrors.Wrap(err, pkgErrors.ErrorTypeStorage,
			"failed to get bucket attributes")
	}

	// Check if already locked
	if attrs.RetentionPolicy != nil && attrs.RetentionPolicy.IsLocked {
		return pkgErrors.New(pkgErrors.ErrorTypeValidation,
			"cannot modify locked retention policy")
	}

	// Set retention policy
	bucketUpdate := storage.BucketAttrsToUpdate{
		RetentionPolicy: &storage.RetentionPolicy{
			RetentionPeriod: retentionPeriod,
		},
	}

	if _, err := p.bucket.Update(ctx, bucketUpdate); err != nil {
		return pkgErrors.Wrap(err, pkgErrors.ErrorTypeStorage,
			"failed to set bucket retention policy")
	}

	return nil
}

// LockBucketRetentionPolicy locks the retention policy (irreversible)
func (p *GCSProvider) LockBucketRetentionPolicy(ctx context.Context) error {
	// WARNING: This operation is IRREVERSIBLE
	// Once locked, the retention policy cannot be removed or reduced

	// Get current attributes to get the metageneration
	attrs, err := p.bucket.Attrs(ctx)
	if err != nil {
		return pkgErrors.Wrap(err, pkgErrors.ErrorTypeStorage,
			"failed to get bucket attributes")
	}

	if attrs.RetentionPolicy == nil {
		return pkgErrors.New(pkgErrors.ErrorTypeValidation,
			"no retention policy to lock")
	}

	if attrs.RetentionPolicy.IsLocked {
		return nil // Already locked
	}

	// Lock the policy
	if err := p.bucket.If(storage.BucketConditions{MetagenerationMatch: attrs.MetaGeneration}).
		LockRetentionPolicy(ctx); err != nil {
		return pkgErrors.Wrap(err, pkgErrors.ErrorTypeStorage,
			"failed to lock retention policy")
	}

	return nil
}

// RemoveBucketRetentionPolicy removes an unlocked retention policy
func (p *GCSProvider) RemoveBucketRetentionPolicy(ctx context.Context) error {
	// Get current attributes
	attrs, err := p.bucket.Attrs(ctx)
	if err != nil {
		return pkgErrors.Wrap(err, pkgErrors.ErrorTypeStorage,
			"failed to get bucket attributes")
	}

	// Check if locked
	if attrs.RetentionPolicy != nil && attrs.RetentionPolicy.IsLocked {
		return pkgErrors.New(pkgErrors.ErrorTypeValidation,
			"cannot remove locked retention policy")
	}

	// Remove retention policy
	bucketUpdate := storage.BucketAttrsToUpdate{
		RetentionPolicy: &storage.RetentionPolicy{
			RetentionPeriod: 0, // Setting to 0 removes the policy
		},
	}

	if _, err := p.bucket.Update(ctx, bucketUpdate); err != nil {
		return pkgErrors.Wrap(err, pkgErrors.ErrorTypeStorage,
			"failed to remove retention policy")
	}

	return nil
}

// GetBucketRetentionPolicy retrieves the bucket retention policy
func (p *GCSProvider) GetBucketRetentionPolicy(ctx context.Context) (*RetentionPolicyConfig, error) {
	attrs, err := p.bucket.Attrs(ctx)
	if err != nil {
		return nil, pkgErrors.Wrap(err, pkgErrors.ErrorTypeStorage,
			"failed to get bucket attributes")
	}

	if attrs.RetentionPolicy == nil {
		return nil, nil // No retention policy
	}

	return &RetentionPolicyConfig{
		RetentionPeriod: attrs.RetentionPolicy.RetentionPeriod,
		EffectiveTime:   attrs.RetentionPolicy.EffectiveTime,
		IsLocked:        attrs.RetentionPolicy.IsLocked,
	}, nil
}

// SetObjectRetention sets retention on a specific object
func (p *GCSProvider) SetObjectRetention(ctx context.Context, objectName string,
	retainUntilTime time.Time, mode RetentionMode) error {

	obj := p.bucket.Object(objectName)

	// Object retention in GCS is set via ObjectAttrsToUpdate
	// This requires object versioning to be enabled
	update := storage.ObjectAttrsToUpdate{
		Retention: &storage.ObjectRetention{
			Mode:        string(mode),
			RetainUntil: retainUntilTime,
		},
	}

	if _, err := obj.Update(ctx, update); err != nil {
		return pkgErrors.Wrap(err, pkgErrors.ErrorTypeStorage,
			fmt.Sprintf("failed to set retention on object: %s", objectName))
	}

	return nil
}

// RemoveObjectRetention removes object retention (only if unlocked)
func (p *GCSProvider) RemoveObjectRetention(ctx context.Context, objectName string) error {
	obj := p.bucket.Object(objectName)

	// Get current attributes to check if locked
	attrs, err := obj.Attrs(ctx)
	if err != nil {
		if err == storage.ErrObjectNotExist {
			return pkgErrors.New(pkgErrors.ErrorTypeNotFound,
				fmt.Sprintf("object not found: %s", objectName))
		}
		return pkgErrors.Wrap(err, pkgErrors.ErrorTypeStorage,
			"failed to get object attributes")
	}

	// Check if retention is locked
	if attrs.Retention != nil && attrs.Retention.Mode == "Locked" {
		return pkgErrors.New(pkgErrors.ErrorTypeValidation,
			"cannot remove locked retention policy")
	}

	// Remove retention
	update := storage.ObjectAttrsToUpdate{
		Retention: &storage.ObjectRetention{},
	}

	if _, err := obj.Update(ctx, update); err != nil {
		return pkgErrors.Wrap(err, pkgErrors.ErrorTypeStorage,
			"failed to remove object retention")
	}

	return nil
}

// SetObjectHold sets event-based hold and/or temporary hold on an object
func (p *GCSProvider) SetObjectHold(ctx context.Context, objectName string, holdConfig *ObjectHoldConfig) error {
	obj := p.bucket.Object(objectName)

	update := storage.ObjectAttrsToUpdate{}

	if holdConfig.EventBasedHold {
		update.EventBasedHold = true
	} else {
		update.EventBasedHold = false
	}

	if holdConfig.TemporaryHold {
		update.TemporaryHold = true
	} else {
		update.TemporaryHold = false
	}

	if _, err := obj.Update(ctx, update); err != nil {
		return pkgErrors.Wrap(err, pkgErrors.ErrorTypeStorage,
			fmt.Sprintf("failed to set hold on object: %s", objectName))
	}

	return nil
}

// GetObjectRetention retrieves object retention and hold information
func (p *GCSProvider) GetObjectRetention(ctx context.Context, objectName string) (*GCSObjectRetentionInfo, error) {
	obj := p.bucket.Object(objectName)

	attrs, err := obj.Attrs(ctx)
	if err != nil {
		if err == storage.ErrObjectNotExist {
			return nil, pkgErrors.New(pkgErrors.ErrorTypeNotFound,
				fmt.Sprintf("object not found: %s", objectName))
		}
		return nil, pkgErrors.Wrap(err, pkgErrors.ErrorTypeStorage,
			"failed to get object attributes")
	}

	info := &GCSObjectRetentionInfo{
		ObjectName:      objectName,
		Size:            attrs.Size,
		Created:         attrs.Created,
		Updated:         attrs.Updated,
		EventBasedHold:  attrs.EventBasedHold,
		TemporaryHold:   attrs.TemporaryHold,
		StorageClass:    attrs.StorageClass,
		ContentType:     attrs.ContentType,
	}

	// Set retention info if available
	if attrs.Retention != nil {
		info.RetentionMode = string(attrs.Retention.Mode)
		info.RetainUntilTime = &attrs.Retention.RetainUntil
	}

	// Check if bucket has default retention
	bucketAttrs, err := p.bucket.Attrs(ctx)
	if err == nil && bucketAttrs.RetentionPolicy != nil {
		info.BucketRetentionPeriod = bucketAttrs.RetentionPolicy.RetentionPeriod
		info.BucketRetentionLocked = bucketAttrs.RetentionPolicy.IsLocked
	}

	return info, nil
}

// ListObjectsWithRetention lists all objects with retention information
func (p *GCSProvider) ListObjectsWithRetention(ctx context.Context, prefix string) ([]GCSObjectRetentionInfo, error) {
	var objects []GCSObjectRetentionInfo

	query := &storage.Query{Prefix: prefix}
	it := p.bucket.Objects(ctx, query)

	for {
		attrs, err := it.Next()
		if err != nil {
			if err.Error() == "no more items in iterator" {
				break
			}
			return nil, pkgErrors.Wrap(err, pkgErrors.ErrorTypeStorage,
				"failed to list objects")
		}

		// Skip directories
		if attrs.Name == "" || attrs.Name[len(attrs.Name)-1] == '/' {
			continue
		}

		info := GCSObjectRetentionInfo{
			ObjectName:     attrs.Name,
			Size:           attrs.Size,
			Created:        attrs.Created,
			Updated:        attrs.Updated,
			EventBasedHold: attrs.EventBasedHold,
			TemporaryHold:  attrs.TemporaryHold,
			StorageClass:   attrs.StorageClass,
			ContentType:    attrs.ContentType,
		}

		// Set retention info if available
		if attrs.Retention != nil {
			info.RetentionMode = string(attrs.Retention.Mode)
			info.RetainUntilTime = &attrs.Retention.RetainUntil
		}

		objects = append(objects, info)
	}

	// Get bucket retention info
	bucketAttrs, err := p.bucket.Attrs(ctx)
	if err == nil && bucketAttrs.RetentionPolicy != nil {
		// Update all objects with bucket retention info
		for i := range objects {
			objects[i].BucketRetentionPeriod = bucketAttrs.RetentionPolicy.RetentionPeriod
			objects[i].BucketRetentionLocked = bucketAttrs.RetentionPolicy.IsLocked
		}
	}

	return objects, nil
}

// GCSObjectRetentionInfo represents information about an object with retention
type GCSObjectRetentionInfo struct {
	ObjectName              string
	Size                    int64
	Created                 time.Time
	Updated                 time.Time
	EventBasedHold          bool
	TemporaryHold           bool
	RetentionMode           string
	RetainUntilTime         *time.Time
	BucketRetentionPeriod   time.Duration
	BucketRetentionLocked   bool
	StorageClass            string
	ContentType             string
}

// IsProtected returns true if the object is currently protected from deletion
func (i *GCSObjectRetentionInfo) IsProtected() bool {
	// Protected by event-based hold
	if i.EventBasedHold {
		return true
	}

	// Protected by temporary hold
	if i.TemporaryHold {
		return true
	}

	// Protected by object retention
	if i.RetainUntilTime != nil && time.Now().Before(*i.RetainUntilTime) {
		return true
	}

	// Protected by bucket retention policy
	if i.BucketRetentionPeriod > 0 {
		// Object is protected if it was created within the retention period
		retentionExpiry := i.Created.Add(i.BucketRetentionPeriod)
		if time.Now().Before(retentionExpiry) {
			return true
		}
	}

	return false
}

// DaysUntilUnlock returns the number of days until the object can be deleted
func (i *GCSObjectRetentionInfo) DaysUntilUnlock() int {
	// Event-based hold or temporary hold = indefinite
	if i.EventBasedHold || i.TemporaryHold {
		return -1
	}

	var unlockTime time.Time

	// Check object retention
	if i.RetainUntilTime != nil {
		unlockTime = *i.RetainUntilTime
	}

	// Check bucket retention
	if i.BucketRetentionPeriod > 0 {
		bucketUnlock := i.Created.Add(i.BucketRetentionPeriod)
		if unlockTime.IsZero() || bucketUnlock.After(unlockTime) {
			unlockTime = bucketUnlock
		}
	}

	if unlockTime.IsZero() {
		return 0 // Not locked
	}

	duration := time.Until(unlockTime)
	days := int(duration.Hours() / 24)

	if days < 0 {
		return 0
	}

	return days
}

// IsLocked returns true if the object has a locked retention policy
func (i *GCSObjectRetentionInfo) IsLocked() bool {
	return i.RetentionMode == string(RetentionModeLocked) || i.BucketRetentionLocked
}
