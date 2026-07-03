// Package gamification provides gamification and engagement features
package gamification

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Engine is the core gamification engine.
type Engine struct {
	achievementMgr *AchievementManager
	streakMgr      *StreakManager
	xpMgr          *XPManager
	leaderboardMgr *LeaderboardManager
	challengeMgr   *ChallengeManager
	rewardMgr      *RewardManager
	statsMgr       *StatsManager
	notifier       EventNotifier
	mu             sync.RWMutex
}

// Config holds gamification engine configuration.
type Config struct {
	EnableAchievements  bool
	EnableStreaks       bool
	EnableXP            bool
	EnableLeaderboards  bool
	EnableChallenges    bool
	EnableRewards       bool
	NotifyOnAchievement bool
}

// EventNotifier sends notifications for gamification events.
type EventNotifier interface {
	NotifyAchievement(ctx context.Context, userID string, achievement *Achievement) error
	NotifyLevelUp(ctx context.Context, userID string, level int) error
	NotifyStreakMilestone(ctx context.Context, userID string, streak int) error
	NotifyReward(ctx context.Context, userID string, reward *Reward) error
}

// NewEngine creates a new gamification engine.
func NewEngine(cfg *Config, notifier EventNotifier) *Engine {
	return &Engine{
		achievementMgr: NewAchievementManager(),
		streakMgr:      NewStreakManager(),
		xpMgr:          NewXPManager(),
		leaderboardMgr: NewLeaderboardManager(),
		challengeMgr:   NewChallengeManager(),
		rewardMgr:      NewRewardManager(),
		statsMgr:       NewStatsManager(),
		notifier:       notifier,
	}
}

// RecordBackup records a backup completion event.
func (e *Engine) RecordBackup(ctx context.Context, userID string, backupData *BackupData) (*Result, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	result := &Result{
		UserID:        userID,
		Timestamp:     time.Now(),
		Achievements:  make([]*Achievement, 0),
		XPGained:      0,
		LevelUps:      0,
		RewardsEarned: make([]*Reward, 0),
	}

	// Update stats
	e.statsMgr.RecordBackup(userID, backupData)

	// Check and update streak
	streak := e.streakMgr.UpdateStreak(userID, time.Now())
	result.CurrentStreak = streak

	// Award XP
	xp := e.calculateXP(backupData)
	levelUpResult := e.xpMgr.AddXP(userID, xp)
	result.XPGained = xp
	result.LevelUps = levelUpResult.LevelsGained
	result.CurrentLevel = levelUpResult.CurrentLevel

	// Check achievements
	newAchievements := e.achievementMgr.CheckAchievements(userID, backupData, e.statsMgr.GetStats(userID))
	result.Achievements = newAchievements

	// Update leaderboards
	e.leaderboardMgr.UpdateScore(userID, backupData)

	// Check active challenges
	completedChallenges := e.challengeMgr.UpdateProgress(userID, backupData)
	result.CompletedChallenges = completedChallenges

	// Award rewards
	for _, achievement := range newAchievements {
		if reward := e.rewardMgr.GetRewardForAchievement(achievement.ID); reward != nil {
			e.rewardMgr.AwardReward(userID, reward)
			result.RewardsEarned = append(result.RewardsEarned, reward)
		}
	}

	// Send notifications (best-effort: notification failures must not fail the backup recording)
	if e.notifier != nil {
		for _, achievement := range newAchievements {
			if err := e.notifier.NotifyAchievement(ctx, userID, achievement); err != nil {
				result.NotificationErrors = append(result.NotificationErrors, err)
			}
		}

		if result.LevelUps > 0 {
			if err := e.notifier.NotifyLevelUp(ctx, userID, result.CurrentLevel); err != nil {
				result.NotificationErrors = append(result.NotificationErrors, err)
			}
		}

		// Notify on streak milestones (every 7 days)
		if streak%7 == 0 && streak > 0 {
			if err := e.notifier.NotifyStreakMilestone(ctx, userID, streak); err != nil {
				result.NotificationErrors = append(result.NotificationErrors, err)
			}
		}
	}

	return result, nil
}

// calculateXP calculates XP earned from a backup.
func (e *Engine) calculateXP(data *BackupData) int {
	baseXP := 10

	// Bonus for size (1 XP per GB)
	sizeXP := int(data.Size / (1024 * 1024 * 1024))

	// Bonus for compression
	compressionBonus := 0
	if data.CompressionRatio > 0.5 {
		compressionBonus = 5
	}

	// Bonus for encryption
	encryptionBonus := 0
	if data.Encrypted {
		encryptionBonus = 5
	}

	// Bonus for speed
	speedBonus := 0
	if data.Duration < 60 { // Under 1 minute
		speedBonus = 10
	} else if data.Duration < 300 { // Under 5 minutes
		speedBonus = 5
	}

	return baseXP + sizeXP + compressionBonus + encryptionBonus + speedBonus
}

