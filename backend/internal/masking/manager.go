package masking

import (
	"fmt"
	"sync"
)

// MaskingRule represents a rule for masking a specific column.
type MaskingRule struct {
	ID             string
	Table          string
	Column         string
	Strategy       MaskingStrategy
	Config         *MaskingConfig
	Priority       int
	Enabled        bool
	Description    string
	PreserveNulls  bool
	ReferentialKey string // For referential integrity preservation
	Metadata       map[string]interface{}
}

// ReferentialIntegrityRule represents a referential integrity relationship.
type ReferentialIntegrityRule struct {
	ID              string
	SourceTable     string
	SourceColumn    string
	TargetTable     string
	TargetColumn    string
	MaskingStrategy MaskingStrategy
	Description     string
}

// MaskingJob represents a masking operation job.
type MaskingJob struct {
	ID            string
	Database      string
	Tables        []string
	Rules         []*MaskingRule
	Status        string
	TotalRows     int64
	ProcessedRows int64
	MaskedFields  int64
	StartTime     string
	EndTime       string
	Error         string
	Metadata      map[string]interface{}
}

// MaskingManager manages data masking operations.
type MaskingManager struct {
	mu                          sync.RWMutex
	detector                    *PIIDetector
	factory                     *MaskerFactory
	rules                       map[string]*MaskingRule
	referentialRules            map[string]*ReferentialIntegrityRule
	referentialMappings         map[string]map[string]string // key: source_table:source_column:value -> masked_value
	jobs                        map[string]*MaskingJob
	autoDetect                  bool
	autoDetectThreshold         float64
	enforceReferentialIntegrity bool
}

// NewMaskingManager creates a new masking manager.
func NewMaskingManager() *MaskingManager {
	return &MaskingManager{
		detector:                    NewPIIDetector(),
		factory:                     NewMaskerFactory(),
		rules:                       make(map[string]*MaskingRule),
		referentialRules:            make(map[string]*ReferentialIntegrityRule),
		referentialMappings:         make(map[string]map[string]string),
		jobs:                        make(map[string]*MaskingJob),
		autoDetect:                  true,
		autoDetectThreshold:         0.7,
		enforceReferentialIntegrity: true,
	}
}

// AddRule adds a masking rule.
func (mm *MaskingManager) AddRule(rule *MaskingRule) error {
	if rule == nil {
		return fmt.Errorf("rule cannot be nil")
	}

	if rule.Table == "" || rule.Column == "" {
		return fmt.Errorf("table and column must be specified")
	}

	mm.mu.Lock()
	defer mm.mu.Unlock()

	ruleKey := fmt.Sprintf("%s.%s", rule.Table, rule.Column)
	if rule.ID == "" {
		rule.ID = ruleKey
	}

	if rule.Config == nil {
		rule.Config = &MaskingConfig{
			Strategy: rule.Strategy,
			Metadata: make(map[string]interface{}),
		}
	}

	mm.rules[ruleKey] = rule
	return nil
}

// GetRule retrieves a masking rule.
func (mm *MaskingManager) GetRule(table, column string) (*MaskingRule, bool) {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	ruleKey := fmt.Sprintf("%s.%s", table, column)
	rule, exists := mm.rules[ruleKey]
	return rule, exists
}

// RemoveRule removes a masking rule.
func (mm *MaskingManager) RemoveRule(table, column string) {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	ruleKey := fmt.Sprintf("%s.%s", table, column)
	delete(mm.rules, ruleKey)
}

// AddReferentialIntegrityRule adds a referential integrity rule.
func (mm *MaskingManager) AddReferentialIntegrityRule(rule *ReferentialIntegrityRule) error {
	if rule == nil {
		return fmt.Errorf("referential integrity rule cannot be nil")
	}

	mm.mu.Lock()
	defer mm.mu.Unlock()

	ruleKey := fmt.Sprintf("%s.%s->%s.%s",
		rule.SourceTable, rule.SourceColumn,
		rule.TargetTable, rule.TargetColumn)

	if rule.ID == "" {
		rule.ID = ruleKey
	}

	mm.referentialRules[ruleKey] = rule

	// Initialize mapping for this relationship
	mappingKey := fmt.Sprintf("%s:%s", rule.SourceTable, rule.SourceColumn)
	if _, exists := mm.referentialMappings[mappingKey]; !exists {
		mm.referentialMappings[mappingKey] = make(map[string]string)
	}

	return nil
}

