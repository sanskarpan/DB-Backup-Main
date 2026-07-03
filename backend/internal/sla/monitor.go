package sla

import (
	"fmt"
	"sync"
	"time"

	"github.com/sanskarpan/db-backup/pkg/uid"
)

// SLALevel represents the criticality level of an SLA.
type SLALevel string

const (
	SLALevelCritical SLALevel = "critical" // Mission-critical, 99.99% uptime
	SLALevelHigh     SLALevel = "high"     // High priority, 99.9% uptime
	SLALevelMedium   SLALevel = "medium"   // Medium priority, 99.5% uptime
	SLALevelLow      SLALevel = "low"      // Low priority, 99% uptime
)

// SLADefinition defines the SLA requirements for a database.
type SLADefinition struct {
	ID                   string
	DatabaseID           string
	DatabaseName         string
	Level                SLALevel
	SuccessRateTarget    float64       // Target success rate (e.g., 99.9%)
	MaxBackupDuration    time.Duration // Maximum acceptable backup duration
	RTOTarget            time.Duration // Recovery Time Objective
	RPOTarget            time.Duration // Recovery Point Objective
	BackupFrequency      time.Duration // Expected backup frequency
	RetentionPeriod      time.Duration // Data retention requirement
	ComplianceStandards  []string      // e.g., ["SOC2", "HIPAA", "GDPR"]
	AlertOnViolation     bool
	AlertThreshold       int // Number of violations before alerting
	ViolationGracePeriod time.Duration
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

// SLAMetrics represents the actual metrics for a database.
type SLAMetrics struct {
	DatabaseID             string
	DatabaseName           string
	TotalBackups           int64
	SuccessfulBackups      int64
	FailedBackups          int64
	SuccessRate            float64
	AverageBackupDuration  time.Duration
	MinBackupDuration      time.Duration
	MaxBackupDuration      time.Duration
	LastBackupTime         time.Time
	LastSuccessfulBackup   time.Time
	ConsecutiveFailures    int
	TotalDataBackedUp      int64 // in bytes
	MeasurementPeriodStart time.Time
	MeasurementPeriodEnd   time.Time
}

// RTOMetrics represents RTO compliance metrics.
type RTOMetrics struct {
	DatabaseID         string
	TargetRTO          time.Duration
	ActualRTO          time.Duration
	LastRestoreTime    time.Time
	RestoreSuccessRate float64
	AverageRestoreTime time.Duration
	FastestRestore     time.Duration
	SlowestRestore     time.Duration
	TotalRestoreTests  int
	PassedRestoreTests int
	FailedRestoreTests int
	IsCompliant        bool
}

// RPOMetrics represents RPO compliance metrics.
type RPOMetrics struct {
	DatabaseID            string
	TargetRPO             time.Duration
	ActualRPO             time.Duration
	DataLossRisk          float64 // Percentage
	BackupFrequency       time.Duration
	LastBackupTimestamp   time.Time
	MaxTimeBetweenBackups time.Duration
	IsCompliant           bool
}

// SLAViolation represents an SLA violation event.
type SLAViolation struct {
	ID                string
	DatabaseID        string
	DatabaseName      string
	ViolationType     string // "success_rate", "duration", "rto", "rpo", "frequency"
	Severity          string // "critical", "high", "medium", "low"
	DetectedAt        time.Time
	ResolvedAt        *time.Time
	IsResolved        bool
	Description       string
	ActualValue       interface{}
	ExpectedValue     interface{}
	Impact            string
	RecommendedAction string
	Metadata          map[string]interface{}
}

// SLAMonitor monitors and tracks SLA compliance.
type SLAMonitor struct {
	mu               sync.RWMutex
	definitions      map[string]*SLADefinition
	metrics          map[string]*SLAMetrics
	rtoMetrics       map[string]*RTOMetrics
	rpoMetrics       map[string]*RPOMetrics
	violations       []*SLAViolation
	alertHandlers    []AlertHandler
	violationCounter int64
}

// AlertHandler is a function that handles SLA violations.
type AlertHandler func(*SLAViolation) error

// NewSLAMonitor creates a new SLA monitor.
func NewSLAMonitor() *SLAMonitor {
	return &SLAMonitor{
		definitions:   make(map[string]*SLADefinition),
		metrics:       make(map[string]*SLAMetrics),
		rtoMetrics:    make(map[string]*RTOMetrics),
		rpoMetrics:    make(map[string]*RPOMetrics),
		violations:    make([]*SLAViolation, 0),
		alertHandlers: make([]AlertHandler, 0),
	}
}

// AddSLADefinition adds or updates an SLA definition.
func (sm *SLAMonitor) AddSLADefinition(def *SLADefinition) error {
	if def == nil {
		return fmt.Errorf("SLA definition cannot be nil")
	}

	if def.DatabaseID == "" {
		return fmt.Errorf("database ID is required")
	}

	// Set default values based on SLA level if not provided
	if def.SuccessRateTarget == 0 {
		def.SuccessRateTarget = sm.getDefaultSuccessRate(def.Level)
	}

	if def.ID == "" {
		def.ID = fmt.Sprintf("sla-%s-%s", def.DatabaseID, uid.Hex(8))
	}

	def.CreatedAt = time.Now()
	def.UpdatedAt = time.Now()

	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.definitions[def.DatabaseID] = def

	// Initialize metrics if not exists
	if _, exists := sm.metrics[def.DatabaseID]; !exists {
		sm.metrics[def.DatabaseID] = &SLAMetrics{
			DatabaseID:             def.DatabaseID,
			DatabaseName:           def.DatabaseName,
			MeasurementPeriodStart: time.Now(),
		}
	}

	return nil
}

// RecordBackup records a backup event and updates metrics.
func (sm *SLAMonitor) RecordBackup(databaseID string, success bool, duration time.Duration, dataSize int64) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	metrics, exists := sm.metrics[databaseID]
	if !exists {
		metrics = &SLAMetrics{
			DatabaseID:             databaseID,
			MeasurementPeriodStart: time.Now(),
		}
		sm.metrics[databaseID] = metrics
	}

	metrics.TotalBackups++
	metrics.LastBackupTime = time.Now()
	metrics.TotalDataBackedUp += dataSize

	if success {
		metrics.SuccessfulBackups++
		metrics.LastSuccessfulBackup = time.Now()
		metrics.ConsecutiveFailures = 0
	} else {
		metrics.FailedBackups++
		metrics.ConsecutiveFailures++
	}

	// Update success rate
	metrics.SuccessRate = float64(metrics.SuccessfulBackups) / float64(metrics.TotalBackups) * 100

	// Update duration metrics
	if metrics.TotalBackups == 1 {
		metrics.MinBackupDuration = duration
		metrics.MaxBackupDuration = duration
		metrics.AverageBackupDuration = duration
	} else {
		if duration < metrics.MinBackupDuration {
			metrics.MinBackupDuration = duration
		}
		if duration > metrics.MaxBackupDuration {
			metrics.MaxBackupDuration = duration
		}
		// Calculate rolling average
		totalDuration := metrics.AverageBackupDuration.Nanoseconds() * (metrics.TotalBackups - 1)
		metrics.AverageBackupDuration = time.Duration((totalDuration + duration.Nanoseconds()) / metrics.TotalBackups)
	}

	metrics.MeasurementPeriodEnd = time.Now()

	// Check for SLA violations
	sm.checkViolations(databaseID)

	return nil
}

