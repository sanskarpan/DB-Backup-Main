package ai

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

// ==================== 4. Auto-Categorization & Tagging ====================

// AutoCategorizer automatically categorizes and tags backups using NLP
type AutoCategorizer struct {
	categories map[string]*Category
	tagRules   []*TagRule
	mu         sync.RWMutex
}

// Category represents a backup category
type Category struct {
	Name        string
	Description string
	Keywords    []string
	Priority    int
	Tags        []string
}

// TagRule represents a rule for automatic tagging
type TagRule struct {
	Pattern     *regexp.Regexp
	Tags        []string
	Confidence  float64
	Description string
}

// CategoryResult represents the result of categorization
type CategoryResult struct {
	PrimaryCategory   string
	SecondaryCategory string
	Confidence        float64
	SuggestedTags     []string
	Reasoning         []string
}

// NewAutoCategorizer creates a new auto-categorizer
func NewAutoCategorizer() *AutoCategorizer {
	ac := &AutoCategorizer{
		categories: make(map[string]*Category),
		tagRules:   make([]*TagRule, 0),
	}

	// Initialize default categories
	ac.initializeDefaultCategories()
	ac.initializeTagRules()

	return ac
}

// initializeDefaultCategories sets up default categories
func (ac *AutoCategorizer) initializeDefaultCategories() {
	categories := []*Category{
		{
			Name:        "Production",
			Description: "Production database backups",
			Keywords:    []string{"prod", "production", "live", "master"},
			Priority:    1,
			Tags:        []string{"critical", "high-priority"},
		},
		{
			Name:        "Staging",
			Description: "Staging environment backups",
			Keywords:    []string{"staging", "stage", "stg", "uat"},
			Priority:    2,
			Tags:        []string{"testing", "pre-production"},
		},
		{
			Name:        "Development",
			Description: "Development database backups",
			Keywords:    []string{"dev", "development", "local", "test"},
			Priority:    3,
			Tags:        []string{"low-priority", "development"},
		},
		{
			Name:        "Analytics",
			Description: "Analytics and reporting databases",
			Keywords:    []string{"analytics", "reporting", "warehouse", "bi", "olap"},
			Priority:    2,
			Tags:        []string{"analytics", "reporting"},
		},
		{
			Name:        "Archive",
			Description: "Archived or historical backups",
			Keywords:    []string{"archive", "historical", "old", "deprecated"},
			Priority:    4,
			Tags:        []string{"archive", "historical"},
		},
		{
			Name:        "Compliance",
			Description: "Compliance-required backups",
			Keywords:    []string{"compliance", "audit", "gdpr", "hipaa", "sox"},
			Priority:    1,
			Tags:        []string{"compliance", "regulatory", "legal"},
		},
		{
			Name:        "Disaster Recovery",
			Description: "Disaster recovery backups",
			Keywords:    []string{"dr", "disaster", "recovery", "failover", "replica"},
			Priority:    1,
			Tags:        []string{"dr", "critical", "failover"},
		},
	}

	for _, category := range categories {
		ac.categories[category.Name] = category
	}
}

// initializeTagRules sets up automatic tagging rules
func (ac *AutoCategorizer) initializeTagRules() {
	rules := []struct {
		pattern     string
		tags        []string
		description string
	}{
		{`(?i)(customer|user|account)`, []string{"customer-data", "sensitive"}, "Contains customer data"},
		{`(?i)(payment|transaction|billing)`, []string{"financial", "sensitive", "encrypted"}, "Contains financial data"},
		{`(?i)(mysql|postgres|mongodb)`, []string{"database"}, "Database type"},
		{`(?i)(full|complete)`, []string{"full-backup"}, "Full backup"},
		{`(?i)(incremental|delta)`, []string{"incremental-backup"}, "Incremental backup"},
		{`(?i)(daily|nightly)`, []string{"scheduled", "automated"}, "Scheduled backup"},
		{`(?i)(manual|adhoc|ondemand)`, []string{"manual", "on-demand"}, "Manual backup"},
		{`(?i)(encrypted|secure)`, []string{"encrypted", "secure"}, "Encrypted backup"},
		{`(?i)(compressed|gzip|zstd)`, []string{"compressed"}, "Compressed backup"},
		{`(?i)(cloud|s3|azure|gcs)`, []string{"cloud-storage"}, "Cloud storage"},
	}

	for _, rule := range rules {
		pattern, err := regexp.Compile(rule.pattern)
		if err != nil {
			continue
		}
		ac.tagRules = append(ac.tagRules, &TagRule{
			Pattern:     pattern,
			Tags:        rule.tags,
			Confidence:  0.95,
			Description: rule.description,
		})
	}
}

