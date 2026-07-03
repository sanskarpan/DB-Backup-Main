package residency

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/sanskarpan/db-backup/pkg/uid"
)

// Region represents a geographic region for data residency.
type Region string

const (
	RegionUSEast        Region = "us-east"
	RegionUSWest        Region = "us-west"
	RegionEUWest        Region = "eu-west"
	RegionEUCentral     Region = "eu-central"
	RegionAPSouth       Region = "ap-south"
	RegionAPNortheast   Region = "ap-northeast"
	RegionAPSoutheast   Region = "ap-southeast"
	RegionCanadaCentral Region = "canada-central"
	RegionSouthAmerica  Region = "south-america"
	RegionMiddleEast    Region = "middle-east"
)

// DataClassification represents the sensitivity level of data.
type DataClassification string

const (
	ClassificationPublic       DataClassification = "public"
	ClassificationInternal     DataClassification = "internal"
	ClassificationConfidential DataClassification = "confidential"
	ClassificationRestricted   DataClassification = "restricted"
)

// ResidencyPolicy defines where data can be stored and processed.
type ResidencyPolicy struct {
	ID                   string             `json:"id"`
	Name                 string             `json:"name"`
	Description          string             `json:"description"`
	PrimaryRegion        Region             `json:"primary_region"`
	AllowedRegions       []Region           `json:"allowed_regions"`
	RestrictedRegions    []Region           `json:"restricted_regions"`
	DataClassification   DataClassification `json:"data_classification"`
	CrossBorderTransfer  bool               `json:"cross_border_transfer"`
	EncryptionRequired   bool               `json:"encryption_required"`
	RequiresApproval     bool               `json:"requires_approval"`
	ComplianceFrameworks []string           `json:"compliance_frameworks"` // GDPR, CCPA, HIPAA, etc.
	CreatedAt            time.Time          `json:"created_at"`
	UpdatedAt            time.Time          `json:"updated_at"`
	Active               bool               `json:"active"`
}

// ResidencyViolation represents a data residency policy violation.
type ResidencyViolation struct {
	ID                string     `json:"id"`
	PolicyID          string     `json:"policy_id"`
	ResourceID        string     `json:"resource_id"`
	ResourceType      string     `json:"resource_type"` // backup, database, logs
	ViolationType     string     `json:"violation_type"`
	ExpectedRegion    Region     `json:"expected_region"`
	ActualRegion      Region     `json:"actual_region"`
	Severity          string     `json:"severity"` // low, medium, high, critical
	DetectedAt        time.Time  `json:"detected_at"`
	ResolvedAt        *time.Time `json:"resolved_at,omitempty"`
	Status            string     `json:"status"` // detected, investigating, resolved, false_positive
	RemediationAction string     `json:"remediation_action,omitempty"`
}

// TransferRequest represents a cross-border data transfer request.
type TransferRequest struct {
	ID                string             `json:"id"`
	ResourceID        string             `json:"resource_id"`
	ResourceType      string             `json:"resource_type"`
	SourceRegion      Region             `json:"source_region"`
	DestinationRegion Region             `json:"destination_region"`
	RequestedBy       string             `json:"requested_by"`
	RequestedAt       time.Time          `json:"requested_at"`
	Status            TransferStatus     `json:"status"`
	ApprovedBy        string             `json:"approved_by,omitempty"`
	ApprovedAt        *time.Time         `json:"approved_at,omitempty"`
	RejectedReason    string             `json:"rejected_reason,omitempty"`
	LegalBasis        string             `json:"legal_basis"`
	DataCategory      DataClassification `json:"data_category"`
	EstimatedSize     int64              `json:"estimated_size"`
}

// TransferStatus represents the status of a transfer request.
type TransferStatus string

const (
	TransferStatusPending   TransferStatus = "pending"
	TransferStatusApproved  TransferStatus = "approved"
	TransferStatusRejected  TransferStatus = "rejected"
	TransferStatusCompleted TransferStatus = "completed"
	TransferStatusFailed    TransferStatus = "failed"
)

// ResidencyManager manages data residency policies and compliance.
type ResidencyManager struct {
	policyStore    PolicyStore
	violationStore ViolationStore
	transferStore  TransferStore
	regionProvider RegionProvider
}

// PolicyStore defines the interface for storing residency policies.
type PolicyStore interface {
	Save(ctx context.Context, policy *ResidencyPolicy) error
	Get(ctx context.Context, policyID string) (*ResidencyPolicy, error)
	GetAll(ctx context.Context) ([]*ResidencyPolicy, error)
	GetByRegion(ctx context.Context, region Region) ([]*ResidencyPolicy, error)
	Update(ctx context.Context, policy *ResidencyPolicy) error
	Delete(ctx context.Context, policyID string) error
}

