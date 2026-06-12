package gamification

import (
	"context"
	"testing"
	"time"
)

// =============================================================================
// Engine Tests
// =============================================================================

func TestNewEngine(t *testing.T) {
	cfg := &Config{
		EnableAchievements: true,
		EnableStreaks:      true,
		EnableXP:           true,
	}

	engine := NewEngine(cfg, nil)
	if engine == nil {
		t.Fatal("expected engine to be created")
	}

	if engine.achievementMgr == nil {
		t.Error("achievement manager not initialized")
	}

	if engine.streakMgr == nil {
		t.Error("streak manager not initialized")
	}
}

func TestEngineRecordBackup(t *testing.T) {
	engine := NewEngine(&Config{}, &mockNotifier{})

	backupData := &BackupData{
		Size:             1024 * 1024 * 1024, // 1 GB
		Duration:         30,
		Database:         "test-db",
		CompressionRatio: 0.6,
		Encrypted:        true,
		DatabaseType:     "postgres",
	}

	result, err := engine.RecordBackup(context.Background(), "user-1", backupData)
	if err != nil {
		t.Fatalf("failed to record backup: %v", err)
	}

	if result.XPGained == 0 {
		t.Error("expected XP to be gained")
	}

	if result.CurrentLevel < 1 {
		t.Error("expected user to have at least level 1")
	}
}

func TestEngineGetUserProgress(t *testing.T) {
	engine := NewEngine(&Config{}, nil)

	// Record some backups
	backupData := &BackupData{
		Size:         1024 * 1024,
		Duration:     10,
		Database:     "test-db",
		Encrypted:    true,
		DatabaseType: "postgres",
	}

	engine.RecordBackup(context.Background(), "user-1", backupData)

	progress := engine.GetUserProgress("user-1")
	if progress == nil {
		t.Fatal("expected user progress")
	}

	if progress.Level < 1 {
		t.Error("expected at least level 1")
	}

	if progress.TotalBackups != 1 {
		t.Errorf("expected 1 backup, got %d", progress.TotalBackups)
	}
}

func TestEngineShareAchievement(t *testing.T) {
	engine := NewEngine(&Config{}, nil)

	// Unlock an achievement first
	backupData := &BackupData{
		Size:         1024,
		Duration:     10,
		Database:     "test-db",
		DatabaseType: "postgres",
	}

	engine.RecordBackup(context.Background(), "user-1", backupData)

	// Try to share (will fail if no achievement unlocked)
	_, err := engine.ShareAchievement("user-1", "first_backup")
	if err == nil {
		// Achievement was unlocked, check shareable content
		// This is okay
	}
}

// =============================================================================
// Achievement Tests
// =============================================================================

func TestAchievementManager(t *testing.T) {
	am := NewAchievementManager()

	if len(am.achievements) == 0 {
		t.Error("expected default achievements to be registered")
	}
}