// Categorize categorizes a backup based on its metadata
func (ac *AutoCategorizer) Categorize(ctx context.Context, backupName, databaseName, description string, metadata map[string]interface{}) (*CategoryResult, error) {
	ac.mu.RLock()
	defer ac.mu.RUnlock()

	// Combine all text for analysis
	allText := strings.ToLower(fmt.Sprintf("%s %s %s", backupName, databaseName, description))

	// Score each category
	type categoryScore struct {
		category string
		score    float64
	}

	scores := make([]categoryScore, 0)

	for name, category := range ac.categories {
		score := 0.0
		matchedKeywords := 0

		for _, keyword := range category.Keywords {
			if strings.Contains(allText, strings.ToLower(keyword)) {
				score += 1.0
				matchedKeywords++
			}
		}

		// Bonus for priority
		score += float64(5-category.Priority) * 0.1

		if score > 0 {
			scores = append(scores, categoryScore{
				category: name,
				score:    score,
			})
		}
	}

	if len(scores) == 0 {
		return &CategoryResult{
			PrimaryCategory: "Uncategorized",
			Confidence:      0.0,
			SuggestedTags:   []string{"uncategorized"},
			Reasoning:       []string{"No matching category found"},
		}, nil
	}

	// Sort by score
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].score > scores[j].score
	})

	// Get suggested tags from rules
	suggestedTags := make(map[string]bool)
	reasoning := make([]string, 0)

	for _, rule := range ac.tagRules {
		if rule.Pattern.MatchString(allText) {
			for _, tag := range rule.Tags {
				suggestedTags[tag] = true
			}
			reasoning = append(reasoning, rule.Description)
		}
	}

	// Add category tags
	if category, exists := ac.categories[scores[0].category]; exists {
		for _, tag := range category.Tags {
			suggestedTags[tag] = true
		}
	}

	// Convert to slice
	tags := make([]string, 0)
	for tag := range suggestedTags {
		tags = append(tags, tag)
	}
	sort.Strings(tags)

	// Calculate confidence (based on score strength)
	confidence := math.Min(100.0, (scores[0].score/float64(len(ac.categories[scores[0].category].Keywords)))*100)

	result := &CategoryResult{
		PrimaryCategory: scores[0].category,
		Confidence:      confidence,
		SuggestedTags:   tags,
		Reasoning:       reasoning,
	}

	if len(scores) > 1 {
		result.SecondaryCategory = scores[1].category
	}

	return result, nil
}

// ==================== 5. Duplicate Detection ====================

// DuplicateDetector detects duplicate backups across databases
type DuplicateDetector struct {
	backupHashes map[string]*BackupHash // hash -> backup info
	mu           sync.RWMutex
}

// BackupHash represents a backup's content hash
type BackupHash struct {
	Hash         string
	DatabaseName string
	BackupID     string
	Size         int64
	Timestamp    time.Time
	ContentHash  string // SHA-256 of content
	MetadataHash string // SHA-256 of metadata
}

// DuplicateReport represents detected duplicates
type DuplicateReport struct {
	BackupID          string
	DatabaseName      string
	Duplicates        []*DuplicateMatch
	TotalDuplicates   int
	PotentialSavings  int64 // bytes
	Confidence        float64
	Recommendations   []string
}

// DuplicateMatch represents a duplicate match
type DuplicateMatch struct {
	BackupID       string
	DatabaseName   string
	MatchType      string  // "exact", "partial", "similar"
	Similarity     float64 // 0-1
	SizeDifference int64
	TimeDifference time.Duration
}

// NewDuplicateDetector creates a new duplicate detector
func NewDuplicateDetector() *DuplicateDetector {
	return &DuplicateDetector{
		backupHashes: make(map[string]*BackupHash),
	}
}

