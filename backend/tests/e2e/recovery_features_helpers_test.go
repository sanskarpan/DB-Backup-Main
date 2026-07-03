package e2e

import (
	"context"
	"database/sql"
	"errors"

	aiRecovery "github.com/sanskarpan/db-backup/internal/ai/recovery"
	"github.com/sanskarpan/db-backup/internal/database"
	"github.com/sanskarpan/db-backup/internal/models"
	"github.com/sanskarpan/db-backup/internal/recovery/cleanroom"
	"github.com/sanskarpan/db-backup/internal/restore"
	"github.com/sanskarpan/db-backup/internal/security/ransomware"
)

// highThreatScanner is a restore/cleanroom/golden Scanner that always reports a
// HIGH threat, used to prove fail-closed behavior. It is a legitimate test
// double for a scanner backend, not a production shortcut.
type highThreatScanner struct{}

// ScanFile always returns a HIGH threat report for the given path.
func (highThreatScanner) ScanFile(_ context.Context, path string) (*ransomware.ThreatReport, error) {
	return &ransomware.ThreatReport{ThreatLevel: ransomware.ThreatLevelHigh, FilePath: path}, nil
}

// okValidator is a golden.Validator that always approves a candidate, used to
// isolate the scan/promotion behavior under test.
type okValidator struct{}

// Validate always reports the candidate as valid.
func (okValidator) Validate(_ context.Context, _ *models.BackupMetadata) error { return nil }

// backupPointSource adapts a set of real backups into the AI assistant's
// RecoveryPointSource, exposing each backup as a full recovery point.
type backupPointSource struct {
	backups map[string]*models.BackupMetadata
	dbName  string
}

// AvailablePoints returns a recovery point for every backup of the named
// database.
func (s *backupPointSource) AvailablePoints(_ context.Context, dbName string) ([]aiRecovery.Point, error) {
	points := make([]aiRecovery.Point, 0, len(s.backups))
	for _, meta := range s.backups {
		if dbName != "" && s.dbName != dbName {
			continue
		}
		points = append(points, aiRecovery.Point{
			BackupID:  meta.ID,
			DBName:    dbName,
			Time:      meta.EndTime,
			Kind:      aiRecovery.KindFull,
			SizeBytes: meta.Size,
			Immutable: meta.Immutable,
		})
	}
	return points, nil
}

// cleanRoomAdapter adapts the real clean-room orchestrator into the AI
// assistant's CleanRoomValidator.
type cleanRoomAdapter struct {
	orch    *cleanroom.Orchestrator
	backups map[string]*models.BackupMetadata
	baseDir string
}

// Validate runs a real clean-room recovery for the point's backup and reports
// whether the backup is promotable.
func (c *cleanRoomAdapter) Validate(ctx context.Context, rp *aiRecovery.Point) (ok bool, detail string, err error) {
	meta, ok := c.backups[rp.BackupID]
	if !ok {
		return false, "", errors.New("unknown backup for recovery point")
	}
	report, err := c.orch.Recover(ctx, meta, &cleanroom.Options{
		BaseDir:        c.baseDir,
		ExpectedTables: []string{"users"},
		Cleanup:        true,
	})
	if err != nil {
		return false, string(report.Verdict), err
	}
	return report.Verdict == cleanroom.VerdictPromotable, string(report.Verdict), nil
}

// restoreAdapter adapts the real restore engine into the AI assistant's
// Restorer.
type restoreAdapter struct {
	engine  *restore.Engine
	backups map[string]*models.BackupMetadata
}

// Restore restores the point's backup into the target path via the real
// restore engine.
func (r *restoreAdapter) Restore(ctx context.Context, rp *aiRecovery.Point, target string) error {
	meta, ok := r.backups[rp.BackupID]
	if !ok {
		return errors.New("unknown backup for recovery point")
	}
	result, err := r.engine.RestoreBackup(ctx, meta, &restore.RestoreOptions{TargetDatabase: target})
	if err != nil {
		return err
	}
	if result.Status != database.RestoreStatusSuccess {
		return errors.New("restore did not complete successfully")
	}
	return nil
}

// openSQLite opens the sqlite database at path for reading.
func openSQLite(path string) (*sql.DB, error) {
	return sql.Open("sqlite3", path)
}

// sqliteVerifier is the AI assistant's Verifier: it confirms a restored sqlite
// database is queryable and non-empty.
type sqliteVerifier struct{}

// Verify opens the restored database and confirms the users table is readable.
func (sqliteVerifier) Verify(ctx context.Context, target string) (ok bool, detail string, err error) {
	db, err := openSQLite(target)
	if err != nil {
		return false, "open failed", err
	}
	defer func() { _ = db.Close() }()
	var count int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM users;").Scan(&count); err != nil {
		return false, "query failed", err
	}
	if count == 0 {
		return false, "restored database is empty", nil
	}
	return true, "ok", nil
}