// RecordRestoreTest records a restore test for RTO tracking.
func (sm *SLAMonitor) RecordRestoreTest(databaseID string, success bool, duration time.Duration) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	rto, exists := sm.rtoMetrics[databaseID]
	if !exists {
		def := sm.definitions[databaseID]
		rto = &RTOMetrics{
			DatabaseID:     databaseID,
			TargetRTO:      def.RTOTarget,
			FastestRestore: duration,
			SlowestRestore: duration,
		}
		sm.rtoMetrics[databaseID] = rto
	}

	rto.TotalRestoreTests++
	rto.LastRestoreTime = time.Now()
	rto.ActualRTO = duration

	if success {
		rto.PassedRestoreTests++
	} else {
		rto.FailedRestoreTests++
	}

	rto.RestoreSuccessRate = float64(rto.PassedRestoreTests) / float64(rto.TotalRestoreTests) * 100

	// Update duration metrics
	if duration < rto.FastestRestore {
		rto.FastestRestore = duration
	}
	if duration > rto.SlowestRestore {
		rto.SlowestRestore = duration
	}

	// Calculate average
	totalDuration := rto.AverageRestoreTime.Nanoseconds() * int64(rto.TotalRestoreTests-1)
	rto.AverageRestoreTime = time.Duration((totalDuration + duration.Nanoseconds()) / int64(rto.TotalRestoreTests))

	// Check RTO compliance
	rto.IsCompliant = duration <= rto.TargetRTO

	if !rto.IsCompliant {
		sm.recordViolation(databaseID, "rto", "high", fmt.Sprintf("Actual RTO (%v) exceeds target (%v)", duration, rto.TargetRTO))
	}

	return nil
}

