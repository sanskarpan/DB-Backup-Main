package gamification

import (
	"fmt"
	"math"
	"sync"
	"time"
)

// ============================================================================
// Streak Manager
// ============================================================================

// StreakManager manages backup streaks
type StreakManager struct {
	streaks map[string]*StreakData
	mu      sync.RWMutex
}

// StreakData holds streak information for a user
type StreakData struct {
	Current       int
	Longest       int
	LastBackupDate time.Time
	StartDate     time.Time
}

// NewStreakManager creates a new streak manager
func NewStreakManager() *StreakManager {
	return &StreakManager{
		streaks: make(map[string]*StreakData),
	}
}

// UpdateStreak updates a user's streak
func (sm *StreakManager) UpdateStreak(userID string, backupTime time.Time) int {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.streaks[userID] == nil {
		sm.streaks[userID] = &StreakData{
			Current:       1,
			Longest:       1,
			LastBackupDate: backupTime,
			StartDate:     backupTime,
		}
		return 1
	}

	streak := sm.streaks[userID]

	// Check if this is the same day
	if isSameDay(streak.LastBackupDate, backupTime) {
		return streak.Current
	}

	// Check if this is the next day
	if isNextDay(streak.LastBackupDate, backupTime) {
		streak.Current++
		streak.LastBackupDate = backupTime
		if streak.Current > streak.Longest {
			streak.Longest = streak.Current
		}
	} else {
		// Streak broken, reset
		streak.Current = 1
		streak.StartDate = backupTime
		streak.LastBackupDate = backupTime
	}

	return streak.Current
}

// GetStreak returns the current streak
func (sm *StreakManager) GetStreak(userID string) int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if streak, exists := sm.streaks[userID]; exists {
		// Check if streak is still valid (within last 24 hours)
		if time.Since(streak.LastBackupDate) > 48*time.Hour {
			return 0 // Streak expired
		}
		return streak.Current
	}
	return 0
}

// GetLongestStreak returns the longest streak achieved
func (sm *StreakManager) GetLongestStreak(userID string) int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if streak, exists := sm.streaks[userID]; exists {
		return streak.Longest
	}
	return 0
}

// Helper functions for date comparison
func isSameDay(t1, t2 time.Time) bool {
	y1, m1, d1 := t1.Date()
	y2, m2, d2 := t2.Date()
	return y1 == y2 && m1 == m2 && d1 == d2
}

func isNextDay(t1, t2 time.Time) bool {
	nextDay := t1.AddDate(0, 0, 1)
	return isSameDay(nextDay, t2)
}

// ============================================================================
// XP and Level Manager
// ============================================================================

// XPManager manages experience points and levels
type XPManager struct {
	userXP map[string]*XPData
	mu     sync.RWMutex
}

// XPData holds XP information for a user
type XPData struct {
	TotalXP      int
	CurrentLevel int
}

// LevelUpResult contains level up information
type LevelUpResult struct {
	CurrentLevel int
	LevelsGained int
	TotalXP      int
}

// NewXPManager creates a new XP manager
func NewXPManager() *XPManager {
	return &XPManager{
		userXP: make(map[string]*XPData),
	}
}

// AddXP adds XP to a user and handles level ups
func (xm *XPManager) AddXP(userID string, xp int) *LevelUpResult {
	xm.mu.Lock()
	defer xm.mu.Unlock()

	if xm.userXP[userID] == nil {
		xm.userXP[userID] = &XPData{
			TotalXP:      0,
			CurrentLevel: 1,
		}
	}

	data := xm.userXP[userID]
	oldLevel := data.CurrentLevel
	data.TotalXP += xp

	// Calculate new level
	newLevel := xm.calculateLevel(data.TotalXP)
	data.CurrentLevel = newLevel

	return &LevelUpResult{
		CurrentLevel: newLevel,
		LevelsGained: newLevel - oldLevel,
		TotalXP:      data.TotalXP,
	}
}

// calculateLevel calculates level from total XP
// Uses exponential curve: XP = 100 * (level^1.5)
func (xm *XPManager) calculateLevel(totalXP int) int {
	if totalXP < 100 {
		return 1
	}

	// Binary search for level
	level := 1
	for {
		requiredXP := xm.getXPForLevel(level + 1)
		if totalXP < requiredXP {
			return level
		}
		level++
		if level > 1000 { // Safety cap
			return 1000
		}
	}
}

// getXPForLevel returns XP required to reach a level
func (xm *XPManager) getXPForLevel(level int) int {
	if level <= 1 {
		return 0
	}
	return int(100 * math.Pow(float64(level), 1.5))
}

// GetLevel returns current level
func (xm *XPManager) GetLevel(userID string) int {
	xm.mu.RLock()
	defer xm.mu.RUnlock()

	if data, exists := xm.userXP[userID]; exists {
		return data.CurrentLevel
	}
	return 1
}

