// Package gdpr provides GDPR compliance features
package gdpr

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

// ProcessingRecord represents a GDPR Article 30 compliant data processing record
// This implements the requirement to maintain records of processing activities
type ProcessingRecord struct {
	ID                    string                 `json:"id"`
	Name                  string                 `json:"name"` // Name of processing activity
	Description           string                 `json:"description"`
	Controller            *ControllerInfo        `json:"controller"`
	DataProtectionOfficer *DPOInfo               `json:"dpo,omitempty"`

	// Purpose and legal basis (Art 30.1.b)
	Purposes              []ProcessingPurpose    `json:"purposes"`
	LegalBases            []LegalBasis           `json:"legal_bases"`

	// Categories of data (Art 30.1.c)
	DataSubjectCategories []string               `json:"data_subject_categories"` // e.g., "customers", "employees"
	PersonalDataCategories []string              `json:"personal_data_categories"` // e.g., "email", "name", "SSN"

	// Recipients (Art 30.1.d)
	Recipients            []Recipient            `json:"recipients,omitempty"`

	// Transfers (Art 30.1.e)
	InternationalTransfers []InternationalTransfer `json:"international_transfers,omitempty"`

	// Retention (Art 30.1.f)
	RetentionPeriod       string                 `json:"retention_period"` // e.g., "7 years"
	RetentionCriteria     string                 `json:"retention_criteria"`

	// Security measures (Art 30.1.g)
	TechnicalMeasures     []string               `json:"technical_measures"`
	OrganizationalMeasures []string              `json:"organizational_measures"`

	// Metadata
	CreatedAt             time.Time              `json:"created_at"`
	UpdatedAt             time.Time              `json:"updated_at"`
	CreatedBy             string                 `json:"created_by"`
	Version               int                    `json:"version"`
	Status                ProcessingRecordStatus `json:"status"`
	ReviewDate            *time.Time             `json:"review_date,omitempty"`
	NextReviewDate        *time.Time             `json:"next_review_date,omitempty"`
}

// ControllerInfo contains information about the data controller
type ControllerInfo struct {
	Name              string   `json:"name"`
	LegalForm         string   `json:"legal_form,omitempty"`
	Address           string   `json:"address"`
	Country           string   `json:"country"`
	ContactPerson     string   `json:"contact_person"`
	Email             string   `json:"email"`
	Phone             string   `json:"phone,omitempty"`
	Representatives   []string `json:"representatives,omitempty"`
}

// DPOInfo contains information about the Data Protection Officer
type DPOInfo struct {
	Name    string `json:"name"`
	Email   string `json:"email"`
	Phone   string `json:"phone,omitempty"`
	Address string `json:"address,omitempty"`
}

// ProcessingPurpose defines a purpose for processing
type ProcessingPurpose struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	Category    string `json:"category"` // e.g., "backup", "analytics", "security"
}

// LegalBasis defines the legal basis for processing
type LegalBasis struct {
	Type        LegalBasisType `json:"type"`
	Description string         `json:"description"`
	Reference   string         `json:"reference,omitempty"` // Legal reference
}

// LegalBasisType represents GDPR legal bases for processing
type LegalBasisType string

const (
	LegalBasisConsent             LegalBasisType = "consent"              // Art 6.1.a
	LegalBasisContract            LegalBasisType = "contract"             // Art 6.1.b
	LegalBasisLegalObligation     LegalBasisType = "legal_obligation"     // Art 6.1.c
	LegalBasisVitalInterests      LegalBasisType = "vital_interests"      // Art 6.1.d
	LegalBasisPublicTask          LegalBasisType = "public_task"          // Art 6.1.e
	LegalBasisLegitimateInterests LegalBasisType = "legitimate_interests" // Art 6.1.f
)

// Recipient represents a recipient of personal data
type Recipient struct {
	Name        string   `json:"name"`
	Type        string   `json:"type"` // e.g., "processor", "third_party", "public_authority"
	Category    string   `json:"category"`
	Country     string   `json:"country"`
	Safeguards  []string `json:"safeguards,omitempty"`
	ContactInfo string   `json:"contact_info,omitempty"`
}

