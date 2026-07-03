package sla

import (
	"sync"
	"testing"
	"time"
)

func TestNewSLAMonitor(t *testing.T) {
	monitor := NewSLAMonitor()

	if monitor == nil {
		t.Fatal("Expected monitor to be created, got nil")
	}

	if monitor.definitions == nil {
		t.Error("Expected definitions map to be initialized")
	}

	if monitor.metrics == nil {
		t.Error("Expected metrics map to be initialized")
	}

	if monitor.rtoMetrics == nil {
		t.Error("Expected rtoMetrics map to be initialized")
	}

	if monitor.rpoMetrics == nil {
		t.Error("Expected rpoMetrics map to be initialized")
	}

	if monitor.violations == nil {
		t.Error("Expected violations slice to be initialized")
	}
}

func TestAddSLADefinition(t *testing.T) {
	monitor := NewSLAMonitor()

	tests := []struct {
		name    string
		def     *SLADefinition
		wantErr bool
	}{
		{
			name: "Valid critical SLA",
			def: &SLADefinition{
				DatabaseID:   "db-001",
				DatabaseName: "Production DB",
				Level:        SLALevelCritical,
			},
			wantErr: false,
		},
		{
			name: "Valid high SLA with custom targets",
			def: &SLADefinition{
				DatabaseID:        "db-002",
				DatabaseName:      "Analytics DB",
				Level:             SLALevelHigh,
				SuccessRateTarget: 99.95,
				MaxBackupDuration: 2 * time.Hour,
				RTOTarget:         4 * time.Hour,
				RPOTarget:         30 * time.Minute,
			},
			wantErr: false,
		},
		{
			name:    "Nil definition",
			def:     nil,
			wantErr: true,
		},
		{
			name: "Missing database ID",
			def: &SLADefinition{
				DatabaseName: "Test DB",
				Level:        SLALevelMedium,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := monitor.AddSLADefinition(tt.def)

			if (err != nil) != tt.wantErr {
				t.Errorf("AddSLADefinition() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.def != nil {
				// Verify definition was added
				def, exists := monitor.definitions[tt.def.DatabaseID]
				if !exists {
					t.Error("Expected definition to be added to monitor")
				}

				// Verify default success rate was set
				if def.SuccessRateTarget == 0 {
					t.Error("Expected default success rate to be set")
				}

				// Verify metrics were initialized
				if _, exists := monitor.metrics[tt.def.DatabaseID]; !exists {
					t.Error("Expected metrics to be initialized")
				}
			}
		})
	}
}

func TestDefaultSuccessRates(t *testing.T) {
	monitor := NewSLAMonitor()

	tests := []struct {
		level        SLALevel
		expectedRate float64
	}{
		{SLALevelCritical, 99.99},
		{SLALevelHigh, 99.9},
		{SLALevelMedium, 99.5},
		{SLALevelLow, 99.0},
	}

	for _, tt := range tests {
		t.Run(string(tt.level), func(t *testing.T) {
			def := &SLADefinition{
				DatabaseID:   "test-db",
				DatabaseName: "Test",
				Level:        tt.level,
			}

			err := monitor.AddSLADefinition(def)
			if err != nil {
				t.Fatalf("Failed to add SLA definition: %v", err)
			}

			storedDef := monitor.definitions["test-db"]
			if storedDef.SuccessRateTarget != tt.expectedRate {
				t.Errorf("Expected success rate %.2f, got %.2f", tt.expectedRate, storedDef.SuccessRateTarget)
			}

			// Clean up for next iteration
			delete(monitor.definitions, "test-db")
		})
	}
}

func TestRecordBackup(t *testing.T) {
	monitor := NewSLAMonitor()

	// Add SLA definition
	def := &SLADefinition{
		DatabaseID:        "db-001",
		DatabaseName:      "Test DB",
		Level:             SLALevelHigh,
		SuccessRateTarget: 99.9,
		MaxBackupDuration: 1 * time.Hour,
	}

	err := monitor.AddSLADefinition(def)
	if err != nil {
		t.Fatalf("Failed to add SLA definition: %v", err)
	}

	// Record successful backups
	for i := 0; i < 10; i++ {
		err = monitor.RecordBackup("db-001", true, 30*time.Minute, 1024*1024*100)
		if err != nil {
			t.Errorf("Failed to record backup: %v", err)
		}
	}

	// Record a failed backup
	err = monitor.RecordBackup("db-001", false, 0, 0)
	if err != nil {
		t.Errorf("Failed to record backup: %v", err)
	}

	// Verify metrics
	metrics, exists := monitor.GetMetrics("db-001")
	if !exists {
		t.Fatal("Expected metrics to exist")
	}

	if metrics.TotalBackups != 11 {
		t.Errorf("Expected 11 total backups, got %d", metrics.TotalBackups)
	}

	if metrics.SuccessfulBackups != 10 {
		t.Errorf("Expected 10 successful backups, got %d", metrics.SuccessfulBackups)
	}

	if metrics.FailedBackups != 1 {
		t.Errorf("Expected 1 failed backup, got %d", metrics.FailedBackups)
	}

	expectedSuccessRate := float64(10) / float64(11) * 100
	if metrics.SuccessRate != expectedSuccessRate {
		t.Errorf("Expected success rate %.2f, got %.2f", expectedSuccessRate, metrics.SuccessRate)
	}

	if metrics.ConsecutiveFailures != 1 {
		t.Errorf("Expected 1 consecutive failure, got %d", metrics.ConsecutiveFailures)
	}

	expectedDataSize := int64(1024 * 1024 * 100 * 10)
	if metrics.TotalDataBackedUp != expectedDataSize {
		t.Errorf("Expected total data backed up %d, got %d", expectedDataSize, metrics.TotalDataBackedUp)
	}
}

func TestRecordBackup_DurationMetrics(t *testing.T) {
	monitor := NewSLAMonitor()

	def := &SLADefinition{
		DatabaseID:   "db-001",
		DatabaseName: "Test DB",
		Level:        SLALevelHigh,
	}

	monitor.AddSLADefinition(def)

	// Record backups with different durations
	durations := []time.Duration{
		30 * time.Minute,
		45 * time.Minute,
		20 * time.Minute,
		60 * time.Minute,
		40 * time.Minute,
	}

	for _, duration := range durations {
		monitor.RecordBackup("db-001", true, duration, 0)
	}

	metrics, _ := monitor.GetMetrics("db-001")

	if metrics.MinBackupDuration != 20*time.Minute {
		t.Errorf("Expected min duration 20m, got %v", metrics.MinBackupDuration)
	}

	if metrics.MaxBackupDuration != 60*time.Minute {
		t.Errorf("Expected max duration 60m, got %v", metrics.MaxBackupDuration)
	}

	// Average should be (30+45+20+60+40)/5 = 39 minutes
	expectedAvg := 39 * time.Minute
	if metrics.AverageBackupDuration != expectedAvg {
		t.Errorf("Expected average duration %v, got %v", expectedAvg, metrics.AverageBackupDuration)
	}
}

func TestRecordRestoreTest(t *testing.T) {
	monitor := NewSLAMonitor()

	def := &SLADefinition{
		DatabaseID:   "db-001",
		DatabaseName: "Test DB",
		Level:        SLALevelCritical,
		RTOTarget:    2 * time.Hour,
	}

	monitor.AddSLADefinition(def)

	// Record successful restore within RTO
	err := monitor.RecordRestoreTest("db-001", true, 1*time.Hour)
	if err != nil {
		t.Fatalf("Failed to record restore test: %v", err)
	}

	rto, exists := monitor.GetRTOMetrics("db-001")
	if !exists {
		t.Fatal("Expected RTO metrics to exist")
	}

	if rto.TotalRestoreTests != 1 {
		t.Errorf("Expected 1 restore test, got %d", rto.TotalRestoreTests)
	}

	if rto.PassedRestoreTests != 1 {
		t.Errorf("Expected 1 passed test, got %d", rto.PassedRestoreTests)
	}

	if !rto.IsCompliant {
		t.Error("Expected restore to be compliant")
	}

	// Record restore that exceeds RTO
	monitor.RecordRestoreTest("db-001", true, 3*time.Hour)

	rto, _ = monitor.GetRTOMetrics("db-001")

	if rto.IsCompliant {
		t.Error("Expected restore to be non-compliant")
	}

	if rto.TotalRestoreTests != 2 {
		t.Errorf("Expected 2 restore tests, got %d", rto.TotalRestoreTests)
	}

	// Verify violation was recorded
	violations := monitor.GetViolations("db-001", true)
	foundRTOViolation := false
	for _, v := range violations {
		if v.ViolationType == "rto" {
			foundRTOViolation = true
			break
		}
	}

	if !foundRTOViolation {
		t.Error("Expected RTO violation to be recorded")
	}
}

func TestUpdateRPOMetrics(t *testing.T) {
	monitor := NewSLAMonitor()

	def := &SLADefinition{
		DatabaseID:      "db-001",
		DatabaseName:    "Test DB",
		Level:           SLALevelHigh,
		RPOTarget:       1 * time.Hour,
		BackupFrequency: 30 * time.Minute,
	}

	monitor.AddSLADefinition(def)

	// Record a backup
	monitor.RecordBackup("db-001", true, 30*time.Minute, 0)

	// Update RPO metrics immediately (should be compliant)
	err := monitor.UpdateRPOMetrics("db-001")
	if err != nil {
		t.Fatalf("Failed to update RPO metrics: %v", err)
	}

	rpo, exists := monitor.GetRPOMetrics("db-001")
	if !exists {
		t.Fatal("Expected RPO metrics to exist")
	}

	if !rpo.IsCompliant {
		t.Error("Expected RPO to be compliant immediately after backup")
	}

	// Simulate time passing (manually set last backup time)
	monitor.mu.Lock()
	metrics := monitor.metrics["db-001"]
	metrics.LastSuccessfulBackup = time.Now().Add(-2 * time.Hour)
	monitor.mu.Unlock()

	// Update RPO metrics (should now be non-compliant)
	monitor.UpdateRPOMetrics("db-001")

	rpo, _ = monitor.GetRPOMetrics("db-001")

	if rpo.IsCompliant {
		t.Error("Expected RPO to be non-compliant after time elapsed")
	}

	if rpo.DataLossRisk == 0 {
		t.Error("Expected data loss risk to be > 0")
	}
}

func TestViolationDetection(t *testing.T) {
	monitor := NewSLAMonitor()

	def := &SLADefinition{
		DatabaseID:        "db-001",
		DatabaseName:      "Test DB",
		Level:             SLALevelCritical,
		SuccessRateTarget: 99.99,
		MaxBackupDuration: 30 * time.Minute,
		BackupFrequency:   1 * time.Hour,
		AlertOnViolation:  true,
	}

	monitor.AddSLADefinition(def)

	// Record enough failures to trigger success rate violation
	for i := 0; i < 10; i++ {
		monitor.RecordBackup("db-001", false, 0, 0)
	}

	violations := monitor.GetViolations("db-001", true)
	if len(violations) == 0 {
		t.Error("Expected violations to be recorded")
	}

	foundSuccessRateViolation := false
	for _, v := range violations {
		if v.ViolationType == "success_rate" {
			foundSuccessRateViolation = true
			if v.Severity != "critical" {
				t.Errorf("Expected severity 'critical', got '%s'", v.Severity)
			}
			if v.DatabaseID != "db-001" {
				t.Errorf("Expected database ID 'db-001', got '%s'", v.DatabaseID)
			}
			if v.IsResolved {
				t.Error("Expected violation to be unresolved")
			}
		}
	}

	if !foundSuccessRateViolation {
		t.Error("Expected success rate violation to be detected")
	}

	// Test duration violation with a separate database
	def2 := &SLADefinition{
		DatabaseID:        "db-002",
		DatabaseName:      "Duration Test DB",
		Level:             SLALevelMedium,
		SuccessRateTarget: 90.0,
		MaxBackupDuration: 30 * time.Minute,
	}
	monitor.AddSLADefinition(def2)

	// Record backups with excessive duration
	for i := 0; i < 10; i++ {
		monitor.RecordBackup("db-002", true, 60*time.Minute, 0)
	}

	violations = monitor.GetViolations("db-002", true)
	foundDurationViolation := false
	for _, v := range violations {
		if v.ViolationType == "duration" {
			foundDurationViolation = true
			break
		}
	}

	if !foundDurationViolation {
		t.Error("Expected duration violation to be detected")
	}
}

func TestResolveViolation(t *testing.T) {
	monitor := NewSLAMonitor()

	def := &SLADefinition{
		DatabaseID:        "db-001",
		DatabaseName:      "Test DB",
		Level:             SLALevelHigh,
		SuccessRateTarget: 99.0,
	}

	monitor.AddSLADefinition(def)

	// Trigger violation
	for i := 0; i < 10; i++ {
		monitor.RecordBackup("db-001", false, 0, 0)
	}

	violations := monitor.GetViolations("db-001", true)
	if len(violations) == 0 {
		t.Fatal("Expected violation to be created")
	}

	violationID := violations[0].ID

	// Resolve violation
	err := monitor.ResolveViolation(violationID)
	if err != nil {
		t.Fatalf("Failed to resolve violation: %v", err)
	}

	// Verify violation is resolved
	allViolations := monitor.GetViolations("db-001", false)
	for _, v := range allViolations {
		if v.ID == violationID {
			if !v.IsResolved {
				t.Error("Expected violation to be marked as resolved")
			}
			if v.ResolvedAt == nil {
				t.Error("Expected ResolvedAt to be set")
			}
		}
	}

	// Verify unresolved violations no longer include this one
	unresolvedViolations := monitor.GetViolations("db-001", true)
	for _, v := range unresolvedViolations {
		if v.ID == violationID {
			t.Error("Resolved violation should not appear in unresolved list")
		}
	}

	// Test resolving non-existent violation
	err = monitor.ResolveViolation("non-existent-id")
	if err == nil {
		t.Error("Expected error when resolving non-existent violation")
	}
}

func TestAlertHandler(t *testing.T) {
	monitor := NewSLAMonitor()

	var mu sync.Mutex
	alertCount := 0
	var lastViolation *SLAViolation

	handler := func(v *SLAViolation) error {
		mu.Lock()
		alertCount++
		lastViolation = v
		mu.Unlock()
		return nil
	}

	monitor.RegisterAlertHandler(handler)

	def := &SLADefinition{
		DatabaseID:        "db-001",
		DatabaseName:      "Test DB",
		Level:             SLALevelCritical,
		SuccessRateTarget: 99.0,
		AlertOnViolation:  true,
	}

	monitor.AddSLADefinition(def)

	// Trigger violation
	for i := 0; i < 10; i++ {
		monitor.RecordBackup("db-001", false, 0, 0)
	}

	// Give alert handler time to execute (it runs in goroutine)
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	count := alertCount
	last := lastViolation
	mu.Unlock()

	if count == 0 {
		t.Error("Expected alert handler to be called")
	}

	if last == nil {
		t.Error("Expected violation to be passed to handler")
	}

	if last != nil && last.DatabaseID != "db-001" {
		t.Errorf("Expected violation for db-001, got %s", last.DatabaseID)
	}
}

func TestGetComplianceSummary(t *testing.T) {
	monitor := NewSLAMonitor()

	// Add multiple databases with different compliance states
	defs := []*SLADefinition{
		{
			DatabaseID:        "db-001",
			DatabaseName:      "Compliant DB 1",
			Level:             SLALevelCritical,
			SuccessRateTarget: 99.0,
		},
		{
			DatabaseID:        "db-002",
			DatabaseName:      "Compliant DB 2",
			Level:             SLALevelHigh,
			SuccessRateTarget: 99.0,
		},
		{
			DatabaseID:        "db-003",
			DatabaseName:      "Non-Compliant DB",
			Level:             SLALevelMedium,
			SuccessRateTarget: 99.0,
		},
	}

	for _, def := range defs {
		monitor.AddSLADefinition(def)
	}

	// Record backups for compliant databases
	for i := 0; i < 100; i++ {
		monitor.RecordBackup("db-001", true, 30*time.Minute, 0)
		monitor.RecordBackup("db-002", true, 30*time.Minute, 0)
	}

	// Record backups for non-compliant database (low success rate)
	for i := 0; i < 50; i++ {
		monitor.RecordBackup("db-003", false, 30*time.Minute, 0)
	}

	summary := monitor.GetComplianceSummary()

	totalDatabases, _ := summary["total_databases"].(int)
	if totalDatabases != 3 {
		t.Errorf("Expected 3 total databases, got %d", totalDatabases)
	}

	compliantDatabases, _ := summary["compliant_databases"].(int)
	if compliantDatabases != 2 {
		t.Errorf("Expected 2 compliant databases, got %d", compliantDatabases)
	}

	avgSuccessRate, _ := summary["average_success_rate"].(float64)
	if avgSuccessRate < 60 || avgSuccessRate > 70 {
		t.Errorf("Expected average success rate around 66%%, got %.2f%%", avgSuccessRate)
	}

	unresolvedViolations, _ := summary["unresolved_violations"].(int)
	if unresolvedViolations == 0 {
		t.Error("Expected some unresolved violations")
	}
}

func TestComplianceReporter_GenerateExecutiveReport(t *testing.T) {
	monitor := NewSLAMonitor()
	reporter := NewComplianceReporter(monitor)

	// Add SLA definitions
	def := &SLADefinition{
		DatabaseID:          "db-001",
		DatabaseName:        "Production DB",
		Level:               SLALevelCritical,
		SuccessRateTarget:   99.9,
		ComplianceStandards: []string{"SOC2", "HIPAA"},
	}

	monitor.AddSLADefinition(def)

	// Record some backups
	for i := 0; i < 100; i++ {
		monitor.RecordBackup("db-001", true, 30*time.Minute, 1024*1024*100)
	}

	// Generate executive report
	periodStart := time.Now().Add(-24 * time.Hour)
	periodEnd := time.Now()

	tests := []struct {
		name   string
		format ReportFormat
	}{
		{"JSON format", ReportFormatJSON},
		{"Markdown format", ReportFormatMarkdown},
		{"HTML format", ReportFormatHTML},
		{"Text format", ReportFormatText},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report, err := reporter.GenerateReport(ReportTypeExecutive, tt.format, periodStart, periodEnd)
			if err != nil {
				t.Fatalf("Failed to generate executive report: %v", err)
			}

			if report == nil {
				t.Fatal("Expected report to be generated")
			}

			if report.Type != ReportTypeExecutive {
				t.Errorf("Expected report type %s, got %s", ReportTypeExecutive, report.Type)
			}

			if report.Format != tt.format {
				t.Errorf("Expected format %s, got %s", tt.format, report.Format)
			}

			if report.Content == "" {
				t.Error("Expected report content to be non-empty")
			}

			if report.Title == "" {
				t.Error("Expected report title to be set")
			}
		})
	}
}