// UpdateRPOMetrics updates RPO metrics based on backup frequency.
func (sm *SLAMonitor) UpdateRPOMetrics(databaseID string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	def, defExists := sm.definitions[databaseID]
	metrics, metricsExist := sm.metrics[databaseID]

	if !defExists || !metricsExist {
		return fmt.Errorf("no SLA definition or metrics for database %s", databaseID)
	}

	rpo, exists := sm.rpoMetrics[databaseID]
	if !exists {
		rpo = &RPOMetrics{
			DatabaseID: databaseID,
			TargetRPO:  def.RPOTarget,
		}
		sm.rpoMetrics[databaseID] = rpo
	}

	// Calculate actual RPO (time since last successful backup)
	rpo.ActualRPO = time.Since(metrics.LastSuccessfulBackup)
	rpo.LastBackupTimestamp = metrics.LastBackupTime
	rpo.BackupFrequency = def.BackupFrequency

	// Calculate data loss risk
	if rpo.ActualRPO > rpo.TargetRPO {
		rpo.DataLossRisk = float64(rpo.ActualRPO-rpo.TargetRPO) / float64(rpo.TargetRPO) * 100
		rpo.IsCompliant = false

		sm.recordViolation(databaseID, "rpo", "critical", fmt.Sprintf("Actual RPO (%v) exceeds target (%v)", rpo.ActualRPO, rpo.TargetRPO))
	} else {
		rpo.DataLossRisk = 0
		rpo.IsCompliant = true
	}

	return nil
}

// checkViolations checks for SLA violations.
func (sm *SLAMonitor) checkViolations(databaseID string) {
	def, defExists := sm.definitions[databaseID]
	metrics, metricsExist := sm.metrics[databaseID]

	if !defExists || !metricsExist {
		return
	}

	// Check success rate violation
	if metrics.SuccessRate < def.SuccessRateTarget {
		severity := sm.getSeverity(def.Level)
		description := fmt.Sprintf("Success rate (%.2f%%) below target (%.2f%%)", metrics.SuccessRate, def.SuccessRateTarget)
		sm.recordViolation(databaseID, "success_rate", severity, description)
	}

	// Check backup duration violation
	if def.MaxBackupDuration > 0 && metrics.AverageBackupDuration > def.MaxBackupDuration {
		description := fmt.Sprintf("Average backup duration (%v) exceeds maximum (%v)", metrics.AverageBackupDuration, def.MaxBackupDuration)
		sm.recordViolation(databaseID, "duration", "medium", description)
	}

	// Check backup frequency violation
	if def.BackupFrequency > 0 {
		timeSinceLastBackup := time.Since(metrics.LastBackupTime)
		if timeSinceLastBackup > def.BackupFrequency*2 {
			description := fmt.Sprintf("Last backup was %v ago, expected every %v", timeSinceLastBackup, def.BackupFrequency)
			sm.recordViolation(databaseID, "frequency", "high", description)
		}
	}
}