// ViolationStore defines the interface for storing violations.
type ViolationStore interface {
	Save(ctx context.Context, violation *ResidencyViolation) error
	Get(ctx context.Context, violationID string) (*ResidencyViolation, error)
	GetByPolicy(ctx context.Context, policyID string) ([]*ResidencyViolation, error)
	GetUnresolved(ctx context.Context) ([]*ResidencyViolation, error)
	Update(ctx context.Context, violation *ResidencyViolation) error
}

// TransferStore defines the interface for storing transfer requests.
type TransferStore interface {
	Save(ctx context.Context, request *TransferRequest) error
	Get(ctx context.Context, requestID string) (*TransferRequest, error)
	GetPending(ctx context.Context) ([]*TransferRequest, error)
	Update(ctx context.Context, request *TransferRequest) error
}

// RegionProvider defines the interface for determining resource regions.
type RegionProvider interface {
	GetResourceRegion(ctx context.Context, resourceType, resourceID string) (Region, error)
}

// NewResidencyManager creates a new residency manager.
func NewResidencyManager(policyStore PolicyStore, violationStore ViolationStore, transferStore TransferStore, regionProvider RegionProvider) *ResidencyManager {
	return &ResidencyManager{
		policyStore:    policyStore,
		violationStore: violationStore,
		transferStore:  transferStore,
		regionProvider: regionProvider,
	}
}

// CreatePolicy creates a new data residency policy.
func (rm *ResidencyManager) CreatePolicy(ctx context.Context, policy *ResidencyPolicy) error {
	if policy.Name == "" {
		return errors.New("policy name is required")
	}

	if policy.PrimaryRegion == "" {
		return errors.New("primary region is required")
	}

	policy.ID = generateID()
	policy.CreatedAt = time.Now()
	policy.UpdatedAt = time.Now()
	policy.Active = true

	return rm.policyStore.Save(ctx, policy)
}

// UpdatePolicy updates an existing policy.
func (rm *ResidencyManager) UpdatePolicy(ctx context.Context, policy *ResidencyPolicy) error {
	existing, err := rm.policyStore.Get(ctx, policy.ID)
	if err != nil {
		return err
	}

	policy.CreatedAt = existing.CreatedAt
	policy.UpdatedAt = time.Now()

	return rm.policyStore.Update(ctx, policy)
}

// GetPolicy retrieves a policy.
func (rm *ResidencyManager) GetPolicy(ctx context.Context, policyID string) (*ResidencyPolicy, error) {
	return rm.policyStore.Get(ctx, policyID)
}

// GetAllPolicies retrieves all policies.
func (rm *ResidencyManager) GetAllPolicies(ctx context.Context) ([]*ResidencyPolicy, error) {
	return rm.policyStore.GetAll(ctx)
}

// ValidateResidency checks if a resource complies with residency policies.
func (rm *ResidencyManager) ValidateResidency(ctx context.Context, resourceType, resourceID string, policyID string) (bool, error) {
	policy, err := rm.policyStore.Get(ctx, policyID)
	if err != nil {
		return false, err
	}

	if !policy.Active {
		return false, errors.New("policy is not active")
	}

	// Get the current region of the resource
	currentRegion, err := rm.regionProvider.GetResourceRegion(ctx, resourceType, resourceID)
	if err != nil {
		return false, err
	}

	// Check if the region is restricted
	for _, restricted := range policy.RestrictedRegions {
		if currentRegion == restricted {
			// Log violation
			violation := &ResidencyViolation{
				ID:             generateID(),
				PolicyID:       policyID,
				ResourceID:     resourceID,
				ResourceType:   resourceType,
				ViolationType:  "restricted_region",
				ExpectedRegion: policy.PrimaryRegion,
				ActualRegion:   currentRegion,
				Severity:       "high",
				DetectedAt:     time.Now(),
				Status:         "detected",
			}
			_ = rm.violationStore.Save(ctx, violation)
			return false, fmt.Errorf("resource in restricted region: %s", currentRegion)
		}
	}

	// Check if the region is in allowed regions (if specified)
	if len(policy.AllowedRegions) > 0 {
		allowed := false
		for _, allowedRegion := range policy.AllowedRegions {
			if currentRegion == allowedRegion {
				allowed = true
				break
			}
		}

		if !allowed {
			violation := &ResidencyViolation{
				ID:             generateID(),
				PolicyID:       policyID,
				ResourceID:     resourceID,
				ResourceType:   resourceType,
				ViolationType:  "disallowed_region",
				ExpectedRegion: policy.PrimaryRegion,
				ActualRegion:   currentRegion,
				Severity:       "medium",
				DetectedAt:     time.Now(),
				Status:         "detected",
			}
			_ = rm.violationStore.Save(ctx, violation)
			return false, fmt.Errorf("resource not in allowed region: %s", currentRegion)
		}
	}

	return true, nil
}

