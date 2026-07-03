package residency

import (
	"context"
	"errors"
	"testing"
)

// Mock stores for testing.
type mockPolicyStore struct {
	policies map[string]*ResidencyPolicy
}

func newMockPolicyStore() *mockPolicyStore {
	return &mockPolicyStore{
		policies: make(map[string]*ResidencyPolicy),
	}
}

func (m *mockPolicyStore) Save(ctx context.Context, policy *ResidencyPolicy) error {
	m.policies[policy.ID] = policy
	return nil
}

func (m *mockPolicyStore) Get(ctx context.Context, policyID string) (*ResidencyPolicy, error) {
	policy, ok := m.policies[policyID]
	if !ok {
		return nil, errors.New("policy not found")
	}
	return policy, nil
}

func (m *mockPolicyStore) GetAll(ctx context.Context) ([]*ResidencyPolicy, error) {
	result := make([]*ResidencyPolicy, 0, len(m.policies))
	for _, policy := range m.policies {
		result = append(result, policy)
	}
	return result, nil
}

func (m *mockPolicyStore) GetByRegion(ctx context.Context, region Region) ([]*ResidencyPolicy, error) {
	var result []*ResidencyPolicy
	for _, policy := range m.policies {
		if policy.PrimaryRegion == region {
			result = append(result, policy)
		}
	}
	return result, nil
}

func (m *mockPolicyStore) Update(ctx context.Context, policy *ResidencyPolicy) error {
	if _, ok := m.policies[policy.ID]; !ok {
		return errors.New("policy not found")
	}
	m.policies[policy.ID] = policy
	return nil
}

func (m *mockPolicyStore) Delete(ctx context.Context, policyID string) error {
	delete(m.policies, policyID)
	return nil
}

type mockViolationStore struct {
	violations map[string]*ResidencyViolation
}

func newMockViolationStore() *mockViolationStore {
	return &mockViolationStore{
		violations: make(map[string]*ResidencyViolation),
	}
}

func (m *mockViolationStore) Save(ctx context.Context, violation *ResidencyViolation) error {
	m.violations[violation.ID] = violation
	return nil
}

func (m *mockViolationStore) Get(ctx context.Context, violationID string) (*ResidencyViolation, error) {
	violation, ok := m.violations[violationID]
	if !ok {
		return nil, errors.New("violation not found")
	}
	return violation, nil
}

func (m *mockViolationStore) GetByPolicy(ctx context.Context, policyID string) ([]*ResidencyViolation, error) {
	var result []*ResidencyViolation
	for _, violation := range m.violations {
		if violation.PolicyID == policyID {
			result = append(result, violation)
		}
	}
	return result, nil
}

func (m *mockViolationStore) GetUnresolved(ctx context.Context) ([]*ResidencyViolation, error) {
	var result []*ResidencyViolation
	for _, violation := range m.violations {
		if violation.Status != "resolved" && violation.Status != "false_positive" {
			result = append(result, violation)
		}
	}
	return result, nil
}

func (m *mockViolationStore) Update(ctx context.Context, violation *ResidencyViolation) error {
	if _, ok := m.violations[violation.ID]; !ok {
		return errors.New("violation not found")
	}
	m.violations[violation.ID] = violation
	return nil
}

type mockTransferStore struct {
	transfers map[string]*TransferRequest
}

func newMockTransferStore() *mockTransferStore {
	return &mockTransferStore{
		transfers: make(map[string]*TransferRequest),
	}
}

func (m *mockTransferStore) Save(ctx context.Context, request *TransferRequest) error {
	m.transfers[request.ID] = request
	return nil
}

func (m *mockTransferStore) Get(ctx context.Context, requestID string) (*TransferRequest, error) {
	request, ok := m.transfers[requestID]
	if !ok {
		return nil, errors.New("transfer request not found")
	}
	return request, nil
}

func (m *mockTransferStore) GetPending(ctx context.Context) ([]*TransferRequest, error) {
	var result []*TransferRequest
	for _, request := range m.transfers {
		if request.Status == TransferStatusPending {
			result = append(result, request)
		}
	}
	return result, nil
}

func (m *mockTransferStore) Update(ctx context.Context, request *TransferRequest) error {
	if _, ok := m.transfers[request.ID]; !ok {
		return errors.New("transfer request not found")
	}
	m.transfers[request.ID] = request
	return nil
}

type mockRegionProvider struct {
	regions map[string]Region
}

func newMockRegionProvider() *mockRegionProvider {
	return &mockRegionProvider{
		regions: make(map[string]Region),
	}
}

func (m *mockRegionProvider) GetResourceRegion(ctx context.Context, resourceType, resourceID string) (Region, error) {
	key := resourceType + ":" + resourceID
	region, ok := m.regions[key]
	if !ok {
		return "", errors.New("resource not found")
	}
	return region, nil
}

