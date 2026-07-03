// Package azure provides Azure Blob Storage with immutable backup support
package azure

import (
	"context"
	"fmt"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/bloberror"

	st "github.com/sanskarpan/db-backup/internal/storage"
	pkgErrors "github.com/sanskarpan/db-backup/pkg/errors"
)

// SetRetention locks the blob until the given time using the given mode,
// satisfying storage.ImmutableProvider. GOVERNANCE maps to Azure's Unlocked
// (modifiable) policy and COMPLIANCE to Azure's Locked policy. It is a thin
// adapter over SetBlobImmutabilityPolicy.
func (p *AzureProvider) SetRetention(ctx context.Context, remotePath string, until time.Time, mode string) error {
	policyMode := ImmutabilityPolicyModeUnlocked
	if mode == st.LockModeCompliance {
		policyMode = ImmutabilityPolicyModeLocked
	}
	return p.SetBlobImmutabilityPolicy(ctx, remotePath, until, policyMode)
}

// GetRetention returns the retention expiry and mode for the blob, satisfying
// storage.ImmutableProvider. Azure's Locked policy maps to COMPLIANCE and
// Unlocked to GOVERNANCE. It returns storage.ErrNoRetention when the blob has
// no immutability policy.
func (p *AzureProvider) GetRetention(ctx context.Context, remotePath string) (time.Time, string, error) {
	info, err := p.GetBlobImmutabilityPolicy(ctx, remotePath)
	if err != nil {
		return time.Time{}, "", err
	}
	if info.ImmutabilityExpiresOn == nil {
		return time.Time{}, "", st.ErrNoRetention
	}

	mode := st.LockModeGovernance
	if info.ImmutabilityPolicyMode == string(ImmutabilityPolicyModeLocked) {
		mode = st.LockModeCompliance
	}
	return *info.ImmutabilityExpiresOn, mode, nil
}

// SetLegalHold turns a legal hold on or off for the blob, satisfying
// storage.ImmutableProvider. It is a thin adapter over SetBlobLegalHold.
func (p *AzureProvider) SetLegalHold(ctx context.Context, remotePath string, on bool) error {
	return p.SetBlobLegalHold(ctx, remotePath, on)
}

// GetLegalHold reports whether a legal hold is on for the blob, satisfying
// storage.ImmutableProvider. It is a thin adapter over GetBlobLegalHold.
func (p *AzureProvider) GetLegalHold(ctx context.Context, remotePath string) (bool, error) {
	return p.GetBlobLegalHold(ctx, remotePath)
}

// ImmutabilityPolicyMode defines the immutability policy mode.
type ImmutabilityPolicyMode string

const (
	// ImmutabilityPolicyModeUnlocked allows policy modification.
	ImmutabilityPolicyModeUnlocked ImmutabilityPolicyMode = "Unlocked"
	// ImmutabilityPolicyModeLocked prevents policy modification.
	ImmutabilityPolicyModeLocked ImmutabilityPolicyMode = "Locked"
)

// ImmutableBlobConfig represents Azure immutable blob configuration.
type ImmutableBlobConfig struct {
	// ImmutabilityPeriodDays specifies retention period in days
	ImmutabilityPeriodDays int

	// LegalHold if true, prevents deletion indefinitely
	LegalHold bool

	// Mode specifies the immutability policy mode
	Mode ImmutabilityPolicyMode

	// AllowProtectedAppendWrites allows append operations even when immutable
	AllowProtectedAppendWrites bool
}

// ContainerImmutabilityPolicy represents container-level immutability
// Note: Container-level policies require Azure Management API, not available in data plane SDK.
type ContainerImmutabilityPolicy struct {
	// ImmutabilityPeriodDays is the retention period
	ImmutabilityPeriodDays int

	// State is the policy state (Locked or Unlocked)
	State string

	// AllowProtectedAppendWrites allows append operations
	AllowProtectedAppendWrites bool

	// AllowProtectedAppendWritesAll allows all append writes
	AllowProtectedAppendWritesAll bool
}

// EnableVersioning enables blob versioning which is required for immutability.
func (p *AzureProvider) EnableVersioning(ctx context.Context) error {
	// Blob versioning is a prerequisite for immutability
	// This is typically done at the storage account level via Azure Portal or ARM templates
	// The data plane SDK doesn't provide direct versioning enablement
	// In production, you would use Azure Management API (azure-sdk-for-go/sdk/resourcemanager/storage)
	return pkgErrors.New(pkgErrors.ErrorTypeOperation,
		"versioning must be enabled at storage account level via Azure Management API or Portal")
}