func TestComplianceReporter_GenerateDetailedReport(t *testing.T) {
	monitor := NewSLAMonitor()
	reporter := NewComplianceReporter(monitor)

	// Add multiple databases
	databases := []struct {
		id      string
		name    string
		level   SLALevel
		success bool
	}{
		{"db-001", "DB 1", SLALevelCritical, true},
		{"db-002", "DB 2", SLALevelHigh, true},
		{"db-003", "DB 3", SLALevelMedium, false},
	}

	for _, db := range databases {
		def := &SLADefinition{
			DatabaseID:        db.id,
			DatabaseName:      db.name,
			Level:             db.level,
			SuccessRateTarget: 99.0,
		}
		monitor.AddSLADefinition(def)

		if db.success {
			for i := 0; i < 100; i++ {
				monitor.RecordBackup(db.id, true, 30*time.Minute, 0)
			}
		} else {
			for i := 0; i < 50; i++ {
				monitor.RecordBackup(db.id, false, 0, 0)
			}
		}
	}

	periodStart := time.Now().Add(-24 * time.Hour)
	periodEnd := time.Now()

	report, err := reporter.GenerateReport(ReportTypeDetailed, ReportFormatMarkdown, periodStart, periodEnd)
	if err != nil {
		t.Fatalf("Failed to generate detailed report: %v", err)
	}

	if report.Content == "" {
		t.Error("Expected report content to be non-empty")
	}

	// Verify report contains database names
	for _, db := range databases {
		if !containsString(report.Content, db.name) {
			t.Errorf("Expected report to contain database name '%s'", db.name)
		}
	}
}

