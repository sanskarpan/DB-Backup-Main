package backuphealth

import (
	"testing"
	"time"

	"github.com/sanskarpan/db-backup/internal/backup"
	"github.com/sanskarpan/db-backup/internal/database"
	"github.com/sanskarpan/db-backup/internal/dbregistry"
	"github.com/sanskarpan/db-backup/internal/models"
	"github.com/sanskarpan/db-backup/internal/scheduler"
)

// dailyCron runs once a day at 02:00 (5-field standard cron).
const dailyCron = "0 2 * * *"

func job(enabled bool, nextRun time.Time) *scheduler.ScheduledJob {
	return &scheduler.ScheduledJob{
		ID:         "j1",
		Name:       "j1",
		Schedule:   dailyCron,
		Enabled:    enabled,
		NextRun:    nextRun,
		BackupOpts: &backup.CreateOptions{Database: "app"},
	}
}

func successBackup(end time.Time) *models.BackupMetadata {
	return &models.BackupMetadata{
		Database:  "app",
		Status:    database.BackupStatusSuccess,
		StartTime: end.Add(-time.Minute),
		EndTime:   end,
	}
}

func regDB() *dbregistry.Database {
	return &dbregistry.Database{ID: "d1", Name: "app-db", Type: "postgres", Host: "h", Database: "app"}
}

func TestAnalyze_RecentSuccessNotStale(t *testing.T) {
	now := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	jobs := []*scheduler.ScheduledJob{job(true, now.Add(12*time.Hour))}
	backups := []*models.BackupMetadata{successBackup(now.Add(-1 * time.Hour))}

	rep := Analyze(jobs, backups, nil, now, defaultWindow())

	if len(rep.Schedules) != 1 {
		t.Fatalf("expected 1 schedule, got %d", len(rep.Schedules))
	}
	sh := rep.Schedules[0]
	if sh.IsStale {
		t.Errorf("expected recent success to be not stale, reason=%q", sh.Reason)
	}
	if sh.LastSuccessTime == nil {
		t.Errorf("expected last success time to be set")
	}
	if sh.ExpectedIntervalSeconds != 24*3600 {
		t.Errorf("expected daily interval of 86400s, got %v", sh.ExpectedIntervalSeconds)
	}
	if rep.Summary.StaleSchedules != 0 || rep.Summary.ProtectionStatus != "healthy" {
		t.Errorf("expected healthy summary, got %+v", rep.Summary)
	}
}

func TestAnalyze_OldSuccessIsStale(t *testing.T) {
	now := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	// Last success 3 days ago, daily schedule -> older than 2x interval.
	jobs := []*scheduler.ScheduledJob{job(true, now.Add(12*time.Hour))}
	backups := []*models.BackupMetadata{successBackup(now.Add(-72 * time.Hour))}

	rep := Analyze(jobs, backups, nil, now, defaultWindow())

	sh := rep.Schedules[0]
	if !sh.IsStale {
		t.Errorf("expected stale for 3-day-old success on daily schedule")
	}
	if rep.Summary.StaleSchedules != 1 {
		t.Errorf("expected 1 stale schedule, got %d", rep.Summary.StaleSchedules)
	}
	if rep.Summary.ProtectionStatus != "degraded" {
		t.Errorf("expected degraded status, got %q", rep.Summary.ProtectionStatus)
	}
}

func TestAnalyze_MissedRunIsStale(t *testing.T) {
	now := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	// Recent success (not stale by age) but NextRun is 2 days in the past ->
	// overdue by more than one daily interval => missed runs.
	jobs := []*scheduler.ScheduledJob{job(true, now.Add(-48*time.Hour))}
	backups := []*models.BackupMetadata{successBackup(now.Add(-1 * time.Hour))}

	rep := Analyze(jobs, backups, nil, now, defaultWindow())

	if !rep.Schedules[0].IsStale {
		t.Errorf("expected stale due to overdue next run")
	}
	if rep.Schedules[0].Reason != "scheduled run overdue (missed runs)" {
		t.Errorf("unexpected reason %q", rep.Schedules[0].Reason)
	}
}

