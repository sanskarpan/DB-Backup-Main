package sla

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/sanskarpan/db-backup/pkg/uid"
)

// ReportFormat represents the format of a compliance report.
type ReportFormat string

const (
	ReportFormatJSON     ReportFormat = "json"
	ReportFormatMarkdown ReportFormat = "markdown"
	ReportFormatHTML     ReportFormat = "html"
	ReportFormatText     ReportFormat = "text"
)

// ReportType represents the type of compliance report.
type ReportType string

const (
	ReportTypeExecutive  ReportType = "executive"  // High-level summary for executives
	ReportTypeDetailed   ReportType = "detailed"   // Detailed compliance report
	ReportTypeTrend      ReportType = "trend"      // Historical trend analysis
	ReportTypeViolation  ReportType = "violation"  // Violation-focused report
	ReportTypeCompliance ReportType = "compliance" // Compliance standard report
)

// ComplianceReport represents a generated compliance report.
type ComplianceReport struct {
	ID                string
	Type              ReportType
	Format            ReportFormat
	GeneratedAt       time.Time
	ReportPeriodStart time.Time
	ReportPeriodEnd   time.Time
	Title             string
	Summary           string
	Content           string
	Metadata          map[string]interface{}
}

// ExecutiveSummary represents high-level metrics for executives.
type ExecutiveSummary struct {
	OverallComplianceRate float64
	TotalDatabases        int
	CompliantDatabases    int
	NonCompliantDatabases int
	AverageSuccessRate    float64
	TotalBackups          int64
	SuccessfulBackups     int64
	FailedBackups         int64
	TotalViolations       int
	CriticalViolations    int
	UnresolvedViolations  int
	TopIssues             []string
	Recommendations       []string
	ComplianceByLevel     map[SLALevel]int
	ComplianceByStandard  map[string]bool
}

// DatabaseComplianceDetail represents detailed compliance info for a database.
type DatabaseComplianceDetail struct {
	DatabaseID          string
	DatabaseName        string
	SLALevel            SLALevel
	IsCompliant         bool
	SuccessRate         float64
	SuccessRateTarget   float64
	AverageBackupTime   time.Duration
	MaxBackupTime       time.Duration
	RTOCompliant        bool
	RPOCompliant        bool
	ActualRTO           time.Duration
	TargetRTO           time.Duration
	ActualRPO           time.Duration
	TargetRPO           time.Duration
	ViolationCount      int
	LastBackupTime      time.Time
	ConsecutiveFailures int
	ComplianceStandards []string
	Issues              []string
	Recommendations     []string
}

// TrendData represents historical trend information.
type TrendData struct {
	Date              time.Time
	SuccessRate       float64
	AverageBackupTime time.Duration
	ViolationCount    int
	ComplianceRate    float64
}

// ComplianceReporter generates compliance reports.
type ComplianceReporter struct {
	monitor       *SLAMonitor
	reportHistory []*ComplianceReport
}

// NewComplianceReporter creates a new compliance reporter.
func NewComplianceReporter(monitor *SLAMonitor) *ComplianceReporter {
	return &ComplianceReporter{
		monitor:       monitor,
		reportHistory: make([]*ComplianceReport, 0),
	}
}

// GenerateReport generates a compliance report.
func (cr *ComplianceReporter) GenerateReport(reportType ReportType, format ReportFormat, periodStart, periodEnd time.Time) (*ComplianceReport, error) {
	if cr.monitor == nil {
		return nil, fmt.Errorf("SLA monitor not initialized")
	}

	report := &ComplianceReport{
		ID:                uid.New("report"),
		Type:              reportType,
		Format:            format,
		GeneratedAt:       time.Now(),
		ReportPeriodStart: periodStart,
		ReportPeriodEnd:   periodEnd,
		Metadata:          make(map[string]interface{}),
	}

	var content string
	var err error

	switch reportType {
	case ReportTypeExecutive:
		content, err = cr.generateExecutiveReport(format, periodStart, periodEnd)
		report.Title = "Executive SLA Compliance Summary"
		report.Summary = "High-level overview of SLA compliance and backup operations"
	case ReportTypeDetailed:
		content, err = cr.generateDetailedReport(format, periodStart, periodEnd)
		report.Title = "Detailed SLA Compliance Report"
		report.Summary = "Comprehensive analysis of SLA compliance across all databases"
	case ReportTypeTrend:
		content, err = cr.generateTrendReport(format, periodStart, periodEnd)
		report.Title = "SLA Compliance Trend Analysis"
		report.Summary = "Historical trends and patterns in backup operations"
	case ReportTypeViolation:
		content, err = cr.generateViolationReport(format, periodStart, periodEnd)
		report.Title = "SLA Violation Report"
		report.Summary = "Detailed analysis of SLA violations and incidents"
	case ReportTypeCompliance:
		content, err = cr.generateComplianceStandardReport(format, periodStart, periodEnd)
		report.Title = "Compliance Standard Report"
		report.Summary = "Compliance status against industry standards"
	default:
		return nil, fmt.Errorf("unsupported report type: %s", reportType)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to generate %s report: %w", reportType, err)
	}

	report.Content = content
	cr.reportHistory = append(cr.reportHistory, report)

	return report, nil
}

