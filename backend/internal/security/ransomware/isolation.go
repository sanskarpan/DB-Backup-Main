// Package ransomware provides ransomware detection and protection mechanisms
package ransomware

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	pkgErrors "github.com/sanskarpan/db-backup/pkg/errors"
)

// IsolationAction represents an action taken during backup isolation
type IsolationAction string

const (
	// IsolationActionQuarantine moves backup to quarantine
	IsolationActionQuarantine IsolationAction = "QUARANTINE"
	// IsolationActionBlock prevents access to backup
	IsolationActionBlock IsolationAction = "BLOCK"
	// IsolationActionAlert sends notification only
	IsolationActionAlert IsolationAction = "ALERT"
	// IsolationActionDelete removes threatened backup
	IsolationActionDelete IsolationAction = "DELETE"
)

// IsolationPolicy defines how to handle threatened backups
type IsolationPolicy struct {
	// Enabled determines if isolation is active
	Enabled bool

	// DefaultAction is the default isolation action
	DefaultAction IsolationAction

	// QuarantinePath is where isolated backups are moved
	QuarantinePath string

	// AutoIsolateThresholds defines thresholds for automatic isolation
	AutoIsolateThresholds map[ThreatLevel]IsolationAction

	// PreserveIsolatedBackups determines if isolated backups are kept
	PreserveIsolatedBackups bool

	// IsolationRetentionDays is how long to keep isolated backups
	IsolationRetentionDays int

	// NotifyOnIsolation sends alerts when backups are isolated
	NotifyOnIsolation bool
}

// DefaultIsolationPolicy returns a default isolation policy
func DefaultIsolationPolicy(quarantinePath string) *IsolationPolicy {
	return &IsolationPolicy{
		Enabled:        true,
		DefaultAction:  IsolationActionQuarantine,
		QuarantinePath: quarantinePath,
		AutoIsolateThresholds: map[ThreatLevel]IsolationAction{
			ThreatLevelCritical: IsolationActionQuarantine,
			ThreatLevelHigh:     IsolationActionQuarantine,
			ThreatLevelMedium:   IsolationActionBlock,
			ThreatLevelLow:      IsolationActionAlert,
			ThreatLevelNone:     IsolationActionAlert,
		},
		PreserveIsolatedBackups: true,
		IsolationRetentionDays:  30,
		NotifyOnIsolation:       true,
	}
}

// IsolationManager manages backup isolation
type IsolationManager struct {
	policy   *IsolationPolicy
	detector *Detector
	mu       sync.RWMutex

	// Isolated backups tracking
	isolatedBackups map[string]*IsolatedBackup

	// Callbacks for notifications
	onIsolation func(*IsolatedBackup)
}

// IsolatedBackup represents a backup that has been isolated
type IsolatedBackup struct {
	OriginalPath   string
	QuarantinePath string
	ThreatReport   *ThreatReport
	IsolatedAt     time.Time
	Action         IsolationAction
	Restored       bool
	RestoredAt     *time.Time
}

// NewIsolationManager creates a new isolation manager
func NewIsolationManager(policy *IsolationPolicy, detector *Detector) *IsolationManager {
	if policy == nil {
		policy = DefaultIsolationPolicy("/var/quarantine")
	}

	return &IsolationManager{
		policy:          policy,
		detector:        detector,
		isolatedBackups: make(map[string]*IsolatedBackup),
	}
}

// SetNotificationCallback sets a callback for isolation events
func (im *IsolationManager) SetNotificationCallback(callback func(*IsolatedBackup)) {
	im.mu.Lock()
	defer im.mu.Unlock()
	im.onIsolation = callback
}