func TestCreatePolicy(t *testing.T) {
	policyStore := newMockPolicyStore()
	violationStore := newMockViolationStore()
	transferStore := newMockTransferStore()
	regionProvider := newMockRegionProvider()
	rm := NewResidencyManager(policyStore, violationStore, transferStore, regionProvider)
	ctx := context.Background()

	tests := []struct {
		name        string
		policy      *ResidencyPolicy
		expectError bool
	}{
		{
			name: "Valid policy",
			policy: &ResidencyPolicy{
				Name:                 "EU GDPR Policy",
				Description:          "GDPR compliance policy for EU regions",
				PrimaryRegion:        RegionEUWest,
				AllowedRegions:       []Region{RegionEUWest, RegionEUCentral},
				RestrictedRegions:    []Region{RegionUSEast},
				DataClassification:   ClassificationConfidential,
				CrossBorderTransfer:  false,
				EncryptionRequired:   true,
				ComplianceFrameworks: []string{"GDPR"},
			},
			expectError: false,
		},
		{
			name: "Missing name",
			policy: &ResidencyPolicy{
				PrimaryRegion: RegionEUWest,
			},
			expectError: true,
		},
		{
			name: "Missing primary region",
			policy: &ResidencyPolicy{
				Name: "Test Policy",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := rm.CreatePolicy(ctx, tt.policy)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if tt.policy.ID == "" {
				t.Errorf("Expected policy ID to be generated")
			}

			if !tt.policy.Active {
				t.Errorf("Expected policy to be active")
			}
		})
	}
}

func TestValidateResidency(t *testing.T) {
	policyStore := newMockPolicyStore()
	violationStore := newMockViolationStore()
	transferStore := newMockTransferStore()
	regionProvider := newMockRegionProvider()
	rm := NewResidencyManager(policyStore, violationStore, transferStore, regionProvider)
	ctx := context.Background()

	// Create a policy
	policy := &ResidencyPolicy{
		ID:                "policy-1",
		Name:              "EU Policy",
		PrimaryRegion:     RegionEUWest,
		AllowedRegions:    []Region{RegionEUWest, RegionEUCentral},
		RestrictedRegions: []Region{RegionUSEast},
		Active:            true,
	}
	policyStore.Save(ctx, policy)

	tests := []struct {
		name           string
		resourceType   string
		resourceID     string
		resourceRegion Region
		expectValid    bool
	}{
		{
			name:           "Resource in allowed region",
			resourceType:   "backup",
			resourceID:     "backup-1",
			resourceRegion: RegionEUWest,
			expectValid:    true,
		},
		{
			name:           "Resource in restricted region",
			resourceType:   "backup",
			resourceID:     "backup-2",
			resourceRegion: RegionUSEast,
			expectValid:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up the region for this resource
			regionProvider.regions[tt.resourceType+":"+tt.resourceID] = tt.resourceRegion

			valid, err := rm.ValidateResidency(ctx, tt.resourceType, tt.resourceID, policy.ID)

			if tt.expectValid {
				if !valid {
					t.Errorf("Expected validation to pass, got: %v", err)
				}
			} else {
				if valid {
					t.Errorf("Expected validation to fail")
				}
			}
		})
	}
}

func TestRequestTransfer(t *testing.T) {
	policyStore := newMockPolicyStore()
	violationStore := newMockViolationStore()
	transferStore := newMockTransferStore()
	regionProvider := newMockRegionProvider()
	rm := NewResidencyManager(policyStore, violationStore, transferStore, regionProvider)
	ctx := context.Background()

	tests := []struct {
		name        string
		resourceID  string
		requestedBy string
		sourceReg   Region
		targetReg   Region
		expectError bool
	}{
		{
			name:        "Valid transfer request",
			resourceID:  "backup-1",
			requestedBy: "user-123",
			sourceReg:   RegionEUWest,
			targetReg:   RegionEUCentral,
			expectError: false,
		},
		{
			name:        "Same source and target region",
			resourceID:  "backup-1",
			requestedBy: "user-123",
			sourceReg:   RegionEUWest,
			targetReg:   RegionEUWest,
			expectError: true,
		},
		{
			name:        "Missing resource ID",
			resourceID:  "",
			requestedBy: "user-123",
			sourceReg:   RegionEUWest,
			targetReg:   RegionEUCentral,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request, err := rm.RequestTransfer(ctx, "backup", tt.resourceID, tt.requestedBy, tt.sourceReg, tt.targetReg, "contract", ClassificationInternal)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if request.Status != TransferStatusPending {
				t.Errorf("Expected status %s, got %s", TransferStatusPending, request.Status)
			}
		})
	}
}

