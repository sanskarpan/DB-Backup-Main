package gdpr

import (
	"context"
	"errors"
	"testing"
	"time"
)

// Mock ConsentStore for testing.
type mockConsentStore struct {
	consents map[string]*Consent
	records  map[string][]*ConsentRecord
}

func newMockConsentStore() *mockConsentStore {
	return &mockConsentStore{
		consents: make(map[string]*Consent),
		records:  make(map[string][]*ConsentRecord),
	}
}

func (m *mockConsentStore) Save(ctx context.Context, consent *Consent) error {
	if consent.ID == "" {
		return errors.New("consent ID is required")
	}
	m.consents[consent.ID] = consent
	return nil
}

func (m *mockConsentStore) Get(ctx context.Context, consentID string) (*Consent, error) {
	consent, ok := m.consents[consentID]
	if !ok {
		return nil, errors.New("consent not found")
	}
	return consent, nil
}

func (m *mockConsentStore) GetByUserID(ctx context.Context, userID string) ([]*Consent, error) {
	var result []*Consent
	for _, consent := range m.consents {
		if consent.UserID == userID {
			result = append(result, consent)
		}
	}
	return result, nil
}

func (m *mockConsentStore) GetByUserIDAndPurpose(ctx context.Context, userID string, purpose ConsentPurpose) (*Consent, error) {
	for _, consent := range m.consents {
		if consent.UserID == userID && consent.Purpose == purpose {
			return consent, nil
		}
	}
	return nil, errors.New("consent not found")
}

func (m *mockConsentStore) Update(ctx context.Context, consent *Consent) error {
	if _, ok := m.consents[consent.ID]; !ok {
		return errors.New("consent not found")
	}
	m.consents[consent.ID] = consent
	return nil
}

func (m *mockConsentStore) Delete(ctx context.Context, consentID string) error {
	delete(m.consents, consentID)
	return nil
}

func (m *mockConsentStore) SaveRecord(ctx context.Context, record *ConsentRecord) error {
	m.records[record.ConsentID] = append(m.records[record.ConsentID], record)
	return nil
}

func (m *mockConsentStore) GetRecords(ctx context.Context, consentID string) ([]*ConsentRecord, error) {
	return m.records[consentID], nil
}

func TestGrantConsent(t *testing.T) {
	store := newMockConsentStore()
	cm := NewConsentManager(store)
	ctx := context.Background()

	tests := []struct {
		name        string
		userID      string
		purpose     ConsentPurpose
		legalBasis  string
		ipAddress   string
		userAgent   string
		method      string
		version     string
		expiresIn   *time.Duration
		expectError bool
	}{
		{
			name:        "Valid consent grant",
			userID:      "user-123",
			purpose:     PurposeBackupStorage,
			legalBasis:  "contract",
			ipAddress:   "192.168.1.1",
			userAgent:   "Mozilla/5.0",
			method:      "web",
			version:     "1.0",
			expiresIn:   nil,
			expectError: false,
		},
		{
			name:        "Missing user ID",
			userID:      "",
			purpose:     PurposeDataProcessing,
			expectError: true,
		},
		{
			name:        "Missing purpose",
			userID:      "user-123",
			purpose:     "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			consent, err := cm.GrantConsent(ctx, tt.userID, tt.purpose, tt.legalBasis, tt.ipAddress, tt.userAgent, tt.method, tt.version, tt.expiresIn)

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

			if consent.UserID != tt.userID {
				t.Errorf("Expected user ID %s, got %s", tt.userID, consent.UserID)
			}

			if consent.Purpose != tt.purpose {
				t.Errorf("Expected purpose %s, got %s", tt.purpose, consent.Purpose)
			}

			if consent.Status != ConsentStatusGranted {
				t.Errorf("Expected status %s, got %s", ConsentStatusGranted, consent.Status)
			}
		})
	}
}