func TestComplianceReporter_GenerateViolationReport(t *testing.T) {
	monitor := NewSLAMonitor()
	reporter := NewComplianceReporter(monitor)

	def := &SLADefinition{
		DatabaseID:        "db-001",
		DatabaseName:      "Test DB",
		Level:             SLALevelCritical,
		SuccessRateTarget: 99.9,
	}

	monitor.AddSLADefinition(def)

	// Trigger violations
	for i := 0; i < 10; i++ {
		monitor.RecordBackup("db-001", false, 0, 0)
	}

	periodStart := time.Now().Add(-24 * time.Hour)
	periodEnd := time.Now()

	report, err := reporter.GenerateReport(ReportTypeViolation, ReportFormatMarkdown, periodStart, periodEnd)
	if err != nil {
		t.Fatalf("Failed to generate violation report: %v", err)
	}

	if report.Content == "" {
		t.Error("Expected report content to be non-empty")
	}

	if !containsString(report.Content, "success_rate") {
		t.Error("Expected report to mention success_rate violation")
	}
}

func TestComplianceReporter_GenerateTrendReport(t *testing.T) {
	monitor := NewSLAMonitor()
	reporter := NewComplianceReporter(monitor)

	def := &SLADefinition{
		DatabaseID:        "db-001",
		DatabaseName:      "Test DB",
		Level:             SLALevelHigh,
		SuccessRateTarget: 99.0,
	}

	monitor.AddSLADefinition(def)

	for i := 0; i < 50; i++ {
		monitor.RecordBackup("db-001", true, 30*time.Minute, 0)
	}

	periodStart := time.Now().Add(-7 * 24 * time.Hour)
	periodEnd := time.Now()

	report, err := reporter.GenerateReport(ReportTypeTrend, ReportFormatMarkdown, periodStart, periodEnd)
	if err != nil {
		t.Fatalf("Failed to generate trend report: %v", err)
	}

	if report.Content == "" {
		t.Error("Expected report content to be non-empty")
	}
}