// generateExecutiveReport generates an executive summary report.
//
//nolint:unparam // period bounds kept for a uniform report-generator signature; used once historical querying is implemented
func (cr *ComplianceReporter) generateExecutiveReport(format ReportFormat, periodStart, periodEnd time.Time) (string, error) {
	summary := cr.getExecutiveSummary()

	switch format {
	case ReportFormatJSON:
		data, err := json.MarshalIndent(summary, "", "  ")
		if err != nil {
			return "", err
		}
		return string(data), nil

	case ReportFormatMarkdown:
		return cr.formatExecutiveMarkdown(summary), nil

	case ReportFormatHTML:
		return cr.formatExecutiveHTML(summary), nil

	case ReportFormatText:
		return cr.formatExecutiveText(summary), nil

	default:
		return "", fmt.Errorf("unsupported format: %s", format)
	}
}

// generateDetailedReport generates a detailed compliance report.
//
//nolint:unparam // period bounds kept for a uniform report-generator signature; used once historical querying is implemented
func (cr *ComplianceReporter) generateDetailedReport(format ReportFormat, periodStart, periodEnd time.Time) (string, error) {
	details := cr.getDatabaseComplianceDetails()

	switch format {
	case ReportFormatJSON:
		data, err := json.MarshalIndent(details, "", "  ")
		if err != nil {
			return "", err
		}
		return string(data), nil

	case ReportFormatMarkdown:
		return cr.formatDetailedMarkdown(details), nil

	case ReportFormatHTML:
		return cr.formatDetailedHTML(details), nil

	case ReportFormatText:
		return cr.formatDetailedText(details), nil

	default:
		return "", fmt.Errorf("unsupported format: %s", format)
	}
}

// generateTrendReport generates a trend analysis report.
func (cr *ComplianceReporter) generateTrendReport(format ReportFormat, periodStart, periodEnd time.Time) (string, error) {
	trends := cr.getTrendData(periodStart, periodEnd)

	switch format {
	case ReportFormatJSON:
		data, err := json.MarshalIndent(trends, "", "  ")
		if err != nil {
			return "", err
		}
		return string(data), nil

	case ReportFormatMarkdown:
		return cr.formatTrendMarkdown(trends), nil

	case ReportFormatHTML:
		return cr.formatTrendHTML(trends), nil

	case ReportFormatText:
		return cr.formatTrendText(trends), nil

	default:
		return "", fmt.Errorf("unsupported format: %s", format)
	}
}

// generateViolationReport generates a violation-focused report.
func (cr *ComplianceReporter) generateViolationReport(format ReportFormat, periodStart, periodEnd time.Time) (string, error) {
	violations := cr.monitor.GetViolations("", false)

	// Filter by period
	periodViolations := make([]*SLAViolation, 0)
	for _, v := range violations {
		if v.DetectedAt.After(periodStart) && v.DetectedAt.Before(periodEnd) {
			periodViolations = append(periodViolations, v)
		}
	}

	switch format {
	case ReportFormatJSON:
		data, err := json.MarshalIndent(periodViolations, "", "  ")
		if err != nil {
			return "", err
		}
		return string(data), nil

	case ReportFormatMarkdown:
		return cr.formatViolationMarkdown(periodViolations), nil

	case ReportFormatHTML:
		return cr.formatViolationHTML(periodViolations), nil

	case ReportFormatText:
		return cr.formatViolationText(periodViolations), nil

	default:
		return "", fmt.Errorf("unsupported format: %s", format)
	}
}

