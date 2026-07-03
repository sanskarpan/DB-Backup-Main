package enhanced

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"sync"
	"time"
)

// AIEngine provides AI-powered notification intelligence.
type AIEngine struct {
	mu           sync.RWMutex
	modelFile    string
	userPatterns map[string]*UserPattern
	globalStats  *GlobalStats
}

// UserPattern represents learned user notification patterns.
type UserPattern struct {
	UserID             string                       `json:"user_id"`
	PreferredChannels  map[DeliveryChannel]float64  `json:"preferred_channels"` // Channel preference scores
	PreferredTimes     []TimeWindow                 `json:"preferred_times"`    // When user typically engages
	TypeEngagement     map[NotificationType]float64 `json:"type_engagement"`    // Engagement by type
	PriorityResponse   map[Priority]float64         `json:"priority_response"`  // Response rate by priority
	ReadTimeByType     map[NotificationType]float64 `json:"read_time_by_type"`  // Avg seconds to read
	ActionRate         float64                      `json:"action_rate"`        // % of notifications acted upon
	DismissRate        float64                      `json:"dismiss_rate"`       // % of notifications dismissed
	TotalNotifications int                          `json:"total_notifications"`
	LastUpdated        time.Time                    `json:"last_updated"`
}

// TimeWindow represents a time window when user is active.
type TimeWindow struct {
	StartHour int     `json:"start_hour"` // 0-23
	EndHour   int     `json:"end_hour"`
	Weight    float64 `json:"weight"` // How likely to engage (0-1)
}

// GlobalStats represents global notification statistics.
type GlobalStats struct {
	AverageReadTime      float64                     `json:"average_read_time"`
	AverageActionRate    float64                     `json:"average_action_rate"`
	TypePopularity       map[NotificationType]int    `json:"type_popularity"`
	ChannelEffectiveness map[DeliveryChannel]float64 `json:"channel_effectiveness"`
	OptimalTimes         []TimeWindow                `json:"optimal_times"`
}

// NewAIEngine creates a new AI engine.
func NewAIEngine(modelFile string) *AIEngine {
	return &AIEngine{
		modelFile:    modelFile,
		userPatterns: make(map[string]*UserPattern),
		globalStats: &GlobalStats{
			TypePopularity:       make(map[NotificationType]int),
			ChannelEffectiveness: make(map[DeliveryChannel]float64),
		},
	}
}

