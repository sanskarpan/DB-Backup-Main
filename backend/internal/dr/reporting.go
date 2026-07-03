// Package dr provides disaster recovery testing and validation
package dr

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/sanskarpan/db-backup/pkg/uid"
)

// ReportGenerator generates compliance reports for DR tests.
type ReportGenerator struct {
	mu      sync.RWMutex
	history []TestResult
}

// NewReportGenerator creates a new report generator.
func NewReportGenerator() *ReportGenerator {
	return &ReportGenerator{
		history: make([]TestResult, 0),
	}
}

// AddTestResult adds a test result to the history.
//
//nolint:gocritic // hugeParam: keeps the exported value-semantics API stable.
func (rg *ReportGenerator) AddTestResult(result TestResult) {
	rg.mu.Lock()
	defer rg.mu.Unlock()

	rg.history = append(rg.history, result)

	// Keep only last 1000 results
	if len(rg.history) > 1000 {
		rg.history = rg.history[len(rg.history)-1000:]
	}
}

// GetTestHistory returns test results for a date range.
func (rg *ReportGenerator) GetTestHistory(startDate, endDate time.Time) []TestResult {
	rg.mu.RLock()
	defer rg.mu.RUnlock()

	results := make([]TestResult, 0)
	for i := range rg.history {
		result := &rg.history[i]
		if result.StartTime.After(startDate) && result.StartTime.Before(endDate) {
			results = append(results, *result)
		}
	}

	return results
}

// GetTestHistoryByDatabase returns test results for a specific database.
func (rg *ReportGenerator) GetTestHistoryByDatabase(dbName string, limit int) []TestResult {
	rg.mu.RLock()
	defer rg.mu.RUnlock()

	results := make([]TestResult, 0)
	count := 0

	// Iterate in reverse to get most recent first
	for i := len(rg.history) - 1; i >= 0 && count < limit; i-- {
		if rg.history[i].DatabaseName == dbName {
			results = append(results, rg.history[i])
			count++
		}
	}

	return results
}

// ComplianceReport represents a compliance report.
type ComplianceReport struct {
	ReportID         string
	GeneratedAt      time.Time
	ReportPeriod     ReportPeriod
	DatabaseName     string
	TotalTests       int
	SuccessfulTests  int
	FailedTests      int
	SuccessRate      float64
	AverageRTO       time.Duration
	AverageRPO       time.Duration
	RTOCompliance    float64 // Percentage of tests meeting RTO
	RPOCompliance    float64 // Percentage of tests meeting RPO
	TestResults      []TestResult
	Recommendations  []string
	ComplianceStatus ComplianceStatus
}

// ReportPeriod represents the time period for a report.
type ReportPeriod struct {
	StartDate time.Time
	EndDate   time.Time
	Label     string // e.g., "Q4 2025", "December 2025"
}

// ComplianceStatus represents overall compliance status.
type ComplianceStatus string

const (
	ComplianceStatusPassing ComplianceStatus = "PASSING"
	ComplianceStatusWarning ComplianceStatus = "WARNING"
	ComplianceStatusFailing ComplianceStatus = "FAILING"
)

// GenerateComplianceReport generates a compliance report for a date range.
func (rg *ReportGenerator) GenerateComplianceReport(ctx context.Context, dbName string, startDate, endDate time.Time) (*ComplianceReport, error) {
	results := rg.GetTestHistory(startDate, endDate)

	// Filter by database if specified
	if dbName != "" {
		filtered := make([]TestResult, 0)
		for i := range results {
			if results[i].DatabaseName == dbName {
				filtered = append(filtered, results[i])
			}
		}
		results = filtered
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("no test results found for the specified period")
	}

	report := &ComplianceReport{
		ReportID:    generateReportID(),
		GeneratedAt: time.Now(),
		ReportPeriod: ReportPeriod{
			StartDate: startDate,
			EndDate:   endDate,
			Label:     formatPeriodLabel(startDate, endDate),
		},
		DatabaseName: dbName,
		TestResults:  results,
	}

	// Calculate metrics
	report.calculateMetrics()

	// Generate recommendations
	report.generateRecommendations()

	// Determine compliance status
	report.determineComplianceStatus()

	return report, nil
}

