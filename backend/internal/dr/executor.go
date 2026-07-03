// Package dr provides disaster recovery testing and validation
package dr

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	"github.com/sanskarpan/db-backup/pkg/uid"
)

// RestoreStats holds metrics for a restore operation within a DR test.
type RestoreStats struct {
	Duration  time.Duration
	SizeBytes int64
}

// ValidationResult holds the outcome of a single validation step.
type ValidationResult struct {
	Name    string
	Success bool
	Error   string
}

// TestResult represents the result of a DR test.
type TestResult struct {
	TestID       string
	DatabaseName string
	StartTime    time.Time
	EndTime      time.Time
	Duration     time.Duration
	Success      bool
	Error        error

	// Restoration metrics
	RestoreStartTime time.Time
	RestoreEndTime   time.Time
	RestoreDuration  time.Duration
	RestoreSize      int64
	RestoreStats     *RestoreStats

	// Structured validation results
	Validations []*ValidationResult

	// Legacy validation fields
	SchemaValid      bool
	SchemaErrors     []string
	RowCountValid    bool
	RowCountErrors   []string
	SampleDataValid  bool
	SampleDataErrors []string

	// Performance metrics
	RTO          time.Duration // Actual recovery time
	RPO          time.Duration // Actual data loss window
	RTOThreshold time.Duration
	RPOThreshold time.Duration
	RTOMet       bool
	RPOMet       bool

	// Smoke test results
	SmokeTestsPassed int
	SmokeTestsFailed int
	SmokeTestErrors  []string

	// Environment info
	TestEnvironmentID string
	CleanedUp         bool
}

// TestExecutor executes DR tests.
type TestExecutor struct {
	provisioner     *EnvironmentProvisioner
	validator       *Validator
	notifier        *NotificationService
	reportGenerator *ReportGenerator
	autoRollback    bool
}

// NewTestExecutor creates a new test executor.
func NewTestExecutor(provisioner *EnvironmentProvisioner, validator *Validator) *TestExecutor {
	return &TestExecutor{
		provisioner:     provisioner,
		validator:       validator,
		reportGenerator: NewReportGenerator(),
		autoRollback:    true, // Enable auto-rollback by default
	}
}

// SetNotificationService sets the notification service.
func (te *TestExecutor) SetNotificationService(notifier *NotificationService) {
	te.notifier = notifier
}

// SetAutoRollback enables or disables automatic rollback on test failure.
func (te *TestExecutor) SetAutoRollback(enabled bool) {
	te.autoRollback = enabled
}

// GetReportGenerator returns the report generator.
func (te *TestExecutor) GetReportGenerator() *ReportGenerator {
	return te.reportGenerator
}

