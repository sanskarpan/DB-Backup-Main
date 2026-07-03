package dr

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestResults(count int, databaseName string, successRate float64) []TestResult {
	results := make([]TestResult, count)
	now := time.Now()

	for i := 0; i < count; i++ {
		success := float64(i) < float64(count)*successRate/100.0

		results[i] = TestResult{
			TestID:            fmt.Sprintf("test-%d", i),
			DatabaseName:      databaseName,
			StartTime:         now.Add(-time.Duration(count-i) * 24 * time.Hour),
			EndTime:           now.Add(-time.Duration(count-i)*24*time.Hour + 10*time.Minute),
			Duration:          10 * time.Minute,
			Success:           success,
			RestoreStartTime:  now.Add(-time.Duration(count-i) * 24 * time.Hour),
			RestoreEndTime:    now.Add(-time.Duration(count-i)*24*time.Hour + 5*time.Minute),
			RestoreDuration:   5 * time.Minute,
			RestoreSize:       1000000000,
			SchemaValid:       success,
			SchemaErrors:      []string{},
			RowCountValid:     success,
			RowCountErrors:    []string{},
			SampleDataValid:   success,
			SampleDataErrors:  []string{},
			RTO:               5 * time.Minute,
			RPO:               1 * time.Minute,
			RTOThreshold:      10 * time.Minute,
			RPOThreshold:      5 * time.Minute,
			RTOMet:            true,
			RPOMet:            true,
			SmokeTestsPassed:  3,
			SmokeTestsFailed:  0,
			TestEnvironmentID: fmt.Sprintf("env-%d", i),
			CleanedUp:         true,
		}
	}

	return results
}

func TestNewReportGenerator(t *testing.T) {
	rg := NewReportGenerator()

	assert.NotNil(t, rg)
	assert.NotNil(t, rg.history)
	assert.Empty(t, rg.history)
}

func TestReportGenerator_AddTestResult(t *testing.T) {
	rg := NewReportGenerator()

	result := TestResult{
		TestID:       "test-1",
		DatabaseName: "test-db",
		StartTime:    time.Now(),
		Success:      true,
	}

	rg.AddTestResult(result)

	assert.Len(t, rg.history, 1)
	assert.Equal(t, "test-1", rg.history[0].TestID)
}

func TestReportGenerator_AddTestResultMaxHistory(t *testing.T) {
	rg := NewReportGenerator()

	// Add 1100 results (should keep only last 1000)
	for i := 0; i < 1100; i++ {
		result := TestResult{
			TestID:       fmt.Sprintf("test-%d", i),
			DatabaseName: "test-db",
			StartTime:    time.Now(),
			Success:      true,
		}
		rg.AddTestResult(result)
	}

	assert.Len(t, rg.history, 1000)
	// Should have kept the most recent 1000
	assert.Equal(t, "test-100", rg.history[0].TestID)
	assert.Equal(t, "test-1099", rg.history[999].TestID)
}

func TestReportGenerator_GetTestHistory(t *testing.T) {
	rg := NewReportGenerator()

	now := time.Now()
	for i := 0; i < 10; i++ {
		result := TestResult{
			TestID:       fmt.Sprintf("test-%d", i),
			DatabaseName: "test-db",
			StartTime:    now.Add(-time.Duration(i) * 24 * time.Hour),
			Success:      true,
		}
		rg.AddTestResult(result)
	}

	// Get last 5 days
	startDate := now.Add(-5 * 24 * time.Hour)
	endDate := now.Add(1 * time.Hour)

	results := rg.GetTestHistory(startDate, endDate)

	// Should get tests from last 5 days (0-4)
	assert.Len(t, results, 5)
}

func TestReportGenerator_GetTestHistoryByDatabase(t *testing.T) {
	rg := NewReportGenerator()

	now := time.Now()
	for i := 0; i < 5; i++ {
		rg.AddTestResult(TestResult{
			TestID:       fmt.Sprintf("test-db1-%d", i),
			DatabaseName: "db1",
			StartTime:    now.Add(-time.Duration(i) * time.Hour),
			Success:      true,
		})
		rg.AddTestResult(TestResult{
			TestID:       fmt.Sprintf("test-db2-%d", i),
			DatabaseName: "db2",
			StartTime:    now.Add(-time.Duration(i) * time.Hour),
			Success:      true,
		})
	}

	results := rg.GetTestHistoryByDatabase("db1", 10)

	assert.Len(t, results, 5)
	for _, r := range results {
		assert.Equal(t, "db1", r.DatabaseName)
	}
}