// MaskValue masks a single value according to rules.
func (mm *MaskingManager) MaskValue(table, column, value string) (string, error) {
	// Check for NULL preservation
	if value == "" {
		rule, exists := mm.GetRule(table, column)
		if exists && rule.PreserveNulls {
			return "", nil
		}
	}

	// Check if referential integrity needs to be preserved
	if mm.enforceReferentialIntegrity {
		mappingKey := fmt.Sprintf("%s:%s", table, column)
		mm.mu.RLock()
		if mapping, exists := mm.referentialMappings[mappingKey]; exists {
			if maskedValue, found := mapping[value]; found {
				mm.mu.RUnlock()
				return maskedValue, nil
			}
		}
		mm.mu.RUnlock()
	}

	// Get masking rule for this column
	rule, exists := mm.GetRule(table, column)
	if !exists {
		// Auto-detect if enabled
		if mm.autoDetect {
			return mm.autoMaskValue(table, column, value)
		}
		return value, nil
	}

	if !rule.Enabled {
		return value, nil
	}

	// Apply masking
	masker := mm.factory.GetMasker(rule.Strategy)
	maskedValue, err := masker.Mask(value, rule.Config)
	if err != nil {
		return value, err
	}

	// Store mapping for referential integrity
	if mm.enforceReferentialIntegrity && rule.ReferentialKey != "" {
		mappingKey := fmt.Sprintf("%s:%s", table, column)
		mm.mu.Lock()
		if _, exists := mm.referentialMappings[mappingKey]; !exists {
			mm.referentialMappings[mappingKey] = make(map[string]string)
		}
		mm.referentialMappings[mappingKey][value] = maskedValue
		mm.mu.Unlock()
	}

	return maskedValue, nil
}

// autoMaskValue automatically masks a value based on PII detection.
func (mm *MaskingManager) autoMaskValue(table, column, value string) (string, error) {
	detections := mm.detector.DetectInColumn(table, column, value)
	if len(detections) == 0 {
		return value, nil
	}

	// Use the highest confidence detection
	bestDetection := detections[0]
	for _, detection := range detections {
		if detection.Confidence > bestDetection.Confidence {
			bestDetection = detection
		}
	}

	// Get suggested strategy
	strategy := mm.detector.SuggestMaskingStrategy(bestDetection.Type)

	// Create config
	config := &MaskingConfig{
		Strategy: strategy,
		Metadata: map[string]interface{}{
			"pii_type":      bestDetection.Type,
			"confidence":    bestDetection.Confidence,
			"auto_detected": true,
		},
	}

	// Apply masking
	masker := mm.factory.GetMasker(strategy)
	return masker.Mask(value, config)
}

// MaskRow masks all values in a row according to rules.
func (mm *MaskingManager) MaskRow(table string, row map[string]string) (map[string]string, error) {
	maskedRow := make(map[string]string)

	for column, value := range row {
		maskedValue, err := mm.MaskValue(table, column, value)
		if err != nil {
			return nil, fmt.Errorf("failed to mask %s.%s: %w", table, column, err)
		}
		maskedRow[column] = maskedValue
	}

	return maskedRow, nil
}

// AnalyzeTable analyzes a table to suggest masking rules.
func (mm *MaskingManager) AnalyzeTable(table string, sampleData map[string][]string) []*MaskingRule {
	rules := make([]*MaskingRule, 0)

	for column, samples := range sampleData {
		// Analyze column for PII
		scores := mm.detector.AnalyzeColumn(table, column, samples, mm.autoDetectThreshold)

		for piiType, score := range scores {
			strategy := mm.detector.SuggestMaskingStrategy(piiType)

			rule := &MaskingRule{
				Table:       table,
				Column:      column,
				Strategy:    strategy,
				Enabled:     true,
				Description: fmt.Sprintf("Auto-detected %s with %.1f%% confidence", piiType, score*100),
				Config: &MaskingConfig{
					Strategy: strategy,
					Metadata: map[string]interface{}{
						"pii_type":        piiType,
						"detection_score": score,
						"auto_detected":   true,
					},
				},
			}

			rules = append(rules, rule)
		}
	}

	return rules
}

// ValidateReferentialIntegrity validates that referential integrity will be preserved.
func (mm *MaskingManager) ValidateReferentialIntegrity() []string {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	warnings := make([]string, 0)

	for _, refRule := range mm.referentialRules {
		// Check if source column has a masking rule
		sourceRule, sourceExists := mm.rules[fmt.Sprintf("%s.%s", refRule.SourceTable, refRule.SourceColumn)]
		targetRule, targetExists := mm.rules[fmt.Sprintf("%s.%s", refRule.TargetTable, refRule.TargetColumn)]

		if sourceExists && !targetExists {
			warnings = append(warnings, fmt.Sprintf(
				"Source column %s.%s is masked but target %s.%s is not",
				refRule.SourceTable, refRule.SourceColumn,
				refRule.TargetTable, refRule.TargetColumn,
			))
		}

		if sourceExists && targetExists {
			// Both are masked - ensure they use compatible strategies
			if sourceRule.Strategy != targetRule.Strategy {
				warnings = append(warnings, fmt.Sprintf(
					"Incompatible masking strategies for referential relationship: %s.%s (%s) -> %s.%s (%s)",
					refRule.SourceTable, refRule.SourceColumn, sourceRule.Strategy,
					refRule.TargetTable, refRule.TargetColumn, targetRule.Strategy,
				))
			}
		}
	}

	return warnings
}

