package gdpr

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ErasureRequest represents a user's request to be forgotten.
type ErasureRequest struct {
	ID            string                 `json:"id"`
	UserID        string                 `json:"user_id"`
	RequestedAt   time.Time              `json:"requested_at"`
	CompletedAt   *time.Time             `json:"completed_at,omitempty"`
	Status        ErasureStatus          `json:"status"`
	Reason        string                 `json:"reason,omitempty"`
	VerifiedBy    string                 `json:"verified_by,omitempty"`
	DataTypes     []string               `json:"data_types"`     // backups, logs, metadata, etc.
	RetentionDays int                    `json:"retention_days"` // grace period before actual deletion
	Results       map[string]EraseResult `json:"results,omitempty"`
}

// ErasureStatus represents the status of an erasure request.
type ErasureStatus string

const (
	ErasureStatusPending    ErasureStatus = "pending"
	ErasureStatusVerifying  ErasureStatus = "verifying"
	ErasureStatusProcessing ErasureStatus = "processing"
	ErasureStatusCompleted  ErasureStatus = "completed"
	ErasureStatusFailed     ErasureStatus = "failed"
	ErasureStatusCancelled  ErasureStatus = "canceled"
)

// EraseResult represents the result of erasing data from a specific source.
type EraseResult struct {
	Source    string    `json:"source"` // database, storage, logs, etc.
	Success   bool      `json:"success"`
	DeletedAt time.Time `json:"deleted_at"`
	Error     string    `json:"error,omitempty"`
	Count     int       `json:"count"` // number of records deleted
}

// ErasureManager manages right to be forgotten requests.
type ErasureManager struct {
	store       ErasureStore
	dataErasers []DataEraser
}

// ErasureStore defines the interface for storing erasure requests.
type ErasureStore interface {
	Save(ctx context.Context, request *ErasureRequest) error
	Get(ctx context.Context, requestID string) (*ErasureRequest, error)
	GetByUserID(ctx context.Context, userID string) ([]*ErasureRequest, error)
	Update(ctx context.Context, request *ErasureRequest) error
}

// DataEraser defines the interface for erasing user data from various sources.
type DataEraser interface {
	Name() string
	EraseUserData(ctx context.Context, userID string) (int, error)
}

// NewErasureManager creates a new erasure manager.
func NewErasureManager(store ErasureStore, erasers []DataEraser) *ErasureManager {
	return &ErasureManager{
		store:       store,
		dataErasers: erasers,
	}
}

// CreateErasureRequest creates a new right to be forgotten request.
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

// ProcessErasureRequest processes a right to be forgotten request.
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

// CancelErasureRequest cancels a pending erasure request.
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

// GetErasureRequest retrieves an erasure request.
func (em *ErasureManager) GetErasureRequest(ctx context.Context, requestID string) (*ErasureRequest, error) {
	return em.store.Get(ctx, requestID)
}

// GetUserErasureRequests retrieves all erasure requests for a user.
func (em *ErasureManager) GetUserErasureRequests(ctx context.Context, userID string) ([]*ErasureRequest, error) {
	return em.store.GetByUserID(ctx, userID)
}

// BackupRecord is the minimal description of a backup artifact that the
// BackupDataEraser needs in order to erase it on behalf of a data subject.
type BackupRecord struct {
	ID       string // catalog / repository identifier of the backup record
	UserID   string // owning data subject
	Location string // filesystem path of the physical artifact (optional)
}

// BackupArtifactStore is the catalog / repository the BackupDataEraser deletes
// backup records from. A concrete implementation is provided by the server
// (e.g. backed by internal/repository), but any store satisfying this contract
// works. It is intentionally narrow so the eraser only depends on what it uses.
type BackupArtifactStore interface {
	// ListByUser returns every backup record owned by the given data subject.
	ListByUser(ctx context.Context, userID string) ([]BackupRecord, error)
	// Delete removes the backup record identified by id from the store.
	Delete(ctx context.Context, id string) error
}

// BackupDataEraser erases a data subject's backup artifacts and catalog records.
type BackupDataEraser struct {
	store   BackupArtifactStore
	baseDir string // optional root for physical artifacts; when set, files are removed too
}

// NewBackupDataEraser creates a BackupDataEraser.
//
// store is required: it is the catalog/repository the eraser deletes records
// from. baseDir is optional; when non-empty, the physical artifact referenced by
// each record's Location is also deleted from disk, but only if it resolves to a
// path inside baseDir (defense against path traversal).
func NewBackupDataEraser(store BackupArtifactStore, baseDir string) *BackupDataEraser {
	return &BackupDataEraser{store: store, baseDir: baseDir}
}

func (b *BackupDataEraser) Name() string {
	return "backups"
}