// IndexBackup indexes a backup for duplicate detection
func (dd *DuplicateDetector) IndexBackup(backupID, database string, size int64, content []byte, metadata map[string]interface{}) error {
	dd.mu.Lock()
	defer dd.mu.Unlock()

	// Calculate content hash
	contentHash := sha256.Sum256(content)
	contentHashStr := hex.EncodeToString(contentHash[:])

	// Calculate metadata hash
	metadataStr := fmt.Sprintf("%v", metadata)
	metadataHash := sha256.Sum256([]byte(metadataStr))
	metadataHashStr := hex.EncodeToString(metadataHash[:])

	backupHash := &BackupHash{
		Hash:         contentHashStr,
		DatabaseName: database,
		BackupID:     backupID,
		Size:         size,
		Timestamp:    time.Now(),
		ContentHash:  contentHashStr,
		MetadataHash: metadataHashStr,
	}

	dd.backupHashes[contentHashStr] = backupHash

	return nil
}

// DetectDuplicates finds duplicate backups
func (dd *DuplicateDetector) DetectDuplicates(ctx context.Context, backupID, database string, content []byte) (*DuplicateReport, error) {
	dd.mu.RLock()
	defer dd.mu.RUnlock()

	// Calculate hash for target backup
	targetHash := sha256.Sum256(content)
	targetHashStr := hex.EncodeToString(targetHash[:])

	duplicates := make([]*DuplicateMatch, 0)
	var totalSavings int64

	// Check for exact matches
	for hash, backup := range dd.backupHashes {
		// Skip self
		if backup.BackupID == backupID && backup.DatabaseName == database {
			continue
		}

		similarity := dd.calculateSimilarity(targetHashStr, hash)

		var matchType string
		if similarity > 0.99 {
			matchType = "exact"
		} else if similarity > 0.80 {
			matchType = "similar"
		} else if similarity > 0.50 {
			matchType = "partial"
		} else {
			continue // Not similar enough
		}

		match := &DuplicateMatch{
			BackupID:       backup.BackupID,
			DatabaseName:   backup.DatabaseName,
			MatchType:      matchType,
			Similarity:     similarity,
			SizeDifference: backup.Size - int64(len(content)),
			TimeDifference: time.Since(backup.Timestamp),
		}

		duplicates = append(duplicates, match)

		if matchType == "exact" {
			totalSavings += backup.Size
		}
	}

	// Sort by similarity
	sort.Slice(duplicates, func(i, j int) bool {
		return duplicates[i].Similarity > duplicates[j].Similarity
	})

	// Generate recommendations
	recommendations := make([]string, 0)
	if len(duplicates) > 0 {
		exactDuplicates := 0
		for _, dup := range duplicates {
			if dup.MatchType == "exact" {
				exactDuplicates++
			}
		}

		if exactDuplicates > 0 {
			recommendations = append(recommendations,
				fmt.Sprintf("Found %d exact duplicate(s) - consider deduplication", exactDuplicates))
			recommendations = append(recommendations,
				fmt.Sprintf("Potential storage savings: %s", formatBytes(totalSavings)))
		}

		if len(duplicates) > exactDuplicates {
			recommendations = append(recommendations,
				fmt.Sprintf("Found %d similar backup(s) - review for consolidation", len(duplicates)-exactDuplicates))
		}
	}

	// Calculate confidence
	confidence := 0.0
	if len(duplicates) > 0 {
		// Higher confidence with more exact matches
		for _, dup := range duplicates {
			if dup.MatchType == "exact" {
				confidence += 100.0
			} else if dup.MatchType == "similar" {
				confidence += dup.Similarity * 80.0
			}
		}
		confidence = math.Min(100.0, confidence/float64(len(duplicates)))
	}

	report := &DuplicateReport{
		BackupID:         backupID,
		DatabaseName:     database,
		Duplicates:       duplicates,
		TotalDuplicates:  len(duplicates),
		PotentialSavings: totalSavings,
		Confidence:       confidence,
		Recommendations:  recommendations,
	}

	return report, nil
}

// calculateSimilarity calculates hash similarity (0-1)
func (dd *DuplicateDetector) calculateSimilarity(hash1, hash2 string) float64 {
	if hash1 == hash2 {
		return 1.0
	}

	// Simple character-by-character comparison
	matches := 0
	minLen := len(hash1)
	if len(hash2) < minLen {
		minLen = len(hash2)
	}

	for i := 0; i < minLen; i++ {
		if hash1[i] == hash2[i] {
			matches++
		}
	}

	return float64(matches) / float64(math.Max(float64(len(hash1)), float64(len(hash2))))
}

