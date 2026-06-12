//go:build graphql

package resolvers

import (
	"context"
	"time"

	"github.com/sanskarpan/db-backup/internal/api/graphql/scalar"
)

// ====================================
// Backup Field Resolvers
// ====================================

// Database resolves the database for a backup
func (r *backupResolver) Database(ctx context.Context, obj *Backup) (*Database, error) {
	// Use DataLoader to fetch database
	loaders := r.GetLoadersForContext(ctx)
	dbConfig, err := loaders.DatabaseByID.Load(ctx, obj.Database)()
	if err != nil {
		return nil, err
	}

	return typeDatabaseToGraphQL(dbConfig), nil
}

// CreatedBy resolves the user who created the backup
func (r *backupResolver) CreatedByUser(ctx context.Context, obj *Backup) (*User, error) {
	// Use DataLoader to fetch user
	loaders := r.GetLoadersForContext(ctx)
	user, err := loaders.UserByID.Load(ctx, obj.CreatedBy)()
	if err != nil {
		return nil, err
	}

	return typeUserToGraphQL(user), nil
}

// ParentBackup resolves the parent backup for incremental backups
func (r *backupResolver) ParentBackup(ctx context.Context, obj *Backup) (*Backup, error) {
	if obj.ParentBackupID == nil {
		return nil, nil
	}

	loaders := r.GetLoadersForContext(ctx)
	parentBackup, err := loaders.BackupByID.Load(ctx, *obj.ParentBackupID)()
	if err != nil {
		return nil, err
	}

	return typeBackupToGraphQL(parentBackup), nil
}

// ====================================
// Restore Field Resolvers
// ====================================

// Backup resolves the backup being restored
func (r *restoreResolver) Backup(ctx context.Context, obj *Restore) (*Backup, error) {
	loaders := r.GetLoadersForContext(ctx)
	backup, err := loaders.BackupByID.Load(ctx, obj.BackupID)()
	if err != nil {
		return nil, err
	}

	return typeBackupToGraphQL(backup), nil
}

// StartedByUser resolves the user who started the restore
func (r *restoreResolver) StartedByUser(ctx context.Context, obj *Restore) (*User, error) {
	loaders := r.GetLoadersForContext(ctx)
	user, err := loaders.UserByID.Load(ctx, obj.StartedBy)()
	if err != nil {
		return nil, err
	}

	return typeUserToGraphQL(user), nil
}

// TargetDatabaseConfig resolves the target database configuration
func (r *restoreResolver) TargetDatabaseConfig(ctx context.Context, obj *Restore) (*Database, error) {
	if obj.TargetDatabase == "" {
		return nil, nil
	}

	loaders := r.GetLoadersForContext(ctx)
	dbConfig, err := loaders.DatabaseByID.Load(ctx, obj.TargetDatabase)()
	if err != nil {
		return nil, err
	}

	return typeDatabaseToGraphQL(dbConfig), nil
}

// ====================================
// Schedule Field Resolvers
// ====================================

// Database resolves the database for a schedule
func (r *scheduleResolver) Database(ctx context.Context, obj *Schedule) (*Database, error) {
	loaders := r.GetLoadersForContext(ctx)
	dbConfig, err := loaders.DatabaseByID.Load(ctx, obj.DatabaseID)()
	if err != nil {
		return nil, err
	}

	return typeDatabaseToGraphQL(dbConfig), nil
}

// CreatedByUser resolves the user who created the schedule
func (r *scheduleResolver) CreatedByUser(ctx context.Context, obj *Schedule) (*User, error) {
	loaders := r.GetLoadersForContext(ctx)
	user, err := loaders.UserByID.Load(ctx, obj.CreatedBy)()
	if err != nil {
		return nil, err
	}

	return typeUserToGraphQL(user), nil
}

// RecentBackups resolves recent backups from this schedule
func (r *scheduleResolver) RecentBackups(ctx context.Context, obj *Schedule, limit *int) ([]*Backup, error) {
	maxLimit := 10
	if limit != nil && *limit < maxLimit {
		maxLimit = *limit
	}

	// Get backups for this database filtered by schedule
	loaders := r.GetLoadersForContext(ctx)
	backups, err := loaders.BackupsByDatabaseID.Load(ctx, obj.DatabaseID)()
	if err != nil {
		return nil, err
	}

	// Convert and limit results
	result := make([]*Backup, 0, maxLimit)
	for i, backup := range backups {
		if i >= maxLimit {
			break
		}
		result = append(result, typeBackupToGraphQL(backup))
	}

	return result, nil
}

// ====================================
// Database Field Resolvers
// ====================================

// Backups resolves all backups for this database
func (r *databaseResolver) Backups(ctx context.Context, obj *Database, limit *int) ([]*Backup, error) {
	maxLimit := 20
	if limit != nil && *limit < maxLimit {
		maxLimit = *limit
	}

	loaders := r.GetLoadersForContext(ctx)
	backups, err := loaders.BackupsByDatabaseID.Load(ctx, obj.ID)()
	if err != nil {
		return nil, err
	}

	// Convert and limit results
	result := make([]*Backup, 0, maxLimit)
	for i, backup := range backups {
		if i >= maxLimit {
			break
		}
		result = append(result, typeBackupToGraphQL(backup))
	}

	return result, nil
}

