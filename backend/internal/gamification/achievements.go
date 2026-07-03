package gamification

import (
	"sync"
	"time"
)

// AchievementManager manages achievements and badges.
type AchievementManager struct {
	achievements     map[string]*Achievement
	userAchievements map[string]map[string]*UserAchievement // userID -> achievementID -> UserAchievement
	mu               sync.RWMutex
}

// Achievement represents an achievement/badge.
type Achievement struct {
	ID          string
	Name        string
	Description string
	BadgeURL    string
	Tier        AchievementTier
	Rarity      AchievementRarity
	Category    AchievementCategory
	XPReward    int
	Requirement AchievementRequirement
	Hidden      bool // Secret achievements
}

// UserAchievement represents a user's earned achievement.
type UserAchievement struct {
	AchievementID string
	UnlockedAt    time.Time
	Progress      float64 // 0.0 to 1.0
}

// AchievementTier represents achievement difficulty tiers.
type AchievementTier string

const (
	TierBronze   AchievementTier = "bronze"
	TierSilver   AchievementTier = "silver"
	TierGold     AchievementTier = "gold"
	TierPlatinum AchievementTier = "platinum"
	TierDiamond  AchievementTier = "diamond"
)

// AchievementRarity represents how rare an achievement is.
type AchievementRarity string

const (
	RarityCommon    AchievementRarity = "common"
	RarityUncommon  AchievementRarity = "uncommon"
	RarityRare      AchievementRarity = "rare"
	RarityEpic      AchievementRarity = "epic"
	RarityLegendary AchievementRarity = "legendary"
	RarityMythic    AchievementRarity = "mythic"
)

// AchievementCategory categorizes achievements.
type AchievementCategory string

const (
	CategoryBackups    AchievementCategory = "backups"
	CategoryStreaks    AchievementCategory = "streaks"
	CategorySize       AchievementCategory = "size"
	CategorySpeed      AchievementCategory = "speed"
	CategoryMilestones AchievementCategory = "milestones"
	CategoryChallenges AchievementCategory = "challenges"
	CategorySocial     AchievementCategory = "social"
	CategoryExpertise  AchievementCategory = "expertise"
)

// AchievementRequirement defines what's needed to unlock an achievement.
type AchievementRequirement struct {
	Type         string
	MinValue     int64
	MinStreak    int
	MinLevel     int
	SpecificData map[string]interface{}
}

// NewAchievementManager creates a new achievement manager.
func NewAchievementManager() *AchievementManager {
	am := &AchievementManager{
		achievements:     make(map[string]*Achievement),
		userAchievements: make(map[string]map[string]*UserAchievement),
	}

	// Register default achievements
	am.registerDefaultAchievements()

	return am
}

