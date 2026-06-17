// Package security provides security features for backup system
package security

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sanskarpan/db-backup/pkg/uid"
)

// AirGapManager manages air-gapped backup copies
type AirGapManager struct {
	mu               sync.RWMutex
	airGapCopies     map[string]*AirGapBackup
	storage          AirGapStorage
	verificationFreq time.Duration
	retentionDays    int
}

// AirGapBackup represents an air-gapped backup copy
type AirGapBackup struct {
	ID                string
	OriginalBackupID  string
	DatabaseName      string
	CreatedAt         time.Time
	Size              int64
	Checksum          string
	StorageLocation   string
	IsolationType     IsolationType
	VerifiedAt        time.Time
	VerificationState VerificationState
	Metadata          map[string]string
	RetainUntil       time.Time
	Immutable         bool
	EncryptionKey     string // Encrypted separately
	AccessLog         []AccessLogEntry
}

// IsolationType represents the type of air-gap isolation
type IsolationType string

const (
	IsolationTypePhysical     IsolationType = "physical"      // Physically disconnected storage
	IsolationTypeLogical      IsolationType = "logical"       // Logically isolated network segment
	IsolationTypeWriteOnce    IsolationType = "write_once"    // WORM (Write Once Read Many)
	IsolationTypeOfflineTape  IsolationType = "offline_tape"  // Offline tape storage
	IsolationTypeImmutable    IsolationType = "immutable"     // Cloud immutable storage
	IsolationTypeVaultLock    IsolationType = "vault_lock"    // AWS Glacier Vault Lock
)

// VerificationState represents the verification state of an air-gapped backup
type VerificationState string

const (
	VerificationStatePending  VerificationState = "pending"
	VerificationStateVerified VerificationState = "verified"
	VerificationStateFailed   VerificationState = "failed"
	VerificationStateCorrupted VerificationState = "corrupted"
)

// AccessLogEntry represents an access log entry
type AccessLogEntry struct {
	Timestamp   time.Time
	Action      string
	User        string
	IPAddress   string
	Success     bool
	Reason      string
}

// AirGapStorage interface for different air-gap storage backends
type AirGapStorage interface {
	Store(ctx context.Context, backup *AirGapBackup, data []byte) error
	Retrieve(ctx context.Context, backupID string) ([]byte, error)
	Verify(ctx context.Context, backupID string) (bool, error)
	Delete(ctx context.Context, backupID string) error
	List(ctx context.Context) ([]*AirGapBackup, error)
}

// NewAirGapManager creates a new air-gap manager
func NewAirGapManager(storage AirGapStorage, verificationFreq time.Duration, retentionDays int) *AirGapManager {
	return &AirGapManager{
		airGapCopies:     make(map[string]*AirGapBackup),
		storage:          storage,
		verificationFreq: verificationFreq,
		retentionDays:    retentionDays,
	}
}

// CreateAirGapCopy creates an air-gapped copy of a backup
func (agm *AirGapManager) CreateAirGapCopy(ctx context.Context, originalBackupID, databaseName string, data []byte, isolationType IsolationType) (*AirGapBackup, error) {
	// Calculate checksum
	checksum := calculateChecksum(data)

	// Create air-gap backup record
	airGapBackup := &AirGapBackup{
		ID:               generateAirGapID(),
		OriginalBackupID: originalBackupID,
		DatabaseName:     databaseName,
		CreatedAt:        time.Now(),
		Size:             int64(len(data)),
		Checksum:         checksum,
		IsolationType:    isolationType,
		VerificationState: VerificationStatePending,
		Metadata:         make(map[string]string),
		RetainUntil:      time.Now().AddDate(0, 0, agm.retentionDays),
		Immutable:        true, // Air-gapped backups are always immutable
		AccessLog:        make([]AccessLogEntry, 0),
	}

	// Store the backup in air-gapped storage
	if err := agm.storage.Store(ctx, airGapBackup, data); err != nil {
		return nil, fmt.Errorf("failed to store air-gap copy: %w", err)
	}

	// Immediate verification
	verified, err := agm.storage.Verify(ctx, airGapBackup.ID)
	if err != nil {
		airGapBackup.VerificationState = VerificationStateFailed
	} else if verified {
		airGapBackup.VerificationState = VerificationStateVerified
		airGapBackup.VerifiedAt = time.Now()
	} else {
		airGapBackup.VerificationState = VerificationStateCorrupted
	}

	// Log creation
	airGapBackup.AddAccessLog("CREATE", "system", "", true, "Air-gap copy created")

	// Store in memory
	agm.mu.Lock()
	agm.airGapCopies[airGapBackup.ID] = airGapBackup
	agm.mu.Unlock()

	return airGapBackup, nil
}