func TestAnalyze_NoSuccessScheduleStaleAndDatabaseUnprotected(t *testing.T) {
	now := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	jobs := []*scheduler.ScheduledJob{job(true, now.Add(12*time.Hour))}
	dbs := []*dbregistry.Database{regDB()}

	rep := Analyze(jobs, nil, dbs, now, defaultWindow())

	if !rep.Schedules[0].IsStale || rep.Schedules[0].Reason != "no successful backup recorded" {
		t.Errorf("expected stale with no-success reason, got %+v", rep.Schedules[0])
	}
	if len(rep.ProtectionGaps) != 1 || rep.ProtectionGaps[0].Status != "unprotected" {
		t.Fatalf("expected 1 unprotected gap, got %+v", rep.ProtectionGaps)
	}
	if rep.Summary.UnprotectedDatabases != 1 || rep.Summary.ProtectionStatus != "at_risk" {
		t.Errorf("expected at_risk with 1 unprotected, got %+v", rep.Summary)
	}
}

func TestAnalyze_UnderProtectedDatabase(t *testing.T) {
	now := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	// Success exists but 3 days old, window is 24h -> under-protected.
	backups := []*models.BackupMetadata{successBackup(now.Add(-72 * time.Hour))}
	dbs := []*dbregistry.Database{regDB()}

	rep := Analyze(nil, backups, dbs, now, 24*time.Hour)

	if len(rep.ProtectionGaps) != 1 || rep.ProtectionGaps[0].Status != "under_protected" {
		t.Fatalf("expected 1 under_protected gap, got %+v", rep.ProtectionGaps)
	}
	if rep.ProtectionGaps[0].LastSuccessTime == nil {
		t.Errorf("expected last success time on under-protected gap")
	}
	if rep.Summary.UnderProtectedDatabases != 1 || rep.Summary.ProtectionStatus != "degraded" {
		t.Errorf("expected degraded with 1 under-protected, got %+v", rep.Summary)
	}
}

func TestAnalyze_ProtectedDatabaseNoGap(t *testing.T) {
	now := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	backups := []*models.BackupMetadata{successBackup(now.Add(-1 * time.Hour))}
	dbs := []*dbregistry.Database{regDB()}

	rep := Analyze(nil, backups, dbs, now, 24*time.Hour)

	if len(rep.ProtectionGaps) != 0 {
		t.Errorf("expected no gaps for recently backed-up database, got %+v", rep.ProtectionGaps)
	}
	if rep.Summary.TotalDatabases != 1 || rep.Summary.ProtectionStatus != "healthy" {
		t.Errorf("expected healthy with 1 protected db, got %+v", rep.Summary)
	}
}

func TestAnalyze_DisabledScheduleSkipped(t *testing.T) {
	now := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	jobs := []*scheduler.ScheduledJob{job(false, time.Time{})}

	rep := Analyze(jobs, nil, nil, now, defaultWindow())

	if len(rep.Schedules) != 0 || rep.Summary.TotalSchedules != 0 {
		t.Errorf("expected disabled schedule to be skipped, got %+v", rep.Schedules)
	}
}

func TestAnalyze_FailedBackupDoesNotCount(t *testing.T) {
	now := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	failed := &models.BackupMetadata{Database: "app", Status: database.BackupStatusFailed, EndTime: now.Add(-time.Hour)}
	dbs := []*dbregistry.Database{regDB()}

	rep := Analyze(nil, []*models.BackupMetadata{failed}, dbs, now, 24*time.Hour)

	if rep.Summary.UnprotectedDatabases != 1 {
		t.Errorf("expected failed backup to leave db unprotected, got %+v", rep.Summary)
	}
}

func TestCronInterval(t *testing.T) {
	now := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	got, ok := cronInterval(dailyCron, now)
	if !ok || got != 24*time.Hour {
		t.Errorf("expected 24h interval, got %v ok=%v", got, ok)
	}
	if _, ok := cronInterval("not-a-cron", now); ok {
		t.Errorf("expected invalid cron to report unknown interval")
	}
}

func defaultWindow() time.Duration { return 24 * time.Hour }