// InternationalTransfer represents a transfer to a third country
type InternationalTransfer struct {
	Country           string   `json:"country"`
	AdequacyDecision  bool     `json:"adequacy_decision"` // EU adequacy decision exists
	Safeguards        []string `json:"safeguards"`        // e.g., "Standard Contractual Clauses", "Binding Corporate Rules"
	DerogationApplied string   `json:"derogation_applied,omitempty"`
	TransferMechanism string   `json:"transfer_mechanism"`
}

// ProcessingRecordStatus represents the status of a processing record
type ProcessingRecordStatus string

const (
	ProcessingRecordStatusActive   ProcessingRecordStatus = "active"
	ProcessingRecordStatusInactive ProcessingRecordStatus = "inactive"
	ProcessingRecordStatusUnderReview ProcessingRecordStatus = "under_review"
	ProcessingRecordStatusDeprecated ProcessingRecordStatus = "deprecated"
)

// ProcessingRecordManager manages GDPR processing records
type ProcessingRecordManager struct {
	store ProcessingRecordStore
}

// ProcessingRecordStore defines the interface for storing processing records
type ProcessingRecordStore interface {
	Save(ctx context.Context, record *ProcessingRecord) error
	Get(ctx context.Context, recordID string) (*ProcessingRecord, error)
	List(ctx context.Context, filter *ProcessingRecordFilter) ([]*ProcessingRecord, error)
	Update(ctx context.Context, record *ProcessingRecord) error
	Delete(ctx context.Context, recordID string) error
	GetByController(ctx context.Context, controllerName string) ([]*ProcessingRecord, error)
	GetByPurpose(ctx context.Context, purpose string) ([]*ProcessingRecord, error)
}

// ProcessingRecordFilter defines filters for listing processing records
type ProcessingRecordFilter struct {
	Status              *ProcessingRecordStatus
	ControllerName      string
	Purpose             string
	RequiresReview      bool
	InternationalTransfer bool
}

// NewProcessingRecordManager creates a new processing record manager
func NewProcessingRecordManager(store ProcessingRecordStore) *ProcessingRecordManager {
	return &ProcessingRecordManager{
		store: store,
	}
}

// CreateRecord creates a new processing record
func (m *ProcessingRecordManager) CreateRecord(ctx context.Context, record *ProcessingRecord) error {
	if err := m.validateRecord(record); err != nil {
		return err
	}

	record.ID = generateRecordID()
	record.CreatedAt = time.Now()
	record.UpdatedAt = time.Now()
	record.Version = 1
	record.Status = ProcessingRecordStatusActive

	// Set next review date (1 year from now by default)
	nextReview := time.Now().AddDate(1, 0, 0)
	record.NextReviewDate = &nextReview

	return m.store.Save(ctx, record)
}

// UpdateRecord updates an existing processing record
func (m *ProcessingRecordManager) UpdateRecord(ctx context.Context, record *ProcessingRecord) error {
	if err := m.validateRecord(record); err != nil {
		return err
	}

	existing, err := m.store.Get(ctx, record.ID)
	if err != nil {
		return err
	}

	// Increment version
	record.Version = existing.Version + 1
	record.UpdatedAt = time.Now()
	record.CreatedAt = existing.CreatedAt // Preserve original creation date

	return m.store.Update(ctx, record)
}

// GetRecord retrieves a processing record
func (m *ProcessingRecordManager) GetRecord(ctx context.Context, recordID string) (*ProcessingRecord, error) {
	return m.store.Get(ctx, recordID)
}

// ListRecords lists processing records with optional filters
func (m *ProcessingRecordManager) ListRecords(ctx context.Context, filter *ProcessingRecordFilter) ([]*ProcessingRecord, error) {
	return m.store.List(ctx, filter)
}

// MarkForReview marks a record for review
func (m *ProcessingRecordManager) MarkForReview(ctx context.Context, recordID string) error {
	record, err := m.store.Get(ctx, recordID)
	if err != nil {
		return err
	}

	record.Status = ProcessingRecordStatusUnderReview
	now := time.Now()
	record.ReviewDate = &now
	record.UpdatedAt = now

	return m.store.Update(ctx, record)
}