// SetContainerImmutabilityPolicy sets an immutability policy on the container.
func (p *AzureProvider) SetContainerImmutabilityPolicy(ctx context.Context,
	periodDays int, allowAppendWrites bool,
) error {
	// Container-level immutability policies require Azure Management API
	// Not available in the data plane SDK (azblob package)
	// Use Azure Management SDK: github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage
	return pkgErrors.New(pkgErrors.ErrorTypeOperation,
		"container-level policies require Azure Management API (use armstorage package)")
}

// LockContainerImmutabilityPolicy locks the immutability policy (irreversible).
func (p *AzureProvider) LockContainerImmutabilityPolicy(ctx context.Context) error {
	// WARNING: This operation would be IRREVERSIBLE
	// Container-level operations require Azure Management API
	return pkgErrors.New(pkgErrors.ErrorTypeOperation,
		"container-level policy locking requires Azure Management API")
}

// ExtendContainerImmutabilityPolicy extends the immutability period.
func (p *AzureProvider) ExtendContainerImmutabilityPolicy(ctx context.Context, newPeriodDays int) error {
	// Container-level operations require Azure Management API
	return pkgErrors.New(pkgErrors.ErrorTypeOperation,
		"container-level policy extension requires Azure Management API")
}

// DeleteContainerImmutabilityPolicy deletes an unlocked immutability policy.
func (p *AzureProvider) DeleteContainerImmutabilityPolicy(ctx context.Context) error {
	// Container-level operations require Azure Management API
	return pkgErrors.New(pkgErrors.ErrorTypeOperation,
		"container-level policy deletion requires Azure Management API")
}

// SetBlobLegalHold sets or removes legal hold on a blob.
func (p *AzureProvider) SetBlobLegalHold(ctx context.Context, blobName string, enabled bool) error {
	blobClient := p.container.NewBlobClient(blobName)

	_, err := blobClient.SetLegalHold(ctx, enabled, &blob.SetLegalHoldOptions{})
	if err != nil {
		return pkgErrors.Wrap(err, pkgErrors.ErrorTypeStorage,
			fmt.Sprintf("failed to set legal hold to %v", enabled))
	}

	return nil
}

// GetBlobLegalHold retrieves legal hold status for a blob.
func (p *AzureProvider) GetBlobLegalHold(ctx context.Context, blobName string) (bool, error) {
	blobClient := p.container.NewBlobClient(blobName)

	props, err := blobClient.GetProperties(ctx, nil)
	if err != nil {
		if bloberror.HasCode(err, bloberror.BlobNotFound) {
			return false, pkgErrors.New(pkgErrors.ErrorTypeNotFound,
				fmt.Sprintf("blob not found: %s", blobName))
		}
		return false, pkgErrors.Wrap(err, pkgErrors.ErrorTypeStorage,
			"failed to get blob properties")
	}

	// Check legal hold status from properties
	// Note: Legal hold status is returned in the properties
	if props.LegalHold != nil && *props.LegalHold {
		return true, nil
	}

	return false, nil
}

// SetBlobImmutabilityPolicy sets an immutability policy on a specific blob.
func (p *AzureProvider) SetBlobImmutabilityPolicy(ctx context.Context, blobName string,
	expiryTime time.Time, mode ImmutabilityPolicyMode,
) error {
	blobClient := p.container.NewBlobClient(blobName)

	// Convert our mode to Azure SDK mode
	var policySetting blob.ImmutabilityPolicySetting
	if mode == ImmutabilityPolicyModeLocked {
		policySetting = blob.ImmutabilityPolicySettingLocked
	} else {
		policySetting = blob.ImmutabilityPolicySettingUnlocked
	}

	_, err := blobClient.SetImmutabilityPolicy(ctx, expiryTime, &blob.SetImmutabilityPolicyOptions{
		Mode: &policySetting,
	})
	if err != nil {
		return pkgErrors.Wrap(err, pkgErrors.ErrorTypeStorage,
			"failed to set blob immutability policy")
	}

	return nil
}

// DeleteBlobImmutabilityPolicy deletes an unlocked blob immutability policy.
func (p *AzureProvider) DeleteBlobImmutabilityPolicy(ctx context.Context, blobName string) error {
	blobClient := p.container.NewBlobClient(blobName)

	_, err := blobClient.DeleteImmutabilityPolicy(ctx, nil)
	if err != nil {
		return pkgErrors.Wrap(err, pkgErrors.ErrorTypeStorage,
			"failed to delete blob immutability policy")
	}

	return nil
}

