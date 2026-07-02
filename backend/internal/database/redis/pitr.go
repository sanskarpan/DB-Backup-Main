package redis

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sanskarpan/db-backup/internal/database"
)

// ErrPITRNotSupported indicates that Redis cannot perform true, time-based
// point-in-time recovery. Redis has no continuous, timestamped write log that
// can be replayed to an arbitrary instant: RDB captures discrete snapshots and
// standard AOF files do not expose per-command timestamps that could be
// replayed up to a specific second.
var ErrPITRNotSupported = errors.New("redis: time-based point-in-time recovery is not supported; Redis cannot be replayed to an arbitrary timestamp")

// ErrPITRGranularity indicates that the requested target time cannot be honored
// because recovery is limited to whole backup snapshots rather than arbitrary
// timestamps.
var ErrPITRGranularity = errors.New("redis: point-in-time recovery granularity is limited to whole backup snapshots")

// PITRManager handles snapshot-based recovery for Redis.
//
// IMPORTANT: Redis does not support true transactional point-in-time recovery.
// The only artifacts available are RDB snapshots and AOF files, neither of
// which can be replayed to an arbitrary second. This manager therefore offers
// best-effort recovery to the state captured by a whole backup snapshot and is
// honest (returns an error) when a request cannot be satisfied.
type PITRManager struct {
	driver *RedisDriver
}

// NewPITRManager creates a new PITR manager
func NewPITRManager(driver *RedisDriver) *PITRManager {
	return &PITRManager{
		driver: driver,
	}
}

// EnablePITR enables point-in-time recovery by enabling AOF
func (p *PITRManager) EnablePITR(ctx context.Context) error {
	// Enable AOF
	if err := p.driver.client.ConfigSet(ctx, "appendonly", "yes").Err(); err != nil {
		return fmt.Errorf("failed to enable AOF: %w", err)
	}

	// Set AOF sync policy to everysec for balance between performance and durability
	if err := p.driver.client.ConfigSet(ctx, "appendfsync", "everysec").Err(); err != nil {
		return fmt.Errorf("failed to set AOF sync policy: %w", err)
	}

	return nil
}

// DisablePITR disables point-in-time recovery
func (p *PITRManager) DisablePITR(ctx context.Context) error {
	return p.driver.client.ConfigSet(ctx, "appendonly", "no").Err()
}

// RestoreToPIT performs best-effort recovery toward a target time.
//
// Redis cannot be replayed to an arbitrary timestamp (see ErrPITRNotSupported).
// The realistic behavior, and the one implemented here, is to restore the whole
// backup snapshot at backupPath, provided that snapshot represents state at or
// before targetTime. Recovery granularity is therefore limited to the discrete
// moments at which backups were taken, NOT arbitrary timestamps: the resulting
// state reflects the snapshot's creation time, which will be at or before
// targetTime.
//
// If the snapshot is newer than targetTime it cannot satisfy the request (it
// contains writes that happened after the requested moment) and an honest
// ErrPITRGranularity error is returned instead of silently restoring the wrong
// data. The target time is never ignored.
func (p *PITRManager) RestoreToPIT(ctx context.Context, targetTime time.Time, backupPath string) error {
	// Use the backup file's modification time as a proxy for the moment the
	// snapshot captured state. copyFile creates the backup at capture time, so
	// this is a faithful, honest approximation of the snapshot instant.
	info, err := os.Stat(backupPath)
	if err != nil {
		return fmt.Errorf("failed to stat backup file %q: %w", backupPath, err)
	}

	snapshotTime := info.ModTime()
	if snapshotTime.After(targetTime) {
		return fmt.Errorf("%w: backup %q was captured at %s, which is after the requested target time %s; "+
			"Redis recovery can only use whole snapshots taken at or before the target time",
			ErrPITRGranularity, backupPath,
			snapshotTime.UTC().Format(time.RFC3339), targetTime.UTC().Format(time.RFC3339))
	}

	// Restore the whole snapshot. This recovers state as of snapshotTime, which
	// is guaranteed to be at or before targetTime.
	restoreOpts := &database.RestoreOptions{
		BackupPath:   backupPath,
		DropExisting: true,
	}
	result := &database.RestoreResult{}

	if strings.HasSuffix(backupPath, ".aof") {
		return p.driver.restoreAOF(ctx, restoreOpts, result)
	}
	return p.driver.restoreRDB(ctx, restoreOpts, result)
}

// CreatePITRCheckpoint creates a consistent checkpoint for PITR
func (p *PITRManager) CreatePITRCheckpoint(ctx context.Context, outputDir string) (string, error) {
	// Trigger AOF rewrite to create a clean checkpoint
	if err := p.driver.client.BgRewriteAOF(ctx).Err(); err != nil {
		return "", fmt.Errorf("failed to trigger AOF rewrite: %w", err)
	}

	// Wait for rewrite to complete
	if err := p.driver.waitForAOFRewrite(ctx); err != nil {
		return "", err
	}

	// Get AOF file location
	configDir, err := p.driver.client.ConfigGet(ctx, "dir").Result()
	if err != nil {
		return "", fmt.Errorf("failed to get AOF directory: %w", err)
	}

	configAOFFilename, err := p.driver.client.ConfigGet(ctx, "appendfilename").Result()
	if err != nil {
		return "", fmt.Errorf("failed to get AOF filename: %w", err)
	}

	var aofDir, aofFilename string
	if len(configDir) >= 2 {
		aofDir = configDir[1].(string)
	}
	if len(configAOFFilename) >= 2 {
		aofFilename = configAOFFilename[1].(string)
	}

	aofPath := filepath.Join(aofDir, aofFilename)

	// Copy to checkpoint location
	checkpointPath := filepath.Join(outputDir, fmt.Sprintf("checkpoint_%s.aof", time.Now().Format("20060102_150405")))
	if err := copyFile(aofPath, checkpointPath); err != nil {
		return "", fmt.Errorf("failed to copy AOF checkpoint: %w", err)
	}

	return checkpointPath, nil
}

// GetRecoveryRange reports the continuous window over which time-based recovery
// is possible.
//
// Redis maintains no such continuous, timestamped window: RDB snapshots capture
// discrete moments and standard AOF files do not expose per-command timestamps
// that could bound a recovery interval. Rather than returning a fabricated range
// (the previous implementation hardcoded now .. now-24h), this returns an honest
// error so callers do not act on a range that does not exist. Recovery is
// snapshot-based; see RestoreToPIT.
func (p *PITRManager) GetRecoveryRange(_ context.Context) (start, end time.Time, err error) {
	return time.Time{}, time.Time{}, fmt.Errorf("%w: no continuous recovery range exists; recovery is limited to whole backup snapshots", ErrPITRNotSupported)
}