// generateComplianceStandardReport generates a compliance standard report.
//
//nolint:unparam // period bounds kept for a uniform report-generator signature; used once historical querying is implemented
func (cr *ComplianceReporter) generateComplianceStandardReport(format ReportFormat, periodStart, periodEnd time.Time) (string, error) {
	complianceData := cr.getComplianceStandardData()

	switch format {
	case ReportFormatJSON:
		data, err := json.MarshalIndent(complianceData, "", "  ")
		if err != nil {
			return "", err
		}
		return string(data), nil

	case ReportFormatMarkdown:
		return cr.formatComplianceStandardMarkdown(complianceData), nil

	default:
		return "", fmt.Errorf("unsupported format for compliance standard report: %s", format)
	}
}

// getExecutiveSummary generates executive summary data.
func (cr *ComplianceReporter) getExecutiveSummary() *ExecutiveSummary {
	summary := cr.monitor.GetComplianceSummary()

	totalDatabases, _ := summary["total_databases"].(int)
	compliantDatabases, _ := summary["compliant_databases"].(int)
	averageSuccessRate, _ := summary["average_success_rate"].(float64)
	totalViolations, _ := summary["violations_count"].(int)
	unresolvedViolations, _ := summary["unresolved_violations"].(int)

	exec := &ExecutiveSummary{
		TotalDatabases:       totalDatabases,
		CompliantDatabases:   compliantDatabases,
		AverageSuccessRate:   averageSuccessRate,
		TotalViolations:      totalViolations,
		UnresolvedViolations: unresolvedViolations,
		ComplianceByLevel:    make(map[SLALevel]int),
		ComplianceByStandard: make(map[string]bool),
		TopIssues:            make([]string, 0),
		Recommendations:      make([]string, 0),
	}

	exec.NonCompliantDatabases = exec.TotalDatabases - exec.CompliantDatabases
	if exec.TotalDatabases > 0 {
		exec.OverallComplianceRate = float64(exec.CompliantDatabases) / float64(exec.TotalDatabases) * 100
	}

	// Aggregate backup statistics
	cr.monitor.mu.RLock()
	for _, metrics := range cr.monitor.metrics {
		exec.TotalBackups += metrics.TotalBackups
		exec.SuccessfulBackups += metrics.SuccessfulBackups
		exec.FailedBackups += metrics.FailedBackups
	}

	// Count compliance by level
	for _, def := range cr.monitor.definitions {
		exec.ComplianceByLevel[def.Level]++
	}

	// Count critical violations
	for _, v := range cr.monitor.violations {
		if v.Severity == "critical" && !v.IsResolved {
			exec.CriticalViolations++
		}
	}
	cr.monitor.mu.RUnlock()

	// Generate top issues and recommendations
	exec.TopIssues, exec.Recommendations = cr.analyzeIssuesAndRecommendations()

	return exec
}

// getDatabaseComplianceDetails gets detailed compliance info for all databases.
func (cr *ComplianceReporter) getDatabaseComplianceDetails() []*DatabaseComplianceDetail {
	details := make([]*DatabaseComplianceDetail, 0)

	cr.monitor.mu.RLock()
	defer cr.monitor.mu.RUnlock()

	for dbID, def := range cr.monitor.definitions {
		metrics := cr.monitor.metrics[dbID]
		rtoMetrics := cr.monitor.rtoMetrics[dbID]
		rpoMetrics := cr.monitor.rpoMetrics[dbID]

		detail := &DatabaseComplianceDetail{
			DatabaseID:          dbID,
			DatabaseName:        def.DatabaseName,
			SLALevel:            def.Level,
			SuccessRateTarget:   def.SuccessRateTarget,
			TargetRTO:           def.RTOTarget,
			TargetRPO:           def.RPOTarget,
			ComplianceStandards: def.ComplianceStandards,
			Issues:              make([]string, 0),
			Recommendations:     make([]string, 0),
		}

		if metrics != nil {
			detail.SuccessRate = metrics.SuccessRate
			detail.AverageBackupTime = metrics.AverageBackupDuration
			detail.MaxBackupTime = metrics.MaxBackupDuration
			detail.LastBackupTime = metrics.LastBackupTime
			detail.ConsecutiveFailures = metrics.ConsecutiveFailures
			detail.IsCompliant = metrics.SuccessRate >= def.SuccessRateTarget
		}

		if rtoMetrics != nil {
			detail.RTOCompliant = rtoMetrics.IsCompliant
			detail.ActualRTO = rtoMetrics.ActualRTO
		}

		if rpoMetrics != nil {
			detail.RPOCompliant = rpoMetrics.IsCompliant
			detail.ActualRPO = rpoMetrics.ActualRPO
		}

		// Count violations for this database
		for _, v := range cr.monitor.violations {
			if v.DatabaseID == dbID && !v.IsResolved {
				detail.ViolationCount++
				detail.Issues = append(detail.Issues, fmt.Sprintf("%s: %s", v.ViolationType, v.Description))
			}
		}

		// Generate recommendations
		detail.Recommendations = cr.generateRecommendations(detail)

		details = append(details, detail)
	}

	// Sort by compliance status (non-compliant first) and then by name
	sort.Slice(details, func(i, j int) bool {
		if details[i].IsCompliant != details[j].IsCompliant {
			return !details[i].IsCompliant
		}
		return details[i].DatabaseName < details[j].DatabaseName
	})

	return details
}