// ==================== 6. NLP Smart Search ====================

// NLPSearchEngine provides natural language search capabilities
type NLPSearchEngine struct {
	indexedBackups map[string]*SearchableBackup
	invertedIndex  map[string][]string // term -> backup IDs
	mu             sync.RWMutex
}

// SearchableBackup represents a backup in the search index
type SearchableBackup struct {
	BackupID     string
	DatabaseName string
	Description  string
	Tags         []string
	Metadata     map[string]interface{}
	Timestamp    time.Time
	Size         int64
	Tokens       []string // Tokenized text for search
}

// SearchResult represents a search result
type SearchResult struct {
	BackupID       string
	DatabaseName   string
	Score          float64 // Relevance score 0-100
	MatchedTerms   []string
	Snippet        string
	Highlights     []string
	Timestamp      time.Time
}

// SearchResponse represents the complete search response
type SearchResponse struct {
	Query            string
	Results          []*SearchResult
	TotalResults     int
	SearchTime       time.Duration
	DidYouMean       []string
	RelatedQueries   []string
	Filters          map[string]interface{}
}

// NewNLPSearchEngine creates a new NLP search engine
func NewNLPSearchEngine() *NLPSearchEngine {
	return &NLPSearchEngine{
		indexedBackups: make(map[string]*SearchableBackup),
		invertedIndex:  make(map[string][]string),
	}
}

// IndexBackup indexes a backup for searching
func (nse *NLPSearchEngine) IndexBackup(backup *SearchableBackup) error {
	nse.mu.Lock()
	defer nse.mu.Unlock()

	// Tokenize text
	allText := fmt.Sprintf("%s %s %s", backup.BackupID, backup.DatabaseName, backup.Description)
	for _, tag := range backup.Tags {
		allText += " " + tag
	}

	backup.Tokens = nse.tokenize(allText)

	// Update inverted index
	for _, token := range backup.Tokens {
		if _, exists := nse.invertedIndex[token]; !exists {
			nse.invertedIndex[token] = make([]string, 0)
		}
		nse.invertedIndex[token] = append(nse.invertedIndex[token], backup.BackupID)
	}

	nse.indexedBackups[backup.BackupID] = backup

	return nil
}

// Search performs a natural language search
func (nse *NLPSearchEngine) Search(ctx context.Context, query string, filters map[string]interface{}) (*SearchResponse, error) {
	startTime := time.Now()

	nse.mu.RLock()
	defer nse.mu.RUnlock()

	// Tokenize query
	queryTokens := nse.tokenize(query)

	// Score each backup
	type backupScore struct {
		backupID string
		score    float64
		matched  []string
	}

	scores := make([]backupScore, 0)

	for backupID, backup := range nse.indexedBackups {
		score := 0.0
		matchedTerms := make([]string, 0)

		// Term frequency scoring
		for _, queryToken := range queryTokens {
			for _, backupToken := range backup.Tokens {
				if queryToken == backupToken {
					score += 10.0
					matchedTerms = append(matchedTerms, queryToken)
				} else if strings.Contains(backupToken, queryToken) {
					score += 5.0
					matchedTerms = append(matchedTerms, queryToken)
				}
			}
		}

		// Boost recent backups
		ageHours := time.Since(backup.Timestamp).Hours()
		if ageHours < 24 {
			score *= 1.5
		} else if ageHours < 168 { // 1 week
			score *= 1.2
		}

		// Tag match bonus
		for _, tag := range backup.Tags {
			for _, queryToken := range queryTokens {
				if strings.Contains(strings.ToLower(tag), queryToken) {
					score += 15.0
					matchedTerms = append(matchedTerms, queryToken)
				}
			}
		}

		if score > 0 {
			scores = append(scores, backupScore{
				backupID: backupID,
				score:    score,
				matched:  matchedTerms,
			})
		}
	}

	// Sort by score
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].score > scores[j].score
	})

	// Build results
	results := make([]*SearchResult, 0)
	for _, score := range scores {
		backup := nse.indexedBackups[score.backupID]

		// Normalize score to 0-100
		normalizedScore := math.Min(100.0, score.score/float64(len(queryTokens))*10)

		result := &SearchResult{
			BackupID:     score.backupID,
			DatabaseName: backup.DatabaseName,
			Score:        normalizedScore,
			MatchedTerms: score.matched,
			Snippet:      nse.generateSnippet(backup, queryTokens),
			Highlights:   score.matched,
			Timestamp:    backup.Timestamp,
		}

		results = append(results, result)
	}

	// Generate related queries
	relatedQueries := nse.generateRelatedQueries(query, queryTokens)

	// Implement spell checking for query terms
	backups := make([]*SearchableBackup, 0, len(nse.indexedBackups))
	for _, b := range nse.indexedBackups {
		backups = append(backups, b)
	}
	didYouMean := nse.generateSpellingSuggestions(query, queryTokens, backups)

	response := &SearchResponse{
		Query:          query,
		Results:        results,
		TotalResults:   len(results),
		SearchTime:     time.Since(startTime),
		DidYouMean:     didYouMean,
		RelatedQueries: relatedQueries,
		Filters:        filters,
	}

	return response, nil
}