// calculateMetrics calculates report metrics.
func (cr *ComplianceReport) calculateMetrics() {
	cr.TotalTests = len(cr.TestResults)

	var totalRTO, totalRPO time.Duration
	rtoMet := 0
	rpoMet := 0

	for i := range cr.TestResults {
		result := &cr.TestResults[i]
		if result.Success {
			cr.SuccessfulTests++
		} else {
			cr.FailedTests++
		}

		totalRTO += result.RTO
		totalRPO += result.RPO

		if result.RTOMet {
			rtoMet++
		}
		if result.RPOMet {
			rpoMet++
		}
	}

	if cr.TotalTests > 0 {
		cr.SuccessRate = float64(cr.SuccessfulTests) / float64(cr.TotalTests) * 100.0
		cr.AverageRTO = totalRTO / time.Duration(cr.TotalTests)
		cr.AverageRPO = totalRPO / time.Duration(cr.TotalTests)
		cr.RTOCompliance = float64(rtoMet) / float64(cr.TotalTests) * 100.0
		cr.RPOCompliance = float64(rpoMet) / float64(cr.TotalTests) * 100.0
	}
}

// generateRecommendations generates recommendations based on test results.
func (cr *ComplianceReport) generateRecommendations() {
	cr.Recommendations = make([]string, 0)

	// Success rate recommendations
	if cr.SuccessRate < 80.0 {
		cr.Recommendations = append(cr.Recommendations,
			fmt.Sprintf("CRITICAL: Success rate (%.1f%%) is below 80%%. Investigate backup/restore infrastructure immediately.", cr.SuccessRate))
	} else if cr.SuccessRate < 95.0 {
		cr.Recommendations = append(cr.Recommendations,
			fmt.Sprintf("WARNING: Success rate (%.1f%%) is below 95%%. Review recent failures and address root causes.", cr.SuccessRate))
	}

	// RTO compliance recommendations
	if cr.RTOCompliance < 90.0 {
		cr.Recommendations = append(cr.Recommendations,
			fmt.Sprintf("RTO compliance (%.1f%%) is below 90%%. Consider optimizing restore procedures or adjusting RTO thresholds.", cr.RTOCompliance))
	}

	// RPO compliance recommendations
	if cr.RPOCompliance < 90.0 {
		cr.Recommendations = append(cr.Recommendations,
			fmt.Sprintf("RPO compliance (%.1f%%) is below 90%%. Review backup frequency and retention policies.", cr.RPOCompliance))
	}

	// Test frequency recommendations
	if cr.TotalTests < 4 {
		cr.Recommendations = append(cr.Recommendations,
			fmt.Sprintf("Only %d DR test(s) conducted this period. Increase testing frequency to ensure preparedness.", cr.TotalTests))
	}

	// Add positive feedback if all is well
	if cr.SuccessRate >= 95.0 && cr.RTOCompliance >= 90.0 && cr.RPOCompliance >= 90.0 {
		cr.Recommendations = append(cr.Recommendations,
			"Excellent DR testing performance. Continue current testing schedule and procedures.")
	}
}

// determineComplianceStatus determines overall compliance status.
func (cr *ComplianceReport) determineComplianceStatus() {
	// Failing if success rate < 80% or RTO/RPO compliance < 75%
	if cr.SuccessRate < 80.0 || cr.RTOCompliance < 75.0 || cr.RPOCompliance < 75.0 {
		cr.ComplianceStatus = ComplianceStatusFailing
		return
	}

	// Warning if success rate < 95% or RTO/RPO compliance < 90%
	if cr.SuccessRate < 95.0 || cr.RTOCompliance < 90.0 || cr.RPOCompliance < 90.0 {
		cr.ComplianceStatus = ComplianceStatusWarning
		return
	}

	cr.ComplianceStatus = ComplianceStatusPassing
}