// ScanAndIsolate scans a backup and isolates if threats are detected
func (im *IsolationManager) ScanAndIsolate(ctx context.Context, backupPath string) (*ThreatReport, error) {
	if !im.policy.Enabled {
		return nil, nil
	}

	// Scan for threats
	report, err := im.detector.ScanFile(ctx, backupPath)
	if err != nil {
		return nil, pkgErrors.Wrap(err, pkgErrors.ErrorTypeStorage,
			"failed to scan backup for threats")
	}

	// Check if threat level requires isolation
	if report.ThreatLevel == ThreatLevelNone {
		return report, nil
	}

	// Determine isolation action based on threat level
	action, exists := im.policy.AutoIsolateThresholds[report.ThreatLevel]
	if !exists {
		action = im.policy.DefaultAction
	}

	// Execute isolation action
	isolated, err := im.executeIsolation(backupPath, report, action)
	if err != nil {
		return report, err
	}

	// Notify if callback is set
	if im.policy.NotifyOnIsolation && im.onIsolation != nil {
		go im.onIsolation(isolated)
	}

	return report, nil
}

// executeIsolation executes the isolation action
func (im *IsolationManager) executeIsolation(backupPath string, report *ThreatReport, action IsolationAction) (*IsolatedBackup, error) {
	im.mu.Lock()
	defer im.mu.Unlock()

	isolated := &IsolatedBackup{
		OriginalPath: backupPath,
		ThreatReport: report,
		IsolatedAt:   time.Now(),
		Action:       action,
	}

	switch action {
	case IsolationActionQuarantine:
		quarantinePath, err := im.moveToQuarantine(backupPath)
		if err != nil {
			return nil, err
		}
		isolated.QuarantinePath = quarantinePath

	case IsolationActionBlock:
		if err := im.blockAccess(backupPath); err != nil {
			return nil, err
		}

	case IsolationActionDelete:
		if err := os.Remove(backupPath); err != nil {
			return nil, pkgErrors.Wrap(err, pkgErrors.ErrorTypeStorage,
				"failed to delete threatened backup")
		}

	case IsolationActionAlert:
		// Alert only - no file system changes
	}

	// Track isolated backup
	im.isolatedBackups[backupPath] = isolated

	return isolated, nil
}

// moveToQuarantine moves a backup to quarantine directory
func (im *IsolationManager) moveToQuarantine(backupPath string) (string, error) {
	// Ensure quarantine directory exists
	if err := os.MkdirAll(im.policy.QuarantinePath, 0700); err != nil {
		return "", pkgErrors.Wrap(err, pkgErrors.ErrorTypeStorage,
			"failed to create quarantine directory")
	}

	// Generate quarantine filename with timestamp
	filename := filepath.Base(backupPath)
	timestamp := time.Now().Format("20060102_150405")
	quarantineFilename := fmt.Sprintf("%s_%s", timestamp, filename)
	quarantinePath := filepath.Join(im.policy.QuarantinePath, quarantineFilename)

	// Move file to quarantine
	if err := os.Rename(backupPath, quarantinePath); err != nil {
		// If rename fails (cross-device), try copy and delete
		if err := im.copyAndDelete(backupPath, quarantinePath); err != nil {
			return "", pkgErrors.Wrap(err, pkgErrors.ErrorTypeStorage,
				"failed to move backup to quarantine")
		}
	}

	// Set restrictive permissions on quarantined file
	if err := os.Chmod(quarantinePath, 0400); err != nil {
		return quarantinePath, pkgErrors.Wrap(err, pkgErrors.ErrorTypeStorage,
			"failed to set quarantine file permissions")
	}

	return quarantinePath, nil
}

// copyAndDelete copies a file and deletes the original
func (im *IsolationManager) copyAndDelete(src, dst string) error {
	// Read source file
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	// Write to destination
	if err := os.WriteFile(dst, data, 0400); err != nil {
		return err
	}

	// Delete source
	return os.Remove(src)
}

// blockAccess blocks access to a backup by setting restrictive permissions
func (im *IsolationManager) blockAccess(backupPath string) error {
	// Set file to no permissions (000)
	if err := os.Chmod(backupPath, 0000); err != nil {
		return pkgErrors.Wrap(err, pkgErrors.ErrorTypeStorage,
			"failed to block access to backup")
	}
	return nil
}

