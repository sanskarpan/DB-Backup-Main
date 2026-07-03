// Package backuphealth analyzes scheduled jobs, backup history and the set of
// registered databases to surface stale/missed backups and protection gaps.
//
// The heart of the package is Analyze: a pure function that takes the current
// jobs, backups and databases as inputs and returns a Report. It performs no
// I/O, so it is fully unit-testable. The API layer gathers the inputs from the
// scheduler, backup engine and database registry and calls Analyze.
package backuphealth

import (
	"time"

	"github.com/robfig/cron/v3"

	"github.com/sanskarpan/db-backup/internal/database"
	"github.com/sanskarpan/db-backup/internal/dbregistry"
	"github.com/sanskarpan/db-backup/internal/models"
	"github.com/sanskarpan/db-backup/internal/scheduler"
)

// staleIntervalMultiplier defines how many expected intervals may elapse since
// the last successful backup before a schedule is considered stale.
const staleIntervalMultiplier = 2

// Report is the backup-health analysis returned to API callers.
type Report struct {
	GeneratedAt         time.Time        `json:"generated_at"`
	Schedules           []ScheduleHealth `json:"schedules"`
	ProtectionGaps      []ProtectionGap  `json:"protection_gaps"`
	Summary             Summary          `json:"summary"`
	RecentWindowSeconds float64          `json:"recent_window_seconds"`
}

// ScheduleHealth describes the freshness of a single enabled schedule.
type ScheduleHealth struct {
	NextRun                 time.Time  `json:"next_run"`
	LastSuccessTime         *time.Time `json:"last_success_time,omitempty"`
	ScheduleID              string     `json:"schedule_id"`
	ScheduleName            string     `json:"schedule_name"`
	Database                string     `json:"database"`
	CronExpression          string     `json:"cron_expression"`
	Reason                  string     `json:"reason"`
	ExpectedIntervalSeconds float64    `json:"expected_interval_seconds"`
	OverdueBySeconds        float64    `json:"overdue_by_seconds"`
	IsStale                 bool       `json:"is_stale"`
}

// ProtectionGap describes a registered database that is unprotected (no
// successful backup at all) or under-protected (no success within the window).
type ProtectionGap struct {
	LastSuccessTime *time.Time `json:"last_success_time,omitempty"`
	DatabaseID      string     `json:"database_id"`
	Name            string     `json:"name"`
	Type            string     `json:"type"`
	Host            string     `json:"host"`
	Database        string     `json:"database"`
	Status          string     `json:"status"`
}

// Summary is the top-level rollup of the report.
type Summary struct {
	ProtectionStatus        string `json:"protection_status"`
	TotalSchedules          int    `json:"total_schedules"`
	StaleSchedules          int    `json:"stale_schedules"`
	TotalDatabases          int    `json:"total_databases"`
	UnprotectedDatabases    int    `json:"unprotected_databases"`
	UnderProtectedDatabases int    `json:"under_protected_databases"`
}

// Analyze produces a backup-health report from the current jobs, backups and
// registered databases. now is the reference time (injected for testability)
// and recentWindow is the protection-gap window (e.g. 24h): a database with no
// successful backup within it is flagged under-protected.
func Analyze(
	jobs []*scheduler.ScheduledJob,
	backups []*models.BackupMetadata,
	databases []*dbregistry.Database,
	now time.Time,
	recentWindow time.Duration,
) Report {
	lastSuccess := lastSuccessByDatabase(backups)
	schedules, stale := analyzeSchedules(jobs, lastSuccess, now)
	gaps, unprotected, underProtected := analyzeProtectionGaps(databases, lastSuccess, now, recentWindow)

	return Report{
		GeneratedAt:         now,
		Schedules:           schedules,
		ProtectionGaps:      gaps,
		RecentWindowSeconds: recentWindow.Seconds(),
		Summary: Summary{
			ProtectionStatus:        overallStatus(stale, unprotected, underProtected),
			TotalSchedules:          len(schedules),
			StaleSchedules:          stale,
			TotalDatabases:          len(databases),
			UnprotectedDatabases:    unprotected,
			UnderProtectedDatabases: underProtected,
		},
	}
}

// lastSuccessByDatabase maps each database name to the completion time of its
// most recent successful backup.
func lastSuccessByDatabase(backups []*models.BackupMetadata) map[string]time.Time {
	m := make(map[string]time.Time)
	for _, b := range backups {
		if b == nil || !isSuccessful(b.Status) {
			continue
		}
		t := completionTime(b)
		if existing, ok := m[b.Database]; !ok || t.After(existing) {
			m[b.Database] = t
		}
	}
	return m
}

// isSuccessful reports whether a backup status counts as a durable success.
func isSuccessful(status database.BackupStatus) bool {
	return status == database.BackupStatusSuccess || status == database.BackupStatusCompleted
}

// completionTime returns the time a backup finished, falling back to its start
// time when the end time was never recorded.
func completionTime(b *models.BackupMetadata) time.Time {
	if !b.EndTime.IsZero() {
		return b.EndTime
	}
	return b.StartTime
}

