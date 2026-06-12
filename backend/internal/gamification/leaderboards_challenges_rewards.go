package gamification

import (
	"fmt"
	"sort"
	"sync"
	"time"
)

// ============================================================================
// Leaderboard Manager
// ============================================================================

// LeaderboardManager manages leaderboards
type LeaderboardManager struct {
	entries map[string]*LeaderboardUserData
	teams   map[string]*TeamData
	mu      sync.RWMutex
}

// LeaderboardUserData holds leaderboard data for a user
type LeaderboardUserData struct{
	UserID          string
	Username        string
	TeamID          string
	TotalXP         int
	Level           int
	TotalBackups    int
	TotalSize       int64
	CurrentStreak   int
	AchievementCount int
	LastUpdated     time.Time
}

// TeamData holds team information
type TeamData struct {
	TeamID      string
	TeamName    string
	Members     []string
	TotalXP     int
	TotalBackups int
}

// LeaderboardEntry represents an entry in the leaderboard
type LeaderboardEntry struct {
	Rank            int
	UserID          string
	Username        string
	TeamID          string
	Score           int
	Level           int
	Achievements    int
	Streak          int
}

// LeaderboardType defines different leaderboard types
type LeaderboardType string

const (
	LeaderboardXP           LeaderboardType = "xp"
	LeaderboardLevel        LeaderboardType = "level"
	LeaderboardBackups      LeaderboardType = "backups"
	LeaderboardSize         LeaderboardType = "size"
	LeaderboardStreak       LeaderboardType = "streak"
	LeaderboardAchievements LeaderboardType = "achievements"
)

// NewLeaderboardManager creates a new leaderboard manager
func NewLeaderboardManager() *LeaderboardManager {
	return &LeaderboardManager{
		entries: make(map[string]*LeaderboardUserData),
		teams:   make(map[string]*TeamData),
	}
}

// UpdateScore updates a user's leaderboard data
func (lm *LeaderboardManager) UpdateScore(userID string, data *BackupData) {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	if lm.entries[userID] == nil {
		lm.entries[userID] = &LeaderboardUserData{
			UserID: userID,
		}
	}

	entry := lm.entries[userID]
	entry.TotalBackups++
	entry.TotalSize += data.Size
	entry.LastUpdated = time.Now()
}

// GetLeaderboard returns the leaderboard for a given type
func (lm *LeaderboardManager) GetLeaderboard(leaderboardType LeaderboardType, limit int) []*LeaderboardEntry {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	entries := make([]*LeaderboardEntry, 0, len(lm.entries))

	for _, userData := range lm.entries {
		entry := &LeaderboardEntry{
			UserID:       userData.UserID,
			Username:     userData.Username,
			TeamID:       userData.TeamID,
			Level:        userData.Level,
			Achievements: userData.AchievementCount,
			Streak:       userData.CurrentStreak,
		}

		// Set score based on type
		switch leaderboardType {
		case LeaderboardXP:
			entry.Score = userData.TotalXP
		case LeaderboardLevel:
			entry.Score = userData.Level
		case LeaderboardBackups:
			entry.Score = userData.TotalBackups
		case LeaderboardSize:
			entry.Score = int(userData.TotalSize / (1024 * 1024)) // MB
		case LeaderboardStreak:
			entry.Score = userData.CurrentStreak
		case LeaderboardAchievements:
			entry.Score = userData.AchievementCount
		}

		entries = append(entries, entry)
	}

	// Sort by score descending
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Score > entries[j].Score
	})

	// Assign ranks
	for i, entry := range entries {
		entry.Rank = i + 1
	}

	// Apply limit
	if limit > 0 && len(entries) > limit {
		entries = entries[:limit]
	}

	return entries
}

// GetTeamLeaderboard returns team-specific leaderboard
func (lm *LeaderboardManager) GetTeamLeaderboard(teamID string, limit int) []*LeaderboardEntry {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	entries := make([]*LeaderboardEntry, 0)

	for _, userData := range lm.entries {
		if userData.TeamID == teamID {
			entry := &LeaderboardEntry{
				UserID:       userData.UserID,
				Username:     userData.Username,
				TeamID:       userData.TeamID,
				Score:        userData.TotalXP,
				Level:        userData.Level,
				Achievements: userData.AchievementCount,
				Streak:       userData.CurrentStreak,
			}
			entries = append(entries, entry)
		}
	}

	// Sort by XP
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Score > entries[j].Score
	})

	// Assign ranks
	for i, entry := range entries {
		entry.Rank = i + 1
	}

	if limit > 0 && len(entries) > limit {
		entries = entries[:limit]
	}

	return entries
}