func TestComplianceReporter_GenerateComplianceStandardReport(t *testing.T) {
	monitor := NewSLAMonitor()
	reporter := NewComplianceReporter(monitor)

	databases := []struct {
		id        string
		name      string
		standards []string
	}{
		{"db-001", "DB 1", []string{"SOC2", "HIPAA"}},
		{"db-002", "DB 2", []string{"GDPR", "SOC2"}},
		{"db-003", "DB 3", []string{"PCI-DSS"}},
	}

	for _, db := range databases {
		def := &SLADefinition{
			DatabaseID:          db.id,
			DatabaseName:        db.name,
			Level:               SLALevelHigh,
			SuccessRateTarget:   99.0,
			ComplianceStandards: db.standards,
		}
		monitor.AddSLADefinition(def)

		for i := 0; i < 100; i++ {
			monitor.RecordBackup(db.id, true, 30*time.Minute, 0)
		}
	}

	periodStart := time.Now().Add(-24 * time.Hour)
	periodEnd := time.Now()

	report, err := reporter.GenerateReport(ReportTypeCompliance, ReportFormatMarkdown, periodStart, periodEnd)
	if err != nil {
		t.Fatalf("Failed to generate compliance standard report: %v", err)
	}

	if report.Content == "" {
		t.Error("Expected report content to be non-empty")
	}

	// Verify report contains compliance standards
	standards := []string{"SOC2", "HIPAA", "GDPR", "PCI-DSS"}
	for _, std := range standards {
		if !containsString(report.Content, std) {
			t.Errorf("Expected report to contain standard '%s'", std)
		}
	}
}