// recordViolation records an SLA violation.
func (sm *SLAMonitor) recordViolation(databaseID, violationType, severity, description string) {
	def := sm.definitions[databaseID]

	sm.violationCounter++
	violation := &SLAViolation{
		ID:            fmt.Sprintf("viol-%d", sm.violationCounter),
		DatabaseID:    databaseID,
		DatabaseName:  def.DatabaseName,
		ViolationType: violationType,
		Severity:      severity,
		DetectedAt:    time.Now(),
		IsResolved:    false,
		Description:   description,
		Metadata:      make(map[string]interface{}),
	}

	sm.violations = append(sm.violations, violation)

	// Trigger alerts if enabled
	if def.AlertOnViolation {
		for _, handler := range sm.alertHandlers {
			go handler(violation)
		}
	}
}

// GetMetrics returns the SLA metrics for a database.
func (sm *SLAMonitor) GetMetrics(databaseID string) (*SLAMetrics, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	metrics, exists := sm.metrics[databaseID]
	return metrics, exists
}

// GetRTOMetrics returns the RTO metrics for a database.
func (sm *SLAMonitor) GetRTOMetrics(databaseID string) (*RTOMetrics, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	rto, exists := sm.rtoMetrics[databaseID]
	return rto, exists
}

// GetRPOMetrics returns the RPO metrics for a database.
func (sm *SLAMonitor) GetRPOMetrics(databaseID string) (*RPOMetrics, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	rpo, exists := sm.rpoMetrics[databaseID]
	return rpo, exists
}

// GetViolations returns all violations, optionally filtered.
func (sm *SLAMonitor) GetViolations(databaseID string, unresolvedOnly bool) []*SLAViolation {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	violations := make([]*SLAViolation, 0)

	for _, v := range sm.violations {
		if databaseID != "" && v.DatabaseID != databaseID {
			continue
		}
		if unresolvedOnly && v.IsResolved {
			continue
		}
		violations = append(violations, v)
	}

	return violations
}

// ResolveViolation marks a violation as resolved.
func (sm *SLAMonitor) ResolveViolation(violationID string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	for _, v := range sm.violations {
		if v.ID == violationID {
			now := time.Now()
			v.ResolvedAt = &now
			v.IsResolved = true
			return nil
		}
	}

	return fmt.Errorf("violation %s not found", violationID)
}

// RegisterAlertHandler registers an alert handler.
func (sm *SLAMonitor) RegisterAlertHandler(handler AlertHandler) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.alertHandlers = append(sm.alertHandlers, handler)
}

// getDefaultSuccessRate returns the default success rate for an SLA level.
func (sm *SLAMonitor) getDefaultSuccessRate(level SLALevel) float64 {
	switch level {
	case SLALevelCritical:
		return 99.99
	case SLALevelHigh:
		return 99.9
	case SLALevelMedium:
		return 99.5
	case SLALevelLow:
		return 99.0
	default:
		return 99.0
	}
}

// getSeverity returns the severity string for an SLA level.
func (sm *SLAMonitor) getSeverity(level SLALevel) string {
	return string(level)
}

// GetComplianceSummary returns a compliance summary for all databases.
func (sm *SLAMonitor) GetComplianceSummary() map[string]interface{} {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	summary := map[string]interface{}{
		"total_databases":       len(sm.definitions),
		"compliant_databases":   0,
		"violations_count":      len(sm.violations),
		"unresolved_violations": 0,
		"average_success_rate":  0.0,
	}

	totalSuccessRate := 0.0
	compliantCount := 0

	for dbID, def := range sm.definitions {
		metrics := sm.metrics[dbID]
		if metrics != nil && metrics.SuccessRate >= def.SuccessRateTarget {
			compliantCount++
		}
		if metrics != nil {
			totalSuccessRate += metrics.SuccessRate
		}
	}

	summary["compliant_databases"] = compliantCount
	if len(sm.metrics) > 0 {
		summary["average_success_rate"] = totalSuccessRate / float64(len(sm.metrics))
	}

	unresolvedCount := 0
	for _, v := range sm.violations {
		if !v.IsResolved {
			unresolvedCount++
		}
	}
	summary["unresolved_violations"] = unresolvedCount

	return summary
}