// Load loads AI model from file.
func (ai *AIEngine) Load() error {
	ai.mu.Lock()
	defer ai.mu.Unlock()

	data, err := os.ReadFile(ai.modelFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	type state struct {
		UserPatterns map[string]*UserPattern `json:"user_patterns"`
		GlobalStats  *GlobalStats            `json:"global_stats"`
	}

	var s state
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	ai.userPatterns = s.UserPatterns
	ai.globalStats = s.GlobalStats

	return nil
}

// Save saves AI model to file.
func (ai *AIEngine) Save() error {
	ai.mu.RLock()
	defer ai.mu.RUnlock()

	type state struct {
		UserPatterns map[string]*UserPattern `json:"user_patterns"`
		GlobalStats  *GlobalStats            `json:"global_stats"`
	}

	s := state{
		UserPatterns: ai.userPatterns,
		GlobalStats:  ai.globalStats,
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(ai.modelFile, data, 0o600)
}

// CalculateRelevance calculates how relevant a notification is to a user (0-1).
func (ai *AIEngine) CalculateRelevance(notification *Notification, prefs *NotificationPreferences) float64 {
	ai.mu.RLock()
	defer ai.mu.RUnlock()

	pattern, exists := ai.userPatterns[notification.UserID]
	if !exists {
		// New user - use global stats
		return ai.calculateRelevanceFromGlobal(notification)
	}

	score := 0.0

	// Factor 1: Type engagement (30%)
	if typeScore, ok := pattern.TypeEngagement[notification.Type]; ok {
		score += typeScore * 0.3
	} else {
		score += 0.5 * 0.3 // Neutral
	}

	// Factor 2: Priority response (25%)
	if priorScore, ok := pattern.PriorityResponse[notification.Priority]; ok {
		score += priorScore * 0.25
	} else {
		score += 0.5 * 0.25
	}

	// Factor 3: Current time alignment (20%)
	timeScore := ai.calculateTimeScore(pattern.PreferredTimes)
	score += timeScore * 0.2

	// Factor 4: Channel preference (15%)
	channelScore := 0.0
	for _, channel := range notification.Channels {
		if chanScore, ok := pattern.PreferredChannels[channel]; ok {
			channelScore = math.Max(channelScore, chanScore)
		}
	}
	score += channelScore * 0.15

	// Factor 5: Recent engagement trend (10%)
	recentScore := 1.0 - pattern.DismissRate
	score += recentScore * 0.1

	return score
}

// PredictOptimalDeliveryTime predicts the best time to deliver a notification.
func (ai *AIEngine) PredictOptimalDeliveryTime(userID string) *time.Time {
	ai.mu.RLock()
	defer ai.mu.RUnlock()

	pattern, exists := ai.userPatterns[userID]
	if !exists {
		// Use global optimal times
		return ai.predictFromGlobalTimes()
	}

	// Find the highest weight time window
	var bestWindow *TimeWindow
	maxWeight := 0.0

	for i := range pattern.PreferredTimes {
		if pattern.PreferredTimes[i].Weight > maxWeight {
			maxWeight = pattern.PreferredTimes[i].Weight
			bestWindow = &pattern.PreferredTimes[i]
		}
	}

	if bestWindow == nil {
		return nil
	}

	// Calculate next occurrence of this time window
	now := time.Now()
	targetHour := bestWindow.StartHour

	nextTime := time.Date(now.Year(), now.Month(), now.Day(), targetHour, 0, 0, 0, now.Location())

	if nextTime.Before(now) {
		nextTime = nextTime.Add(24 * time.Hour)
	}

	return &nextTime
}

// LearnFromDelivery learns from notification delivery results.
func (ai *AIEngine) LearnFromDelivery(notification *Notification, results []DeliveryResult) {
	ai.mu.Lock()
	defer ai.mu.Unlock()

	pattern, exists := ai.userPatterns[notification.UserID]
	if !exists {
		pattern = &UserPattern{
			UserID:            notification.UserID,
			PreferredChannels: make(map[DeliveryChannel]float64),
			TypeEngagement:    make(map[NotificationType]float64),
			PriorityResponse:  make(map[Priority]float64),
			ReadTimeByType:    make(map[NotificationType]float64),
		}
		ai.userPatterns[notification.UserID] = pattern
	}

	// Update channel preferences based on delivery success
	for _, result := range results {
		if result.Success {
			// Increase preference for successful channels
			current := pattern.PreferredChannels[result.Channel]
			pattern.PreferredChannels[result.Channel] = ai.lerp(current, 1.0, 0.1)

			// Update global stats
			globalScore := ai.globalStats.ChannelEffectiveness[result.Channel]
			ai.globalStats.ChannelEffectiveness[result.Channel] = ai.lerp(globalScore, 1.0, 0.05)
		} else {
			// Decrease preference for failed channels
			current := pattern.PreferredChannels[result.Channel]
			pattern.PreferredChannels[result.Channel] = ai.lerp(current, 0.0, 0.1)

			globalScore := ai.globalStats.ChannelEffectiveness[result.Channel]
			ai.globalStats.ChannelEffectiveness[result.Channel] = ai.lerp(globalScore, 0.0, 0.05)
		}
	}

	pattern.TotalNotifications++
	pattern.LastUpdated = time.Now()

	// Auto-save periodically
	if pattern.TotalNotifications%10 == 0 {
		go func() {
			if err := ai.Save(); err != nil {
				fmt.Printf("Error saving AI model: %v\n", err)
			}
		}()
	}
}

// LearnFromEngagement learns from user engagement with notification.
func (ai *AIEngine) LearnFromEngagement(userID string, notification *Notification, action string, timeToAction time.Duration) {
	ai.mu.Lock()
	defer ai.mu.Unlock()

	pattern, exists := ai.userPatterns[userID]
	if !exists {
		return
	}

	switch action {
	case string(StatusRead):
		// Update type engagement
		current := pattern.TypeEngagement[notification.Type]
		pattern.TypeEngagement[notification.Type] = ai.lerp(current, 1.0, 0.15)

		// Update priority response
		priorCurrent := pattern.PriorityResponse[notification.Priority]
		pattern.PriorityResponse[notification.Priority] = ai.lerp(priorCurrent, 1.0, 0.15)

		// Update read time
		readSeconds := timeToAction.Seconds()
		if currentTime, ok := pattern.ReadTimeByType[notification.Type]; ok {
			pattern.ReadTimeByType[notification.Type] = (currentTime + readSeconds) / 2
		} else {
			pattern.ReadTimeByType[notification.Type] = readSeconds
		}

		// Update time preferences
		ai.updateTimePreferences(pattern, notification.CreatedAt)

	case "action":
		// User took action - very positive signal
		pattern.ActionRate = ai.lerp(pattern.ActionRate, 1.0, 0.2)

		current := pattern.TypeEngagement[notification.Type]
		pattern.TypeEngagement[notification.Type] = ai.lerp(current, 1.0, 0.25)

	case "dismiss":
		// User dismissed - negative signal
		pattern.DismissRate = ai.lerp(pattern.DismissRate, 1.0, 0.15)

		current := pattern.TypeEngagement[notification.Type]
		pattern.TypeEngagement[notification.Type] = ai.lerp(current, 0.0, 0.1)

	case "snooze":
		// User snoozed - not highly relevant right now
		current := pattern.TypeEngagement[notification.Type]
		pattern.TypeEngagement[notification.Type] = ai.lerp(current, 0.3, 0.1)
	}

	pattern.LastUpdated = time.Now()

	go func() {
		if err := ai.Save(); err != nil {
			fmt.Printf("Error saving AI model: %v\n", err)
		}
	}()
}

// SuggestQuietHours suggests optimal quiet hours based on user patterns.
func (ai *AIEngine) SuggestQuietHours(userID string) []QuietHoursPeriod {
	ai.mu.RLock()
	defer ai.mu.RUnlock()

	pattern, exists := ai.userPatterns[userID]
	if !exists {
		// Return default quiet hours
		return []QuietHoursPeriod{
			{
				Start:    "22:00",
				End:      "08:00",
				Days:     []string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"},
				Timezone: "UTC",
			},
		}
	}

	// Find hours with lowest engagement
	hourEngagement := make(map[int]float64)
	for _, tw := range pattern.PreferredTimes {
		for h := tw.StartHour; h <= tw.EndHour; h++ {
			hourEngagement[h] = tw.Weight
		}
	}

	// Find consecutive hours with low engagement
	quietPeriods := make([]QuietHoursPeriod, 0)

	// Simplified - would use more sophisticated analysis
	quietPeriods = append(quietPeriods, QuietHoursPeriod{
		Start:    "22:00",
		End:      "08:00",
		Days:     []string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"},
		Timezone: "UTC",
	})

	return quietPeriods
}

// Helper methods

func (ai *AIEngine) calculateRelevanceFromGlobal(notification *Notification) float64 {
	// Use global statistics for new users
	score := 0.5 // Base score

	// Adjust based on priority (urgent is always relevant)
	if notification.Priority == PriorityUrgent {
		score = 1.0
	} else if notification.Priority == PriorityHigh {
		score = 0.8
	}

	// Adjust based on global type popularity
	if popularity, ok := ai.globalStats.TypePopularity[notification.Type]; ok {
		// Normalize by max popularity
		maxPop := 0
		for _, pop := range ai.globalStats.TypePopularity {
			if pop > maxPop {
				maxPop = pop
			}
		}
		if maxPop > 0 {
			score += (float64(popularity) / float64(maxPop)) * 0.3
		}
	}

	return math.Min(score, 1.0)
}

func (ai *AIEngine) calculateTimeScore(timeWindows []TimeWindow) float64 {
	if len(timeWindows) == 0 {
		return 0.5
	}

	now := time.Now()
	currentHour := now.Hour()

	maxScore := 0.0
	for _, tw := range timeWindows {
		if currentHour >= tw.StartHour && currentHour <= tw.EndHour {
			if tw.Weight > maxScore {
				maxScore = tw.Weight
			}
		}
	}

	return maxScore
}

func (ai *AIEngine) updateTimePreferences(pattern *UserPattern, notificationTime time.Time) {
	hour := notificationTime.Hour()

	// Find or create time window
	found := false
	for i := range pattern.PreferredTimes {
		if hour >= pattern.PreferredTimes[i].StartHour && hour <= pattern.PreferredTimes[i].EndHour {
			// Increase weight
			pattern.PreferredTimes[i].Weight = ai.lerp(pattern.PreferredTimes[i].Weight, 1.0, 0.1)
			found = true
			break
		}
	}

	if !found {
		// Create new time window (2-hour window)
		pattern.PreferredTimes = append(pattern.PreferredTimes, TimeWindow{
			StartHour: hour,
			EndHour:   (hour + 2) % 24,
			Weight:    0.7,
		})
	}
}

func (ai *AIEngine) predictFromGlobalTimes() *time.Time {
	if len(ai.globalStats.OptimalTimes) == 0 {
		// Default to 9 AM next day
		now := time.Now()
		next := time.Date(now.Year(), now.Month(), now.Day()+1, 9, 0, 0, 0, now.Location())
		return &next
	}

	// Use first optimal time
	now := time.Now()
	targetHour := ai.globalStats.OptimalTimes[0].StartHour

	nextTime := time.Date(now.Year(), now.Month(), now.Day(), targetHour, 0, 0, 0, now.Location())

	if nextTime.Before(now) {
		nextTime = nextTime.Add(24 * time.Hour)
	}

	return &nextTime
}

// lerp performs linear interpolation.
func (ai *AIEngine) lerp(current, target, alpha float64) float64 {
	return current + alpha*(target-current)
}