func TestComplianceReporter_GetReportHistory(t *testing.T) {
	monitor := NewSLAMonitor()
	reporter := NewComplianceReporter(monitor)

	def := &SLADefinition{
		DatabaseID:   "db-001",
		DatabaseName: "Test DB",
		Level:        SLALevelHigh,
	}

	monitor.AddSLADefinition(def)

	periodStart := time.Now().Add(-24 * time.Hour)
	periodEnd := time.Now()

	// Generate multiple reports
	reporter.GenerateReport(ReportTypeExecutive, ReportFormatJSON, periodStart, periodEnd)
	reporter.GenerateReport(ReportTypeDetailed, ReportFormatMarkdown, periodStart, periodEnd)
	reporter.GenerateReport(ReportTypeViolation, ReportFormatText, periodStart, periodEnd)

	history := reporter.GetReportHistory()

	if len(history) != 3 {
		t.Errorf("Expected 3 reports in history, got %d", len(history))
	}
}

func TestComplianceReporter_GetReport(t *testing.T) {
	monitor := NewSLAMonitor()
	reporter := NewComplianceReporter(monitor)

	def := &SLADefinition{
		DatabaseID:   "db-001",
		DatabaseName: "Test DB",
		Level:        SLALevelHigh,
	}

	monitor.AddSLADefinition(def)

	periodStart := time.Now().Add(-24 * time.Hour)
	periodEnd := time.Now()

	generatedReport, _ := reporter.GenerateReport(ReportTypeExecutive, ReportFormatJSON, periodStart, periodEnd)

	// Retrieve report by ID
	retrievedReport, exists := reporter.GetReport(generatedReport.ID)

	if !exists {
		t.Fatal("Expected report to be found")
	}

	if retrievedReport.ID != generatedReport.ID {
		t.Errorf("Expected report ID %s, got %s", generatedReport.ID, retrievedReport.ID)
	}

	// Try to get non-existent report
	_, exists = reporter.GetReport("non-existent-id")

	if exists {
		t.Error("Expected report not to be found")
	}
}