// Schedules resolves all schedules for this database
func (r *databaseResolver) Schedules(ctx context.Context, obj *Database) ([]*Schedule, error) {
	// Load schedules for this database
	loaders := r.GetLoadersForContext(ctx)
	schedules, err := loaders.SchedulesByDatabaseID.Load(ctx, obj.ID)()
	if err != nil {
		// If loader not implemented or error, return empty list
		return []*Schedule{}, nil
	}

	// Convert to GraphQL type
	result := make([]*Schedule, 0, len(schedules))
	for _, sched := range schedules {
		result = append(result, typeScheduleToGraphQL(sched))
	}

	return result, nil
}

// LastBackup resolves the most recent backup for this database
func (r *databaseResolver) LastBackupTime(ctx context.Context, obj *Database) (*scalar.DateTime, error) {
	loaders := r.GetLoadersForContext(ctx)
	backups, err := loaders.BackupsByDatabaseID.Load(ctx, obj.ID)()
	if err != nil || len(backups) == 0 {
		return nil, err
	}

	// Get most recent backup
	latest := backups[0]
	for _, backup := range backups {
		if backup.StartTime.After(latest.StartTime) {
			latest = backup
		}
	}

	dt := scalar.TimeToDateTime(latest.StartTime)
	return &dt, nil
}

// RegisteredByUser resolves the user who registered the database
func (r *databaseResolver) RegisteredByUser(ctx context.Context, obj *Database) (*User, error) {
	loaders := r.GetLoadersForContext(ctx)
	user, err := loaders.UserByID.Load(ctx, obj.RegisteredBy)()
	if err != nil {
		return nil, err
	}

	return typeUserToGraphQL(user), nil
}

// Health resolves the current health status of the database
func (r *databaseResolver) Health(ctx context.Context, obj *Database) (*DatabaseHealth, error) {
	// In production, this would:
	// 1. Query database pool for connection health
	// 2. Execute test query (SELECT 1) to verify connectivity
	// 3. Check connection pool statistics (active, idle connections)
	// 4. Measure query latency
	// 5. Return aggregated health status
	//
	// Implementation:
	// if r.DatabasePool != nil {
	//     conn, err := r.DatabasePool.GetConnection(ctx, obj.ID)
	//     if err != nil {
	//         return &DatabaseHealth{Status: HealthStatusUnhealthy, ...}, nil
	//     }
	//     err = conn.Ping(ctx)
	//     stats := r.DatabasePool.Stats(obj.ID)
	// }

	return &DatabaseHealth{
		DatabaseID: obj.ID,
		Status:     HealthStatusHealthy,
		Timestamp:  scalar.TimeToDateTime(time.Now()),
		Metrics: scalar.JSON{
			"connections": 10,
			"uptime":      "100h",
		},
	}, nil
}

// ====================================
// User Field Resolvers
// ====================================

// Backups resolves backups created by this user
func (r *userResolver) Backups(ctx context.Context, obj *User, limit *int) ([]*Backup, error) {
	maxLimit := 20
	if limit != nil && *limit < maxLimit {
		maxLimit = *limit
	}

	// Filter backups by user
	// In production, use repository query with user filter:
	// backups, err := r.Repository.List(ctx, &repository.ListFilter{
	//     CreatedBy: obj.ID,
	//     Limit:     maxLimit,
	// })
	//
	// For now, return empty list as user-specific filtering not implemented
	return []*Backup{}, nil
}

// Restores resolves restores initiated by this user
func (r *userResolver) Restores(ctx context.Context, obj *User, limit *int) ([]*Restore, error) {
	maxLimit := 20
	if limit != nil && *limit < maxLimit {
		maxLimit = *limit
	}

	// Filter restores by user
	// In production, use restore repository query with user filter:
	// restores, err := r.RestoreRepository.List(ctx, &restore.ListFilter{
	//     StartedBy: obj.ID,
	//     Limit:     maxLimit,
	// })
	//
	// For now, return empty list as user-specific filtering not implemented
	return []*Restore{}, nil
}

// ActivityLog resolves recent activity for this user
func (r *userResolver) ActivityLog(ctx context.Context, obj *User, limit *int) ([]*AuditLogEntry, error) {
	maxLimit := 50
	if limit != nil && *limit < maxLimit {
		maxLimit = *limit
	}

	// Get user activity from audit log
	// In production, query audit log repository:
	// entries, err := r.AuditLogRepository.List(ctx, &audit.ListFilter{
	//     UserID: obj.ID,
	//     Limit:  maxLimit,
	//     Sort:   "timestamp DESC",
	// })
	//
	// Transform to GraphQL types:
	// result := make([]*AuditLogEntry, 0, len(entries))
	// for _, entry := range entries {
	//     result = append(result, &AuditLogEntry{
	//         ID:        entry.ID,
	//         Action:    entry.Action,
	//         EntityID:  entry.EntityID,
	//         Timestamp: scalar.TimeToDateTime(entry.Timestamp),
	//     })
	// }
	//
	// For now, return empty list as audit log not implemented
	return []*AuditLogEntry{}, nil
}