func TestReportGenerator_GetTestHistoryByDatabaseLimit(t *testing.T) {
	rg := NewReportGenerator()

	now := time.Now()
	for i := 0; i < 10; i++ {
		rg.AddTestResult(TestResult{
			TestID:       fmt.Sprintf("test-%d", i),
			DatabaseName: "test-db",
			StartTime:    now.Add(-time.Duration(i) * time.Hour),
			Success:      true,
		})
	}

	results := rg.GetTestHistoryByDatabase("test-db", 5)

	assert.Len(t, results, 5)
}

func TestReportGenerator_GenerateComplianceReport(t *testing.T) {
	rg := NewReportGenerator()

	results := createTestResults(10, "test-db", 100.0)
	for _, r := range results {
		rg.AddTestResult(r)
	}

	ctx := context.Background()
	startDate := time.Now().Add(-30 * 24 * time.Hour)
	endDate := time.Now().Add(1 * time.Hour)

	report, err := rg.GenerateComplianceReport(ctx, "test-db", startDate, endDate)

	assert.NoError(t, err)
	assert.NotNil(t, report)
	assert.NotEmpty(t, report.ReportID)
	assert.Equal(t, "test-db", report.DatabaseName)
	assert.Equal(t, 10, report.TotalTests)
	assert.Equal(t, 10, report.SuccessfulTests)
	assert.Equal(t, 0, report.FailedTests)
	assert.Equal(t, 100.0, report.SuccessRate)
}

func TestReportGenerator_GenerateComplianceReportNoResults(t *testing.T) {
	rg := NewReportGenerator()

	ctx := context.Background()
	startDate := time.Now().Add(-30 * 24 * time.Hour)
	endDate := time.Now().Add(1 * time.Hour)

	_, err := rg.GenerateComplianceReport(ctx, "test-db", startDate, endDate)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no test results found")
}

func TestComplianceReport_CalculateMetrics(t *testing.T) {
	results := createTestResults(10, "test-db", 80.0) // 80% success rate

	report := &ComplianceReport{
		TestResults: results,
	}

	report.calculateMetrics()

	assert.Equal(t, 10, report.TotalTests)
	assert.Equal(t, 8, report.SuccessfulTests)
	assert.Equal(t, 2, report.FailedTests)
	assert.Equal(t, 80.0, report.SuccessRate)
	assert.Greater(t, report.AverageRTO, time.Duration(0))
	assert.Greater(t, report.AverageRPO, time.Duration(0))
}

func TestComplianceReport_GenerateRecommendations(t *testing.T) {
	testCases := []struct {
		name            string
		successRate     float64
		rtoCompliance   float64
		rpoCompliance   float64
		expectedCount   int
		expectedContain string
	}{
		{
			name:            "Critical - Low success rate",
			successRate:     70.0,
			rtoCompliance:   95.0,
			rpoCompliance:   95.0,
			expectedCount:   1,
			expectedContain: "CRITICAL",
		},
		{
			name:            "Warning - Medium success rate",
			successRate:     85.0,
			rtoCompliance:   95.0,
			rpoCompliance:   95.0,
			expectedCount:   1,
			expectedContain: "WARNING",
		},
		{
			name:            "Excellent performance",
			successRate:     100.0,
			rtoCompliance:   100.0,
			rpoCompliance:   100.0,
			expectedCount:   1,
			expectedContain: "Excellent",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			report := &ComplianceReport{
				SuccessRate:   tc.successRate,
				RTOCompliance: tc.rtoCompliance,
				RPOCompliance: tc.rpoCompliance,
				TotalTests:    10,
			}

			report.generateRecommendations()

			assert.GreaterOrEqual(t, len(report.Recommendations), tc.expectedCount)
			found := false
			for _, rec := range report.Recommendations {
				if strings.Contains(rec, tc.expectedContain) {
					found = true
					break
				}
			}
			assert.True(t, found, "Expected to find recommendation containing: %s", tc.expectedContain)
		})
	}
}