// registerDefaultAchievements registers all built-in achievements.
func (am *AchievementManager) registerDefaultAchievements() {
	achievements := []*Achievement{
		// First Steps
		{
			ID:          "first_backup",
			Name:        "First Steps",
			Description: "Complete your first backup",
			Tier:        TierBronze,
			Rarity:      RarityCommon,
			Category:    CategoryBackups,
			XPReward:    50,
			Requirement: AchievementRequirement{Type: "total_backups", MinValue: 1},
		},
		{
			ID:          "backup_10",
			Name:        "Getting Started",
			Description: "Complete 10 backups",
			Tier:        TierBronze,
			Rarity:      RarityCommon,
			Category:    CategoryBackups,
			XPReward:    100,
			Requirement: AchievementRequirement{Type: "total_backups", MinValue: 10},
		},
		{
			ID:          "backup_100",
			Name:        "Backup Enthusiast",
			Description: "Complete 100 backups",
			Tier:        TierSilver,
			Rarity:      RarityUncommon,
			Category:    CategoryBackups,
			XPReward:    500,
			Requirement: AchievementRequirement{Type: "total_backups", MinValue: 100},
		},
		{
			ID:          "backup_1000",
			Name:        "Backup Master",
			Description: "Complete 1,000 backups",
			Tier:        TierGold,
			Rarity:      RarityRare,
			Category:    CategoryBackups,
			XPReward:    2000,
			Requirement: AchievementRequirement{Type: "total_backups", MinValue: 1000},
		},
		{
			ID:          "backup_10000",
			Name:        "Backup Legend",
			Description: "Complete 10,000 backups",
			Tier:        TierPlatinum,
			Rarity:      RarityEpic,
			Category:    CategoryBackups,
			XPReward:    10000,
			Requirement: AchievementRequirement{Type: "total_backups", MinValue: 10000},
		},

		// Streak Achievements
		{
			ID:          "streak_7",
			Name:        "Week Warrior",
			Description: "Maintain a 7-day backup streak",
			Tier:        TierBronze,
			Rarity:      RarityUncommon,
			Category:    CategoryStreaks,
			XPReward:    200,
			Requirement: AchievementRequirement{Type: "streak", MinStreak: 7},
		},
		{
			ID:          "streak_30",
			Name:        "Monthly Maven",
			Description: "Maintain a 30-day backup streak",
			Tier:        TierSilver,
			Rarity:      RarityRare,
			Category:    CategoryStreaks,
			XPReward:    1000,
			Requirement: AchievementRequirement{Type: "streak", MinStreak: 30},
		},
		{
			ID:          "streak_100",
			Name:        "Century Streak",
			Description: "Maintain a 100-day backup streak",
			Tier:        TierGold,
			Rarity:      RarityEpic,
			Category:    CategoryStreaks,
			XPReward:    5000,
			Requirement: AchievementRequirement{Type: "streak", MinStreak: 100},
		},
		{
			ID:          "streak_365",
			Name:        "Year-Long Dedication",
			Description: "Maintain a 365-day backup streak",
			Tier:        TierDiamond,
			Rarity:      RarityLegendary,
			Category:    CategoryStreaks,
			XPReward:    25000,
			Requirement: AchievementRequirement{Type: "streak", MinStreak: 365},
		},

		// Size Achievements
		{
			ID:          "size_1gb",
			Name:        "Gigabyte Guardian",
			Description: "Back up 1 GB of data",
			Tier:        TierBronze,
			Rarity:      RarityCommon,
			Category:    CategorySize,
			XPReward:    100,
			Requirement: AchievementRequirement{Type: "total_size", MinValue: 1 * 1024 * 1024 * 1024},
		},
		{
			ID:          "size_1tb",
			Name:        "Terabyte Titan",
			Description: "Back up 1 TB of data",
			Tier:        TierGold,
			Rarity:      RarityRare,
			Category:    CategorySize,
			XPReward:    5000,
			Requirement: AchievementRequirement{Type: "total_size", MinValue: 1 * 1024 * 1024 * 1024 * 1024},
		},
		{
			ID:          "size_10tb",
			Name:        "Petabyte Pioneer",
			Description: "Back up 10 TB of data",
			Tier:        TierDiamond,
			Rarity:      RarityLegendary,
			Category:    CategorySize,
			XPReward:    50000,
			Requirement: AchievementRequirement{Type: "total_size", MinValue: 10 * 1024 * 1024 * 1024 * 1024},
		},

		// Speed Achievements
		{
			ID:          "speed_demon",
			Name:        "Speed Demon",
			Description: "Complete a backup in under 10 seconds",
			Tier:        TierSilver,
			Rarity:      RarityRare,
			Category:    CategorySpeed,
			XPReward:    500,
			Requirement: AchievementRequirement{Type: "backup_speed", MinValue: 10},
		},
		{
			ID:          "lightning_fast",
			Name:        "Lightning Fast",
			Description: "Complete a backup in under 1 second",
			Tier:        TierGold,
			Rarity:      RarityEpic,
			Category:    CategorySpeed,
			XPReward:    2000,
			Requirement: AchievementRequirement{Type: "backup_speed", MinValue: 1},
		},

		// Expertise Achievements
		{
			ID:          "encryption_expert",
			Name:        "Encryption Expert",
			Description: "Complete 50 encrypted backups",
			Tier:        TierGold,
			Rarity:      RarityRare,
			Category:    CategoryExpertise,
			XPReward:    1000,
			Requirement: AchievementRequirement{Type: "encrypted_backups", MinValue: 50},
		},
		{
			ID:          "compression_king",
			Name:        "Compression King",
			Description: "Achieve 90%+ compression ratio",
			Tier:        TierGold,
			Rarity:      RarityEpic,
			Category:    CategoryExpertise,
			XPReward:    2000,
			Requirement: AchievementRequirement{Type: "compression_ratio", MinValue: 90},
		},
		{
			ID:          "multi_database",
			Name:        "Database Polyglot",
			Description: "Back up 5 different database types",
			Tier:        TierPlatinum,
			Rarity:      RarityEpic,
			Category:    CategoryExpertise,
			XPReward:    5000,
			Requirement: AchievementRequirement{Type: "database_types", MinValue: 5},
		},

		// Secret/Hidden Achievements
		{
			ID:          "midnight_backup",
			Name:        "Night Owl",
			Description: "Complete a backup at midnight",
			Tier:        TierSilver,
			Rarity:      RarityRare,
			Category:    CategoryMilestones,
			XPReward:    500,
			Requirement: AchievementRequirement{Type: "midnight_backup", MinValue: 1},
			Hidden:      true,
		},
		{
			ID:          "perfect_month",
			Name:        "Perfectionist",
			Description: "Complete a backup every day for a month with 100% success rate",
			Tier:        TierDiamond,
			Rarity:      RarityLegendary,
			Category:    CategoryMilestones,
			XPReward:    10000,
			Requirement: AchievementRequirement{Type: "perfect_month", MinValue: 1},
			Hidden:      true,
		},
	}

	for _, achievement := range achievements {
		am.achievements[achievement.ID] = achievement
	}
}

