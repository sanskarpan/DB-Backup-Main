// Package security provides security features for backup system
package security

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// ThreatIntelligenceFeed provides threat intelligence integration
type ThreatIntelligenceFeed struct {
	mu              sync.RWMutex
	feedURLs        []string
	httpClient      *http.Client
	indicators      map[string]*ThreatIndicator
	lastUpdate      time.Time
	updateInterval  time.Duration
	stopChan        chan struct{}
	running         bool
}

// ThreatIndicator represents a threat indicator
type ThreatIndicator struct {
	ID              string
	Type            IndicatorType
	Value           string
	Confidence      float64 // 0-100
	Severity        Severity
	Description     string
	Source          string
	FirstSeen       time.Time
	LastSeen        time.Time
	Tags            []string
	RansomwareFamily string // e.g., "WannaCry", "Ryuk", "LockBit"
	Active          bool
}

// IndicatorType represents the type of threat indicator
type IndicatorType string

const (
	IndicatorTypeFileHash       IndicatorType = "file_hash"
	IndicatorTypeIPAddress      IndicatorType = "ip_address"
	IndicatorTypeDomain         IndicatorType = "domain"
	IndicatorTypeFilePattern    IndicatorType = "file_pattern"
	IndicatorTypeFileExtension  IndicatorType = "file_extension"
	IndicatorTypeBehavior       IndicatorType = "behavior"
	IndicatorTypeRansomwareNote IndicatorType = "ransom_note"
)

// Severity represents threat severity
type Severity string

const (
	SeverityLow      Severity = "LOW"
	SeverityMedium   Severity = "MEDIUM"
	SeverityHigh     Severity = "HIGH"
	SeverityCritical Severity = "CRITICAL"
)

// ThreatIntelResponse represents a threat intelligence feed response
type ThreatIntelResponse struct {
	Indicators []ThreatIndicator `json:"indicators"`
	UpdatedAt  time.Time         `json:"updated_at"`
	Source     string            `json:"source"`
}

// NewThreatIntelligenceFeed creates a new threat intelligence feed
func NewThreatIntelligenceFeed(feedURLs []string, updateInterval time.Duration) *ThreatIntelligenceFeed {
	return &ThreatIntelligenceFeed{
		feedURLs:  feedURLs,
		indicators: make(map[string]*ThreatIndicator),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		updateInterval: updateInterval,
		stopChan:       make(chan struct{}),
	}
}

// Start starts the threat intelligence feed updater
func (tif *ThreatIntelligenceFeed) Start(ctx context.Context) error {
	tif.mu.Lock()
	if tif.running {
		tif.mu.Unlock()
		return fmt.Errorf("threat intelligence feed already running")
	}
	tif.running = true
	tif.mu.Unlock()

	// Initial update
	if err := tif.Update(ctx); err != nil {
		return fmt.Errorf("initial feed update failed: %w", err)
	}

	// Start background updater
	go tif.backgroundUpdater(ctx)

	return nil
}

// Stop stops the threat intelligence feed updater
func (tif *ThreatIntelligenceFeed) Stop() {
	tif.mu.Lock()
	defer tif.mu.Unlock()

	if tif.running {
		close(tif.stopChan)
		tif.running = false
	}
}

// backgroundUpdater periodically updates threat intelligence feeds
func (tif *ThreatIntelligenceFeed) backgroundUpdater(ctx context.Context) {
	ticker := time.NewTicker(tif.updateInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-tif.stopChan:
			return
		case <-ticker.C:
			if err := tif.Update(ctx); err != nil {
				// Log error but continue
				fmt.Printf("Failed to update threat feeds: %v\n", err)
			}
		}
	}
}

// Update updates threat intelligence from all configured feeds
func (tif *ThreatIntelligenceFeed) Update(ctx context.Context) error {
	tif.mu.Lock()
	defer tif.mu.Unlock()

	newIndicators := make(map[string]*ThreatIndicator)

	// Fetch from all feeds
	for _, feedURL := range tif.feedURLs {
		indicators, err := tif.fetchFeed(ctx, feedURL)
		if err != nil {
			// Log error but continue with other feeds
			fmt.Printf("Failed to fetch feed %s: %v\n", feedURL, err)
			continue
		}

		// Add to combined indicators
		for _, indicator := range indicators {
			// Use ID or value as key
			key := indicator.ID
			if key == "" {
				key = fmt.Sprintf("%s:%s", indicator.Type, indicator.Value)
			}
			newIndicators[key] = &indicator
		}
	}

	// If we successfully fetched from at least one feed, update
	if len(newIndicators) > 0 {
		tif.indicators = newIndicators
		tif.lastUpdate = time.Now()
	}

	return nil
}