// GetUserRank returns a user's rank on the global leaderboard
func (lm *LeaderboardManager) GetUserRank(userID string) int {
	leaderboard := lm.GetLeaderboard(LeaderboardXP, 0)
	for _, entry := range leaderboard {
		if entry.UserID == userID {
			return entry.Rank
		}
	}
	return 0
}

// ============================================================================
// Challenge Manager
// ============================================================================

// ChallengeManager manages challenges
type ChallengeManager struct {
	challenges     map[string]*Challenge
	userChallenges map[string]map[string]*ChallengeProgress // userID -> challengeID -> progress
	mu             sync.RWMutex
}

// Challenge represents a challenge
type Challenge struct {
	ID          string
	Name        string
	Description string
	Type        ChallengeType
	StartTime   time.Time
	EndTime     time.Time
	Goal        int64
	GoalType    string // "backups", "size", "streak", etc.
	XPReward    int
	Active      bool
}

// ChallengeType defines challenge duration
type ChallengeType string

const (
	ChallengeTypeDaily     ChallengeType = "daily"
	ChallengeTypeWeekly    ChallengeType = "weekly"
	ChallengeTypeMonthly   ChallengeType = "monthly"
	ChallengeTypeSeasonal  ChallengeType = "seasonal"
	ChallengeTypeSpecial   ChallengeType = "special"
)

// ChallengeProgress tracks user progress on a challenge
type ChallengeProgress struct {
	ChallengeID string
	Progress    int64
	Completed   bool
	CompletedAt time.Time
}

// NewChallengeManager creates a new challenge manager
func NewChallengeManager() *ChallengeManager {
	cm := &ChallengeManager{
		challenges:     make(map[string]*Challenge),
		userChallenges: make(map[string]map[string]*ChallengeProgress),
	}

	// Create default challenges
	cm.createDefaultChallenges()

	return cm
}

// createDefaultChallenges creates default challenges
func (cm *ChallengeManager) createDefaultChallenges() {
	now := time.Now()

	challenges := []*Challenge{
		{
			ID:          "daily_backup_3",
			Name:        "Daily Grind",
			Description: "Complete 3 backups today",
			Type:        ChallengeTypeDaily,
			StartTime:   now,
			EndTime:     now.Add(24 * time.Hour),
			Goal:        3,
			GoalType:    "backups",
			XPReward:    50,
			Active:      true,
		},
		{
			ID:          "weekly_backup_20",
			Name:        "Weekly Warrior",
			Description: "Complete 20 backups this week",
			Type:        ChallengeTypeWeekly,
			StartTime:   now,
			EndTime:     now.Add(7 * 24 * time.Hour),
			Goal:        20,
			GoalType:    "backups",
			XPReward:    500,
			Active:      true,
		},
		{
			ID:          "monthly_size_1tb",
			Name:        "Terabyte Challenge",
			Description: "Back up 1 TB of data this month",
			Type:        ChallengeTypeMonthly,
			StartTime:   now,
			EndTime:     now.AddDate(0, 1, 0),
			Goal:        1024 * 1024 * 1024 * 1024, // 1 TB in bytes
			GoalType:    "size",
			XPReward:    2000,
			Active:      true,
		},
	}

	for _, challenge := range challenges {
		cm.challenges[challenge.ID] = challenge
	}
}

// CreateChallenge creates a new challenge
func (cm *ChallengeManager) CreateChallenge(challenge *Challenge) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if _, exists := cm.challenges[challenge.ID]; exists {
		return fmt.Errorf("challenge %s already exists", challenge.ID)
	}

	cm.challenges[challenge.ID] = challenge
	return nil
}

// UpdateProgress updates challenge progress for a user
func (cm *ChallengeManager) UpdateProgress(userID string, data *BackupData) []*Challenge {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.userChallenges[userID] == nil {
		cm.userChallenges[userID] = make(map[string]*ChallengeProgress)
	}

	completedChallenges := make([]*Challenge, 0)

	for _, challenge := range cm.challenges {
		if !challenge.Active {
			continue
		}

		// Check if challenge is still active
		if time.Now().After(challenge.EndTime) {
			challenge.Active = false
			continue
		}

		// Initialize progress if needed
		if cm.userChallenges[userID][challenge.ID] == nil {
			cm.userChallenges[userID][challenge.ID] = &ChallengeProgress{
				ChallengeID: challenge.ID,
			}
		}

		progress := cm.userChallenges[userID][challenge.ID]

		// Skip if already completed
		if progress.Completed {
			continue
		}

		// Update progress based on goal type
		switch challenge.GoalType {
		case "backups":
			progress.Progress++
		case "size":
			progress.Progress += data.Size
		}

		// Check if completed
		if progress.Progress >= challenge.Goal {
			progress.Completed = true
			progress.CompletedAt = time.Now()
			completedChallenges = append(completedChallenges, challenge)
		}
	}

	return completedChallenges
}