func TestComplianceReport_DetermineComplianceStatus(t *testing.T) {
	testCases := []struct {
		name           string
		successRate    float64
		rtoCompliance  float64
		rpoCompliance  float64
		expectedStatus ComplianceStatus
	}{
		{
			name:           "Passing",
			successRate:    100.0,
			rtoCompliance:  100.0,
			rpoCompliance:  100.0,
			expectedStatus: ComplianceStatusPassing,
		},
		{
			name:           "Warning - Low success rate",
			successRate:    85.0,
			rtoCompliance:  95.0,
			rpoCompliance:  95.0,
			expectedStatus: ComplianceStatusWarning,
		},
		{
			name:           "Failing - Very low success rate",
			successRate:    70.0,
			rtoCompliance:  95.0,
			rpoCompliance:  95.0,
			expectedStatus: ComplianceStatusFailing,
		},
		{
			name:           "Failing - Low RTO compliance",
			successRate:    95.0,
			rtoCompliance:  70.0,
			rpoCompliance:  95.0,
			expectedStatus: ComplianceStatusFailing,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			report := &ComplianceReport{
				SuccessRate:   tc.successRate,
				RTOCompliance: tc.rtoCompliance,
				RPOCompliance: tc.rpoCompliance,
			}

			report.determineComplianceStatus()

			assert.Equal(t, tc.expectedStatus, report.ComplianceStatus)
		})
	}
}

func TestComplianceReport_ExportToCSV(t *testing.T) {
	results := createTestResults(3, "test-db", 100.0)

	report := &ComplianceReport{
		ReportID:    "report-123",
		GeneratedAt: time.Now(),
		ReportPeriod: ReportPeriod{
			StartDate: time.Now().Add(-30 * 24 * time.Hour),
			EndDate:   time.Now(),
			Label:     "December 2025",
		},
		DatabaseName:     "test-db",
		TotalTests:       3,
		SuccessfulTests:  3,
		FailedTests:      0,
		SuccessRate:      100.0,
		AverageRTO:       5 * time.Minute,
		AverageRPO:       1 * time.Minute,
		RTOCompliance:    100.0,
		RPOCompliance:    100.0,
		TestResults:      results,
		Recommendations:  []string{"Excellent performance"},
		ComplianceStatus: ComplianceStatusPassing,
	}

	csv, err := report.ExportToCSV()

	assert.NoError(t, err)
	assert.NotNil(t, csv)

	csvString := string(csv)
	assert.Contains(t, csvString, "Disaster Recovery Compliance Report")
	assert.Contains(t, csvString, "report-123")
	assert.Contains(t, csvString, "test-db")
	assert.Contains(t, csvString, "100.00%")
	assert.Contains(t, csvString, "PASSING")
	assert.Contains(t, csvString, "Excellent performance")
}

func TestComplianceReport_ExportToPDF(t *testing.T) {
	results := createTestResults(2, "prod-db", 100.0)

	report := &ComplianceReport{
		ReportID:    "report-456",
		GeneratedAt: time.Now(),
		ReportPeriod: ReportPeriod{
			StartDate: time.Now().Add(-30 * 24 * time.Hour),
			EndDate:   time.Now(),
			Label:     "Q4 2025",
		},
		DatabaseName:     "prod-db",
		TotalTests:       2,
		SuccessfulTests:  2,
		FailedTests:      0,
		SuccessRate:      100.0,
		AverageRTO:       3 * time.Minute,
		AverageRPO:       30 * time.Second,
		RTOCompliance:    100.0,
		RPOCompliance:    100.0,
		TestResults:      results,
		Recommendations:  []string{"Continue current procedures"},
		ComplianceStatus: ComplianceStatusPassing,
	}

	pdf, err := report.ExportToPDF()

	assert.NoError(t, err)
	assert.NotNil(t, pdf)

	pdfString := string(pdf)
	assert.Contains(t, pdfString, "DISASTER RECOVERY COMPLIANCE REPORT")
	assert.Contains(t, pdfString, "report-456")
	assert.Contains(t, pdfString, "prod-db")
	assert.Contains(t, pdfString, "COMPLIANCE STATUS: PASSING")
	assert.Contains(t, pdfString, "100.00%")
}

func TestReportGenerator_GetComplianceSummary(t *testing.T) {
	rg := NewReportGenerator()

	results := createTestResults(10, "test-db", 90.0)
	for _, r := range results {
		rg.AddTestResult(r)
	}

	summary := rg.GetComplianceSummary(30)

	assert.Equal(t, 30, summary["period_days"])
	assert.Equal(t, 10, summary["total_tests"])
	assert.Equal(t, 90.0, summary["success_rate"])
	assert.Greater(t, summary["rto_compliance"], 0.0)
	assert.Greater(t, summary["rpo_compliance"], 0.0)

	databases, ok := summary["databases"].(map[string]int)
	require.True(t, ok)
	assert.Equal(t, 10, databases["test-db"])
}