// tokenize splits text into searchable tokens
func (nse *NLPSearchEngine) tokenize(text string) []string {
	// Convert to lowercase
	text = strings.ToLower(text)

	// Remove special characters
	reg := regexp.MustCompile(`[^a-z0-9\s-]`)
	text = reg.ReplaceAllString(text, " ")

	// Split by whitespace
	tokens := strings.Fields(text)

	// Remove stopwords
	stopwords := map[string]bool{
		"a": true, "an": true, "and": true, "are": true, "as": true, "at": true,
		"be": true, "by": true, "for": true, "from": true, "has": true, "he": true,
		"in": true, "is": true, "it": true, "its": true, "of": true, "on": true,
		"that": true, "the": true, "to": true, "was": true, "will": true, "with": true,
	}

	filtered := make([]string, 0)
	for _, token := range tokens {
		if !stopwords[token] && len(token) > 2 {
			filtered = append(filtered, token)
		}
	}

	return filtered
}

// generateSnippet creates a text snippet highlighting matches
func (nse *NLPSearchEngine) generateSnippet(backup *SearchableBackup, queryTokens []string) string {
	text := backup.Description
	if len(text) == 0 {
		text = fmt.Sprintf("%s backup from %s", backup.DatabaseName, backup.BackupID)
	}

	// Truncate to 150 characters
	if len(text) > 150 {
		text = text[:147] + "..."
	}

	// Highlight matched terms
	for _, token := range queryTokens {
		pattern := regexp.MustCompile(fmt.Sprintf(`(?i)(%s)`, regexp.QuoteMeta(token)))
		text = pattern.ReplaceAllString(text, "**$1**")
	}

	return text
}

// generateRelatedQueries suggests related search queries
func (nse *NLPSearchEngine) generateRelatedQueries(original string, tokens []string) []string {
	related := make([]string, 0)

	// Time-based variations
	related = append(related, original+" last week")
	related = append(related, original+" last month")
	related = append(related, "recent "+original)

	// Type-based variations
	related = append(related, original+" full backup")
	related = append(related, original+" incremental")

	return related
}

// ==================== 7. Smart Retention Policy Generator ====================

// SmartRetentionPolicyGenerator generates intelligent retention policies
type SmartRetentionPolicyGenerator struct {
	mu sync.RWMutex
}

// RetentionPolicy represents a smart retention policy
type RetentionPolicy struct {
	Name              string
	DatabaseName      string
	DailyRetention    int // days
	WeeklyRetention   int // weeks
	MonthlyRetention  int // months
	YearlyRetention   int // years
	GFSEnabled        bool // Grandfather-Father-Son
	TotalRetentionDays int
	EstimatedCost     float64
	StorageTier       string
	Reasoning         []string
	ComplianceRules   []string
}

// RetentionRecommendation represents a policy recommendation
type RetentionRecommendation struct {
	RecommendedPolicy *RetentionPolicy
	Alternatives      []*RetentionPolicy
	CostAnalysis      *CostAnalysis
	RiskAssessment    *RiskAssessment
	Confidence        float64
}

// CostAnalysis represents storage cost analysis
type CostAnalysis struct {
	MonthlyStorageCost float64
	AnnualStorageCost  float64
	CostPerBackup      float64
	PotentialSavings   float64
	Optimizations      []string
}