func TestRevokeConsent(t *testing.T) {
	store := newMockConsentStore()
	cm := NewConsentManager(store)
	ctx := context.Background()

	// First grant a consent
	consent, err := cm.GrantConsent(ctx, "user-123", PurposeBackupStorage, "contract", "192.168.1.1", "Mozilla/5.0", "web", "1.0", nil)
	if err != nil {
		t.Fatalf("Failed to grant consent: %v", err)
	}

	// Revoke the consent
	err = cm.RevokeConsent(ctx, consent.ID, "User requested revocation")
	if err != nil {
		t.Errorf("Failed to revoke consent: %v", err)
	}

	// Verify the consent is revoked
	revokedConsent, err := store.Get(ctx, consent.ID)
	if err != nil {
		t.Errorf("Failed to get consent: %v", err)
	}

	if revokedConsent.Status != ConsentStatusRevoked {
		t.Errorf("Expected status %s, got %s", ConsentStatusRevoked, revokedConsent.Status)
	}

	if revokedConsent.RevokedAt == nil {
		t.Errorf("Expected RevokedAt to be set")
	}

	// Try to revoke already revoked consent
	err = cm.RevokeConsent(ctx, consent.ID, "Already revoked")
	if err == nil {
		t.Errorf("Expected error when revoking already revoked consent")
	}
}

func TestCheckConsent(t *testing.T) {
	store := newMockConsentStore()
	cm := NewConsentManager(store)
	ctx := context.Background()

	// Grant a consent
	_, err := cm.GrantConsent(ctx, "user-123", PurposeBackupStorage, "contract", "192.168.1.1", "Mozilla/5.0", "web", "1.0", nil)
	if err != nil {
		t.Fatalf("Failed to grant consent: %v", err)
	}

	tests := []struct {
		name           string
		userID         string
		purpose        ConsentPurpose
		expectedResult bool
		expectError    bool
	}{
		{
			name:           "Valid consent check",
			userID:         "user-123",
			purpose:        PurposeBackupStorage,
			expectedResult: true,
			expectError:    false,
		},
		{
			name:           "No consent for purpose",
			userID:         "user-123",
			purpose:        PurposeAnalytics,
			expectedResult: false,
			expectError:    false,
		},
		{
			name:           "No consent for user",
			userID:         "user-999",
			purpose:        PurposeBackupStorage,
			expectedResult: false,
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasConsent, err := cm.CheckConsent(ctx, tt.userID, tt.purpose)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if hasConsent != tt.expectedResult {
				t.Errorf("Expected %v, got %v", tt.expectedResult, hasConsent)
			}
		})
	}
}

func TestExportUserConsents(t *testing.T) {
	store := newMockConsentStore()
	cm := NewConsentManager(store)
	ctx := context.Background()

	// Grant multiple consents
	purposes := []ConsentPurpose{PurposeBackupStorage, PurposeDataProcessing, PurposeAnalytics}
	for _, purpose := range purposes {
		_, err := cm.GrantConsent(ctx, "user-123", purpose, "contract", "192.168.1.1", "Mozilla/5.0", "web", "1.0", nil)
		if err != nil {
			t.Fatalf("Failed to grant consent: %v", err)
		}
	}

	// Export consents
	data, err := cm.ExportUserConsents(ctx, "user-123")
	if err != nil {
		t.Errorf("Failed to export consents: %v", err)
	}

	if len(data) == 0 {
		t.Errorf("Expected non-empty export data")
	}

	// Verify JSON structure (basic check)
	if string(data[:1]) != "{" {
		t.Errorf("Expected JSON output")
	}
}

func TestValidateConsent(t *testing.T) {
	cm := NewConsentManager(nil)

	tests := []struct {
		name          string
		consent       *Consent
		expectedValid bool
		expectedMsg   string
	}{
		{
			name: "Valid consent",
			consent: &Consent{
				Status:    ConsentStatusGranted,
				ExpiresAt: nil,
				RevokedAt: nil,
			},
			expectedValid: true,
			expectedMsg:   "",
		},
		{
			name: "Revoked consent",
			consent: &Consent{
				Status: ConsentStatusRevoked,
			},
			expectedValid: false,
			expectedMsg:   "consent not granted",
		},
		{
			name: "Expired consent",
			consent: &Consent{
				Status:    ConsentStatusGranted,
				ExpiresAt: timePtr(time.Now().Add(-24 * time.Hour)),
			},
			expectedValid: false,
			expectedMsg:   "consent expired",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid, msg := cm.ValidateConsent(tt.consent)

			if valid != tt.expectedValid {
				t.Errorf("Expected valid=%v, got valid=%v", tt.expectedValid, valid)
			}

			if msg != tt.expectedMsg {
				t.Errorf("Expected message '%s', got '%s'", tt.expectedMsg, msg)
			}
		})
	}
}

func timePtr(t time.Time) *time.Time {
	return &t
}