// ExportToCSV exports the compliance report to CSV format.
func (cr *ComplianceReport) ExportToCSV() ([]byte, error) {
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	// Write header section
	header := [][]string{
		{"Disaster Recovery Compliance Report"},
		{"Report ID", cr.ReportID},
		{"Generated At", cr.GeneratedAt.Format(time.RFC3339)},
		{"Report Period", cr.ReportPeriod.Label},
		{"Database", cr.DatabaseName},
		{""},
		{"Summary Metrics"},
		{"Total Tests", fmt.Sprintf("%d", cr.TotalTests)},
		{"Successful Tests", fmt.Sprintf("%d", cr.SuccessfulTests)},
		{"Failed Tests", fmt.Sprintf("%d", cr.FailedTests)},
		{"Success Rate", fmt.Sprintf("%.2f%%", cr.SuccessRate)},
		{"Average RTO", cr.AverageRTO.String()},
		{"Average RPO", cr.AverageRPO.String()},
		{"RTO Compliance", fmt.Sprintf("%.2f%%", cr.RTOCompliance)},
		{"RPO Compliance", fmt.Sprintf("%.2f%%", cr.RPOCompliance)},
		{"Compliance Status", string(cr.ComplianceStatus)},
		{""},
		{"Recommendations"},
	}

	for _, row := range header {
		if err := writer.Write(row); err != nil {
			return nil, fmt.Errorf("failed to write header: %w", err)
		}
	}

	// Write recommendations
	for _, rec := range cr.Recommendations {
		if err := writer.Write([]string{rec}); err != nil {
			return nil, fmt.Errorf("failed to write recommendation: %w", err)
		}
	}

	// Write empty row and detailed test result headers
	detailRows := [][]string{
		{""},
		{"Detailed Test Results"},
		{
			"Test ID",
			"Database",
			"Start Time",
			"Duration",
			"Success",
			"RTO",
			"RTO Met",
			"RPO",
			"RPO Met",
			"Schema Valid",
			"Row Count Valid",
			"Sample Data Valid",
			"Smoke Tests Passed",
			"Smoke Tests Failed",
			"Cleaned Up",
		},
	}
	for _, row := range detailRows {
		if err := writer.Write(row); err != nil {
			return nil, fmt.Errorf("failed to write detail header: %w", err)
		}
	}

	// Sort results by start time
	sortedResults := make([]TestResult, len(cr.TestResults))
	copy(sortedResults, cr.TestResults)
	sort.Slice(sortedResults, func(i, j int) bool {
		return sortedResults[i].StartTime.Before(sortedResults[j].StartTime)
	})

	for i := range sortedResults {
		result := &sortedResults[i]
		row := []string{
			result.TestID,
			result.DatabaseName,
			result.StartTime.Format(time.RFC3339),
			result.Duration.String(),
			boolToString(result.Success),
			result.RTO.String(),
			boolToString(result.RTOMet),
			result.RPO.String(),
			boolToString(result.RPOMet),
			boolToString(result.SchemaValid),
			boolToString(result.RowCountValid),
			boolToString(result.SampleDataValid),
			fmt.Sprintf("%d", result.SmokeTestsPassed),
			fmt.Sprintf("%d", result.SmokeTestsFailed),
			boolToString(result.CleanedUp),
		}
		if err := writer.Write(row); err != nil {
			return nil, fmt.Errorf("failed to write test result: %w", err)
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, fmt.Errorf("CSV writer error: %w", err)
	}

	return buf.Bytes(), nil
}

// ExportToPDF exports the compliance report to PDF format.
func (cr *ComplianceReport) ExportToPDF() ([]byte, error) {
	// Generate PDF content as HTML-like format
	// In production, use a proper PDF library like github.com/jung-kurt/gofpdf
	// or github.com/signintech/gopdf

	var buf bytes.Buffer

	// Write PDF header (simplified - real implementation would use PDF library)
	buf.WriteString("%PDF-1.4\n")
	buf.WriteString("% Disaster Recovery Compliance Report\n\n")

	// Title
	buf.WriteString("DISASTER RECOVERY COMPLIANCE REPORT\n")
	buf.WriteString(fmt.Sprintf("Report ID: %s\n", cr.ReportID))
	buf.WriteString(fmt.Sprintf("Generated: %s\n", cr.GeneratedAt.Format("January 2, 2006 15:04:05 MST")))
	buf.WriteString(fmt.Sprintf("Period: %s\n", cr.ReportPeriod.Label))
	if cr.DatabaseName != "" {
		buf.WriteString(fmt.Sprintf("Database: %s\n", cr.DatabaseName))
	} else {
		buf.WriteString("Database: All Databases\n")
	}
	buf.WriteString("\n")

	// Compliance Status Banner
	buf.WriteString("═══════════════════════════════════════════════════════════\n")
	buf.WriteString(fmt.Sprintf("COMPLIANCE STATUS: %s\n", cr.ComplianceStatus))
	buf.WriteString("═══════════════════════════════════════════════════════════\n\n")

	// Executive Summary
	buf.WriteString("EXECUTIVE SUMMARY\n")
	buf.WriteString("─────────────────────────────────────────────────────────\n")
	buf.WriteString(fmt.Sprintf("Total DR Tests Conducted:    %d\n", cr.TotalTests))
	buf.WriteString(fmt.Sprintf("Successful Tests:            %d\n", cr.SuccessfulTests))
	buf.WriteString(fmt.Sprintf("Failed Tests:                %d\n", cr.FailedTests))
	buf.WriteString(fmt.Sprintf("Success Rate:                %.2f%%\n", cr.SuccessRate))
	buf.WriteString("\n")

	// Performance Metrics
	buf.WriteString("PERFORMANCE METRICS\n")
	buf.WriteString("─────────────────────────────────────────────────────────\n")
	buf.WriteString(fmt.Sprintf("Average RTO (Recovery Time): %s\n", cr.AverageRTO))
	buf.WriteString(fmt.Sprintf("Average RPO (Data Loss):     %s\n", cr.AverageRPO))
	buf.WriteString(fmt.Sprintf("RTO Compliance Rate:         %.2f%%\n", cr.RTOCompliance))
	buf.WriteString(fmt.Sprintf("RPO Compliance Rate:         %.2f%%\n", cr.RPOCompliance))
	buf.WriteString("\n")

	// Recommendations
	if len(cr.Recommendations) > 0 {
		buf.WriteString("RECOMMENDATIONS\n")
		buf.WriteString("─────────────────────────────────────────────────────────\n")
		for i, rec := range cr.Recommendations {
			buf.WriteString(fmt.Sprintf("%d. %s\n", i+1, rec))
		}
		buf.WriteString("\n")
	}

	// Test History Summary
	buf.WriteString("DETAILED TEST HISTORY\n")
	buf.WriteString("─────────────────────────────────────────────────────────\n")

	// Sort results by start time
	sortedResults := make([]TestResult, len(cr.TestResults))
	copy(sortedResults, cr.TestResults)
	sort.Slice(sortedResults, func(i, j int) bool {
		return sortedResults[i].StartTime.Before(sortedResults[j].StartTime)
	})

	for i := range sortedResults {
		result := &sortedResults[i]
		status := "✓ PASS"
		if !result.Success {
			status = "✗ FAIL"
		}

		buf.WriteString(fmt.Sprintf("\nTest: %s [%s]\n", result.TestID, status))
		buf.WriteString(fmt.Sprintf("  Date: %s\n", result.StartTime.Format("2006-01-02 15:04:05")))
		buf.WriteString(fmt.Sprintf("  Database: %s\n", result.DatabaseName))
		buf.WriteString(fmt.Sprintf("  Duration: %s\n", result.Duration))
		buf.WriteString(fmt.Sprintf("  RTO: %s (threshold: %s) - %s\n",
			result.RTO, result.RTOThreshold, boolToPassFail(result.RTOMet)))
		buf.WriteString(fmt.Sprintf("  RPO: %s (threshold: %s) - %s\n",
			result.RPO, result.RPOThreshold, boolToPassFail(result.RPOMet)))
		buf.WriteString(fmt.Sprintf("  Schema Valid: %s\n", boolToPassFail(result.SchemaValid)))
		buf.WriteString(fmt.Sprintf("  Row Counts Valid: %s\n", boolToPassFail(result.RowCountValid)))
		buf.WriteString(fmt.Sprintf("  Sample Data Valid: %s\n", boolToPassFail(result.SampleDataValid)))
		buf.WriteString(fmt.Sprintf("  Smoke Tests: %d passed, %d failed\n",
			result.SmokeTestsPassed, result.SmokeTestsFailed))

		// Include errors if any
		if len(result.SchemaErrors) > 0 {
			buf.WriteString("  Schema Errors:\n")
			for _, err := range result.SchemaErrors {
				buf.WriteString(fmt.Sprintf("    - %s\n", err))
			}
		}
		if len(result.RowCountErrors) > 0 {
			buf.WriteString("  Row Count Errors:\n")
			for _, err := range result.RowCountErrors {
				buf.WriteString(fmt.Sprintf("    - %s\n", err))
			}
		}
		if len(result.SampleDataErrors) > 0 {
			buf.WriteString("  Sample Data Errors:\n")
			for _, err := range result.SampleDataErrors {
				buf.WriteString(fmt.Sprintf("    - %s\n", err))
			}
		}
	}

	buf.WriteString("\n")
	buf.WriteString("═══════════════════════════════════════════════════════════\n")
	buf.WriteString("End of Report\n")
	buf.WriteString("Generated by DB-Backup DR Testing Framework v1.0\n")
	buf.WriteString("═══════════════════════════════════════════════════════════\n")

	return buf.Bytes(), nil
}

// GetComplianceSummary returns a summary of recent compliance.
func (rg *ReportGenerator) GetComplianceSummary(days int) map[string]interface{} {
	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -days)

	results := rg.GetTestHistory(startDate, endDate)

	summary := map[string]interface{}{
		"period_days":    days,
		"total_tests":    len(results),
		"databases":      make(map[string]int),
		"success_rate":   0.0,
		"rto_compliance": 0.0,
		"rpo_compliance": 0.0,
	}

	if len(results) == 0 {
		return summary
	}

	successful := 0
	rtoMet := 0
	rpoMet := 0
	dbCounts := make(map[string]int)

	for i := range results {
		result := &results[i]
		if result.Success {
			successful++
		}
		if result.RTOMet {
			rtoMet++
		}
		if result.RPOMet {
			rpoMet++
		}
		dbCounts[result.DatabaseName]++
	}

	summary["success_rate"] = float64(successful) / float64(len(results)) * 100.0
	summary["rto_compliance"] = float64(rtoMet) / float64(len(results)) * 100.0
	summary["rpo_compliance"] = float64(rpoMet) / float64(len(results)) * 100.0
	summary["databases"] = dbCounts

	return summary
}

// Helper functions

func generateReportID() string {
	return uid.New("report")
}

func formatPeriodLabel(startDate, endDate time.Time) string {
	if startDate.Year() == endDate.Year() {
		if startDate.Month() == endDate.Month() {
			return fmt.Sprintf("%s %d", startDate.Month().String(), startDate.Year())
		}
		return fmt.Sprintf("%s - %s %d",
			startDate.Month().String()[:3],
			endDate.Month().String()[:3],
			startDate.Year())
	}
	return fmt.Sprintf("%s %d - %s %d",
		startDate.Month().String()[:3], startDate.Year(),
		endDate.Month().String()[:3], endDate.Year())
}

func boolToString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

func boolToPassFail(b bool) string {
	if b {
		return "PASS"
	}
	return "FAIL"
}
