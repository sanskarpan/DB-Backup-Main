// Package cleanroom performs clean-room isolated recovery: it restores a backup
// into a throwaway, isolated environment, runs a malware/ransomware scan and an
// integrity validation there, and only reports the backup as promotable when it
// is proven clean and structurally valid. The production target is never
// touched, so a poisoned or corrupt backup can be vetted safely before it is
// ever promoted into a live system.
package cleanroom

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	// Register the go-sqlite3 SQL driver so the isolated integrity validation can
	// open the recovered SQLite database.
	_ "github.com/mattn/go-sqlite3"

	"github.com/sanskarpan/db-backup/internal/database"
	"github.com/sanskarpan/db-backup/internal/models"
	"github.com/sanskarpan/db-backup/internal/restore"
	"github.com/sanskarpan/db-backup/internal/security/ransomware"
)

const (
	// sqliteDriverName is the database/sql driver used to open recovered SQLite
	// artifacts for the integrity check.
	sqliteDriverName = "sqlite3"

	// recoveredArtifactName is the file name of the restored database inside the
	// isolated working directory.
	recoveredArtifactName = "recovered.db"

	// integrityOK is the value PRAGMA integrity_check returns for a healthy
	// SQLite database.
	integrityOK = "ok"

	// userTableQuery lists user tables (excluding SQLite internal tables).
	userTableQuery = "SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'"
)

// Step names recorded in a Report.
const (
	stepIsolate   = "isolate"
	stepRestore   = "restore"
	stepScan      = "scan"
	stepIntegrity = "integrity"
)

// ErrScanFailed indicates the clean-room malware scan could not be completed, so
// the backup is rejected fail-closed rather than promoted on incomplete
// evidence.
var ErrScanFailed = errors.New("clean-room malware scan failed")

// ErrIntegrityFailed indicates the recovered database failed its structural
// integrity validation.
var ErrIntegrityFailed = errors.New("clean-room integrity validation failed")

// ErrRestoreFailed indicates the backup could not be restored into the isolated
// environment.
var ErrRestoreFailed = errors.New("clean-room restore failed")

// Verdict is the clean-room recovery decision for a backup.
type Verdict string

const (
	// VerdictPromotable means the backup restored cleanly, passed the malware
	// scan below the configured threshold, and passed integrity validation.
	VerdictPromotable Verdict = "PROMOTABLE"
	// VerdictQuarantined means the malware scan detected a threat at or above the
	// configured severity threshold; the backup must not be promoted.
	VerdictQuarantined Verdict = "QUARANTINED"
	// VerdictFailed means the clean-room recovery could not reach a clean
	// promotable state (restore, scan, or integrity failure); fail-closed.
	VerdictFailed Verdict = "FAILED"
)

// Scanner scans an on-disk artifact for malware/ransomware indicators. It is
// satisfied by *ransomware.Detector.
type Scanner interface {
	// ScanFile scans the file at path and returns a threat report.
	ScanFile(ctx context.Context, path string) (*ransomware.ThreatReport, error)
}

// Restorer restores a backup into a target database. It is satisfied by
// *restore.Engine.
type Restorer interface {
	// RestoreBackup restores backupMetadata into the target described by opts.
	RestoreBackup(ctx context.Context, backupMetadata *models.BackupMetadata, opts *restore.RestoreOptions) (*restore.RestoreResult, error)
}

// Step records the outcome of a single clean-room recovery phase.
type Step struct {
	// Name identifies the phase (isolate, restore, scan, integrity).
	Name string
	// OK reports whether the phase succeeded.
	OK bool
	// Detail carries a human-readable note about the phase outcome.
	Detail string
}

// Options configures a clean-room recovery.
type Options struct {
	// BaseDir is the parent directory under which the isolated working directory
	// is created. When empty, the OS temporary directory is used.
	BaseDir string
	// Threshold is the minimum threat severity that quarantines the backup. When
	// empty it defaults to ransomware.ThreatLevelHigh.
	Threshold ransomware.ThreatLevel
	// ExpectedTables, when non-empty, requires that at least one of the named
	// tables exists in the recovered database for integrity validation to pass.
	// When empty, integrity validation requires at least one user table.
	ExpectedTables []string
	// Cleanup, when true, removes the isolated working directory after recovery.
	Cleanup bool
}