// GetUserProgress returns comprehensive user progress.
func (e *Engine) GetUserProgress(userID string) *UserProgress {
	e.mu.RLock()
	defer e.mu.RUnlock()

	return &UserProgress{
		UserID:            userID,
		Level:             e.xpMgr.GetLevel(userID),
		XP:                e.xpMgr.GetXP(userID),
		XPToNextLevel:     e.xpMgr.GetXPToNextLevel(userID),
		Achievements:      e.achievementMgr.GetUserAchievements(userID),
		CurrentStreak:     e.streakMgr.GetStreak(userID),
		LongestStreak:     e.streakMgr.GetLongestStreak(userID),
		TotalBackups:      e.statsMgr.GetTotalBackups(userID),
		TotalDataBackedUp: e.statsMgr.GetTotalSize(userID),
		ActiveChallenges:  e.challengeMgr.GetActiveChallenges(userID),
		AvailableRewards:  e.rewardMgr.GetAvailableRewards(userID),
		LeaderboardRank:   e.leaderboardMgr.GetUserRank(userID),
		Stats:             e.statsMgr.GetStats(userID),
	}
}

// GetLeaderboard returns leaderboard data.
func (e *Engine) GetLeaderboard(leaderboardType LeaderboardType, limit int) []*LeaderboardEntry {
	e.mu.RLock()
	defer e.mu.RUnlock()

	return e.leaderboardMgr.GetLeaderboard(leaderboardType, limit)
}

// GetTeamLeaderboard returns team-specific leaderboard.
func (e *Engine) GetTeamLeaderboard(teamID string, limit int) []*LeaderboardEntry {
	e.mu.RLock()
	defer e.mu.RUnlock()

	return e.leaderboardMgr.GetTeamLeaderboard(teamID, limit)
}

// CreateChallenge creates a new challenge.
func (e *Engine) CreateChallenge(challenge *Challenge) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	return e.challengeMgr.CreateChallenge(challenge)
}

// ShareAchievement generates shareable content for an achievement.
func (e *Engine) ShareAchievement(userID, achievementID string) (*ShareableContent, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	achievement := e.achievementMgr.GetAchievement(userID, achievementID)
	if achievement == nil {
		return nil, fmt.Errorf("achievement not found")
	}

	userProgress := e.GetUserProgress(userID)

	return &ShareableContent{
		Type:        "achievement",
		Title:       achievement.Name,
		Description: achievement.Description,
		ImageURL:    achievement.BadgeURL,
		Stats: map[string]interface{}{
			"level":         userProgress.Level,
			"total_backups": userProgress.TotalBackups,
			"streak":        userProgress.CurrentStreak,
		},
		ShareURL: fmt.Sprintf("https://backup-manager.com/share/achievement/%s", achievementID),
		Hashtags: []string{"#BackupHero", "#DataSafety", "#Achievement"},
	}, nil
}

// BackupData contains information about a backup.
type BackupData struct {
	Size             int64
	Duration         int64
	Database         string
	CompressionRatio float64
	Encrypted        bool
	Verified         bool
	Tables           int
	DatabaseType     string
}

// Result contains the results of a gamification event.
type Result struct {
	UserID              string
	Timestamp           time.Time
	XPGained            int
	LevelUps            int
	CurrentLevel        int
	Achievements        []*Achievement
	CurrentStreak       int
	CompletedChallenges []*Challenge
	RewardsEarned       []*Reward
	// NotificationErrors holds any best-effort notification failures; they do not
	// fail the event recording.
	NotificationErrors []error
}

// UserProgress represents comprehensive user progress.
type UserProgress struct {
	UserID            string
	Level             int
	XP                int
	XPToNextLevel     int
	Achievements      []*Achievement
	CurrentStreak     int
	LongestStreak     int
	TotalBackups      int
	TotalDataBackedUp int64
	ActiveChallenges  []*Challenge
	AvailableRewards  []*Reward
	LeaderboardRank   int
	Stats             *UserStats
}

// ShareableContent represents content that can be shared socially.
type ShareableContent struct {
	Type        string
	Title       string
	Description string
	ImageURL    string
	Stats       map[string]interface{}
	ShareURL    string
	Hashtags    []string
}