func TestApproveTransfer(t *testing.T) {
	policyStore := newMockPolicyStore()
	violationStore := newMockViolationStore()
	transferStore := newMockTransferStore()
	regionProvider := newMockRegionProvider()
	rm := NewResidencyManager(policyStore, violationStore, transferStore, regionProvider)
	ctx := context.Background()

	// Create a transfer request
	request, err := rm.RequestTransfer(ctx, "backup", "backup-1", "user-123", RegionEUWest, RegionEUCentral, "contract", ClassificationInternal)
	if err != nil {
		t.Fatalf("Failed to create transfer request: %v", err)
	}

	// Approve the transfer
	err = rm.ApproveTransfer(ctx, request.ID, "approver-123")
	if err != nil {
		t.Errorf("Failed to approve transfer: %v", err)
	}

	// Verify the transfer was approved
	approvedRequest, err := transferStore.Get(ctx, request.ID)
	if err != nil {
		t.Errorf("Failed to get transfer request: %v", err)
	}

	if approvedRequest.Status != TransferStatusApproved {
		t.Errorf("Expected status %s, got %s", TransferStatusApproved, approvedRequest.Status)
	}

	if approvedRequest.ApprovedBy != "approver-123" {
		t.Errorf("Expected approved by 'approver-123', got '%s'", approvedRequest.ApprovedBy)
	}
}

func TestRejectTransfer(t *testing.T) {
	policyStore := newMockPolicyStore()
	violationStore := newMockViolationStore()
	transferStore := newMockTransferStore()
	regionProvider := newMockRegionProvider()
	rm := NewResidencyManager(policyStore, violationStore, transferStore, regionProvider)
	ctx := context.Background()

	// Create a transfer request
	request, err := rm.RequestTransfer(ctx, "backup", "backup-1", "user-123", RegionEUWest, RegionUSEast, "contract", ClassificationInternal)
	if err != nil {
		t.Fatalf("Failed to create transfer request: %v", err)
	}

	// Reject the transfer
	reason := "Does not meet GDPR requirements"
	err = rm.RejectTransfer(ctx, request.ID, reason)
	if err != nil {
		t.Errorf("Failed to reject transfer: %v", err)
	}

	// Verify the transfer was rejected
	rejectedRequest, err := transferStore.Get(ctx, request.ID)
	if err != nil {
		t.Errorf("Failed to get transfer request: %v", err)
	}

	if rejectedRequest.Status != TransferStatusRejected {
		t.Errorf("Expected status %s, got %s", TransferStatusRejected, rejectedRequest.Status)
	}

	if rejectedRequest.RejectedReason != reason {
		t.Errorf("Expected rejected reason '%s', got '%s'", reason, rejectedRequest.RejectedReason)
	}
}

func TestResolveViolation(t *testing.T) {
	policyStore := newMockPolicyStore()
	violationStore := newMockViolationStore()
	transferStore := newMockTransferStore()
	regionProvider := newMockRegionProvider()
	rm := NewResidencyManager(policyStore, violationStore, transferStore, regionProvider)
	ctx := context.Background()

	// Create a violation
	violation := &ResidencyViolation{
		ID:             "violation-1",
		PolicyID:       "policy-1",
		ResourceID:     "backup-1",
		ResourceType:   "backup",
		ViolationType:  "restricted_region",
		ExpectedRegion: RegionEUWest,
		ActualRegion:   RegionUSEast,
		Status:         "detected",
	}
	violationStore.Save(ctx, violation)

	// Resolve the violation
	remediationAction := "Moved backup to EU-West region"
	err := rm.ResolveViolation(ctx, violation.ID, remediationAction)
	if err != nil {
		t.Errorf("Failed to resolve violation: %v", err)
	}

	// Verify the violation was resolved
	resolvedViolation, err := violationStore.Get(ctx, violation.ID)
	if err != nil {
		t.Errorf("Failed to get violation: %v", err)
	}

	if resolvedViolation.Status != "resolved" {
		t.Errorf("Expected status 'resolved', got '%s'", resolvedViolation.Status)
	}

	if resolvedViolation.RemediationAction != remediationAction {
		t.Errorf("Expected remediation action '%s', got '%s'", remediationAction, resolvedViolation.RemediationAction)
	}

	if resolvedViolation.ResolvedAt == nil {
		t.Errorf("Expected ResolvedAt to be set")
	}
}

func TestGetRegionCompliance(t *testing.T) {
	policyStore := newMockPolicyStore()
	violationStore := newMockViolationStore()
	transferStore := newMockTransferStore()
	regionProvider := newMockRegionProvider()
	rm := NewResidencyManager(policyStore, violationStore, transferStore, regionProvider)
	ctx := context.Background()

	// Create policies for a region
	policy1 := &ResidencyPolicy{
		ID:            "policy-1",
		Name:          "Policy 1",
		PrimaryRegion: RegionEUWest,
		Active:        true,
	}
	policy2 := &ResidencyPolicy{
		ID:            "policy-2",
		Name:          "Policy 2",
		PrimaryRegion: RegionEUWest,
		Active:        true,
	}
	policyStore.Save(ctx, policy1)
	policyStore.Save(ctx, policy2)

	// Check compliance
	isCompliant, compliant, total, err := rm.GetRegionCompliance(ctx, RegionEUWest)
	if err != nil {
		t.Errorf("Failed to get region compliance: %v", err)
	}

	if total != 2 {
		t.Errorf("Expected 2 total policies, got %d", total)
	}

	if compliant != 2 {
		t.Errorf("Expected 2 compliant policies, got %d", compliant)
	}

	if !isCompliant {
		t.Errorf("Expected region to be compliant")
	}
}