// VerifyAirGapCopy verifies an air-gapped backup
func (agm *AirGapManager) VerifyAirGapCopy(ctx context.Context, backupID string) (bool, error) {
	agm.mu.RLock()
	backup, exists := agm.airGapCopies[backupID]
	agm.mu.RUnlock()

	if !exists {
		return false, fmt.Errorf("air-gap backup not found: %s", backupID)
	}

	// Verify with storage backend
	verified, err := agm.storage.Verify(ctx, backupID)
	if err != nil {
		backup.VerificationState = VerificationStateFailed
		backup.AddAccessLog("VERIFY", "system", "", false, fmt.Sprintf("Verification failed: %v", err))
		return false, err
	}

	if verified {
		backup.VerificationState = VerificationStateVerified
		backup.VerifiedAt = time.Now()
		backup.AddAccessLog("VERIFY", "system", "", true, "Verification successful")
	} else {
		backup.VerificationState = VerificationStateCorrupted
		backup.AddAccessLog("VERIFY", "system", "", false, "Checksum mismatch - backup corrupted")
	}

	return verified, nil
}

// RetrieveAirGapCopy retrieves an air-gapped backup
func (agm *AirGapManager) RetrieveAirGapCopy(ctx context.Context, backupID, user, ipAddress string) ([]byte, error) {
	agm.mu.RLock()
	backup, exists := agm.airGapCopies[backupID]
	agm.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("air-gap backup not found: %s", backupID)
	}

	// Verify before retrieval
	verified, verifyErr := agm.storage.Verify(ctx, backupID)
	if verifyErr != nil || !verified {
		backup.AddAccessLog("RETRIEVE", user, ipAddress, false, "Verification failed before retrieval")
		return nil, fmt.Errorf("backup verification failed: %w", verifyErr)
	}

	// Retrieve data
	data, err := agm.storage.Retrieve(ctx, backupID)
	if err != nil {
		backup.AddAccessLog("RETRIEVE", user, ipAddress, false, fmt.Sprintf("Retrieval failed: %v", err))
		return nil, fmt.Errorf("failed to retrieve air-gap copy: %w", err)
	}

	// Verify checksum
	checksum := calculateChecksum(data)
	if checksum != backup.Checksum {
		backup.VerificationState = VerificationStateCorrupted
		backup.AddAccessLog("RETRIEVE", user, ipAddress, false, "Checksum mismatch after retrieval")
		return nil, fmt.Errorf("checksum mismatch: backup may be corrupted")
	}

	// Log successful retrieval
	backup.AddAccessLog("RETRIEVE", user, ipAddress, true, "Air-gap copy retrieved successfully")

	return data, nil
}

// ListAirGapBackups lists all air-gapped backups
func (agm *AirGapManager) ListAirGapBackups(databaseName string) []*AirGapBackup {
	agm.mu.RLock()
	defer agm.mu.RUnlock()

	backups := make([]*AirGapBackup, 0)
	for _, backup := range agm.airGapCopies {
		if databaseName == "" || backup.DatabaseName == databaseName {
			backups = append(backups, backup)
		}
	}

	return backups
}

// CleanupExpiredBackups removes expired air-gapped backups
func (agm *AirGapManager) CleanupExpiredBackups(ctx context.Context) (int, error) {
	agm.mu.Lock()
	defer agm.mu.Unlock()

	now := time.Now()
	removed := 0

	for id, backup := range agm.airGapCopies {
		if now.After(backup.RetainUntil) {
			// Delete from storage
			if err := agm.storage.Delete(ctx, id); err != nil {
				backup.AddAccessLog("DELETE", "system", "", false, fmt.Sprintf("Cleanup failed: %v", err))
				continue
			}

			// Remove from memory
			delete(agm.airGapCopies, id)
			removed++
		}
	}

	return removed, nil
}

// VerifyAllBackups verifies all air-gapped backups
func (agm *AirGapManager) VerifyAllBackups(ctx context.Context) (verified, failed int, errors []error) {
	agm.mu.RLock()
	backups := make([]*AirGapBackup, 0, len(agm.airGapCopies))
	for _, backup := range agm.airGapCopies {
		backups = append(backups, backup)
	}
	agm.mu.RUnlock()

	for _, backup := range backups {
		ok, err := agm.VerifyAirGapCopy(ctx, backup.ID)
		if err != nil {
			failed++
			errors = append(errors, fmt.Errorf("backup %s: %w", backup.ID, err))
		} else if ok {
			verified++
		} else {
			failed++
			errors = append(errors, fmt.Errorf("backup %s: verification failed", backup.ID))
		}
	}

	return verified, failed, errors
}