// RiskAssessment represents data loss risk assessment
type RiskAssessment struct {
	DataLossRisk      string // "low", "medium", "high"
	RecoveryPointObjective time.Duration // RPO
	RecoveryTimeObjective  time.Duration // RTO
	ComplianceRisk    string
	Recommendations   []string
}

// NewSmartRetentionPolicyGenerator creates a new generator
func NewSmartRetentionPolicyGenerator() *SmartRetentionPolicyGenerator {
	return &SmartRetentionPolicyGenerator{}
}

// GeneratePolicy generates a smart retention policy
func (srpg *SmartRetentionPolicyGenerator) GeneratePolicy(
	ctx context.Context,
	database string,
	avgSize int64,
	backupFrequency string,
	complianceReqs []string,
	budget float64,
) (*RetentionRecommendation, error) {

	// Determine retention based on compliance
	dailyRetention := 7
	weeklyRetention := 4
	monthlyRetention := 12
	yearlyRetention := 7

	reasoning := make([]string, 0)

	// Adjust for compliance
	for _, req := range complianceReqs {
		req = strings.ToLower(req)
		if strings.Contains(req, "gdpr") {
			yearlyRetention = 6 // GDPR: 6 years max
			reasoning = append(reasoning, "GDPR compliance: max 6 years retention")
		} else if strings.Contains(req, "hipaa") {
			yearlyRetention = 6 // HIPAA: 6 years
			reasoning = append(reasoning, "HIPAA compliance: 6 years retention")
		} else if strings.Contains(req, "sox") {
			yearlyRetention = 7 // SOX: 7 years
			reasoning = append(reasoning, "SOX compliance: 7 years retention")
		} else if strings.Contains(req, "pci") {
			yearlyRetention = 1 // PCI-DSS: 1 year minimum
			reasoning = append(reasoning, "PCI-DSS compliance: 1 year retention")
		}
	}

	// Calculate total retention days
	totalDays := dailyRetention + (weeklyRetention * 7) + (monthlyRetention * 30) + (yearlyRetention * 365)

	// Estimate storage cost
	avgBackupsPerYear := 365 // daily backups
	totalStorage := avgSize * int64(avgBackupsPerYear)
	monthlyStorageCost := float64(totalStorage) / (1024 * 1024 * 1024) * 0.023 // S3 standard pricing

	// Select storage tier based on budget
	storageTier := "Hot"
	if budget > 0 && monthlyStorageCost > budget {
		storageTier = "Cool"
		monthlyStorageCost *= 0.543 // Cool tier is 54.3% cheaper
		reasoning = append(reasoning, fmt.Sprintf("Using Cool storage tier to meet budget of $%.2f/month", budget))
	}

	policy := &RetentionPolicy{
		Name:              fmt.Sprintf("%s-retention-policy", database),
		DatabaseName:      database,
		DailyRetention:    dailyRetention,
		WeeklyRetention:   weeklyRetention,
		MonthlyRetention:  monthlyRetention,
		YearlyRetention:   yearlyRetention,
		GFSEnabled:        true,
		TotalRetentionDays: totalDays,
		EstimatedCost:     monthlyStorageCost,
		StorageTier:       storageTier,
		Reasoning:         reasoning,
		ComplianceRules:   complianceReqs,
	}

	// Generate alternatives
	alternatives := make([]*RetentionPolicy, 0)

	// Minimal retention
	minimal := &RetentionPolicy{
		Name:               "Minimal",
		DailyRetention:     3,
		WeeklyRetention:    2,
		MonthlyRetention:   3,
		YearlyRetention:    0,
		GFSEnabled:         false,
		TotalRetentionDays: 3 + 14 + 90,
		EstimatedCost:      monthlyStorageCost * 0.3,
		StorageTier:        "Hot",
	}
	alternatives = append(alternatives, minimal)

	// Extended retention
	extended := &RetentionPolicy{
		Name:               "Extended",
		DailyRetention:     14,
		WeeklyRetention:    8,
		MonthlyRetention:   24,
		YearlyRetention:    10,
		GFSEnabled:         true,
		TotalRetentionDays: 14 + 56 + 720 + 3650,
		EstimatedCost:      monthlyStorageCost * 2.0,
		StorageTier:        "Cool",
	}
	alternatives = append(alternatives, extended)

	// Cost analysis
	costAnalysis := &CostAnalysis{
		MonthlyStorageCost: monthlyStorageCost,
		AnnualStorageCost:  monthlyStorageCost * 12,
		CostPerBackup:      monthlyStorageCost / 30,
		PotentialSavings:   monthlyStorageCost * 0.457, // Savings from using Cool tier
		Optimizations: []string{
			"Use Cool storage tier for older backups",
			"Enable compression to reduce storage by 50-70%",
			"Implement deduplication for incremental backups",
		},
	}

	// Risk assessment
	riskAssessment := &RiskAssessment{
		DataLossRisk:           "low",
		RecoveryPointObjective: 24 * time.Hour,
		RecoveryTimeObjective:  2 * time.Hour,
		ComplianceRisk:         "low",
		Recommendations: []string{
			"Test recovery procedures monthly",
			"Maintain multiple backup copies (3-2-1 rule)",
			"Store backups in multiple geographic locations",
		},
	}

	recommendation := &RetentionRecommendation{
		RecommendedPolicy: policy,
		Alternatives:      alternatives,
		CostAnalysis:      costAnalysis,
		RiskAssessment:    riskAssessment,
		Confidence:        85.0,
	}

	return recommendation, nil
}