// getTrendData generates historical trend data.
//
//nolint:unparam // period bounds kept for the intended historical query implementation (see snapshot note below)
func (cr *ComplianceReporter) getTrendData(periodStart, periodEnd time.Time) []*TrendData {
	trends := make([]*TrendData, 0)

	// For now, generate a snapshot of current data
	// In a real implementation, this would query historical data
	cr.monitor.mu.RLock()
	defer cr.monitor.mu.RUnlock()

	trend := &TrendData{
		Date: time.Now(),
	}

	totalDatabases := len(cr.monitor.definitions)
	compliantCount := 0
	totalSuccessRate := 0.0
	totalBackupTime := int64(0)
	backupCount := int64(0)

	for dbID, def := range cr.monitor.definitions {
		metrics := cr.monitor.metrics[dbID]
		if metrics != nil {
			totalSuccessRate += metrics.SuccessRate
			totalBackupTime += metrics.AverageBackupDuration.Nanoseconds()
			backupCount++

			if metrics.SuccessRate >= def.SuccessRateTarget {
				compliantCount++
			}
		}
	}

	if backupCount > 0 {
		trend.SuccessRate = totalSuccessRate / float64(backupCount)
		trend.AverageBackupTime = time.Duration(totalBackupTime / backupCount)
	}

	if totalDatabases > 0 {
		trend.ComplianceRate = float64(compliantCount) / float64(totalDatabases) * 100
	}

	trend.ViolationCount = len(cr.monitor.violations)

	trends = append(trends, trend)

	return trends
}

// getComplianceStandardData gets compliance data for industry standards.
func (cr *ComplianceReporter) getComplianceStandardData() map[string]interface{} {
	standards := make(map[string][]string)
	complianceStatus := make(map[string]bool)

	cr.monitor.mu.RLock()
	defer cr.monitor.mu.RUnlock()

	for dbID, def := range cr.monitor.definitions {
		for _, standard := range def.ComplianceStandards {
			if standards[standard] == nil {
				standards[standard] = make([]string, 0)
			}
			standards[standard] = append(standards[standard], def.DatabaseName)

			// Check if database is compliant
			metrics := cr.monitor.metrics[dbID]
			if metrics != nil && metrics.SuccessRate >= def.SuccessRateTarget {
				complianceStatus[standard] = true
			}
		}
	}

	return map[string]interface{}{
		"standards":         standards,
		"compliance_status": complianceStatus,
	}
}

// analyzeIssuesAndRecommendations analyzes violations and generates recommendations.
func (cr *ComplianceReporter) analyzeIssuesAndRecommendations() (issues, recommendations []string) {
	issues = make([]string, 0)
	recommendations = make([]string, 0)

	cr.monitor.mu.RLock()
	defer cr.monitor.mu.RUnlock()

	// Analyze violations by type
	violationsByType := make(map[string]int)
	for _, v := range cr.monitor.violations {
		if !v.IsResolved {
			violationsByType[v.ViolationType]++
		}
	}

	// Generate issues from violation patterns
	if count := violationsByType["success_rate"]; count > 0 {
		issues = append(issues, fmt.Sprintf("%d databases below target success rate", count))
		recommendations = append(recommendations, "Review backup configurations and increase retry attempts")
	}

	if count := violationsByType["rto"]; count > 0 {
		issues = append(issues, fmt.Sprintf("%d databases exceeding RTO target", count))
		recommendations = append(recommendations, "Consider optimizing restore procedures or increasing infrastructure capacity")
	}

	if count := violationsByType["rpo"]; count > 0 {
		issues = append(issues, fmt.Sprintf("%d databases exceeding RPO target", count))
		recommendations = append(recommendations, "Increase backup frequency to reduce recovery point objectives")
	}

	if count := violationsByType["duration"]; count > 0 {
		issues = append(issues, fmt.Sprintf("%d databases with slow backup performance", count))
		recommendations = append(recommendations, "Optimize backup windows or implement incremental backups")
	}

	// Check for databases with consecutive failures
	for _, metrics := range cr.monitor.metrics {
		if metrics.ConsecutiveFailures >= 3 {
			issues = append(issues, fmt.Sprintf("Database %s has %d consecutive failures", metrics.DatabaseName, metrics.ConsecutiveFailures))
			recommendations = append(recommendations, "Investigate root cause of failures immediately")
		}
	}

	return issues, recommendations
}