// EraseUserData deletes every backup record owned by userID, removing the
// underlying artifact file from disk when a baseDir is configured. It returns
// the number of backup records actually deleted from the store.
func (b *BackupDataEraser) EraseUserData(ctx context.Context, userID string) (int, error) {
	if b.store == nil {
		return 0, errors.New("backup eraser: no backup artifact store configured")
	}

	records, err := b.store.ListByUser(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("backup eraser: list backups for user %q: %w", userID, err)
	}

	deleted := 0
	var errs []string
	for _, rec := range records {
		// Remove the physical artifact first so we never leave dangling data
		// after the record (the index pointing at it) is gone.
		if b.baseDir != "" && rec.Location != "" {
			if err := removeFileWithinBase(b.baseDir, rec.Location); err != nil {
				errs = append(errs, fmt.Sprintf("artifact %s: %v", rec.ID, err))
				continue
			}
		}

		if err := b.store.Delete(ctx, rec.ID); err != nil {
			errs = append(errs, fmt.Sprintf("record %s: %v", rec.ID, err))
			continue
		}
		deleted++
	}

	if len(errs) > 0 {
		return deleted, fmt.Errorf("backup eraser: %d of %d records erased, errors: %s",
			deleted, len(records), strings.Join(errs, "; "))
	}
	return deleted, nil
}

// LogEntry is the minimal description of a log record referencing a data subject.
type LogEntry struct {
	ID     string
	UserID string
}

// LogStore is the audit/activity log backend the LogDataEraser removes entries
// from. The server supplies a concrete implementation; the eraser only needs to
// enumerate and delete a subject's entries.
type LogStore interface {
	// ListByUser returns every log entry referencing the given data subject.
	ListByUser(ctx context.Context, userID string) ([]LogEntry, error)
	// Delete removes the log entry identified by id.
	Delete(ctx context.Context, id string) error
}

// LogDataEraser erases (deletes) log entries referencing a data subject.
type LogDataEraser struct {
	store LogStore
}

// NewLogDataEraser creates a LogDataEraser backed by the given log store.
func NewLogDataEraser(store LogStore) *LogDataEraser {
	return &LogDataEraser{store: store}
}

func (l *LogDataEraser) Name() string {
	return "logs"
}

// EraseUserData deletes every log entry referencing userID and returns the
// number of entries actually removed.
func (l *LogDataEraser) EraseUserData(ctx context.Context, userID string) (int, error) {
	if l.store == nil {
		return 0, errors.New("log eraser: no log store configured")
	}

	entries, err := l.store.ListByUser(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("log eraser: list logs for user %q: %w", userID, err)
	}

	deleted := 0
	var errs []string
	for _, e := range entries {
		if err := l.store.Delete(ctx, e.ID); err != nil {
			errs = append(errs, fmt.Sprintf("entry %s: %v", e.ID, err))
			continue
		}
		deleted++
	}

	if len(errs) > 0 {
		return deleted, fmt.Errorf("log eraser: %d of %d entries erased, errors: %s",
			deleted, len(entries), strings.Join(errs, "; "))
	}
	return deleted, nil
}

// MetadataStore is the metadata backend the MetadataEraser removes records from.
type MetadataStore interface {
	// DeleteByUser removes all metadata records for the given data subject and
	// returns the number of records deleted.
	DeleteByUser(ctx context.Context, userID string) (int, error)
}

// MetadataEraser erases metadata records associated with a data subject.
type MetadataEraser struct {
	store MetadataStore
}

// NewMetadataEraser creates a MetadataEraser backed by the given metadata store.
func NewMetadataEraser(store MetadataStore) *MetadataEraser {
	return &MetadataEraser{store: store}
}

func (m *MetadataEraser) Name() string {
	return "metadata"
}

// EraseUserData deletes all metadata records for userID and returns the number
// of records actually deleted by the store.
func (m *MetadataEraser) EraseUserData(ctx context.Context, userID string) (int, error) {
	if m.store == nil {
		return 0, errors.New("metadata eraser: no metadata store configured")
	}

	deleted, err := m.store.DeleteByUser(ctx, userID)
	if err != nil {
		return deleted, fmt.Errorf("metadata eraser: delete metadata for user %q: %w", userID, err)
	}
	return deleted, nil
}

// removeFileWithinBase deletes target only if it resolves to a path inside
// baseDir, guarding against path-traversal so erasure can never reach outside
// the configured artifact root. A missing file is treated as already erased.
func removeFileWithinBase(baseDir, target string) error {
	absBase, err := filepath.Abs(baseDir)
	if err != nil {
		return fmt.Errorf("resolve base dir: %w", err)
	}
	absTarget, err := filepath.Abs(target)
	if err != nil {
		return fmt.Errorf("resolve target: %w", err)
	}

	rel, err := filepath.Rel(absBase, absTarget)
	if err != nil {
		return fmt.Errorf("relativise target: %w", err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return fmt.Errorf("refusing to delete %q: outside artifact root %q", target, baseDir)
	}

	if err := os.Remove(absTarget); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove artifact: %w", err)
	}
	return nil
}