func TestReportGenerator_GetComplianceSummaryNoData(t *testing.T) {
	rg := NewReportGenerator()

	summary := rg.GetComplianceSummary(30)

	assert.Equal(t, 30, summary["period_days"])
	assert.Equal(t, 0, summary["total_tests"])
	assert.Equal(t, 0.0, summary["success_rate"])
}

func TestGenerateReportID(t *testing.T) {
	id1 := generateReportID()
	time.Sleep(1 * time.Millisecond)
	id2 := generateReportID()

	assert.NotEmpty(t, id1)
	assert.NotEmpty(t, id2)
	assert.NotEqual(t, id1, id2)
	assert.Contains(t, id1, "report-")
	assert.Contains(t, id2, "report-")
}

func TestFormatPeriodLabel(t *testing.T) {
	testCases := []struct {
		name      string
		startDate time.Time
		endDate   time.Time
		expected  string
	}{
		{
			name:      "Same month",
			startDate: time.Date(2025, 12, 1, 0, 0, 0, 0, time.UTC),
			endDate:   time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC),
			expected:  "December 2025",
		},
		{
			name:      "Same year, different months",
			startDate: time.Date(2025, 10, 1, 0, 0, 0, 0, time.UTC),
			endDate:   time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC),
			expected:  "Oct - Dec 2025",
		},
		{
			name:      "Different years",
			startDate: time.Date(2024, 11, 1, 0, 0, 0, 0, time.UTC),
			endDate:   time.Date(2025, 1, 31, 0, 0, 0, 0, time.UTC),
			expected:  "Nov 2024 - Jan 2025",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			label := formatPeriodLabel(tc.startDate, tc.endDate)
			assert.Equal(t, tc.expected, label)
		})
	}
}

func TestBoolToString(t *testing.T) {
	assert.Equal(t, "true", boolToString(true))
	assert.Equal(t, "false", boolToString(false))
}

func TestBoolToPassFail(t *testing.T) {
	assert.Equal(t, "PASS", boolToPassFail(true))
	assert.Equal(t, "FAIL", boolToPassFail(false))
}

func TestReportPeriod_Fields(t *testing.T) {
	period := ReportPeriod{
		StartDate: time.Date(2025, 12, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC),
		Label:     "December 2025",
	}

	assert.Equal(t, "December 2025", period.Label)
	assert.Equal(t, 2025, period.StartDate.Year())
	assert.Equal(t, time.December, period.StartDate.Month())
	assert.Equal(t, 2025, period.EndDate.Year())
	assert.Equal(t, 31, period.EndDate.Day())
}

func TestComplianceStatus_Values(t *testing.T) {
	assert.Equal(t, ComplianceStatus("PASSING"), ComplianceStatusPassing)
	assert.Equal(t, ComplianceStatus("WARNING"), ComplianceStatusWarning)
	assert.Equal(t, ComplianceStatus("FAILING"), ComplianceStatusFailing)
}

func TestComplianceReport_WithErrors(t *testing.T) {
	results := []TestResult{
		{
			TestID:           "test-1",
			DatabaseName:     "test-db",
			StartTime:        time.Now(),
			Success:          false,
			SchemaValid:      false,
			SchemaErrors:     []string{"Missing table: users", "Missing index: idx_email"},
			RowCountValid:    false,
			RowCountErrors:   []string{"Row count mismatch: orders"},
			SampleDataValid:  false,
			SampleDataErrors: []string{"Data corruption detected"},
			RTO:              15 * time.Minute,
			RPO:              10 * time.Minute,
			RTOThreshold:     10 * time.Minute,
			RPOThreshold:     5 * time.Minute,
			RTOMet:           false,
			RPOMet:           false,
		},
	}

	report := &ComplianceReport{
		ReportID:         "report-error-test",
		GeneratedAt:      time.Now(),
		DatabaseName:     "test-db",
		TestResults:      results,
		ComplianceStatus: ComplianceStatusFailing,
	}

	report.calculateMetrics()

	assert.Equal(t, 1, report.TotalTests)
	assert.Equal(t, 0, report.SuccessfulTests)
	assert.Equal(t, 1, report.FailedTests)
	assert.Equal(t, 0.0, report.SuccessRate)
	assert.Equal(t, 0.0, report.RTOCompliance)
	assert.Equal(t, 0.0, report.RPOCompliance)

	// Test PDF export includes errors
	pdf, err := report.ExportToPDF()
	require.NoError(t, err)

	pdfString := string(pdf)
	assert.Contains(t, pdfString, "Missing table: users")
	assert.Contains(t, pdfString, "Row count mismatch: orders")
	assert.Contains(t, pdfString, "Data corruption detected")
}