// GetXP returns total XP
func (xm *XPManager) GetXP(userID string) int {
	xm.mu.RLock()
	defer xm.mu.RUnlock()

	if data, exists := xm.userXP[userID]; exists {
		return data.TotalXP
	}
	return 0
}

// GetXPToNextLevel returns XP needed to reach next level
func (xm *XPManager) GetXPToNextLevel(userID string) int {
	xm.mu.RLock()
	defer xm.mu.RUnlock()

	if data, exists := xm.userXP[userID]; exists {
		nextLevelXP := xm.getXPForLevel(data.CurrentLevel + 1)
		return nextLevelXP - data.TotalXP
	}
	return 100 // XP to level 2
}

// ============================================================================
// Stats Manager
// ============================================================================

// StatsManager tracks user statistics
type StatsManager struct {
	stats map[string]*UserStats
	mu    sync.RWMutex
}

// UserStats contains detailed user statistics
type UserStats struct {
	TotalBackups      int
	SuccessfulBackups int
	FailedBackups     int
	TotalSize         int64
	TotalDuration     int64
	EncryptedBackups  int
	CurrentStreak     int
	LongestStreak     int
	DatabaseTypes     map[string]int
	AverageSize       int64
	AverageDuration   int64
	FastestBackup     int64
	LargestBackup     int64
	PerfectMonths     int
	LastBackupTime    time.Time
	FirstBackupTime   time.Time
}

// NewStatsManager creates a new stats manager
func NewStatsManager() *StatsManager {
	return &StatsManager{
		stats: make(map[string]*UserStats),
	}
}

// RecordBackup records a backup in statistics
func (sm *StatsManager) RecordBackup(userID string, data *BackupData) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.stats[userID] == nil {
		sm.stats[userID] = &UserStats{
			DatabaseTypes:   make(map[string]int),
			FirstBackupTime: time.Now(),
		}
	}

	stats := sm.stats[userID]
	stats.TotalBackups++
	stats.SuccessfulBackups++
	stats.TotalSize += data.Size
	stats.TotalDuration += data.Duration
	stats.LastBackupTime = time.Now()

	if data.Encrypted {
		stats.EncryptedBackups++
	}

	// Track database types
	if data.DatabaseType != "" {
		stats.DatabaseTypes[data.DatabaseType]++
	}

	// Update records
	if stats.FastestBackup == 0 || data.Duration < stats.FastestBackup {
		stats.FastestBackup = data.Duration
	}

	if data.Size > stats.LargestBackup {
		stats.LargestBackup = data.Size
	}

	// Calculate averages
	stats.AverageSize = stats.TotalSize / int64(stats.TotalBackups)
	stats.AverageDuration = stats.TotalDuration / int64(stats.TotalBackups)
}

// GetStats returns user statistics
func (sm *StatsManager) GetStats(userID string) *UserStats {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if stats, exists := sm.stats[userID]; exists {
		return stats
	}

	return &UserStats{
		DatabaseTypes: make(map[string]int),
	}
}

// GetTotalBackups returns total backups count
func (sm *StatsManager) GetTotalBackups(userID string) int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if stats, exists := sm.stats[userID]; exists {
		return stats.TotalBackups
	}
	return 0
}

// GetTotalSize returns total size backed up
func (sm *StatsManager) GetTotalSize(userID string) int64 {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if stats, exists := sm.stats[userID]; exists {
		return stats.TotalSize
	}
	return 0
}

// GetPersonalInsights generates personalized insights
func (sm *StatsManager) GetPersonalInsights(userID string) *PersonalInsights {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	stats := sm.GetStats(userID)

	insights := &PersonalInsights{
		Insights: make([]string, 0),
		Tips:     make([]string, 0),
	}

	// Generate insights based on stats
	if stats.TotalBackups > 0 {
		successRate := float64(stats.SuccessfulBackups) / float64(stats.TotalBackups) * 100
		insights.Insights = append(insights.Insights,
			fmt.Sprintf("Your backup success rate is %.1f%%", successRate))

		if stats.AverageDuration > 0 {
			insights.Insights = append(insights.Insights,
				fmt.Sprintf("Your average backup takes %d seconds", stats.AverageDuration))
		}

		if len(stats.DatabaseTypes) > 0 {
			insights.Insights = append(insights.Insights,
				fmt.Sprintf("You've backed up %d different database types", len(stats.DatabaseTypes)))
		}

		// Generate tips
		if successRate < 90 {
			insights.Tips = append(insights.Tips,
				"Consider reviewing failed backups to improve success rate")
		}

		if stats.EncryptedBackups < stats.TotalBackups/2 {
			insights.Tips = append(insights.Tips,
				"Enable encryption for better data security")
		}

		if stats.CurrentStreak < 7 {
			insights.Tips = append(insights.Tips,
				"Maintain daily backups to build your streak!")
		}
	}

	return insights
}

// PersonalInsights contains personalized insights for a user
type PersonalInsights struct {
	Insights []string
	Tips     []string
}
