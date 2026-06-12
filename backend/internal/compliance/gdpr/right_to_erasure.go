package gdpr

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// ErasureRequest represents a user's request to be forgotten
type ErasureRequest struct {
	ID            string              `json:"id"`
	UserID        string              `json:"user_id"`
	RequestedAt   time.Time           `json:"requested_at"`
	CompletedAt   *time.Time          `json:"completed_at,omitempty"`
	Status        ErasureStatus       `json:"status"`
	Reason        string              `json:"reason,omitempty"`
	VerifiedBy    string              `json:"verified_by,omitempty"`
	DataTypes     []string            `json:"data_types"` // backups, logs, metadata, etc.
	RetentionDays int                 `json:"retention_days"` // grace period before actual deletion
	Results       map[string]EraseResult `json:"results,omitempty"`
}

// ErasureStatus represents the status of an erasure request
type ErasureStatus string

const (
	ErasureStatusPending    ErasureStatus = "pending"
	ErasureStatusVerifying  ErasureStatus = "verifying"
	ErasureStatusProcessing ErasureStatus = "processing"
	ErasureStatusCompleted  ErasureStatus = "completed"
	ErasureStatusFailed     ErasureStatus = "failed"
	ErasureStatusCancelled  ErasureStatus = "cancelled"
)

// EraseResult represents the result of erasing data from a specific source
type EraseResult struct {
	Source    string    `json:"source"`    // database, storage, logs, etc.
	Success   bool      `json:"success"`
	DeletedAt time.Time `json:"deleted_at"`
	Error     string    `json:"error,omitempty"`
	Count     int       `json:"count"` // number of records deleted
}

// ErasureManager manages right to be forgotten requests
type ErasureManager struct {
	store       ErasureStore
	dataErasers []DataEraser
}

// ErasureStore defines the interface for storing erasure requests
type ErasureStore interface {
	Save(ctx context.Context, request *ErasureRequest) error
	Get(ctx context.Context, requestID string) (*ErasureRequest, error)
	GetByUserID(ctx context.Context, userID string) ([]*ErasureRequest, error)
	Update(ctx context.Context, request *ErasureRequest) error
}

// DataEraser defines the interface for erasing user data from various sources
type DataEraser interface {
	Name() string
	EraseUserData(ctx context.Context, userID string) (int, error)
}

// NewErasureManager creates a new erasure manager
func NewErasureManager(store ErasureStore, erasers []DataEraser) *ErasureManager {
	return &ErasureManager{
		store:       store,
		dataErasers: erasers,
	}
}

// CreateErasureRequest creates a new right to be forgotten request
func (em *ErasureManager) CreateErasureRequest(ctx context.Context, userID, reason string, dataTypes []string, retentionDays int) (*ErasureRequest, error) {
	if userID == "" {
		return nil, errors.New("user ID is required")
	}

	// Check if there's already a pending request
	existing, err := em.store.GetByUserID(ctx, userID)
	if err == nil && len(existing) > 0 {
		for _, req := range existing {
			if req.Status == ErasureStatusPending || req.Status == ErasureStatusProcessing {
				return nil, fmt.Errorf("erasure request already exists: %s", req.ID)
			}
		}
	}

	if dataTypes == nil || len(dataTypes) == 0 {
		dataTypes = []string{"all"}
	}

	if retentionDays == 0 {
		retentionDays = 30 // Default 30-day grace period
	}

	request := &ErasureRequest{
		ID:            generateID(),
		UserID:        userID,
		RequestedAt:   time.Now(),
		Status:        ErasureStatusPending,
		Reason:        reason,
		DataTypes:     dataTypes,
		RetentionDays: retentionDays,
		Results:       make(map[string]EraseResult),
	}

	if err := em.store.Save(ctx, request); err != nil {
		return nil, err
	}

	return request, nil
}

// ProcessErasureRequest processes a right to be forgotten request
func (em *ErasureManager) ProcessErasureRequest(ctx context.Context, requestID, verifiedBy string) error {
	request, err := em.store.Get(ctx, requestID)
	if err != nil {
		return err
	}

	if request.Status != ErasureStatusPending {
		return fmt.Errorf("request not in pending status: %s", request.Status)
	}

	// Update status to processing
	request.Status = ErasureStatusProcessing
	request.VerifiedBy = verifiedBy
	if err := em.store.Update(ctx, request); err != nil {
		return err
	}

	// Check if grace period has passed
	gracePeriodEnd := request.RequestedAt.Add(time.Duration(request.RetentionDays) * 24 * time.Hour)
	if time.Now().Before(gracePeriodEnd) {
		return fmt.Errorf("grace period not expired (expires at %s)", gracePeriodEnd.Format(time.RFC3339))
	}

	// Execute erasure across all data erasers
	hasFailures := false
	for _, eraser := range em.dataErasers {
		count, err := eraser.EraseUserData(ctx, request.UserID)

		result := EraseResult{
			Source:    eraser.Name(),
			DeletedAt: time.Now(),
			Count:     count,
		}

		if err != nil {
			result.Success = false
			result.Error = err.Error()
			hasFailures = true
		} else {
			result.Success = true
		}

		request.Results[eraser.Name()] = result
	}

	// Update final status
	if hasFailures {
		request.Status = ErasureStatusFailed
	} else {
		request.Status = ErasureStatusCompleted
		now := time.Now()
		request.CompletedAt = &now
	}

	return em.store.Update(ctx, request)
}

// CancelErasureRequest cancels a pending erasure request
func (em *ErasureManager) CancelErasureRequest(ctx context.Context, requestID string) error {
	request, err := em.store.Get(ctx, requestID)
	if err != nil {
		return err
	}

	if request.Status != ErasureStatusPending {
		return fmt.Errorf("can only cancel pending requests, current status: %s", request.Status)
	}

	request.Status = ErasureStatusCancelled
	now := time.Now()
	request.CompletedAt = &now

	return em.store.Update(ctx, request)
}

// GetErasureRequest retrieves an erasure request
func (em *ErasureManager) GetErasureRequest(ctx context.Context, requestID string) (*ErasureRequest, error) {
	return em.store.Get(ctx, requestID)
}

// GetUserErasureRequests retrieves all erasure requests for a user
func (em *ErasureManager) GetUserErasureRequests(ctx context.Context, userID string) ([]*ErasureRequest, error) {
	return em.store.GetByUserID(ctx, userID)
}

// BackupDataEraser erases user backup data
type BackupDataEraser struct{}

func (b *BackupDataEraser) Name() string {
	return "backups"
}

func (b *BackupDataEraser) EraseUserData(ctx context.Context, userID string) (int, error) {
	// Implementation would delete all backups owned by the user
	// This is a placeholder
	return 0, nil
}

// LogDataEraser erases user log data
type LogDataEraser struct{}

func (l *LogDataEraser) Name() string {
	return "logs"
}

func (l *LogDataEraser) EraseUserData(ctx context.Context, userID string) (int, error) {
	// Implementation would delete all logs related to the user
	// This is a placeholder
	return 0, nil
}

// MetadataEraser erases user metadata
type MetadataEraser struct{}

func (m *MetadataEraser) Name() string {
	return "metadata"
}

func (m *MetadataEraser) EraseUserData(ctx context.Context, userID string) (int, error) {
	// Implementation would delete all metadata related to the user
	// This is a placeholder
	return 0, nil
}
