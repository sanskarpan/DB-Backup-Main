//go:build graphql

package loader

import (
	"context"
	"time"

	"github.com/graph-gophers/dataloader/v7"
	"github.com/sanskarpan/db-backup/internal/database"
	"github.com/sanskarpan/db-backup/internal/models"
	"github.com/sanskarpan/db-backup/internal/repository"
)

// Context key for loaders
type ctxKey string

const LoaderKey = ctxKey("dataloaders")

// Loaders contains all DataLoaders for the application
type Loaders struct {
	BackupByID          *dataloader.Loader[string, *models.BackupMetadata]
	BackupsByDatabaseID *dataloader.Loader[string, []*models.BackupMetadata]
	DatabaseByID        *dataloader.Loader[string, *database.ConnectionConfig]
	UserByID            *dataloader.Loader[string, *User]
	RestoreByID         *dataloader.Loader[string, *RestoreResult]
	ScheduleByID        *dataloader.Loader[string, *Schedule]
}

// NewLoaders creates all DataLoaders
func NewLoaders(repo repository.Repository, dbPool interface{}) *Loaders {
	// Configure dataloader options
	batchCapacity := 100
	wait := 10 * time.Millisecond

	return &Loaders{
		BackupByID: dataloader.NewBatchedLoader(
			func(ctx context.Context, ids []string) []*dataloader.Result[*models.BackupMetadata] {
				return batchGetBackups(ctx, repo, ids)
			},
			dataloader.WithBatchCapacity[string, *models.BackupMetadata](batchCapacity),
			dataloader.WithWait[string, *models.BackupMetadata](wait),
		),

		BackupsByDatabaseID: dataloader.NewBatchedLoader(
			func(ctx context.Context, dbIDs []string) []*dataloader.Result[[]*models.BackupMetadata] {
				return batchGetBackupsByDatabase(ctx, repo, dbIDs)
			},
			dataloader.WithBatchCapacity[string, []*models.BackupMetadata](batchCapacity),
			dataloader.WithWait[string, []*models.BackupMetadata](wait),
		),

		DatabaseByID: dataloader.NewBatchedLoader(
			func(ctx context.Context, ids []string) []*dataloader.Result[*database.ConnectionConfig] {
				return batchGetDatabases(ctx, dbPool, ids)
			},
			dataloader.WithBatchCapacity[string, *database.ConnectionConfig](batchCapacity),
			dataloader.WithWait[string, *database.ConnectionConfig](wait),
		),

		UserByID: dataloader.NewBatchedLoader(
			func(ctx context.Context, ids []string) []*dataloader.Result[*User] {
				return batchGetUsers(ctx, ids)
			},
			dataloader.WithBatchCapacity[string, *User](batchCapacity),
			dataloader.WithWait[string, *User](wait),
		),

		RestoreByID: dataloader.NewBatchedLoader(
			func(ctx context.Context, ids []string) []*dataloader.Result[*RestoreResult] {
				return batchGetRestores(ctx, repo, ids)
			},
			dataloader.WithBatchCapacity[string, *RestoreResult](batchCapacity),
			dataloader.WithWait[string, *RestoreResult](wait),
		),

		ScheduleByID: dataloader.NewBatchedLoader(
			func(ctx context.Context, ids []string) []*dataloader.Result[*Schedule] {
				return batchGetSchedules(ctx, ids)
			},
			dataloader.WithBatchCapacity[string, *Schedule](batchCapacity),
			dataloader.WithWait[string, *Schedule](wait),
		),
	}
}

// Batch fetch functions
func batchGetBackups(ctx context.Context, repo repository.Repository, ids []string) []*dataloader.Result[*models.BackupMetadata] {
	results := make([]*dataloader.Result[*models.BackupMetadata], len(ids))

	// Create a map to track which IDs we need to fetch
	backupMap := make(map[string]*models.BackupMetadata)

	// Fetch backups by ID individually (in production, repository should support batch fetch by IDs)
	// For now, we fetch all and filter, or use Get for each ID
	for _, id := range ids {
		backup, err := repo.Get(ctx, id)
		if err != nil {
			// Store error but continue fetching others
			backupMap[id] = nil
		} else {
			backupMap[id] = backup
		}
	}

	// Match backups to requested IDs in order
	for i, id := range ids {
		if backup, ok := backupMap[id]; ok && backup != nil {
			results[i] = &dataloader.Result[*models.BackupMetadata]{Data: backup}
		} else {
			results[i] = &dataloader.Result[*models.BackupMetadata]{Error: ErrNotFound}
		}
	}

	return results
}