func TestExecutiveSummary_Recommendations(t *testing.T) {
	monitor := NewSLAMonitor()
	reporter := NewComplianceReporter(monitor)

	// Create database with multiple issues
	def := &SLADefinition{
		DatabaseID:        "db-001",
		DatabaseName:      "Problematic DB",
		Level:             SLALevelCritical,
		SuccessRateTarget: 99.9,
		RTOTarget:         2 * time.Hour,
		RPOTarget:         1 * time.Hour,
		MaxBackupDuration: 30 * time.Minute,
		BackupFrequency:   1 * time.Hour,
	}

	monitor.AddSLADefinition(def)

	// Record failures to trigger success rate violation
	for i := 0; i < 10; i++ {
		monitor.RecordBackup("db-001", false, 0, 0)
	}

	// Record slow backups to trigger duration violation
	for i := 0; i < 5; i++ {
		monitor.RecordBackup("db-001", true, 60*time.Minute, 0)
	}

	// Trigger RTO violation
	monitor.RecordRestoreTest("db-001", true, 3*time.Hour)

	// Trigger RPO violation
	monitor.mu.Lock()
	metrics := monitor.metrics["db-001"]
	metrics.LastSuccessfulBackup = time.Now().Add(-3 * time.Hour)
	monitor.mu.Unlock()
	monitor.UpdateRPOMetrics("db-001")

	summary := reporter.getExecutiveSummary()

	if len(summary.TopIssues) == 0 {
		t.Error("Expected top issues to be identified")
	}

	if len(summary.Recommendations) == 0 {
		t.Error("Expected recommendations to be generated")
	}

	t.Logf("Top Issues: %v", summary.TopIssues)
	t.Logf("Recommendations: %v", summary.Recommendations)
}