// RequestTransfer creates a cross-border data transfer request.
func (rm *ResidencyManager) RequestTransfer(ctx context.Context, resourceType, resourceID, requestedBy string, sourceRegion, destinationRegion Region, legalBasis string, dataCategory DataClassification) (*TransferRequest, error) {
	if resourceID == "" || requestedBy == "" {
		return nil, errors.New("resource ID and requester are required")
	}

	if sourceRegion == destinationRegion {
		return nil, errors.New("source and destination regions must be different")
	}

	request := &TransferRequest{
		ID:                generateID(),
		ResourceID:        resourceID,
		ResourceType:      resourceType,
		SourceRegion:      sourceRegion,
		DestinationRegion: destinationRegion,
		RequestedBy:       requestedBy,
		RequestedAt:       time.Now(),
		Status:            TransferStatusPending,
		LegalBasis:        legalBasis,
		DataCategory:      dataCategory,
	}

	if err := rm.transferStore.Save(ctx, request); err != nil {
		return nil, err
	}

	return request, nil
}

// ApproveTransfer approves a cross-border transfer request.
func (rm *ResidencyManager) ApproveTransfer(ctx context.Context, requestID, approvedBy string) error {
	request, err := rm.transferStore.Get(ctx, requestID)
	if err != nil {
		return err
	}

	if request.Status != TransferStatusPending {
		return fmt.Errorf("transfer request not in pending status: %s", request.Status)
	}

	request.Status = TransferStatusApproved
	request.ApprovedBy = approvedBy
	now := time.Now()
	request.ApprovedAt = &now

	return rm.transferStore.Update(ctx, request)
}

// RejectTransfer rejects a cross-border transfer request.
func (rm *ResidencyManager) RejectTransfer(ctx context.Context, requestID, reason string) error {
	request, err := rm.transferStore.Get(ctx, requestID)
	if err != nil {
		return err
	}

	if request.Status != TransferStatusPending {
		return fmt.Errorf("transfer request not in pending status: %s", request.Status)
	}

	request.Status = TransferStatusRejected
	request.RejectedReason = reason

	return rm.transferStore.Update(ctx, request)
}

// GetTransferRequest retrieves a transfer request.
func (rm *ResidencyManager) GetTransferRequest(ctx context.Context, requestID string) (*TransferRequest, error) {
	return rm.transferStore.Get(ctx, requestID)
}

// GetPendingTransfers retrieves all pending transfer requests.
func (rm *ResidencyManager) GetPendingTransfers(ctx context.Context) ([]*TransferRequest, error) {
	return rm.transferStore.GetPending(ctx)
}

// GetViolations retrieves violations for a policy.
func (rm *ResidencyManager) GetViolations(ctx context.Context, policyID string) ([]*ResidencyViolation, error) {
	return rm.violationStore.GetByPolicy(ctx, policyID)
}

// GetUnresolvedViolations retrieves all unresolved violations.
func (rm *ResidencyManager) GetUnresolvedViolations(ctx context.Context) ([]*ResidencyViolation, error) {
	return rm.violationStore.GetUnresolved(ctx)
}

// ResolveViolation marks a violation as resolved.
func (rm *ResidencyManager) ResolveViolation(ctx context.Context, violationID, remediationAction string) error {
	violation, err := rm.violationStore.Get(ctx, violationID)
	if err != nil {
		return err
	}

	violation.Status = "resolved"
	violation.RemediationAction = remediationAction
	now := time.Now()
	violation.ResolvedAt = &now

	return rm.violationStore.Update(ctx, violation)
}

// GetRegionCompliance checks compliance for all resources in a region.
func (rm *ResidencyManager) GetRegionCompliance(ctx context.Context, region Region) (bool, int, int, error) {
	policies, err := rm.policyStore.GetByRegion(ctx, region)
	if err != nil {
		return false, 0, 0, err
	}

	totalPolicies := len(policies)
	compliantPolicies := 0

	for _, policy := range policies {
		if !policy.Active {
			continue
		}

		violations, err := rm.violationStore.GetByPolicy(ctx, policy.ID)
		if err != nil {
			continue
		}

		hasUnresolvedViolations := false
		for _, v := range violations {
			if v.Status != "resolved" && v.Status != "false_positive" {
				hasUnresolvedViolations = true
				break
			}
		}

		if !hasUnresolvedViolations {
			compliantPolicies++
		}
	}

	isCompliant := compliantPolicies == totalPolicies && totalPolicies > 0
	return isCompliant, compliantPolicies, totalPolicies, nil
}

// generateID generates a unique ID.
func generateID() string {
	return uid.New("res")
}