// Report describes the outcome of a clean-room recovery.
type Report struct {
	// Isolated reports whether an isolated working directory was created.
	Isolated bool
	// Steps records each phase outcome in execution order.
	Steps []Step
	// Verdict is the overall clean-room decision.
	Verdict Verdict
	// ThreatLevel is the highest threat severity observed during the scan.
	ThreatLevel ransomware.ThreatLevel
	// RecoveredPath is the path of the restored database inside the isolated
	// working directory.
	RecoveredPath string
	// Elapsed is the total wall-clock time spent on the recovery.
	Elapsed time.Duration
}

func (r *Report) addStep(name string, ok bool, detail string) {
	r.Steps = append(r.Steps, Step{Name: name, OK: ok, Detail: detail})
}

// Orchestrator drives clean-room isolated recovery using an injected Restorer
// and Scanner.
type Orchestrator struct {
	restorer Restorer
	scanner  Scanner
}

// NewOrchestrator creates an Orchestrator from a Restorer and a Scanner.
func NewOrchestrator(restorer Restorer, scanner Scanner) *Orchestrator {
	return &Orchestrator{restorer: restorer, scanner: scanner}
}

// Recover restores meta into an isolated environment, scans and integrity-checks
// it there, and reports whether the backup is promotable. It never writes to any
// production target. A threat at or above the configured threshold yields a
// Quarantined verdict; restore, scan, or integrity failures yield a Failed
// verdict fail-closed. The returned error is non-nil for operational and
// fail-closed conditions and nil for a clean Promotable or a determinate
// Quarantined verdict.
func (o *Orchestrator) Recover(ctx context.Context, meta *models.BackupMetadata, opts *Options) (*Report, error) {
	start := time.Now()
	if opts == nil {
		opts = &Options{}
	}
	report := &Report{Verdict: VerdictFailed, ThreatLevel: ransomware.ThreatLevelNone}
	defer func() { report.Elapsed = time.Since(start) }()

	isolated, err := o.isolate(opts, report)
	if err != nil {
		return report, err
	}
	if opts.Cleanup {
		defer func() {
			_ = os.RemoveAll(isolated)
		}()
	}

	recoveredPath := filepath.Join(isolated, recoveredArtifactName)
	report.RecoveredPath = recoveredPath

	if restoreErr := o.restoreIsolated(ctx, meta, recoveredPath, report); restoreErr != nil {
		return report, restoreErr
	}

	quarantined, err := o.scan(ctx, recoveredPath, opts, report)
	if err != nil {
		return report, err
	}
	if quarantined {
		report.Verdict = VerdictQuarantined
		return report, nil
	}

	if err = validateIntegrity(ctx, recoveredPath, opts.ExpectedTables); err != nil {
		report.addStep(stepIntegrity, false, err.Error())
		report.Verdict = VerdictFailed
		return report, fmt.Errorf("%w: %w", ErrIntegrityFailed, err)
	}
	report.addStep(stepIntegrity, true, "integrity_check ok")

	report.Verdict = VerdictPromotable
	return report, nil
}

// isolate creates the isolated working directory and records the step.
func (o *Orchestrator) isolate(opts *Options, report *Report) (string, error) {
	base := opts.BaseDir
	if base == "" {
		base = os.TempDir()
	}
	// Create the base directory when a caller supplies one that does not yet
	// exist so callers need not pre-create it before requesting recovery.
	if err := os.MkdirAll(base, 0o750); err != nil {
		report.addStep(stepIsolate, false, err.Error())
		return "", fmt.Errorf("create clean-room base dir: %w", err)
	}
	isolated, err := os.MkdirTemp(base, "cleanroom-")
	if err != nil {
		report.addStep(stepIsolate, false, err.Error())
		return "", fmt.Errorf("create isolated workdir: %w", err)
	}
	report.Isolated = true
	report.addStep(stepIsolate, true, isolated)
	return isolated, nil
}

// restoreIsolated restores the backup into the isolated recovered path.
func (o *Orchestrator) restoreIsolated(ctx context.Context, meta *models.BackupMetadata, recoveredPath string, report *Report) error {
	res, err := o.restorer.RestoreBackup(ctx, meta, &restore.RestoreOptions{TargetDatabase: recoveredPath})
	if err != nil {
		report.addStep(stepRestore, false, err.Error())
		return fmt.Errorf("%w: %w", ErrRestoreFailed, err)
	}
	if res == nil || res.Status != database.RestoreStatusSuccess {
		report.addStep(stepRestore, false, "restore did not report success")
		return fmt.Errorf("%w: restore did not report success", ErrRestoreFailed)
	}
	report.addStep(stepRestore, true, string(res.Status))
	return nil
}

