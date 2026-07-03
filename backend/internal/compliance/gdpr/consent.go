// Package gdpr provides GDPR compliance features
package gdpr

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

// ConsentPurpose defines the purpose for which consent is given.
type ConsentPurpose string

const (
	PurposeBackupStorage     ConsentPurpose = "backup_storage"
	PurposeDataProcessing    ConsentPurpose = "data_processing"
	PurposeDataTransfer      ConsentPurpose = "data_transfer"
	PurposeAnalytics         ConsentPurpose = "analytics"
	PurposeThirdPartySharing ConsentPurpose = "third_party_sharing"
)

// ConsentStatus represents the status of consent.
type ConsentStatus string

const (
	ConsentStatusGranted ConsentStatus = "granted"
	ConsentStatusRevoked ConsentStatus = "revoked"
	ConsentStatusExpired ConsentStatus = "expired"
	ConsentStatusPending ConsentStatus = "pending"
)

// Consent represents a user's consent for data processing.
type Consent struct {
	ID            string                 `json:"id"`
	UserID        string                 `json:"user_id"`
	Purpose       ConsentPurpose         `json:"purpose"`
	Status        ConsentStatus          `json:"status"`
	GrantedAt     time.Time              `json:"granted_at"`
	RevokedAt     *time.Time             `json:"revoked_at,omitempty"`
	ExpiresAt     *time.Time             `json:"expires_at,omitempty"`
	LegalBasis    string                 `json:"legal_basis"`
	IPAddress     string                 `json:"ip_address"`
	UserAgent     string                 `json:"user_agent"`
	ConsentMethod string                 `json:"consent_method"` // web, api, cli, etc.
	Version       string                 `json:"version"`        // consent policy version
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// ConsentRecord stores the history of consent changes.
type ConsentRecord struct {
	ID            string    `json:"id"`
	ConsentID     string    `json:"consent_id"`
	Action        string    `json:"action"` // granted, revoked, updated
	Timestamp     time.Time `json:"timestamp"`
	PreviousState string    `json:"previous_state"`
	NewState      string    `json:"new_state"`
	Reason        string    `json:"reason,omitempty"`
}

// ConsentManager manages user consent for GDPR compliance.
type ConsentManager struct {
	store ConsentStore
}

// ConsentStore defines the interface for storing consent records.
type ConsentStore interface {
	Save(ctx context.Context, consent *Consent) error
	Get(ctx context.Context, consentID string) (*Consent, error)
	GetByUserID(ctx context.Context, userID string) ([]*Consent, error)
	GetByUserIDAndPurpose(ctx context.Context, userID string, purpose ConsentPurpose) (*Consent, error)
	Update(ctx context.Context, consent *Consent) error
	Delete(ctx context.Context, consentID string) error
	SaveRecord(ctx context.Context, record *ConsentRecord) error
	GetRecords(ctx context.Context, consentID string) ([]*ConsentRecord, error)
}

// NewConsentManager creates a new consent manager.
func NewConsentManager(store ConsentStore) *ConsentManager {
	return &ConsentManager{
		store: store,
	}
}

// GrantConsent records user consent for a specific purpose.
func (cm *ConsentManager) GrantConsent(ctx context.Context, userID string, purpose ConsentPurpose, legalBasis, ipAddress, userAgent, method, version string, expiresIn *time.Duration) (*Consent, error) {
	if userID == "" {
		return nil, errors.New("user ID is required")
	}

	if purpose == "" {
		return nil, errors.New("purpose is required")
	}

	// Check if consent already exists
	existing, err := cm.store.GetByUserIDAndPurpose(ctx, userID, purpose)
	if err == nil && existing != nil && existing.Status == ConsentStatusGranted {
		return existing, nil // Already has active consent
	}

	consent := &Consent{
		ID:            generateID(),
		UserID:        userID,
		Purpose:       purpose,
		Status:        ConsentStatusGranted,
		GrantedAt:     time.Now(),
		LegalBasis:    legalBasis,
		IPAddress:     ipAddress,
		UserAgent:     userAgent,
		ConsentMethod: method,
		Version:       version,
		Metadata:      make(map[string]interface{}),
	}

	if expiresIn != nil {
		expiresAt := time.Now().Add(*expiresIn)
		consent.ExpiresAt = &expiresAt
	}

	if err := cm.store.Save(ctx, consent); err != nil {
		return nil, err
	}

	// Record the consent grant
	record := &ConsentRecord{
		ID:            generateID(),
		ConsentID:     consent.ID,
		Action:        "granted",
		Timestamp:     time.Now(),
		PreviousState: "",
		NewState:      string(ConsentStatusGranted),
	}

	if err := cm.store.SaveRecord(ctx, record); err != nil {
		return nil, err
	}

	return consent, nil
}

// RevokeConsent revokes user consent.
func (cm *ConsentManager) RevokeConsent(ctx context.Context, consentID, reason string) error {
	consent, err := cm.store.Get(ctx, consentID)
	if err != nil {
		return err
	}

	if consent.Status == ConsentStatusRevoked {
		return errors.New("consent already revoked")
	}

	previousState := string(consent.Status)
	now := time.Now()
	consent.Status = ConsentStatusRevoked
	consent.RevokedAt = &now

	if err := cm.store.Update(ctx, consent); err != nil {
		return err
	}

	// Record the revocation
	record := &ConsentRecord{
		ID:            generateID(),
		ConsentID:     consent.ID,
		Action:        "revoked",
		Timestamp:     time.Now(),
		PreviousState: previousState,
		NewState:      string(ConsentStatusRevoked),
		Reason:        reason,
	}

	if err := cm.store.SaveRecord(ctx, record); err != nil {
		return err
	}

	return nil
}

// CheckConsent verifies if user has valid consent for a purpose.
func (cm *ConsentManager) CheckConsent(ctx context.Context, userID string, purpose ConsentPurpose) (bool, error) {
	consent, err := cm.store.GetByUserIDAndPurpose(ctx, userID, purpose)
	if err != nil {
		return false, err
	}

	if consent == nil {
		return false, nil
	}

	// Check if consent is granted
	if consent.Status != ConsentStatusGranted {
		return false, nil
	}

	// Check if consent has expired
	if consent.ExpiresAt != nil && time.Now().After(*consent.ExpiresAt) {
		// Auto-expire the consent
		consent.Status = ConsentStatusExpired
		if err := cm.store.Update(ctx, consent); err != nil {
			return false, err
		}
		return false, nil
	}

	return true, nil
}

// GetConsent retrieves a consent record.
func (cm *ConsentManager) GetConsent(ctx context.Context, consentID string) (*Consent, error) {
	return cm.store.Get(ctx, consentID)
}

// GetUserConsents retrieves all consent records for a user.
func (cm *ConsentManager) GetUserConsents(ctx context.Context, userID string) ([]*Consent, error) {
	return cm.store.GetByUserID(ctx, userID)
}

// GetConsentHistory retrieves the history of consent changes.
func (cm *ConsentManager) GetConsentHistory(ctx context.Context, consentID string) ([]*ConsentRecord, error) {
	return cm.store.GetRecords(ctx, consentID)
}

// ExportUserConsents exports all user consents as JSON (for data portability).
func (cm *ConsentManager) ExportUserConsents(ctx context.Context, userID string) ([]byte, error) {
	consents, err := cm.store.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	data := map[string]interface{}{
		"user_id":       userID,
		"exported_at":   time.Now(),
		"consent_count": len(consents),
		"consents":      consents,
	}

	return json.MarshalIndent(data, "", "  ")
}

// ValidateConsent checks if a consent is still valid.
func (cm *ConsentManager) ValidateConsent(consent *Consent) (valid bool, reason string) {
	if consent.Status != ConsentStatusGranted {
		return false, "consent not granted"
	}

	if consent.ExpiresAt != nil && time.Now().After(*consent.ExpiresAt) {
		return false, "consent expired"
	}

	if consent.RevokedAt != nil {
		return false, "consent revoked"
	}

	return true, ""
}

// generateID generates a unique ID for consent records.
func generateID() string {
	return time.Now().Format("20060102150405") + "-" + randomString(8)
}

func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, length)
	// Use UnixNano to get more entropy per character
	nanoTime := time.Now().UnixNano()
	for i := range b {
		// Add index to ensure different chars in same call
		nanoTime = (nanoTime*1103515245 + 12345) // Simple LCG
		// Use absolute value to avoid negative modulo
		idx := int(nanoTime)
		if idx < 0 {
			idx = -idx
		}
		b[i] = charset[idx%len(charset)]
	}
	return string(b)
}
