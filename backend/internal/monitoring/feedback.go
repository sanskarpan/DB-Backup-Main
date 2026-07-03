package monitoring

import (
	"sync"
	"time"
)

// UserFeedback represents user feedback.
type UserFeedback struct {
	ID        string                 `json:"id"`
	UserID    string                 `json:"user_id"`
	Rating    int                    `json:"rating"` // 1-5
	Category  string                 `json:"category"`
	Comment   string                 `json:"comment"`
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// FeedbackCollector collects user feedback.
type FeedbackCollector struct {
	mu            sync.RWMutex
	feedback      []*UserFeedback
	maxFeedback   int
	averageRating float64
}

// NewFeedbackCollector creates a new feedback collector.
func NewFeedbackCollector() *FeedbackCollector {
	return &FeedbackCollector{
		feedback:    make([]*UserFeedback, 0),
		maxFeedback: 10000,
	}
}

// Collect collects user feedback.
func (fc *FeedbackCollector) Collect(feedback *UserFeedback) error {
	fc.mu.Lock()
	defer fc.mu.Unlock()

	if feedback.ID == "" {
		feedback.ID = generateFeedbackID()
	}

	feedback.Timestamp = time.Now()

	fc.feedback = append(fc.feedback, feedback)
	if len(fc.feedback) > fc.maxFeedback {
		fc.feedback = fc.feedback[1:]
	}

	fc.updateAverageRating()

	return nil
}

// GetAverageRating returns the average user rating.
func (fc *FeedbackCollector) GetAverageRating() float64 {
	fc.mu.RLock()
	defer fc.mu.RUnlock()
	return fc.averageRating
}

// GetRecentFeedback returns recent feedback.
func (fc *FeedbackCollector) GetRecentFeedback(limit int) []*UserFeedback {
	fc.mu.RLock()
	defer fc.mu.RUnlock()

	if limit <= 0 || limit > len(fc.feedback) {
		limit = len(fc.feedback)
	}

	start := len(fc.feedback) - limit
	return fc.feedback[start:]
}

// updateAverageRating updates the average rating.
func (fc *FeedbackCollector) updateAverageRating() {
	if len(fc.feedback) == 0 {
		fc.averageRating = 0
		return
	}

	total := 0
	for _, f := range fc.feedback {
		total += f.Rating
	}

	fc.averageRating = float64(total) / float64(len(fc.feedback))
}

// generateFeedbackID generates a unique feedback ID.
func generateFeedbackID() string {
	return time.Now().Format("20060102150405") + "-fb-" + generateRandomString(6)
}

// generateRandomString generates a random alphanumeric string of given length.
func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}