func TestCheckAchievements(t *testing.T) {
	am := NewAchievementManager()

	backupData := &BackupData{
		Size:         1024,
		Duration:     10,
		Database:     "test-db",
		DatabaseType: "postgres",
	}

	stats := &UserStats{
		TotalBackups:  1,
		TotalSize:     1024,
		DatabaseTypes: map[string]int{"postgres": 1},
	}

	achievements := am.CheckAchievements("user-1", backupData, stats)

	// Should unlock "first_backup" achievement
	if len(achievements) == 0 {
		t.Error("expected at least one achievement to be unlocked")
	}

	// Check if first_backup was unlocked
	found := false
	for _, ach := range achievements {
		if ach.ID == "first_backup" {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected 'first_backup' achievement to be unlocked")
	}
}

func TestGetAchievementProgress(t *testing.T) {
	am := NewAchievementManager()

	stats := &UserStats{
		TotalBackups: 50,
		TotalSize:    50 * 1024 * 1024, // 50 MB
	}

	progress := am.GetAchievementProgress("user-1", stats)

	if len(progress) == 0 {
		t.Error("expected progress data")
	}

	// Check progress towards 100 backups achievement
	if p, exists := progress["backup_100"]; exists {
		// Expected: 0.5 (50/100)
		if p < 0.49 || p > 0.51 {
			t.Errorf("expected progress ~0.5, got %.2f", p)
		}
	}
}

// =============================================================================
// Streak Tests
// =============================================================================

func TestStreakManager(t *testing.T) {
	sm := NewStreakManager()

	now := time.Now()

	// Day 1
	streak := sm.UpdateStreak("user-1", now)
	if streak != 1 {
		t.Errorf("expected streak 1, got %d", streak)
	}

	// Same day (should not increase)
	streak = sm.UpdateStreak("user-1", now.Add(2*time.Hour))
	if streak != 1 {
		t.Errorf("expected streak 1, got %d", streak)
	}

	// Next day
	streak = sm.UpdateStreak("user-1", now.Add(25*time.Hour))
	if streak != 2 {
		t.Errorf("expected streak 2, got %d", streak)
	}

	// Day after next
	streak = sm.UpdateStreak("user-1", now.Add(49*time.Hour))
	if streak != 3 {
		t.Errorf("expected streak 3, got %d", streak)
	}

	// Break streak (skip 2 days)
	streak = sm.UpdateStreak("user-1", now.Add(120*time.Hour))
	if streak != 1 {
		t.Errorf("expected streak reset to 1, got %d", streak)
	}
}

func TestGetLongestStreak(t *testing.T) {
	sm := NewStreakManager()

	now := time.Now()

	// Build a 5-day streak
	for i := 0; i < 5; i++ {
		sm.UpdateStreak("user-1", now.Add(time.Duration(i*24)*time.Hour))
	}

	longest := sm.GetLongestStreak("user-1")
	if longest != 5 {
		t.Errorf("expected longest streak 5, got %d", longest)
	}

	// Break and start new streak
	sm.UpdateStreak("user-1", now.Add(200*time.Hour))

	// Longest should still be 5
	longest = sm.GetLongestStreak("user-1")
	if longest != 5 {
		t.Errorf("expected longest streak still 5, got %d", longest)
	}
}

// =============================================================================
// XP Tests
// =============================================================================

func TestXPManager(t *testing.T) {
	xm := NewXPManager()

	// Add XP
	result := xm.AddXP("user-1", 100)

	if result.CurrentLevel != 1 {
		t.Errorf("expected level 1 at 100 XP, got %d", result.CurrentLevel)
	}

	// Add more XP to level up
	result = xm.AddXP("user-1", 200)

	if result.CurrentLevel < 2 {
		t.Errorf("expected at least level 2 at 300 XP, got %d", result.CurrentLevel)
	}

	if result.LevelsGained < 1 {
		t.Error("expected at least 1 level gained")
	}
}

func TestGetXPToNextLevel(t *testing.T) {
	xm := NewXPManager()

	xm.AddXP("user-1", 100)

	xpNeeded := xm.GetXPToNextLevel("user-1")
	if xpNeeded <= 0 {
		t.Errorf("expected positive XP needed, got %d", xpNeeded)
	}
}

// =============================================================================
// Stats Tests
// =============================================================================

func TestStatsManager(t *testing.T) {
	sm := NewStatsManager()

	backupData := &BackupData{
		Size:         1024 * 1024, // 1 MB
		Duration:     30,
		Encrypted:    true,
		DatabaseType: "postgres",
	}

	sm.RecordBackup("user-1", backupData)

	stats := sm.GetStats("user-1")
	if stats.TotalBackups != 1 {
		t.Errorf("expected 1 backup, got %d", stats.TotalBackups)
	}

	if stats.EncryptedBackups != 1 {
		t.Errorf("expected 1 encrypted backup, got %d", stats.EncryptedBackups)
	}

	if stats.TotalSize != 1024*1024 {
		t.Errorf("expected size 1MB, got %d", stats.TotalSize)
	}

	if stats.DatabaseTypes["postgres"] != 1 {
		t.Error("expected postgres to be tracked")
	}
}

func TestGetPersonalInsights(t *testing.T) {
	sm := NewStatsManager()

	// Record some backups
	for i := 0; i < 10; i++ {
		backupData := &BackupData{
			Size:         1024,
			Duration:     10,
			Encrypted:    i%2 == 0, // 50% encrypted
			DatabaseType: "postgres",
		}
		sm.RecordBackup("user-1", backupData)
	}

	insights := sm.GetPersonalInsights("user-1")

	if len(insights.Insights) == 0 {
		t.Error("expected at least one insight")
	}

	if len(insights.Tips) == 0 {
		t.Error("expected at least one tip")
	}
}

// =============================================================================
// Leaderboard Tests
// =============================================================================

func TestLeaderboardManager(t *testing.T) {
	lm := NewLeaderboardManager()

	backupData1 := &BackupData{Size: 1024 * 1024}
	backupData2 := &BackupData{Size: 2048 * 1024}

	lm.UpdateScore("user-1", backupData1)
	lm.UpdateScore("user-2", backupData2)

	leaderboard := lm.GetLeaderboard(LeaderboardBackups, 10)

	if len(leaderboard) != 2 {
		t.Errorf("expected 2 entries, got %d", len(leaderboard))
	}

	// Both users should have 1 backup
	if leaderboard[0].Rank != 1 {
		t.Error("expected first entry to have rank 1")
	}
}

func TestGetTeamLeaderboard(t *testing.T) {
	lm := NewLeaderboardManager()

	// Add users to team
	lm.entries["user-1"] = &LeaderboardUserData{
		UserID:       "user-1",
		TeamID:       "team-alpha",
		TotalXP:      1000,
		TotalBackups: 10,
	}

	lm.entries["user-2"] = &LeaderboardUserData{
		UserID:       "user-2",
		TeamID:       "team-alpha",
		TotalXP:      2000,
		TotalBackups: 20,
	}

	lm.entries["user-3"] = &LeaderboardUserData{
		UserID:       "user-3",
		TeamID:       "team-beta",
		TotalXP:      1500,
		TotalBackups: 15,
	}

	teamLeaderboard := lm.GetTeamLeaderboard("team-alpha", 10)

	if len(teamLeaderboard) != 2 {
		t.Errorf("expected 2 team members, got %d", len(teamLeaderboard))
	}

	// user-2 should be first (higher XP)
	if teamLeaderboard[0].UserID != "user-2" {
		t.Errorf("expected user-2 to be first, got %s", teamLeaderboard[0].UserID)
	}
}

// =============================================================================
// Challenge Tests
// =============================================================================

func TestChallengeManager(t *testing.T) {
	cm := NewChallengeManager()

	if len(cm.challenges) == 0 {
		t.Error("expected default challenges to be created")
	}

	activeChallenges := cm.GetActiveChallenges("user-1")
	if len(activeChallenges) == 0 {
		t.Error("expected active challenges")
	}
}

func TestCreateChallenge(t *testing.T) {
	cm := NewChallengeManager()

	challenge := &Challenge{
		ID:          "test_challenge",
		Name:        "Test Challenge",
		Description: "Complete 5 backups",
		Type:        ChallengeTypeDaily,
		StartTime:   time.Now(),
		EndTime:     time.Now().Add(24 * time.Hour),
		Goal:        5,
		GoalType:    "backups",
		XPReward:    100,
		Active:      true,
	}

	err := cm.CreateChallenge(challenge)
	if err != nil {
		t.Errorf("failed to create challenge: %v", err)
	}

	// Try to create duplicate
	err = cm.CreateChallenge(challenge)
	if err == nil {
		t.Error("expected error when creating duplicate challenge")
	}
}

func TestUpdateChallengeProgress(t *testing.T) {
	cm := NewChallengeManager()

	backupData := &BackupData{
		Size:     1024,
		Duration: 10,
	}

	// Complete 3 backups for daily challenge
	for i := 0; i < 3; i++ {
		completed := cm.UpdateProgress("user-1", backupData)
		if i == 2 && len(completed) == 0 {
			// Should complete daily_backup_3 challenge
			t.Error("expected challenge to be completed on 3rd backup")
		}
	}
}

// =============================================================================
// Reward Tests
// =============================================================================

func TestRewardManager(t *testing.T) {
	rm := NewRewardManager()

	if len(rm.rewards) == 0 {
		t.Error("expected default rewards to be created")
	}
}

func TestAwardReward(t *testing.T) {
	rm := NewRewardManager()

	reward := &Reward{
		ID:          "test_reward",
		Name:        "Test Reward",
		Type:        RewardTypeBadge,
		Description: "Test badge",
	}

	rm.AwardReward("user-1", reward)

	// Check if reward was awarded
	if len(rm.userRewards["user-1"]) != 1 {
		t.Error("expected reward to be awarded")
	}
}

func TestGetAvailableRewards(t *testing.T) {
	rm := NewRewardManager()

	// Initially all rewards should be available
	available := rm.GetAvailableRewards("user-1")
	initialCount := len(available)

	if initialCount == 0 {
		t.Error("expected some available rewards")
	}

	// Award a reward
	reward := available[0]
	rm.AwardReward("user-1", reward)

	// Check available rewards again
	available = rm.GetAvailableRewards("user-1")
	if len(available) != initialCount-1 {
		t.Errorf("expected %d available rewards, got %d", initialCount-1, len(available))
	}
}

// =============================================================================
// Mock Notifier
// =============================================================================

type mockNotifier struct{}

func (m *mockNotifier) NotifyAchievement(ctx context.Context, userID string, achievement *Achievement) error {
	return nil
}

func (m *mockNotifier) NotifyLevelUp(ctx context.Context, userID string, level int) error {
	return nil
}

func (m *mockNotifier) NotifyStreakMilestone(ctx context.Context, userID string, streak int) error {
	return nil
}

func (m *mockNotifier) NotifyReward(ctx context.Context, userID string, reward *Reward) error {
	return nil
}

// =============================================================================
// Benchmark Tests
// =============================================================================

func BenchmarkRecordBackup(b *testing.B) {
	engine := NewEngine(&Config{}, nil)

	backupData := &BackupData{
		Size:             1024 * 1024,
		Duration:         30,
		Database:         "test-db",
		CompressionRatio: 0.6,
		Encrypted:        true,
		DatabaseType:     "postgres",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine.RecordBackup(context.Background(), "user-1", backupData)
	}
}

func BenchmarkCheckAchievements(b *testing.B) {
	am := NewAchievementManager()

	backupData := &BackupData{
		Size:         1024,
		Duration:     10,
		Database:     "test-db",
		DatabaseType: "postgres",
	}

	stats := &UserStats{
		TotalBackups:  100,
		TotalSize:     100 * 1024 * 1024,
		DatabaseTypes: map[string]int{"postgres": 100},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		am.CheckAchievements("user-1", backupData, stats)
	}
}

func BenchmarkGetLeaderboard(b *testing.B) {
	lm := NewLeaderboardManager()

	// Add 1000 users
	for i := 0; i < 1000; i++ {
		lm.entries[string(rune(i))] = &LeaderboardUserData{
			UserID:       string(rune(i)),
			TotalXP:      i * 100,
			TotalBackups: i,
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lm.GetLeaderboard(LeaderboardXP, 100)
	}
}