// CompleteReview completes a review and sets next review date
func (m *ProcessingRecordManager) CompleteReview(ctx context.Context, recordID string, nextReviewMonths int) error {
	record, err := m.store.Get(ctx, recordID)
	if err != nil {
		return err
	}

	record.Status = ProcessingRecordStatusActive
	nextReview := time.Now().AddDate(0, nextReviewMonths, 0)
	record.NextReviewDate = &nextReview
	record.UpdatedAt = time.Now()

	return m.store.Update(ctx, record)
}

// GetRecordsRequiringReview returns records that need review
func (m *ProcessingRecordManager) GetRecordsRequiringReview(ctx context.Context) ([]*ProcessingRecord, error) {
	filter := &ProcessingRecordFilter{
		RequiresReview: true,
	}
	return m.store.List(ctx, filter)
}

// ExportRecords exports all processing records (for compliance)
func (m *ProcessingRecordManager) ExportRecords(ctx context.Context) ([]byte, error) {
	records, err := m.store.List(ctx, nil)
	if err != nil {
		return nil, err
	}

	export := map[string]interface{}{
		"exported_at":   time.Now(),
		"record_count":  len(records),
		"records":       records,
		"compliance":    "GDPR Article 30",
	}

	return json.MarshalIndent(export, "", "  ")
}

// GenerateComplianceReport generates a compliance report
func (m *ProcessingRecordManager) GenerateComplianceReport(ctx context.Context) (*ComplianceReport, error) {
	records, err := m.store.List(ctx, nil)
	if err != nil {
		return nil, err
	}

	report := &ComplianceReport{
		GeneratedAt:      time.Now(),
		TotalRecords:     len(records),
		ActiveRecords:    0,
		RecordsUnderReview: 0,
		RecordsRequiringReview: 0,
		InternationalTransfers: 0,
		ByLegalBasis:     make(map[string]int),
		ByPurpose:        make(map[string]int),
		ByController:     make(map[string]int),
	}

	for _, record := range records {
		// Count by status
		switch record.Status {
		case ProcessingRecordStatusActive:
			report.ActiveRecords++
		case ProcessingRecordStatusUnderReview:
			report.RecordsUnderReview++
		}

		// Check if review is needed
		if record.NextReviewDate != nil && time.Now().After(*record.NextReviewDate) {
			report.RecordsRequiringReview++
		}

		// Count international transfers
		if len(record.InternationalTransfers) > 0 {
			report.InternationalTransfers++
		}

		// Count by legal basis
		for _, lb := range record.LegalBases {
			report.ByLegalBasis[string(lb.Type)]++
		}

		// Count by purpose
		for _, p := range record.Purposes {
			report.ByPurpose[p.Category]++
		}

		// Count by controller
		if record.Controller != nil {
			report.ByController[record.Controller.Name]++
		}
	}

	return report, nil
}

// validateRecord validates a processing record
func (m *ProcessingRecordManager) validateRecord(record *ProcessingRecord) error {
	if record.Name == "" {
		return errors.New("processing record name is required")
	}

	if record.Controller == nil {
		return errors.New("controller information is required")
	}

	if len(record.Purposes) == 0 {
		return errors.New("at least one processing purpose is required")
	}

	if len(record.LegalBases) == 0 {
		return errors.New("at least one legal basis is required")
	}

	if len(record.DataSubjectCategories) == 0 {
		return errors.New("at least one data subject category is required")
	}

	if len(record.PersonalDataCategories) == 0 {
		return errors.New("at least one personal data category is required")
	}

	if record.RetentionPeriod == "" {
		return errors.New("retention period is required")
	}

	return nil
}

// ComplianceReport represents a GDPR compliance report
type ComplianceReport struct {
	GeneratedAt            time.Time         `json:"generated_at"`
	TotalRecords           int               `json:"total_records"`
	ActiveRecords          int               `json:"active_records"`
	RecordsUnderReview     int               `json:"records_under_review"`
	RecordsRequiringReview int               `json:"records_requiring_review"`
	InternationalTransfers int               `json:"international_transfers"`
	ByLegalBasis           map[string]int    `json:"by_legal_basis"`
	ByPurpose              map[string]int    `json:"by_purpose"`
	ByController           map[string]int    `json:"by_controller"`
}

// generateRecordID generates a unique processing record ID
func generateRecordID() string {
	return "proc-" + time.Now().Format("20060102150405") + "-" + randomString(8)
}