// fetchFeed fetches threat intelligence from a feed URL
func (tif *ThreatIntelligenceFeed) fetchFeed(ctx context.Context, feedURL string) ([]ThreatIndicator, error) {
	// In production, this would fetch from real threat intelligence feeds:
	// - MISP (Malware Information Sharing Platform)
	// - AlienVault OTX
	// - Recorded Future
	// - Commercial feeds (CrowdStrike, Palo Alto, etc.)

	req, err := http.NewRequestWithContext(ctx, "GET", feedURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := tif.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch feed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("feed returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var response ThreatIntelResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse feed: %w", err)
	}

	return response.Indicators, nil
}

// CheckThreat checks if a value matches any threat indicators
func (tif *ThreatIntelligenceFeed) CheckThreat(indicatorType IndicatorType, value string) (*ThreatIndicator, bool) {
	tif.mu.RLock()
	defer tif.mu.RUnlock()

	// Check exact match
	key := fmt.Sprintf("%s:%s", indicatorType, value)
	if indicator, exists := tif.indicators[key]; exists && indicator.Active {
		return indicator, true
	}

	// Check all indicators of this type
	for _, indicator := range tif.indicators {
		if indicator.Type == indicatorType && indicator.Value == value && indicator.Active {
			return indicator, true
		}
	}

	return nil, false
}

// GetIndicatorsByType returns all indicators of a specific type
func (tif *ThreatIntelligenceFeed) GetIndicatorsByType(indicatorType IndicatorType) []*ThreatIndicator {
	tif.mu.RLock()
	defer tif.mu.RUnlock()

	indicators := make([]*ThreatIndicator, 0)
	for _, indicator := range tif.indicators {
		if indicator.Type == indicatorType && indicator.Active {
			indicators = append(indicators, indicator)
		}
	}

	return indicators
}

// GetIndicatorsByRansomwareFamily returns all indicators for a specific ransomware family
func (tif *ThreatIntelligenceFeed) GetIndicatorsByRansomwareFamily(family string) []*ThreatIndicator {
	tif.mu.RLock()
	defer tif.mu.RUnlock()

	indicators := make([]*ThreatIndicator, 0)
	for _, indicator := range tif.indicators {
		if indicator.RansomwareFamily == family && indicator.Active {
			indicators = append(indicators, indicator)
		}
	}

	return indicators
}

// GetRansomwareKnownExtensions returns known ransomware file extensions
func (tif *ThreatIntelligenceFeed) GetRansomwareKnownExtensions() []string {
	tif.mu.RLock()
	defer tif.mu.RUnlock()

	extensions := make([]string, 0)
	for _, indicator := range tif.indicators {
		if indicator.Type == IndicatorTypeFileExtension && indicator.Active {
			extensions = append(extensions, indicator.Value)
		}
	}

	return extensions
}

// GetStatistics returns statistics about the threat intelligence feed
func (tif *ThreatIntelligenceFeed) GetStatistics() map[string]interface{} {
	tif.mu.RLock()
	defer tif.mu.RUnlock()

	stats := map[string]interface{}{
		"total_indicators": len(tif.indicators),
		"last_update":      tif.lastUpdate,
		"feed_count":       len(tif.feedURLs),
		"by_type":          make(map[IndicatorType]int),
		"by_severity":      make(map[Severity]int),
		"ransomware_families": make(map[string]int),
	}

	byType := make(map[IndicatorType]int)
	bySeverity := make(map[Severity]int)
	families := make(map[string]int)

	for _, indicator := range tif.indicators {
		if indicator.Active {
			byType[indicator.Type]++
			bySeverity[indicator.Severity]++
			if indicator.RansomwareFamily != "" {
				families[indicator.RansomwareFamily]++
			}
		}
	}

	stats["by_type"] = byType
	stats["by_severity"] = bySeverity
	stats["ransomware_families"] = families

	return stats
}

// AddCustomIndicator adds a custom threat indicator
func (tif *ThreatIntelligenceFeed) AddCustomIndicator(indicator *ThreatIndicator) {
	tif.mu.Lock()
	defer tif.mu.Unlock()

	key := indicator.ID
	if key == "" {
		key = fmt.Sprintf("%s:%s", indicator.Type, indicator.Value)
		indicator.ID = key
	}

	indicator.Active = true
	indicator.FirstSeen = time.Now()
	indicator.LastSeen = time.Now()

	tif.indicators[key] = indicator
}

// RemoveIndicator removes an indicator
func (tif *ThreatIntelligenceFeed) RemoveIndicator(id string) {
	tif.mu.Lock()
	defer tif.mu.Unlock()

	delete(tif.indicators, id)
}