// ExecuteTest executes a complete DR test.
func (te *TestExecutor) ExecuteTest(ctx context.Context, databaseName string, config *TestConfig) (*TestResult, error) {
	result := &TestResult{
		TestID:       generateTestID(),
		DatabaseName: databaseName,
		StartTime:    time.Now(),
		RTOThreshold: config.MaxRestoreTime,
		RPOThreshold: config.MaxDataLoss,
	}

	// Create timeout context
	testCtx, cancel := context.WithTimeout(ctx, config.MaxTestDuration)
	defer cancel()

	// A real DR test requires a real target to restore into and validate.
	if config.Target == nil {
		err := fmt.Errorf("DR test aborted: no test target configured (TestConfig.Target is nil)")
		result.Success = false
		result.Error = err
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)
		return result, err
	}

	// Step 1: Provision test environment
	env, err := te.provisioner.ProvisionEnvironment(testCtx, &ProvisionConfig{
		DatabaseName:      databaseName,
		IsolatedNetwork:   config.IsolatedNetwork,
		EphemeralDatabase: config.EphemeralDatabase,
		Target:            config.Target,
	})
	if err != nil {
		result.Success = false
		result.Error = fmt.Errorf("environment provisioning failed: %w", err)
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)
		return result, err
	}
	result.TestEnvironmentID = env.ID

	// Ensure cleanup. If a rollback already tore the environment down this is
	// a no-op and CleanedUp is left as set by the rollback.
	defer func() {
		if !config.AutoCleanup || result.CleanedUp {
			return
		}
		cleanupErr := te.provisioner.CleanupEnvironment(context.Background(), env.ID)
		result.CleanedUp = (cleanupErr == nil)
	}()

	// Step 2: Perform restoration
	result.RestoreStartTime = time.Now()
	restoreSize, restoreErr := te.performRestore(testCtx, env, config)
	result.RestoreEndTime = time.Now()
	result.RestoreDuration = result.RestoreEndTime.Sub(result.RestoreStartTime)
	result.RTO = result.RestoreDuration
	result.RestoreSize = restoreSize
	result.RestoreStats = &RestoreStats{
		Duration:  result.RestoreDuration,
		SizeBytes: restoreSize,
	}

	if restoreErr != nil {
		result.Success = false
		result.Error = fmt.Errorf("restore failed: %w", restoreErr)
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)
		return result, restoreErr
	}

	// Check RTO
	result.RTOMet = result.RTO <= config.MaxRestoreTime

	// Step 3: Validate schema
	if config.ValidateSchema {
		schemaValid, schemaErrs := te.validator.ValidateSchema(testCtx, env, databaseName)
		result.SchemaValid = schemaValid
		result.SchemaErrors = schemaErrs
	}

	// Step 4: Validate row counts
	if config.ValidateRowCounts {
		rowCountValid, rowCountErrs := te.validator.ValidateRowCounts(testCtx, env, databaseName)
		result.RowCountValid = rowCountValid
		result.RowCountErrors = rowCountErrs
	}

	// Step 5: Validate sample data
	if config.ValidateSampleData {
		sampleValid, sampleErrs := te.validator.ValidateSampleData(testCtx, env, databaseName, config.SampleDataPercent)
		result.SampleDataValid = sampleValid
		result.SampleDataErrors = sampleErrs
	}

	// Step 6: Run smoke tests
	if config.RunSmokeTests {
		passed, failed, smokeErrs := te.runSmokeTests(testCtx, env, config.CustomQueries)
		result.SmokeTestsPassed = passed
		result.SmokeTestsFailed = failed
		result.SmokeTestErrors = smokeErrs
	}

	// Calculate RPO from real timestamps: the data-loss window is the gap
	// between when the backup was taken and when the restore completed. If no
	// backup timestamp is available the RPO cannot be verified and is treated
	// as unmet.
	if !config.Target.BackupTime.IsZero() {
		result.RPO = result.RestoreEndTime.Sub(config.Target.BackupTime)
		if result.RPO < 0 {
			result.RPO = 0
		}
		result.RPOMet = result.RPO <= config.MaxDataLoss
	} else {
		result.RPO = 0
		result.RPOMet = false
	}

	// Determine overall success
	result.Success = result.RTOMet &&
		result.RPOMet &&
		(!config.ValidateSchema || result.SchemaValid) &&
		(!config.ValidateRowCounts || result.RowCountValid) &&
		(!config.ValidateSampleData || result.SampleDataValid) &&
		(!config.RunSmokeTests || result.SmokeTestsFailed == 0)

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	// Perform automatic rollback if test failed and auto-rollback is enabled
	if !result.Success && te.autoRollback && config.AutoCleanup {
		rollbackErr := te.performRollback(ctx, env, result)
		if rollbackErr != nil {
			// Log rollback error but don't fail the test result
			// The test already failed, rollback failure is secondary
			result.Error = fmt.Errorf("test failed; rollback also failed: %w", rollbackErr)
		}
	}

	// Store test result in report generator
	if te.reportGenerator != nil {
		te.reportGenerator.AddTestResult(*result)
	}

	return result, nil
}

// performRestore performs a real database restore into the provisioned test
// target by applying the backup artifact referenced by the config. It returns
// the size, in bytes, of the restored backup. If no restorable backup is
// configured it returns a real error -- it never simulates success.
func (te *TestExecutor) performRestore(ctx context.Context, env *TestEnvironment, config *TestConfig) (int64, error) {
	if env.db == nil {
		return 0, fmt.Errorf("restore aborted: no live connection to test target")
	}

	target := config.Target
	if target == nil || target.BackupPath == "" {
		return 0, fmt.Errorf("restore aborted: no backup path configured for target")
	}

	info, err := os.Stat(target.BackupPath)
	if err != nil {
		return 0, fmt.Errorf("restore aborted: cannot access backup %q: %w", target.BackupPath, err)
	}

	script, err := os.ReadFile(target.BackupPath)
	if err != nil {
		return 0, fmt.Errorf("restore aborted: cannot read backup %q: %w", target.BackupPath, err)
	}

	applied, err := applySQLScript(ctx, env.db, string(script))
	if err != nil {
		return info.Size(), fmt.Errorf("restore failed after %d statement(s): %w", applied, err)
	}

	// Verify connectivity to the freshly restored database.
	if err := env.db.PingContext(ctx); err != nil {
		return info.Size(), fmt.Errorf("restore verification failed: %w", err)
	}

	return info.Size(), nil
}

// runSmokeTests executes real smoke-test queries against the restored target.
func (te *TestExecutor) runSmokeTests(ctx context.Context, env *TestEnvironment, customQueries []string) (passed, failed int, errors []string) {
	smokeTests := defaultSmokeQueries(env.driver)
	smokeTests = append(smokeTests, customQueries...)

	for _, query := range smokeTests {
		err := te.executeQuery(ctx, env, query)
		if err != nil {
			failed++
			errors = append(errors, fmt.Sprintf("Query failed: %s - Error: %s", query, err.Error()))
		} else {
			passed++
		}
	}

	return passed, failed, errors
}

// defaultSmokeQueries returns driver-appropriate default smoke tests: a
// connectivity probe plus a schema-visibility probe.
func defaultSmokeQueries(driver string) []string {
	switch normalizeDriver(driver) {
	case driverSQLite:
		return []string{
			"SELECT 1",
			"SELECT COUNT(*) FROM sqlite_master WHERE type = 'table'",
		}
	default: // postgres, mysql and other information_schema databases
		return []string{
			"SELECT 1",
			"SELECT COUNT(*) FROM information_schema.tables",
		}
	}
}