// CreateMaskingJob creates a new masking job.
func (mm *MaskingManager) CreateMaskingJob(database string, tables []string) *MaskingJob {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	job := &MaskingJob{
		ID:       fmt.Sprintf("mask-job-%d", len(mm.jobs)),
		Database: database,
		Tables:   tables,
		Rules:    make([]*MaskingRule, 0),
		Status:   "pending",
		Metadata: make(map[string]interface{}),
	}

	// Collect applicable rules
	for _, rule := range mm.rules {
		for _, table := range tables {
			if rule.Table == table && rule.Enabled {
				job.Rules = append(job.Rules, rule)
			}
		}
	}

	mm.jobs[job.ID] = job
	return job
}

// GetJob retrieves a masking job.
func (mm *MaskingManager) GetJob(jobID string) (*MaskingJob, bool) {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	job, exists := mm.jobs[jobID]
	return job, exists
}

// ListJobs returns all masking jobs.
func (mm *MaskingManager) ListJobs() []*MaskingJob {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	jobs := make([]*MaskingJob, 0, len(mm.jobs))
	for _, job := range mm.jobs {
		jobs = append(jobs, job)
	}
	return jobs
}

// UpdateJobProgress updates job progress.
func (mm *MaskingManager) UpdateJobProgress(jobID string, processedRows, maskedFields int64) error {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	job, exists := mm.jobs[jobID]
	if !exists {
		return fmt.Errorf("job not found: %s", jobID)
	}

	job.ProcessedRows = processedRows
	job.MaskedFields = maskedFields

	if job.TotalRows > 0 {
		progress := float64(processedRows) / float64(job.TotalRows) * 100
		job.Metadata["progress_percentage"] = progress
	}

	return nil
}

// CompleteJob marks a job as completed.
func (mm *MaskingManager) CompleteJob(jobID string, success bool, errorMsg string) error {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	job, exists := mm.jobs[jobID]
	if !exists {
		return fmt.Errorf("job not found: %s", jobID)
	}

	if success {
		job.Status = "completed"
	} else {
		job.Status = "failed"
		job.Error = errorMsg
	}

	return nil
}

// GetStatistics returns masking statistics.
func (mm *MaskingManager) GetStatistics() map[string]interface{} {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	stats := map[string]interface{}{
		"total_rules":                   len(mm.rules),
		"total_referential_rules":       len(mm.referentialRules),
		"total_jobs":                    len(mm.jobs),
		"auto_detect_enabled":           mm.autoDetect,
		"auto_detect_threshold":         mm.autoDetectThreshold,
		"referential_integrity_enabled": mm.enforceReferentialIntegrity,
	}

	// Count rules by strategy
	rulesByStrategy := make(map[MaskingStrategy]int)
	enabledRules := 0
	for _, rule := range mm.rules {
		rulesByStrategy[rule.Strategy]++
		if rule.Enabled {
			enabledRules++
		}
	}
	stats["rules_by_strategy"] = rulesByStrategy
	stats["enabled_rules"] = enabledRules

	// Count jobs by status
	jobsByStatus := make(map[string]int)
	for _, job := range mm.jobs {
		jobsByStatus[job.Status]++
	}
	stats["jobs_by_status"] = jobsByStatus

	// Referential mappings stats
	totalMappings := 0
	for _, mapping := range mm.referentialMappings {
		totalMappings += len(mapping)
	}
	stats["total_referential_mappings"] = totalMappings

	return stats
}

// ExportMaskingConfiguration exports masking configuration.
func (mm *MaskingManager) ExportMaskingConfiguration() map[string]interface{} {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	config := map[string]interface{}{
		"rules":                         mm.rules,
		"referential_rules":             mm.referentialRules,
		"auto_detect":                   mm.autoDetect,
		"auto_detect_threshold":         mm.autoDetectThreshold,
		"enforce_referential_integrity": mm.enforceReferentialIntegrity,
	}

	return config
}

// ClearMappings clears all referential integrity mappings.
func (mm *MaskingManager) ClearMappings() {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	mm.referentialMappings = make(map[string]map[string]string)
}

// SetAutoDetect enables or disables auto-detection.
func (mm *MaskingManager) SetAutoDetect(enabled bool, threshold float64) {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	mm.autoDetect = enabled
	if threshold > 0 && threshold <= 1.0 {
		mm.autoDetectThreshold = threshold
	}
}

// SetEnforceReferentialIntegrity enables or disables referential integrity enforcement.
func (mm *MaskingManager) SetEnforceReferentialIntegrity(enabled bool) {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	mm.enforceReferentialIntegrity = enabled
}

// ListRules returns all masking rules.
func (mm *MaskingManager) ListRules() []*MaskingRule {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	rules := make([]*MaskingRule, 0, len(mm.rules))
	for _, rule := range mm.rules {
		rules = append(rules, rule)
	}
	return rules
}

// ListReferentialRules returns all referential integrity rules.
func (mm *MaskingManager) ListReferentialRules() []*ReferentialIntegrityRule {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	rules := make([]*ReferentialIntegrityRule, 0, len(mm.referentialRules))
	for _, rule := range mm.referentialRules {
		rules = append(rules, rule)
	}
	return rules
}
