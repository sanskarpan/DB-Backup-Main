// Package retention provides automated backup retention policy management
package retention

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/sanskarpan/db-backup/pkg/errors"
)

// Policy defines backup retention rules
type Policy struct {
	// DailyRetention: number of daily backups to keep (most recent N days)
	DailyRetention int
	// WeeklyRetention: number of weekly backups to keep (most recent N weeks)
	WeeklyRetention int
	// MonthlyRetention: number of monthly backups to keep (most recent N months)
	MonthlyRetention int
	// YearlyRetention: number of yearly backups to keep (most recent N years)
	YearlyRetention int
}

// Backup represents a backup for retention purposes
type Backup struct {
	ID        string
	Timestamp time.Time
	Size      int64
	Tags      map[string]string
}

// BackupStore interface for backup storage operations
type BackupStore interface {
	// ListBackups returns all backups
	ListBackups(ctx context.Context) ([]*Backup, error)
	// DeleteBackup deletes a backup by ID
	DeleteBackup(ctx context.Context, id string) error
}

// Manager manages backup retention policies
type Manager struct {
	policy *Policy
	store  BackupStore
}

// NewManager creates a new retention manager
func NewManager(policy *Policy, store BackupStore) *Manager {
	return &Manager{
		policy: policy,
		store:  store,
	}
}

// ApplyPolicy applies retention policy and deletes old backups
func (m *Manager) ApplyPolicy(ctx context.Context) (*ApplyResult, error) {
	// Get all backups
	allBackups, err := m.store.ListBackups(ctx)
	if err != nil {
		return nil, errors.ErrRetentionPolicy(fmt.Errorf("failed to list backups: %w", err))
	}

	// Sort backups by timestamp (newest first)
	sort.Slice(allBackups, func(i, j int) bool {
		return allBackups[i].Timestamp.After(allBackups[j].Timestamp)
	})

	// Determine which backups to keep
	toKeep := m.selectBackupsToKeep(allBackups)
	toDelete := m.findBackupsToDelete(allBackups, toKeep)

	// Delete old backups
	result := &ApplyResult{
		TotalBackups:   len(allBackups),
		BackupsKept:    len(toKeep),
		BackupsDeleted: 0,
		SpaceFreed:     0,
		DeletedIDs:     []string{},
		Errors:         []error{},
	}

	for _, backup := range toDelete {
		if err := m.store.DeleteBackup(ctx, backup.ID); err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("failed to delete backup %s: %w", backup.ID, err))
			continue
		}
		result.BackupsDeleted++
		result.SpaceFreed += backup.Size
		result.DeletedIDs = append(result.DeletedIDs, backup.ID)
	}

	return result, nil
}

// selectBackupsToKeep determines which backups to keep based on policy
func (m *Manager) selectBackupsToKeep(backups []*Backup) map[string]bool {
	toKeep := make(map[string]bool)
	now := time.Now()

	// Sort backups by timestamp (newest first)
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].Timestamp.After(backups[j].Timestamp)
	})

	// Keep daily backups (most recent N days)
	if m.policy.DailyRetention > 0 {
		dailyCutoff := now.AddDate(0, 0, -m.policy.DailyRetention)
		for _, backup := range backups {
			if backup.Timestamp.After(dailyCutoff) {
				toKeep[backup.ID] = true
			}
		}
	}

	// Keep weekly backups (one per week for N weeks)
	if m.policy.WeeklyRetention > 0 {
		weeklyCutoff := now.AddDate(0, 0, -m.policy.WeeklyRetention*7)
		weeklyBackups := m.selectWeeklyBackups(backups, weeklyCutoff, m.policy.WeeklyRetention)
		for id := range weeklyBackups {
			toKeep[id] = true
		}
	}

	// Keep monthly backups (one per month for N months)
	if m.policy.MonthlyRetention > 0 {
		monthlyCutoff := now.AddDate(0, -m.policy.MonthlyRetention, 0)
		monthlyBackups := m.selectMonthlyBackups(backups, monthlyCutoff, m.policy.MonthlyRetention)
		for id := range monthlyBackups {
			toKeep[id] = true
		}
	}

	// Keep yearly backups (one per year for N years)
	if m.policy.YearlyRetention > 0 {
		yearlyCutoff := now.AddDate(-m.policy.YearlyRetention, 0, 0)
		yearlyBackups := m.selectYearlyBackups(backups, yearlyCutoff, m.policy.YearlyRetention)
		for id := range yearlyBackups {
			toKeep[id] = true
		}
	}

	return toKeep
}