// GetBlobImmutabilityPolicy retrieves blob immutability policy.
func (p *AzureProvider) GetBlobImmutabilityPolicy(ctx context.Context, blobName string) (*AzureImmutableBlobInfo, error) {
	blobClient := p.container.NewBlobClient(blobName)

	props, err := blobClient.GetProperties(ctx, nil)
	if err != nil {
		if bloberror.HasCode(err, bloberror.BlobNotFound) {
			return nil, pkgErrors.New(pkgErrors.ErrorTypeNotFound,
				fmt.Sprintf("blob not found: %s", blobName))
		}
		return nil, pkgErrors.Wrap(err, pkgErrors.ErrorTypeStorage,
			"failed to get blob properties")
	}

	info := &AzureImmutableBlobInfo{
		BlobName:     blobName,
		LastModified: props.LastModified,
		ETag:         props.ETag,
	}

	// Legal hold status
	if props.LegalHold != nil {
		info.LegalHold = *props.LegalHold
	}

	// Immutability policy
	if props.ImmutabilityPolicyExpiresOn != nil {
		info.ImmutabilityExpiresOn = props.ImmutabilityPolicyExpiresOn
	}

	if props.ImmutabilityPolicyMode != nil {
		info.ImmutabilityPolicyMode = string(*props.ImmutabilityPolicyMode)
	}

	return info, nil
}

// ListImmutableBlobs lists all blobs with immutability information.
func (p *AzureProvider) ListImmutableBlobs(ctx context.Context, prefix string) ([]AzureImmutableBlobInfo, error) {
	// Use the List method from the base provider to get blobs
	files, err := p.List(ctx, prefix)
	if err != nil {
		return nil, err
	}

	blobs := make([]AzureImmutableBlobInfo, 0, len(files))

	// For each blob, get immutability info
	for _, file := range files {
		blobClient := p.container.NewBlobClient(file.Path)
		props, err := blobClient.GetProperties(ctx, nil)
		if err != nil {
			// Skip blobs we can't read
			continue
		}

		info := AzureImmutableBlobInfo{
			BlobName:     file.Path,
			LastModified: &file.LastModified,
		}

		// Legal hold
		if props.LegalHold != nil {
			info.LegalHold = *props.LegalHold
		}

		// Immutability policy
		if props.ImmutabilityPolicyExpiresOn != nil {
			info.ImmutabilityExpiresOn = props.ImmutabilityPolicyExpiresOn
		}

		if props.ImmutabilityPolicyMode != nil {
			info.ImmutabilityPolicyMode = string(*props.ImmutabilityPolicyMode)
		}

		// Set ETag
		info.ETag = props.ETag

		blobs = append(blobs, info)
	}

	return blobs, nil
}

// AzureImmutableBlobInfo represents information about an immutable blob.
//
//nolint:revive // AzureImmutableBlobInfo keeps the public API stable across packages
type AzureImmutableBlobInfo struct {
	BlobName               string
	LastModified           *time.Time
	ETag                   *azcore.ETag
	LegalHold              bool
	ImmutabilityExpiresOn  *time.Time
	ImmutabilityPolicyMode string
}

// IsProtected returns true if the blob is currently protected.
func (i *AzureImmutableBlobInfo) IsProtected() bool {
	// Protected by legal hold
	if i.LegalHold {
		return true
	}

	// Protected by immutability policy
	if i.ImmutabilityExpiresOn != nil && time.Now().Before(*i.ImmutabilityExpiresOn) {
		return true
	}

	return false
}

// DaysUntilUnlock returns the number of days until the blob can be deleted.
func (i *AzureImmutableBlobInfo) DaysUntilUnlock() int {
	if i.LegalHold {
		return -1 // Indefinite
	}

	if i.ImmutabilityExpiresOn == nil {
		return 0 // Not locked
	}

	duration := time.Until(*i.ImmutabilityExpiresOn)
	days := int(duration.Hours() / 24)

	if days < 0 {
		return 0
	}

	return days
}

// IsLocked returns true if the immutability policy is locked.
func (i *AzureImmutableBlobInfo) IsLocked() bool {
	return i.ImmutabilityPolicyMode == string(ImmutabilityPolicyModeLocked)
}