// CheckAchievements checks if any new achievements were unlocked.
func (am *AchievementManager) CheckAchievements(userID string, backupData *BackupData, stats *UserStats) []*Achievement {
	am.mu.Lock()
	defer am.mu.Unlock()

	if am.userAchievements[userID] == nil {
		am.userAchievements[userID] = make(map[string]*UserAchievement)
	}

	newAchievements := make([]*Achievement, 0)

	for _, achievement := range am.achievements {
		// Skip if already unlocked
		if _, unlocked := am.userAchievements[userID][achievement.ID]; unlocked {
			continue
		}

		// Check if requirement is met
		if am.checkRequirement(achievement.Requirement, backupData, stats) {
			// Unlock achievement
			am.userAchievements[userID][achievement.ID] = &UserAchievement{
				AchievementID: achievement.ID,
				UnlockedAt:    time.Now(),
				Progress:      1.0,
			}
			newAchievements = append(newAchievements, achievement)
		}
	}

	return newAchievements
}

// checkRequirement checks if an achievement requirement is met.
func (am *AchievementManager) checkRequirement(req AchievementRequirement, data *BackupData, stats *UserStats) bool {
	switch req.Type {
	case "total_backups":
		return stats.TotalBackups >= int(req.MinValue)
	case "total_size":
		return stats.TotalSize >= req.MinValue
	case "streak":
		return stats.CurrentStreak >= req.MinStreak
	case "backup_speed":
		return data.Duration <= req.MinValue
	case "encrypted_backups":
		return stats.EncryptedBackups >= int(req.MinValue)
	case "compression_ratio":
		return data.CompressionRatio*100 >= float64(req.MinValue)
	case "database_types":
		return len(stats.DatabaseTypes) >= int(req.MinValue)
	case "midnight_backup":
		hour := time.Now().Hour()
		return hour == 0
	case "perfect_month":
		return stats.PerfectMonths >= int(req.MinValue)
	default:
		return false
	}
}

// GetUserAchievements returns all achievements for a user.
func (am *AchievementManager) GetUserAchievements(userID string) []*Achievement {
	am.mu.RLock()
	defer am.mu.RUnlock()

	achievements := make([]*Achievement, 0)
	if userAchievements, exists := am.userAchievements[userID]; exists {
		for achievementID := range userAchievements {
			if achievement, found := am.achievements[achievementID]; found {
				achievements = append(achievements, achievement)
			}
		}
	}

	return achievements
}

// GetAchievement returns a specific achievement for a user.
func (am *AchievementManager) GetAchievement(userID, achievementID string) *Achievement {
	am.mu.RLock()
	defer am.mu.RUnlock()

	if userAchievements, exists := am.userAchievements[userID]; exists {
		if _, unlocked := userAchievements[achievementID]; unlocked {
			return am.achievements[achievementID]
		}
	}

	return nil
}

// GetAchievementProgress returns progress towards all achievements.
func (am *AchievementManager) GetAchievementProgress(userID string, stats *UserStats) map[string]float64 {
	am.mu.RLock()
	defer am.mu.RUnlock()

	progress := make(map[string]float64)

	for id, achievement := range am.achievements {
		// Skip if already unlocked
		if am.userAchievements[userID] != nil {
			if _, unlocked := am.userAchievements[userID][id]; unlocked {
				progress[id] = 1.0
				continue
			}
		}

		// Calculate progress
		progress[id] = am.calculateProgress(achievement.Requirement, stats)
	}

	return progress
}

// calculateProgress calculates progress towards an achievement (0.0 to 1.0).
func (am *AchievementManager) calculateProgress(req AchievementRequirement, stats *UserStats) float64 {
	switch req.Type {
	case "total_backups":
		if req.MinValue == 0 {
			return 0
		}
		progress := float64(stats.TotalBackups) / float64(req.MinValue)
		if progress > 1.0 {
			return 1.0
		}
		return progress
	case "total_size":
		if req.MinValue == 0 {
			return 0
		}
		progress := float64(stats.TotalSize) / float64(req.MinValue)
		if progress > 1.0 {
			return 1.0
		}
		return progress
	case "streak":
		if req.MinStreak == 0 {
			return 0
		}
		progress := float64(stats.CurrentStreak) / float64(req.MinStreak)
		if progress > 1.0 {
			return 1.0
		}
		return progress
	default:
		return 0
	}
}

// GetAllAchievements returns all available achievements (including hidden ones for admins).
func (am *AchievementManager) GetAllAchievements(includeHidden bool) []*Achievement {
	am.mu.RLock()
	defer am.mu.RUnlock()

	achievements := make([]*Achievement, 0, len(am.achievements))
	for _, achievement := range am.achievements {
		if !includeHidden && achievement.Hidden {
			continue
		}
		achievements = append(achievements, achievement)
	}

	return achievements
}