// selectWeeklyBackups selects one backup per week
func (m *Manager) selectWeeklyBackups(backups []*Backup, cutoff time.Time, count int) map[string]bool {
	selected := make(map[string]bool)
	weeksSeen := make(map[string]bool)

	for _, backup := range backups {
		if backup.Timestamp.Before(cutoff) {
			continue
		}

		// Get week identifier (year + ISO week number)
		year, week := backup.Timestamp.ISOWeek()
		weekKey := fmt.Sprintf("%d-W%02d", year, week)

		if !weeksSeen[weekKey] {
			selected[backup.ID] = true
			weeksSeen[weekKey] = true

			if len(weeksSeen) >= count {
				break
			}
		}
	}

	return selected
}

// selectMonthlyBackups selects one backup per month
func (m *Manager) selectMonthlyBackups(backups []*Backup, cutoff time.Time, count int) map[string]bool {
	selected := make(map[string]bool)
	monthsSeen := make(map[string]bool)

	for _, backup := range backups {
		if backup.Timestamp.Before(cutoff) {
			continue
		}

		// Get month identifier (year + month)
		monthKey := fmt.Sprintf("%d-%02d", backup.Timestamp.Year(), backup.Timestamp.Month())

		if !monthsSeen[monthKey] {
			selected[backup.ID] = true
			monthsSeen[monthKey] = true

			if len(monthsSeen) >= count {
				break
			}
		}
	}

	return selected
}

// selectYearlyBackups selects one backup per year
func (m *Manager) selectYearlyBackups(backups []*Backup, cutoff time.Time, count int) map[string]bool {
	selected := make(map[string]bool)
	yearsSeen := make(map[string]bool)

	for _, backup := range backups {
		if backup.Timestamp.Before(cutoff) {
			continue
		}

		// Get year identifier
		yearKey := fmt.Sprintf("%d", backup.Timestamp.Year())

		if !yearsSeen[yearKey] {
			selected[backup.ID] = true
			yearsSeen[yearKey] = true

			if len(yearsSeen) >= count {
				break
			}
		}
	}

	return selected
}

// findBackupsToDelete returns backups that should be deleted
func (m *Manager) findBackupsToDelete(allBackups []*Backup, toKeep map[string]bool) []*Backup {
	var toDelete []*Backup

	for _, backup := range allBackups {
		if !toKeep[backup.ID] {
			toDelete = append(toDelete, backup)
		}
	}

	return toDelete
}

// ApplyResult contains the results of applying a retention policy
type ApplyResult struct {
	TotalBackups   int
	BackupsKept    int
	BackupsDeleted int
	SpaceFreed     int64
	DeletedIDs     []string
	Errors         []error
}

// Summary returns a human-readable summary
func (r *ApplyResult) Summary() string {
	if len(r.Errors) > 0 {
		return fmt.Sprintf("Retention policy applied: %d/%d backups deleted (%d kept), %.2f MB freed, %d errors",
			r.BackupsDeleted, r.TotalBackups, r.BackupsKept,
			float64(r.SpaceFreed)/(1024*1024), len(r.Errors))
	}

	return fmt.Sprintf("Retention policy applied: %d/%d backups deleted (%d kept), %.2f MB freed",
		r.BackupsDeleted, r.TotalBackups, r.BackupsKept,
		float64(r.SpaceFreed)/(1024*1024))
}

// DefaultPolicy returns a sensible default retention policy
func DefaultPolicy() *Policy {
	return &Policy{
		DailyRetention:   7,  // Keep 7 days
		WeeklyRetention:  4,  // Keep 4 weeks
		MonthlyRetention: 12, // Keep 12 months
		YearlyRetention:  2,  // Keep 2 years
	}
}

// AggressivePolicy returns a more aggressive (less storage) retention policy
func AggressivePolicy() *Policy {
	return &Policy{
		DailyRetention:   3,  // Keep 3 days
		WeeklyRetention:  2,  // Keep 2 weeks
		MonthlyRetention: 6,  // Keep 6 months
		YearlyRetention:  1,  // Keep 1 year
	}
}

// ConservativePolicy returns a more conservative (more storage) retention policy
func ConservativePolicy() *Policy {
	return &Policy{
		DailyRetention:   14, // Keep 14 days
		WeeklyRetention:  8,  // Keep 8 weeks
		MonthlyRetention: 24, // Keep 24 months
		YearlyRetention:  5,  // Keep 5 years
	}
}