// GetMockRansomwareIndicators returns mock ransomware indicators for testing
func GetMockRansomwareIndicators() []ThreatIndicator {
	return []ThreatIndicator{
		{
			ID:               "ind-1",
			Type:             IndicatorTypeFileExtension,
			Value:            ".locked",
			Confidence:       95.0,
			Severity:         SeverityCritical,
			Description:      "Common ransomware file extension",
			Source:           "Internal Database",
			FirstSeen:        time.Now().Add(-30 * 24 * time.Hour),
			LastSeen:         time.Now(),
			Tags:             []string{"ransomware", "encryption"},
			RansomwareFamily: "Generic",
			Active:           true,
		},
		{
			ID:               "ind-2",
			Type:             IndicatorTypeFileExtension,
			Value:            ".encrypted",
			Confidence:       90.0,
			Severity:         SeverityCritical,
			Description:      "Common ransomware file extension",
			Source:           "Internal Database",
			FirstSeen:        time.Now().Add(-30 * 24 * time.Hour),
			LastSeen:         time.Now(),
			Tags:             []string{"ransomware", "encryption"},
			RansomwareFamily: "Generic",
			Active:           true,
		},
		{
			ID:               "ind-3",
			Type:             IndicatorTypeFileExtension,
			Value:            ".wannacry",
			Confidence:       100.0,
			Severity:         SeverityCritical,
			Description:      "WannaCry ransomware file extension",
			Source:           "Public Threat Intelligence",
			FirstSeen:        time.Now().Add(-90 * 24 * time.Hour),
			LastSeen:         time.Now().Add(-30 * 24 * time.Hour),
			Tags:             []string{"ransomware", "WannaCry", "encryption"},
			RansomwareFamily: "WannaCry",
			Active:           true,
		},
		{
			ID:               "ind-4",
			Type:             IndicatorTypeFileExtension,
			Value:            ".ryuk",
			Confidence:       100.0,
			Severity:         SeverityCritical,
			Description:      "Ryuk ransomware file extension",
			Source:           "Public Threat Intelligence",
			FirstSeen:        time.Now().Add(-60 * 24 * time.Hour),
			LastSeen:         time.Now().Add(-10 * 24 * time.Hour),
			Tags:             []string{"ransomware", "Ryuk", "encryption"},
			RansomwareFamily: "Ryuk",
			Active:           true,
		},
		{
			ID:               "ind-5",
			Type:             IndicatorTypeFileExtension,
			Value:            ".lockbit",
			Confidence:       100.0,
			Severity:         SeverityCritical,
			Description:      "LockBit ransomware file extension",
			Source:           "Public Threat Intelligence",
			FirstSeen:        time.Now().Add(-45 * 24 * time.Hour),
			LastSeen:         time.Now().Add(-5 * 24 * time.Hour),
			Tags:             []string{"ransomware", "LockBit", "encryption"},
			RansomwareFamily: "LockBit",
			Active:           true,
		},
		{
			ID:               "ind-6",
			Type:             IndicatorTypeRansomwareNote,
			Value:            "README_DECRYPT.txt",
			Confidence:       85.0,
			Severity:         SeverityCritical,
			Description:      "Common ransomware note filename",
			Source:           "Internal Database",
			FirstSeen:        time.Now().Add(-90 * 24 * time.Hour),
			LastSeen:         time.Now(),
			Tags:             []string{"ransomware", "ransom_note"},
			RansomwareFamily: "Generic",
			Active:           true,
		},
		{
			ID:               "ind-7",
			Type:             IndicatorTypeRansomwareNote,
			Value:            "HOW_TO_DECRYPT.html",
			Confidence:       85.0,
			Severity:         SeverityCritical,
			Description:      "Common ransomware note filename",
			Source:           "Internal Database",
			FirstSeen:        time.Now().Add(-90 * 24 * time.Hour),
			LastSeen:         time.Now(),
			Tags:             []string{"ransomware", "ransom_note"},
			RansomwareFamily: "Generic",
			Active:           true,
		},
		{
			ID:               "ind-8",
			Type:             IndicatorTypeFileHash,
			Value:            "a1b2c3d4e5f6g7h8i9j0",
			Confidence:       100.0,
			Severity:         SeverityCritical,
			Description:      "Known ransomware binary hash",
			Source:           "VirusTotal",
			FirstSeen:        time.Now().Add(-120 * 24 * time.Hour),
			LastSeen:         time.Now().Add(-30 * 24 * time.Hour),
			Tags:             []string{"ransomware", "malware", "executable"},
			RansomwareFamily: "WannaCry",
			Active:           true,
		},
	}
}