// GetStatistics returns statistics about air-gapped backups
func (agm *AirGapManager) GetStatistics() map[string]interface{} {
	agm.mu.RLock()
	defer agm.mu.RUnlock()

	stats := map[string]interface{}{
		"total_backups":     len(agm.airGapCopies),
		"total_size":        int64(0),
		"by_database":       make(map[string]int),
		"by_isolation_type": make(map[IsolationType]int),
		"by_state":          make(map[VerificationState]int),
		"expired":           0,
	}

	byDatabase := make(map[string]int)
	byIsolationType := make(map[IsolationType]int)
	byState := make(map[VerificationState]int)
	totalSize := int64(0)
	expired := 0
	now := time.Now()

	for _, backup := range agm.airGapCopies {
		totalSize += backup.Size
		byDatabase[backup.DatabaseName]++
		byIsolationType[backup.IsolationType]++
		byState[backup.VerificationState]++
		if now.After(backup.RetainUntil) {
			expired++
		}
	}

	stats["total_size"] = totalSize
	stats["by_database"] = byDatabase
	stats["by_isolation_type"] = byIsolationType
	stats["by_state"] = byState
	stats["expired"] = expired

	return stats
}

// AddAccessLog adds an access log entry
func (agb *AirGapBackup) AddAccessLog(action, user, ipAddress string, success bool, reason string) {
	entry := AccessLogEntry{
		Timestamp: time.Now(),
		Action:    action,
		User:      user,
		IPAddress: ipAddress,
		Success:   success,
		Reason:    reason,
	}
	agb.AccessLog = append(agb.AccessLog, entry)

	// Keep only last 100 entries
	if len(agb.AccessLog) > 100 {
		agb.AccessLog = agb.AccessLog[len(agb.AccessLog)-100:]
	}
}

// MockAirGapStorage is a mock implementation for testing
type MockAirGapStorage struct {
	mu      sync.RWMutex
	backups map[string][]byte
}

// NewMockAirGapStorage creates a new mock air-gap storage
func NewMockAirGapStorage() *MockAirGapStorage {
	return &MockAirGapStorage{
		backups: make(map[string][]byte),
	}
}

func (mas *MockAirGapStorage) Store(ctx context.Context, backup *AirGapBackup, data []byte) error {
	mas.mu.Lock()
	defer mas.mu.Unlock()

	// Simulate storage delay
	time.Sleep(10 * time.Millisecond)

	mas.backups[backup.ID] = data
	backup.StorageLocation = fmt.Sprintf("mock://%s", backup.ID)

	return nil
}

func (mas *MockAirGapStorage) Retrieve(ctx context.Context, backupID string) ([]byte, error) {
	mas.mu.RLock()
	defer mas.mu.RUnlock()

	// Simulate retrieval delay
	time.Sleep(10 * time.Millisecond)

	data, exists := mas.backups[backupID]
	if !exists {
		return nil, fmt.Errorf("backup not found")
	}

	return data, nil
}

func (mas *MockAirGapStorage) Verify(ctx context.Context, backupID string) (bool, error) {
	mas.mu.RLock()
	defer mas.mu.RUnlock()

	// Simulate verification delay
	time.Sleep(5 * time.Millisecond)

	_, exists := mas.backups[backupID]
	return exists, nil
}

func (mas *MockAirGapStorage) Delete(ctx context.Context, backupID string) error {
	mas.mu.Lock()
	defer mas.mu.Unlock()

	delete(mas.backups, backupID)
	return nil
}

func (mas *MockAirGapStorage) List(ctx context.Context) ([]*AirGapBackup, error) {
	mas.mu.RLock()
	defer mas.mu.RUnlock()

	backups := make([]*AirGapBackup, 0)
	// This would normally return full backup metadata
	return backups, nil
}

// Helper functions

func generateAirGapID() string {
	return uid.New("airgap")
}

func calculateChecksum(data []byte) string {
	// In production, use SHA-256 or similar
	// For now, simple checksum based on length and first/last bytes
	if len(data) == 0 {
		return "empty"
	}
	return fmt.Sprintf("checksum-%d-%x-%x", len(data), data[0], data[len(data)-1])
}