// generateSpellingSuggestions generates "Did you mean?" suggestions using Levenshtein distance
func (nse *NLPSearchEngine) generateSpellingSuggestions(query string, queryTokens []string, backups []*SearchableBackup) []string {
	// Build a corpus of known terms from backups
	corpus := make(map[string]int)

	for _, backup := range backups {
		// Add database names
		tokens := nse.tokenize(backup.DatabaseName)
		for _, token := range tokens {
			corpus[token]++
		}

		// Add tags
		for _, tag := range backup.Tags {
			tagTokens := nse.tokenize(tag)
			for _, token := range tagTokens {
				corpus[token]++
			}
		}

		// Add description words
		descTokens := nse.tokenize(backup.Description)
		for _, token := range descTokens {
			corpus[token]++
		}
	}

	// Find suggestions for each query token
	suggestions := make(map[string]bool)

	for _, queryToken := range queryTokens {
		// Skip if token exists in corpus (correctly spelled)
		if _, exists := corpus[queryToken]; exists {
			continue
		}

		// Find closest matches using Levenshtein distance
		bestMatch := ""
		bestDistance := 999
		threshold := 2 // Maximum edit distance for suggestions

		for corpusWord := range corpus {
			distance := levenshteinDistance(queryToken, corpusWord)
			if distance < bestDistance && distance <= threshold {
				bestDistance = distance
				bestMatch = corpusWord
			}
		}

		// If we found a good match, add it to suggestions
		if bestMatch != "" && bestDistance > 0 {
			// Build suggested query by replacing the token
			suggestedQuery := strings.Replace(strings.ToLower(query), queryToken, bestMatch, 1)
			suggestions[suggestedQuery] = true
		}
	}

	// Convert map to slice
	result := make([]string, 0, len(suggestions))
	for suggestion := range suggestions {
		result = append(result, suggestion)
	}

	// Limit to top 3 suggestions
	if len(result) > 3 {
		result = result[:3]
	}

	return result
}

// levenshteinDistance calculates the Levenshtein distance between two strings
func levenshteinDistance(s1, s2 string) int {
	// Create matrix
	len1 := len(s1)
	len2 := len(s2)

	// Early exit for empty strings
	if len1 == 0 {
		return len2
	}
	if len2 == 0 {
		return len1
	}

	// Initialize matrix
	matrix := make([][]int, len1+1)
	for i := range matrix {
		matrix[i] = make([]int, len2+1)
		matrix[i][0] = i
	}
	for j := range matrix[0] {
		matrix[0][j] = j
	}

	// Calculate distances
	for i := 1; i <= len1; i++ {
		for j := 1; j <= len2; j++ {
			cost := 0
			if s1[i-1] != s2[j-1] {
				cost = 1
			}

			matrix[i][j] = min(
				matrix[i-1][j]+1,      // deletion
				matrix[i][j-1]+1,      // insertion
				matrix[i-1][j-1]+cost, // substitution
			)
		}
	}

	return matrix[len1][len2]
}

// min returns the minimum of three integers
func min(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}

// ==================== Continue with remaining features ====================

// Additional helper types and functions will go in smart_features_part3.go