// RestoreFromQuarantine restores a backup from quarantine
func (im *IsolationManager) RestoreFromQuarantine(originalPath string, destinationPath string) error {
	im.mu.Lock()
	defer im.mu.Unlock()

	isolated, exists := im.isolatedBackups[originalPath]
	if !exists {
		return pkgErrors.New(pkgErrors.ErrorTypeNotFound,
			"backup not found in isolation records")
	}

	if isolated.Action != IsolationActionQuarantine {
		return pkgErrors.New(pkgErrors.ErrorTypeValidation,
			"backup was not quarantined, cannot restore")
	}

	// Move from quarantine back to destination
	if err := os.Rename(isolated.QuarantinePath, destinationPath); err != nil {
		return pkgErrors.Wrap(err, pkgErrors.ErrorTypeStorage,
			"failed to restore backup from quarantine")
	}

	// Restore normal permissions
	if err := os.Chmod(destinationPath, 0644); err != nil {
		return pkgErrors.Wrap(err, pkgErrors.ErrorTypeStorage,
			"failed to restore backup permissions")
	}

	// Mark as restored
	now := time.Now()
	isolated.Restored = true
	isolated.RestoredAt = &now

	return nil
}

// ListIsolatedBackups returns all isolated backups
func (im *IsolationManager) ListIsolatedBackups() []*IsolatedBackup {
	im.mu.RLock()
	defer im.mu.RUnlock()

	backups := make([]*IsolatedBackup, 0, len(im.isolatedBackups))
	for _, isolated := range im.isolatedBackups {
		backups = append(backups, isolated)
	}

	return backups
}

// GetIsolatedBackup retrieves an isolated backup by original path
func (im *IsolationManager) GetIsolatedBackup(originalPath string) (*IsolatedBackup, error) {
	im.mu.RLock()
	defer im.mu.RUnlock()

	isolated, exists := im.isolatedBackups[originalPath]
	if !exists {
		return nil, pkgErrors.New(pkgErrors.ErrorTypeNotFound,
			"isolated backup not found")
	}

	return isolated, nil
}

// CleanupOldQuarantine removes quarantined backups older than retention period
func (im *IsolationManager) CleanupOldQuarantine(ctx context.Context) error {
	im.mu.Lock()
	defer im.mu.Unlock()

	if !im.policy.PreserveIsolatedBackups || im.policy.IsolationRetentionDays <= 0 {
		return nil
	}

	cutoffTime := time.Now().AddDate(0, 0, -im.policy.IsolationRetentionDays)
	var toDelete []string

	for originalPath, isolated := range im.isolatedBackups {
		if isolated.Action != IsolationActionQuarantine {
			continue
		}

		if isolated.IsolatedAt.Before(cutoffTime) && !isolated.Restored {
			// Delete quarantined file
			if err := os.Remove(isolated.QuarantinePath); err != nil && !os.IsNotExist(err) {
				// Log error but continue
				continue
			}

			toDelete = append(toDelete, originalPath)
		}
	}

	// Remove from tracking
	for _, path := range toDelete {
		delete(im.isolatedBackups, path)
	}

	return nil
}

// IsBackupIsolated checks if a backup is currently isolated
func (im *IsolationManager) IsBackupIsolated(backupPath string) bool {
	im.mu.RLock()
	defer im.mu.RUnlock()

	isolated, exists := im.isolatedBackups[backupPath]
	return exists && !isolated.Restored
}

// GetIsolationStats returns statistics about isolated backups
func (im *IsolationManager) GetIsolationStats() *IsolationStats {
	im.mu.RLock()
	defer im.mu.RUnlock()

	stats := &IsolationStats{
		TotalIsolated: len(im.isolatedBackups),
		ByAction:      make(map[IsolationAction]int),
		ByThreatLevel: make(map[ThreatLevel]int),
		Restored:      0,
	}

	for _, isolated := range im.isolatedBackups {
		stats.ByAction[isolated.Action]++
		stats.ByThreatLevel[isolated.ThreatReport.ThreatLevel]++

		if isolated.Restored {
			stats.Restored++
		}
	}

	return stats
}

// IsolationStats represents statistics about isolated backups
type IsolationStats struct {
	TotalIsolated int
	ByAction      map[IsolationAction]int
	ByThreatLevel map[ThreatLevel]int
	Restored      int
}