// executeQuery executes a real query against the test target and consumes any
// result set, returning an error if the query fails.
func (te *TestExecutor) executeQuery(ctx context.Context, env *TestEnvironment, query string) error {
	if env.db == nil {
		return fmt.Errorf("no live connection to test target")
	}

	rows, err := env.db.QueryContext(ctx, query)
	if err != nil {
		return err
	}
	defer rows.Close()

	// Drain the result set so drivers report any deferred errors.
	for rows.Next() {
	}
	return rows.Err()
}

// generateTestID generates a unique test ID.
func generateTestID() string {
	return uid.New("dr-test")
}

// EnvironmentProvisioner provisions test environments.
type EnvironmentProvisioner struct {
	environments map[string]*TestEnvironment
}

// NewEnvironmentProvisioner creates a new environment provisioner.
func NewEnvironmentProvisioner() *EnvironmentProvisioner {
	return &EnvironmentProvisioner{
		environments: make(map[string]*TestEnvironment),
	}
}

// TestEnvironment represents a test environment.
type TestEnvironment struct {
	ID                string
	DatabaseName      string
	ConnectionString  string
	IsolatedNetwork   bool
	EphemeralDatabase bool
	CreatedAt         time.Time
	Status            string // "provisioning", "ready", "cleaning_up", "destroyed"

	// Real connection to the test target. These are populated during
	// provisioning and used for the restore and all validation queries.
	db     *sql.DB
	driver string
	target *TargetConfig
}

// ProvisionConfig represents environment provisioning configuration.
type ProvisionConfig struct {
	DatabaseName      string
	IsolatedNetwork   bool
	EphemeralDatabase bool

	// Target is the real database the drill restores into. Required: full
	// ephemeral provisioning (spinning up a throwaway instance) is out of
	// scope, so a reachable target MUST be supplied.
	Target *TargetConfig
}

// ProvisionEnvironment provisions a new test environment by establishing a
// real, verified connection to the configured target. It does NOT simulate:
// if no target is configured, or the target is unreachable, it returns an
// error.
func (ep *EnvironmentProvisioner) ProvisionEnvironment(ctx context.Context, config *ProvisionConfig) (*TestEnvironment, error) {
	if config.Target == nil {
		return nil, fmt.Errorf("cannot provision DR test environment: no test target configured")
	}

	driver := normalizeDriver(config.Target.Driver)
	dsn, err := buildDSN(config.Target)
	if err != nil {
		return nil, fmt.Errorf("provision environment: %w", err)
	}

	env := &TestEnvironment{
		ID:                generateEnvironmentID(),
		DatabaseName:      config.DatabaseName,
		ConnectionString:  dsn,
		IsolatedNetwork:   config.IsolatedNetwork,
		EphemeralDatabase: config.EphemeralDatabase,
		CreatedAt:         time.Now(),
		Status:            "provisioning",
		driver:            driver,
		target:            config.Target,
	}

	// Open and verify real connectivity to the target.
	db, err := openTarget(ctx, config.Target)
	if err != nil {
		return nil, fmt.Errorf("provision environment: %w", err)
	}
	env.db = db

	env.Status = "ready"
	ep.environments[env.ID] = env

	return env, nil
}

// CleanupEnvironment cleans up a test environment by scrubbing every object
// that the restore created in the target and closing the connection.
func (ep *EnvironmentProvisioner) CleanupEnvironment(ctx context.Context, envID string) error {
	env, exists := ep.environments[envID]
	if !exists {
		return fmt.Errorf("environment not found: %s", envID)
	}

	env.Status = "cleaning_up"

	var scrubErr error
	if env.db != nil {
		// Drop all restored tables so the target is left clean for the next drill.
		scrubErr = dropAllTables(ctx, env.db, env.driver, env.DatabaseName)
		if closeErr := env.db.Close(); closeErr != nil && scrubErr == nil {
			scrubErr = closeErr
		}
		env.db = nil
	}

	env.Status = "destroyed"
	delete(ep.environments, envID)

	if scrubErr != nil {
		return fmt.Errorf("cleanup environment %s: %w", envID, scrubErr)
	}
	return nil
}

// generateEnvironmentID generates a unique environment ID.
func generateEnvironmentID() string {
	return uid.New("env")
}

// performRollback performs a real rollback when a test fails: it scrubs the
// restored objects from the test target and tears down the environment so no
// partial/failed restore is left behind.
func (te *TestExecutor) performRollback(ctx context.Context, env *TestEnvironment, result *TestResult) error {
	// CleanupEnvironment drops every restored table and closes the connection.
	cleanupErr := te.provisioner.CleanupEnvironment(ctx, env.ID)
	if cleanupErr != nil {
		return fmt.Errorf("rollback cleanup failed: %w", cleanupErr)
	}

	// Mark as cleaned up in result
	result.CleanedUp = true

	return nil
}