func TestDatabaseComplianceDetail_Sorting(t *testing.T) {
	monitor := NewSLAMonitor()
	reporter := NewComplianceReporter(monitor)

	// Add databases with different compliance states
	databases := []struct {
		id         string
		name       string
		backupRate float64
	}{
		{"db-001", "Compliant A", 1.0},
		{"db-002", "Non-Compliant Z", 0.5},
		{"db-003", "Compliant B", 1.0},
		{"db-004", "Non-Compliant A", 0.3},
	}

	for _, db := range databases {
		def := &SLADefinition{
			DatabaseID:        db.id,
			DatabaseName:      db.name,
			Level:             SLALevelHigh,
			SuccessRateTarget: 99.0,
		}
		monitor.AddSLADefinition(def)

		successCount := int(db.backupRate * 100)
		for i := 0; i < successCount; i++ {
			monitor.RecordBackup(db.id, true, 30*time.Minute, 0)
		}
		for i := 0; i < 100-successCount; i++ {
			monitor.RecordBackup(db.id, false, 0, 0)
		}
	}

	details := reporter.getDatabaseComplianceDetails()

	// Verify sorting: non-compliant first, then by name
	if len(details) < 4 {
		t.Fatal("Expected 4 database details")
	}

	// First two should be non-compliant
	if details[0].IsCompliant || details[1].IsCompliant {
		t.Error("Expected non-compliant databases first")
	}

	// Last two should be compliant
	if !details[2].IsCompliant || !details[3].IsCompliant {
		t.Error("Expected compliant databases last")
	}

	// Within non-compliant group, should be sorted by name
	if details[0].DatabaseName != "Non-Compliant A" {
		t.Errorf("Expected 'Non-Compliant A' first, got '%s'", details[0].DatabaseName)
	}
}

// Helper function to check if a string contains a substring.
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && stringContains(s, substr))
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