func batchGetBackupsByDatabase(ctx context.Context, repo repository.Repository, dbIDs []string) []*dataloader.Result[[]*models.BackupMetadata] {
	results := make([]*dataloader.Result[[]*models.BackupMetadata], len(dbIDs))

	// Fetch all backups for these databases
	allBackups, err := repo.List(ctx, nil)
	if err != nil {
		for i := range results {
			results[i] = &dataloader.Result[[]*models.BackupMetadata]{Error: err}
		}
		return results
	}

	// Group backups by database ID
	backupsByDB := make(map[string][]*models.BackupMetadata)
	for _, backup := range allBackups {
		backupsByDB[backup.Database] = append(backupsByDB[backup.Database], backup)
	}

	// Match to requested database IDs
	for i, dbID := range dbIDs {
		backups := backupsByDB[dbID]
		if backups == nil {
			backups = []*models.BackupMetadata{}
		}
		results[i] = &dataloader.Result[[]*models.BackupMetadata]{Data: backups}
	}

	return results
}

func batchGetDatabases(ctx context.Context, dbPool interface{}, ids []string) []*dataloader.Result[*database.ConnectionConfig] {
	results := make([]*dataloader.Result[*database.ConnectionConfig], len(ids))

	// In production, this would fetch database configurations from database pool or config repository
	// For now, return placeholder configs with IDs
	for i := range ids {
		results[i] = &dataloader.Result[*database.ConnectionConfig]{
			Data: &database.ConnectionConfig{
				Type: "postgres", // Placeholder
				// Note: ID would be stored separately or in a wrapper struct
			},
		}
	}

	return results
}

func batchGetUsers(ctx context.Context, ids []string) []*dataloader.Result[*User] {
	results := make([]*dataloader.Result[*User], len(ids))

	// In production, this would query user repository or auth service
	// Batch fetch users by IDs from database
	// For now, return placeholder users with IDs
	for i, id := range ids {
		results[i] = &dataloader.Result[*User]{
			Data: &User{
				ID:    id,
				Email: "user@example.com", // Placeholder
				Name:  "User",             // Placeholder
			},
		}
	}

	return results
}

func batchGetRestores(ctx context.Context, repo repository.Repository, ids []string) []*dataloader.Result[*RestoreResult] {
	results := make([]*dataloader.Result[*RestoreResult], len(ids))

	// In production, this would query restore repository or restore service
	// Batch fetch restore operations by IDs
	// For now, return placeholder restore results with IDs
	for i, id := range ids {
		results[i] = &dataloader.Result[*RestoreResult]{
			Data: &RestoreResult{
				ID:       id,
				Database: "", // Placeholder
			},
		}
	}

	return results
}

func batchGetSchedules(ctx context.Context, ids []string) []*dataloader.Result[*Schedule] {
	results := make([]*dataloader.Result[*Schedule], len(ids))

	// In production, this would query scheduler service or schedule repository
	// Batch fetch schedules by IDs
	// For now, return placeholder schedules with IDs
	for i, id := range ids {
		results[i] = &dataloader.Result[*Schedule]{
			Data: &Schedule{
				ID:   id,
				Name: "Schedule", // Placeholder
			},
		}
	}

	return results
}

// Placeholder types
type User struct {
	ID    string
	Email string
	Name  string
}

type Schedule struct {
	ID   string
	Name string
}

type RestoreResult struct {
	ID       string
	Database string
}

// Errors
var (
	ErrNotFound = &NotFoundError{}
)

type NotFoundError struct{}

func (e *NotFoundError) Error() string {
	return "not found"
}