// GetActiveChallenges returns active challenges for a user
func (cm *ChallengeManager) GetActiveChallenges(userID string) []*Challenge {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	activeChallenges := make([]*Challenge, 0)

	for _, challenge := range cm.challenges {
		if !challenge.Active {
			continue
		}

		if time.Now().After(challenge.EndTime) {
			continue
		}

		// Check if not already completed
		if cm.userChallenges[userID] != nil {
			if progress, exists := cm.userChallenges[userID][challenge.ID]; exists && progress.Completed {
				continue
			}
		}

		activeChallenges = append(activeChallenges, challenge)
	}

	return activeChallenges
}

// ============================================================================
// Reward Manager
// ============================================================================

// RewardManager manages rewards and redemption
type RewardManager struct {
	rewards     map[string]*Reward
	userRewards map[string][]*UserReward // userID -> rewards
	mu          sync.RWMutex
}

// Reward represents a redeemable reward
type Reward struct {
	ID          string
	Name        string
	Description string
	Type        RewardType
	Value       string
	CostXP      int
	Limited     bool
	Available   int
	ImageURL    string
}

// RewardType defines different reward types
type RewardType string

const (
	RewardTypeBadge       RewardType = "badge"
	RewardTypeTitle       RewardType = "title"
	RewardTypeAvatar      RewardType = "avatar"
	RewardTypeTheme       RewardType = "theme"
	RewardTypeFeature     RewardType = "feature"
	RewardTypeDiscount    RewardType = "discount"
)

// UserReward represents a reward owned by a user
type UserReward struct {
	RewardID   string
	AwardedAt  time.Time
	Redeemed   bool
	RedeemedAt time.Time
}

// NewRewardManager creates a new reward manager
func NewRewardManager() *RewardManager {
	rm := &RewardManager{
		rewards:     make(map[string]*Reward),
		userRewards: make(map[string][]*UserReward),
	}

	rm.createDefaultRewards()

	return rm
}

// createDefaultRewards creates default rewards
func (rm *RewardManager) createDefaultRewards() {
	rewards := []*Reward{
		{
			ID:          "badge_gold_star",
			Name:        "Gold Star Badge",
			Description: "Exclusive gold star profile badge",
			Type:        RewardTypeBadge,
			Value:       "gold_star",
			CostXP:      1000,
		},
		{
			ID:          "title_backup_master",
			Name:        "Backup Master Title",
			Description: "Display 'Backup Master' on your profile",
			Type:        RewardTypeTitle,
			Value:       "Backup Master",
			CostXP:      5000,
		},
		{
			ID:          "theme_dark_mode",
			Name:        "Dark Mode Theme",
			Description: "Unlock dark mode theme",
			Type:        RewardTypeTheme,
			Value:       "dark_mode",
			CostXP:      500,
		},
		{
			ID:          "feature_priority_support",
			Name:        "Priority Support",
			Description: "Get priority customer support for 30 days",
			Type:        RewardTypeFeature,
			Value:       "priority_support_30d",
			CostXP:      10000,
		},
	}

	for _, reward := range rewards {
		rm.rewards[reward.ID] = reward
	}
}

// AwardReward awards a reward to a user
func (rm *RewardManager) AwardReward(userID string, reward *Reward) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if rm.userRewards[userID] == nil {
		rm.userRewards[userID] = make([]*UserReward, 0)
	}

	userReward := &UserReward{
		RewardID:  reward.ID,
		AwardedAt: time.Now(),
	}

	rm.userRewards[userID] = append(rm.userRewards[userID], userReward)
}

// GetAvailableRewards returns rewards available for a user
func (rm *RewardManager) GetAvailableRewards(userID string) []*Reward {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	available := make([]*Reward, 0)

	for _, reward := range rm.rewards {
		// Check if user already has this reward
		hasReward := false
		if rm.userRewards[userID] != nil {
			for _, userReward := range rm.userRewards[userID] {
				if userReward.RewardID == reward.ID {
					hasReward = true
					break
				}
			}
		}

		if !hasReward {
			available = append(available, reward)
		}
	}

	return available
}

// GetRewardForAchievement returns reward associated with an achievement
func (rm *RewardManager) GetRewardForAchievement(achievementID string) *Reward {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	// Map achievements to rewards
	achievementRewards := map[string]string{
		"backup_1000": "badge_gold_star",
		"streak_365":  "title_backup_master",
	}

	if rewardID, exists := achievementRewards[achievementID]; exists {
		return rm.rewards[rewardID]
	}

	return nil
}