// analyzeSchedules evaluates every enabled schedule, returning the per-schedule
// health entries and the number flagged stale. Disabled schedules are skipped.
func analyzeSchedules(
	jobs []*scheduler.ScheduledJob,
	lastSuccess map[string]time.Time,
	now time.Time,
) (schedules []ScheduleHealth, staleCount int) {
	schedules = make([]ScheduleHealth, 0, len(jobs))
	for _, job := range jobs {
		if job == nil || !job.Enabled {
			continue
		}
		sh := evaluateSchedule(job, lastSuccess, now)
		if sh.IsStale {
			staleCount++
		}
		schedules = append(schedules, sh)
	}
	return schedules, staleCount
}

// evaluateSchedule builds the health entry for a single enabled schedule.
func evaluateSchedule(
	job *scheduler.ScheduledJob,
	lastSuccess map[string]time.Time,
	now time.Time,
) ScheduleHealth {
	dbName := ""
	if job.BackupOpts != nil {
		dbName = job.BackupOpts.Database
	}

	sh := ScheduleHealth{
		ScheduleID:     job.ID,
		ScheduleName:   job.Name,
		Database:       dbName,
		CronExpression: job.Schedule,
		NextRun:        job.NextRun,
	}

	interval, intervalKnown := cronInterval(job.Schedule, now)
	if intervalKnown {
		sh.ExpectedIntervalSeconds = interval.Seconds()
	}

	last, hasSuccess := lastSuccess[dbName]
	if hasSuccess {
		t := last
		sh.LastSuccessTime = &t
		sh.OverdueBySeconds = now.Sub(last).Seconds()
	}

	sh.IsStale, sh.Reason = staleReason(&staleInputs{
		hasSuccess:    hasSuccess,
		intervalKnown: intervalKnown,
		interval:      interval,
		lastSuccess:   last,
		nextRun:       job.NextRun,
		now:           now,
	})
	return sh
}

// staleInputs bundles the values needed to decide whether a schedule is stale.
type staleInputs struct {
	lastSuccess   time.Time
	nextRun       time.Time
	now           time.Time
	interval      time.Duration
	hasSuccess    bool
	intervalKnown bool
}

// staleReason decides whether a schedule is stale and why. A schedule is stale
// when it has never produced a successful backup, when its last success is
// older than staleIntervalMultiplier x the expected interval, or when its next
// run is overdue by more than one interval (indicating missed runs).
func staleReason(in *staleInputs) (isStale bool, reason string) {
	if !in.hasSuccess {
		return true, "no successful backup recorded"
	}
	if in.intervalKnown {
		if in.now.Sub(in.lastSuccess) > staleIntervalMultiplier*in.interval {
			return true, "last success older than 2x expected interval"
		}
		if !in.nextRun.IsZero() && in.now.Sub(in.nextRun) > in.interval {
			return true, "scheduled run overdue (missed runs)"
		}
	}
	return false, "ok"
}

// cronInterval derives the cadence of a cron expression as the gap between the
// next two scheduled occurrences after now (the "next-after-next" gap). It
// parses standard 5-field cron, matching the scheduler's own validation. When
// the expression cannot be parsed the interval is reported as unknown.
func cronInterval(expr string, now time.Time) (interval time.Duration, ok bool) {
	sched, err := cron.ParseStandard(expr)
	if err != nil {
		return 0, false
	}
	first := sched.Next(now)
	if first.IsZero() {
		return 0, false
	}
	second := sched.Next(first)
	if second.IsZero() {
		return 0, false
	}
	return second.Sub(first), true
}

// analyzeProtectionGaps flags registered databases with no successful backup
// (unprotected) or none within recentWindow (under-protected).
func analyzeProtectionGaps(
	databases []*dbregistry.Database,
	lastSuccess map[string]time.Time,
	now time.Time,
	recentWindow time.Duration,
) (gaps []ProtectionGap, unprotected, underProtected int) {
	gaps = make([]ProtectionGap, 0)
	for _, db := range databases {
		if db == nil {
			continue
		}
		last, has := lastSuccess[db.Database]
		if !has {
			unprotected++
			gaps = append(gaps, buildGap(db, nil, "unprotected"))
			continue
		}
		if now.Sub(last) > recentWindow {
			underProtected++
			t := last
			gaps = append(gaps, buildGap(db, &t, "under_protected"))
		}
	}
	return gaps, unprotected, underProtected
}

// buildGap constructs a ProtectionGap entry for a database.
func buildGap(db *dbregistry.Database, lastSuccess *time.Time, status string) ProtectionGap {
	return ProtectionGap{
		DatabaseID:      db.ID,
		Name:            db.Name,
		Type:            db.Type,
		Host:            db.Host,
		Database:        db.Database,
		Status:          status,
		LastSuccessTime: lastSuccess,
	}
}

// overallStatus rolls the counts up into a single protection status.
// Overall protection-status values.
const (
	statusHealthy  = "healthy"
	statusDegraded = "degraded"
	statusAtRisk   = "at_risk"
)

func overallStatus(stale, unprotected, underProtected int) string {
	switch {
	case unprotected > 0:
		return statusAtRisk
	case stale > 0 || underProtected > 0:
		return statusDegraded
	default:
		return statusHealthy
	}
}