// scan runs the malware scan over the recovered artifact. It returns whether the
// backup must be quarantined (a detection at/above the threshold). A scan error
// is fail-closed: it returns a non-nil error so the backup is never promoted.
func (o *Orchestrator) scan(ctx context.Context, recoveredPath string, opts *Options, report *Report) (bool, error) {
	threshold := opts.Threshold
	if threshold == "" {
		threshold = ransomware.ThreatLevelHigh
	}

	threatReport, err := o.scanner.ScanFile(ctx, recoveredPath)
	if err != nil {
		report.addStep(stepScan, false, err.Error())
		report.Verdict = VerdictFailed
		return false, fmt.Errorf("%w: %w", ErrScanFailed, err)
	}
	if threatReport == nil {
		report.addStep(stepScan, false, "scanner returned no report")
		report.Verdict = VerdictFailed
		return false, fmt.Errorf("%w: scanner returned no report", ErrScanFailed)
	}

	report.ThreatLevel = threatReport.ThreatLevel
	if severityRank(threatReport.ThreatLevel) >= severityRank(threshold) {
		report.addStep(stepScan, false, fmt.Sprintf("threat %s >= threshold %s: %s",
			threatReport.ThreatLevel, threshold, threatReport.Description))
		return true, nil
	}
	report.addStep(stepScan, true, fmt.Sprintf("threat level %s below threshold %s",
		threatReport.ThreatLevel, threshold))
	return false, nil
}

// validateIntegrity opens the recovered SQLite database, runs PRAGMA
// integrity_check, and confirms an expected table exists.
func validateIntegrity(ctx context.Context, dbPath string, expectedTables []string) error {
	db, err := sql.Open(sqliteDriverName, dbPath)
	if err != nil {
		return fmt.Errorf("open recovered database: %w", err)
	}
	defer func() {
		_ = db.Close()
	}()

	if err = db.PingContext(ctx); err != nil {
		return fmt.Errorf("ping recovered database: %w", err)
	}

	var check string
	if err = db.QueryRowContext(ctx, "PRAGMA integrity_check").Scan(&check); err != nil {
		return fmt.Errorf("run integrity_check: %w", err)
	}
	if check != integrityOK {
		return fmt.Errorf("integrity_check reported %q", check)
	}

	tables, err := userTables(ctx, db)
	if err != nil {
		return err
	}
	return requireExpectedTable(tables, expectedTables)
}

// userTables returns the user table names in the database.
func userTables(ctx context.Context, db *sql.DB) ([]string, error) {
	rows, err := db.QueryContext(ctx, userTableQuery)
	if err != nil {
		return nil, fmt.Errorf("list tables: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	var tables []string
	for rows.Next() {
		var name string
		if err = rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("scan table name: %w", err)
		}
		tables = append(tables, name)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tables: %w", err)
	}
	return tables, nil
}

// requireExpectedTable confirms the recovered database contains at least one
// expected table (or any user table when none are specified).
func requireExpectedTable(tables, expected []string) error {
	if len(tables) == 0 {
		return errors.New("recovered database contains no user tables")
	}
	if len(expected) == 0 {
		return nil
	}
	present := make(map[string]struct{}, len(tables))
	for _, t := range tables {
		present[t] = struct{}{}
	}
	for _, want := range expected {
		if _, ok := present[want]; ok {
			return nil
		}
	}
	return fmt.Errorf("none of the expected tables %v are present", expected)
}

// severityRank maps a threat level to an orderable rank so severities can be
// compared against the configured quarantine threshold.
func severityRank(level ransomware.ThreatLevel) int {
	switch level {
	case ransomware.ThreatLevelCritical:
		return 4
	case ransomware.ThreatLevelHigh:
		return 3
	case ransomware.ThreatLevelMedium:
		return 2
	case ransomware.ThreatLevelLow:
		return 1
	case ransomware.ThreatLevelNone:
		return 0
	default:
		return 0
	}
}