// generateRecommendations generates recommendations for a specific database.
func (cr *ComplianceReporter) generateRecommendations(detail *DatabaseComplianceDetail) []string {
	recommendations := make([]string, 0)

	if !detail.IsCompliant {
		recommendations = append(recommendations, "Success rate below target - review backup process")
	}

	if detail.ConsecutiveFailures > 0 {
		recommendations = append(recommendations, fmt.Sprintf("Address %d consecutive failures", detail.ConsecutiveFailures))
	}

	if !detail.RTOCompliant {
		recommendations = append(recommendations, "RTO exceeded - optimize restore procedures")
	}

	if !detail.RPOCompliant {
		recommendations = append(recommendations, "RPO exceeded - increase backup frequency")
	}

	if detail.AverageBackupTime > detail.MaxBackupTime {
		recommendations = append(recommendations, "Backup duration excessive - consider incremental backups")
	}

	return recommendations
}

// formatExecutiveMarkdown formats executive summary as markdown.
func (cr *ComplianceReporter) formatExecutiveMarkdown(summary *ExecutiveSummary) string {
	var sb strings.Builder

	sb.WriteString("# Executive SLA Compliance Summary\n\n")
	sb.WriteString(fmt.Sprintf("**Generated:** %s\n\n", time.Now().Format(time.RFC3339)))

	sb.WriteString("## Overall Metrics\n\n")
	sb.WriteString(fmt.Sprintf("- **Overall Compliance Rate:** %.2f%%\n", summary.OverallComplianceRate))
	sb.WriteString(fmt.Sprintf("- **Total Databases:** %d\n", summary.TotalDatabases))
	sb.WriteString(fmt.Sprintf("- **Compliant:** %d\n", summary.CompliantDatabases))
	sb.WriteString(fmt.Sprintf("- **Non-Compliant:** %d\n", summary.NonCompliantDatabases))
	sb.WriteString(fmt.Sprintf("- **Average Success Rate:** %.2f%%\n\n", summary.AverageSuccessRate))

	sb.WriteString("## Backup Statistics\n\n")
	sb.WriteString(fmt.Sprintf("- **Total Backups:** %d\n", summary.TotalBackups))
	sb.WriteString(fmt.Sprintf("- **Successful:** %d\n", summary.SuccessfulBackups))
	sb.WriteString(fmt.Sprintf("- **Failed:** %d\n\n", summary.FailedBackups))

	sb.WriteString("## Violations\n\n")
	sb.WriteString(fmt.Sprintf("- **Total Violations:** %d\n", summary.TotalViolations))
	sb.WriteString(fmt.Sprintf("- **Critical:** %d\n", summary.CriticalViolations))
	sb.WriteString(fmt.Sprintf("- **Unresolved:** %d\n\n", summary.UnresolvedViolations))

	if len(summary.TopIssues) > 0 {
		sb.WriteString("## Top Issues\n\n")
		for _, issue := range summary.TopIssues {
			sb.WriteString(fmt.Sprintf("- %s\n", issue))
		}
		sb.WriteString("\n")
	}

	if len(summary.Recommendations) > 0 {
		sb.WriteString("## Recommendations\n\n")
		for _, rec := range summary.Recommendations {
			sb.WriteString(fmt.Sprintf("- %s\n", rec))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// formatExecutiveText formats executive summary as plain text.
func (cr *ComplianceReporter) formatExecutiveText(summary *ExecutiveSummary) string {
	var sb strings.Builder

	sb.WriteString("EXECUTIVE SLA COMPLIANCE SUMMARY\n")
	sb.WriteString("================================\n\n")
	sb.WriteString(fmt.Sprintf("Generated: %s\n\n", time.Now().Format(time.RFC3339)))

	sb.WriteString("OVERALL METRICS\n")
	sb.WriteString(fmt.Sprintf("Overall Compliance Rate: %.2f%%\n", summary.OverallComplianceRate))
	sb.WriteString(fmt.Sprintf("Total Databases: %d\n", summary.TotalDatabases))
	sb.WriteString(fmt.Sprintf("Compliant: %d\n", summary.CompliantDatabases))
	sb.WriteString(fmt.Sprintf("Non-Compliant: %d\n\n", summary.NonCompliantDatabases))

	return sb.String()
}

// formatExecutiveHTML formats executive summary as HTML.
func (cr *ComplianceReporter) formatExecutiveHTML(summary *ExecutiveSummary) string {
	var sb strings.Builder

	sb.WriteString("<html><head><title>Executive SLA Compliance Summary</title></head><body>\n")
	sb.WriteString("<h1>Executive SLA Compliance Summary</h1>\n")
	sb.WriteString(fmt.Sprintf("<p>Generated: %s</p>\n", time.Now().Format(time.RFC3339)))
	sb.WriteString("<h2>Overall Metrics</h2>\n")
	sb.WriteString("<ul>\n")
	sb.WriteString(fmt.Sprintf("<li>Overall Compliance Rate: %.2f%%</li>\n", summary.OverallComplianceRate))
	sb.WriteString(fmt.Sprintf("<li>Total Databases: %d</li>\n", summary.TotalDatabases))
	sb.WriteString(fmt.Sprintf("<li>Compliant: %d</li>\n", summary.CompliantDatabases))
	sb.WriteString("</ul>\n")
	sb.WriteString("</body></html>")

	return sb.String()
}

// formatDetailedMarkdown formats detailed report as markdown.
func (cr *ComplianceReporter) formatDetailedMarkdown(details []*DatabaseComplianceDetail) string {
	var sb strings.Builder

	sb.WriteString("# Detailed SLA Compliance Report\n\n")
	sb.WriteString(fmt.Sprintf("**Generated:** %s\n\n", time.Now().Format(time.RFC3339)))

	for _, detail := range details {
		status := "✅ COMPLIANT"
		if !detail.IsCompliant {
			status = "❌ NON-COMPLIANT"
		}

		sb.WriteString(fmt.Sprintf("## %s - %s\n\n", detail.DatabaseName, status))
		sb.WriteString(fmt.Sprintf("- **SLA Level:** %s\n", detail.SLALevel))
		sb.WriteString(fmt.Sprintf("- **Success Rate:** %.2f%% (Target: %.2f%%)\n", detail.SuccessRate, detail.SuccessRateTarget))
		sb.WriteString(fmt.Sprintf("- **Average Backup Time:** %v\n", detail.AverageBackupTime))
		sb.WriteString(fmt.Sprintf("- **RTO Compliant:** %v (Actual: %v, Target: %v)\n", detail.RTOCompliant, detail.ActualRTO, detail.TargetRTO))
		sb.WriteString(fmt.Sprintf("- **RPO Compliant:** %v (Actual: %v, Target: %v)\n", detail.RPOCompliant, detail.ActualRPO, detail.TargetRPO))
		sb.WriteString(fmt.Sprintf("- **Violations:** %d\n", detail.ViolationCount))

		if len(detail.Issues) > 0 {
			sb.WriteString("\n**Issues:**\n")
			for _, issue := range detail.Issues {
				sb.WriteString(fmt.Sprintf("- %s\n", issue))
			}
		}

		if len(detail.Recommendations) > 0 {
			sb.WriteString("\n**Recommendations:**\n")
			for _, rec := range detail.Recommendations {
				sb.WriteString(fmt.Sprintf("- %s\n", rec))
			}
		}

		sb.WriteString("\n---\n\n")
	}

	return sb.String()
}

// formatDetailedText formats detailed report as plain text.
func (cr *ComplianceReporter) formatDetailedText(details []*DatabaseComplianceDetail) string {
	var sb strings.Builder

	sb.WriteString("DETAILED SLA COMPLIANCE REPORT\n")
	sb.WriteString("==============================\n\n")

	for _, detail := range details {
		status := "COMPLIANT"
		if !detail.IsCompliant {
			status = "NON-COMPLIANT"
		}

		sb.WriteString(fmt.Sprintf("%s - %s\n", detail.DatabaseName, status))
		sb.WriteString(fmt.Sprintf("Success Rate: %.2f%% (Target: %.2f%%)\n", detail.SuccessRate, detail.SuccessRateTarget))
		sb.WriteString(fmt.Sprintf("Violations: %d\n\n", detail.ViolationCount))
	}

	return sb.String()
}

// formatDetailedHTML formats detailed report as HTML.
func (cr *ComplianceReporter) formatDetailedHTML(details []*DatabaseComplianceDetail) string {
	var sb strings.Builder

	sb.WriteString("<html><head><title>Detailed SLA Compliance Report</title></head><body>\n")
	sb.WriteString("<h1>Detailed SLA Compliance Report</h1>\n")

	for _, detail := range details {
		status := "COMPLIANT"
		if !detail.IsCompliant {
			status = "NON-COMPLIANT"
		}

		sb.WriteString(fmt.Sprintf("<h2>%s - %s</h2>\n", detail.DatabaseName, status))
		sb.WriteString("<ul>\n")
		sb.WriteString(fmt.Sprintf("<li>Success Rate: %.2f%% (Target: %.2f%%)</li>\n", detail.SuccessRate, detail.SuccessRateTarget))
		sb.WriteString(fmt.Sprintf("<li>Violations: %d</li>\n", detail.ViolationCount))
		sb.WriteString("</ul>\n")
	}

	sb.WriteString("</body></html>")

	return sb.String()
}

// formatViolationMarkdown formats violation report as markdown.
func (cr *ComplianceReporter) formatViolationMarkdown(violations []*SLAViolation) string {
	var sb strings.Builder

	sb.WriteString("# SLA Violation Report\n\n")
	sb.WriteString(fmt.Sprintf("**Generated:** %s\n", time.Now().Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("**Total Violations:** %d\n\n", len(violations)))

	// Group by severity
	bySeverity := make(map[string][]*SLAViolation)
	for _, v := range violations {
		bySeverity[v.Severity] = append(bySeverity[v.Severity], v)
	}

	for _, severity := range []string{"critical", "high", "medium", "low"} {
		viols := bySeverity[severity]
		if len(viols) == 0 {
			continue
		}

		sb.WriteString(fmt.Sprintf("## %s Severity (%d)\n\n", strings.ToUpper(severity), len(viols)))

		for _, v := range viols {
			status := "🔴 UNRESOLVED"
			if v.IsResolved {
				status = "✅ RESOLVED"
			}

			sb.WriteString(fmt.Sprintf("### %s - %s\n\n", v.DatabaseName, status))
			sb.WriteString(fmt.Sprintf("- **Type:** %s\n", v.ViolationType))
			sb.WriteString(fmt.Sprintf("- **Description:** %s\n", v.Description))
			sb.WriteString(fmt.Sprintf("- **Detected:** %s\n", v.DetectedAt.Format(time.RFC3339)))
			if v.ResolvedAt != nil {
				sb.WriteString(fmt.Sprintf("- **Resolved:** %s\n", v.ResolvedAt.Format(time.RFC3339)))
			}
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

// formatViolationText formats violation report as plain text.
func (cr *ComplianceReporter) formatViolationText(violations []*SLAViolation) string {
	var sb strings.Builder

	sb.WriteString("SLA VIOLATION REPORT\n")
	sb.WriteString("====================\n\n")
	sb.WriteString(fmt.Sprintf("Total Violations: %d\n\n", len(violations)))

	for _, v := range violations {
		status := "UNRESOLVED"
		if v.IsResolved {
			status = "RESOLVED"
		}

		sb.WriteString(fmt.Sprintf("%s - %s\n", v.DatabaseName, status))
		sb.WriteString(fmt.Sprintf("Type: %s\n", v.ViolationType))
		sb.WriteString(fmt.Sprintf("Description: %s\n\n", v.Description))
	}

	return sb.String()
}

// formatViolationHTML formats violation report as HTML.
func (cr *ComplianceReporter) formatViolationHTML(violations []*SLAViolation) string {
	var sb strings.Builder

	sb.WriteString("<html><head><title>SLA Violation Report</title></head><body>\n")
	sb.WriteString("<h1>SLA Violation Report</h1>\n")
	sb.WriteString(fmt.Sprintf("<p>Total Violations: %d</p>\n", len(violations)))

	for _, v := range violations {
		status := "UNRESOLVED"
		if v.IsResolved {
			status = "RESOLVED"
		}

		sb.WriteString(fmt.Sprintf("<h2>%s - %s</h2>\n", v.DatabaseName, status))
		sb.WriteString("<ul>\n")
		sb.WriteString(fmt.Sprintf("<li>Type: %s</li>\n", v.ViolationType))
		sb.WriteString(fmt.Sprintf("<li>Description: %s</li>\n", v.Description))
		sb.WriteString("</ul>\n")
	}

	sb.WriteString("</body></html>")

	return sb.String()
}

// formatTrendMarkdown formats trend report as markdown.
func (cr *ComplianceReporter) formatTrendMarkdown(trends []*TrendData) string {
	var sb strings.Builder

	sb.WriteString("# SLA Compliance Trend Analysis\n\n")
	sb.WriteString(fmt.Sprintf("**Generated:** %s\n\n", time.Now().Format(time.RFC3339)))

	sb.WriteString("| Date | Success Rate | Avg Backup Time | Compliance Rate | Violations |\n")
	sb.WriteString("|------|--------------|-----------------|-----------------|------------|\n")

	for _, trend := range trends {
		sb.WriteString(fmt.Sprintf(
			"| %s | %.2f%% | %v | %.2f%% | %d |\n",
			trend.Date.Format("2006-01-02"),
			trend.SuccessRate,
			trend.AverageBackupTime,
			trend.ComplianceRate,
			trend.ViolationCount,
		))
	}

	sb.WriteString("\n")

	return sb.String()
}

// formatTrendText formats trend report as plain text.
func (cr *ComplianceReporter) formatTrendText(trends []*TrendData) string {
	var sb strings.Builder

	sb.WriteString("SLA COMPLIANCE TREND ANALYSIS\n")
	sb.WriteString("=============================\n\n")

	for _, trend := range trends {
		sb.WriteString(fmt.Sprintf("Date: %s\n", trend.Date.Format("2006-01-02")))
		sb.WriteString(fmt.Sprintf("Success Rate: %.2f%%\n", trend.SuccessRate))
		sb.WriteString(fmt.Sprintf("Compliance Rate: %.2f%%\n\n", trend.ComplianceRate))
	}

	return sb.String()
}

// formatTrendHTML formats trend report as HTML.
func (cr *ComplianceReporter) formatTrendHTML(trends []*TrendData) string {
	var sb strings.Builder

	sb.WriteString("<html><head><title>SLA Compliance Trend Analysis</title></head><body>\n")
	sb.WriteString("<h1>SLA Compliance Trend Analysis</h1>\n")
	sb.WriteString("<table border='1'>\n")
	sb.WriteString("<tr><th>Date</th><th>Success Rate</th><th>Compliance Rate</th></tr>\n")

	for _, trend := range trends {
		sb.WriteString(fmt.Sprintf(
			"<tr><td>%s</td><td>%.2f%%</td><td>%.2f%%</td></tr>\n",
			trend.Date.Format("2006-01-02"),
			trend.SuccessRate,
			trend.ComplianceRate,
		))
	}

	sb.WriteString("</table>\n")
	sb.WriteString("</body></html>")

	return sb.String()
}

// formatComplianceStandardMarkdown formats compliance standard report as markdown.
func (cr *ComplianceReporter) formatComplianceStandardMarkdown(data map[string]interface{}) string {
	var sb strings.Builder

	sb.WriteString("# Compliance Standard Report\n\n")
	sb.WriteString(fmt.Sprintf("**Generated:** %s\n\n", time.Now().Format(time.RFC3339)))

	standards, _ := data["standards"].(map[string][]string)
	complianceStatus, _ := data["compliance_status"].(map[string]bool)

	sb.WriteString("## Compliance Standards Coverage\n\n")

	for standard, databases := range standards {
		status := "✅ COMPLIANT"
		if !complianceStatus[standard] {
			status = "❌ NON-COMPLIANT"
		}

		sb.WriteString(fmt.Sprintf("### %s - %s\n\n", standard, status))
		sb.WriteString(fmt.Sprintf("**Databases:** %d\n\n", len(databases)))

		for _, db := range databases {
			sb.WriteString(fmt.Sprintf("- %s\n", db))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// GetReportHistory returns the history of generated reports.
func (cr *ComplianceReporter) GetReportHistory() []*ComplianceReport {
	return cr.reportHistory
}

// GetReport returns a specific report by ID.
func (cr *ComplianceReporter) GetReport(reportID string) (*ComplianceReport, bool) {
	for _, report := range cr.reportHistory {
		if report.ID == reportID {
			return report, true
		}
	}
	return nil, false
}
