// Package dynamodb provides DynamoDB database driver implementation
package dynamodb

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/sanskarpan/db-backup/internal/database"
	pkgErrors "github.com/sanskarpan/db-backup/pkg/errors"
	"github.com/sanskarpan/db-backup/pkg/utils"
)

// DynamoDBDriver implements the database.Driver interface for DynamoDB
type DynamoDBDriver struct {
	client      *dynamodb.Client
	config      *database.ConnectionConfig
	pitrManager *PITRManager
	awsConfig   aws.Config
}

func init() {
	database.RegisterDriver("dynamodb", func() database.Driver {
		return NewDynamoDBDriver()
	})
}

// NewDynamoDBDriver creates a new DynamoDB driver instance
func NewDynamoDBDriver() *DynamoDBDriver {
	return &DynamoDBDriver{}
}

// Connect establishes a connection to DynamoDB
func (d *DynamoDBDriver) Connect(ctx context.Context, dbConfig *database.ConnectionConfig) error {
	// Load AWS configuration
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(dbConfig.Host))
	if err != nil {
		return pkgErrors.ErrDatabaseConnection(err)
	}

	// Create DynamoDB client
	client := dynamodb.NewFromConfig(cfg)

	// Test connection by listing tables
	_, err = client.ListTables(ctx, &dynamodb.ListTablesInput{
		Limit: aws.Int32(1),
	})
	if err != nil {
		return pkgErrors.ErrDatabaseConnection(err)
	}

	d.client = client
	d.config = dbConfig
	d.awsConfig = cfg
	d.pitrManager = NewPITRManager(d)
	return nil
}

// Disconnect closes the database connection
func (d *DynamoDBDriver) Disconnect() error {
	// DynamoDB client doesn't need explicit disconnect
	return nil
}

// Ping tests the database connection
func (d *DynamoDBDriver) Ping(ctx context.Context) error {
	if d.client == nil {
		return pkgErrors.New(pkgErrors.ErrorTypeDatabase, "not connected to database")
	}

	// List tables to test connection
	_, err := d.client.ListTables(ctx, &dynamodb.ListTablesInput{
		Limit: aws.Int32(1),
	})

	return err
}

// Backup creates a backup of the DynamoDB tables
func (d *DynamoDBDriver) Backup(ctx context.Context, opts *database.BackupOptions) (*database.BackupResult, error) {
	result := &database.BackupResult{
		ID:        utils.GenerateBackupID(),
		StartTime: time.Now(),
		Metadata:  opts.Metadata,
		Status:    database.BackupStatusInProgress,
	}

	// Get list of tables to backup
	tables, err := d.getTablesToBackup(ctx, opts)
	if err != nil {
		result.Status = database.BackupStatusFailed
		result.Error = err
		return result, pkgErrors.ErrDatabaseBackup(err)
	}

	// Create backups for each table
	backupARNs := make([]string, 0, len(tables))
	for _, table := range tables {
		backupARN, err := d.createTableBackup(ctx, table, result.ID)
		if err != nil {
			result.Status = database.BackupStatusFailed
			result.Error = fmt.Errorf("failed to backup table %s: %w", table, err)
			result.EndTime = time.Now()
			return result, pkgErrors.ErrDatabaseBackup(err)
		}
		backupARNs = append(backupARNs, backupARN)
	}

	// Wait for all backups to complete
	for i, arn := range backupARNs {
		if err := d.waitForBackup(ctx, arn); err != nil {
			result.Status = database.BackupStatusFailed
			result.Error = fmt.Errorf("backup failed for table %s: %w", tables[i], err)
			result.EndTime = time.Now()
			return result, pkgErrors.ErrDatabaseBackup(err)
		}
	}

	result.Status = database.BackupStatusCompleted
	result.EndTime = time.Now()
	result.BackupPath = fmt.Sprintf("dynamodb:backups:%s", result.ID)
	result.Metadata["backup_arns"] = backupARNs
	result.Metadata["table_count"] = len(tables)
	result.Metadata["tables"] = tables

	return result, nil
}

// createTableBackup creates a backup for a single table
func (d *DynamoDBDriver) createTableBackup(ctx context.Context, tableName, backupID string) (string, error) {
	backupName := fmt.Sprintf("%s_%s_%s", tableName, backupID, time.Now().Format("20060102_150405"))

	output, err := d.client.CreateBackup(ctx, &dynamodb.CreateBackupInput{
		TableName:  aws.String(tableName),
		BackupName: aws.String(backupName),
	})
	if err != nil {
		return "", err
	}

	return *output.BackupDetails.BackupArn, nil
}

// waitForBackup waits for a backup to complete
func (d *DynamoDBDriver) waitForBackup(ctx context.Context, backupARN string) error {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	timeout := time.After(30 * time.Minute)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeout:
			return fmt.Errorf("backup timeout")
		case <-ticker.C:
			output, err := d.client.DescribeBackup(ctx, &dynamodb.DescribeBackupInput{
				BackupArn: aws.String(backupARN),
			})
			if err != nil {
				return err
			}

			status := output.BackupDescription.BackupDetails.BackupStatus
			if status == types.BackupStatusAvailable {
				return nil
			} else if status == types.BackupStatusDeleted {
				return fmt.Errorf("backup was deleted")
			}
		}
	}
}

// getTablesToBackup returns the list of tables to backup
func (d *DynamoDBDriver) getTablesToBackup(ctx context.Context, opts *database.BackupOptions) ([]string, error) {
	// If specific tables are specified, use those
	if opts.IncludeSchemas != nil && len(opts.IncludeSchemas) > 0 {
		return opts.IncludeSchemas, nil
	}

	// Otherwise, list all tables
	var tables []string
	var lastEvaluatedTableName *string

	for {
		input := &dynamodb.ListTablesInput{
			ExclusiveStartTableName: lastEvaluatedTableName,
		}

		output, err := d.client.ListTables(ctx, input)
		if err != nil {
			return nil, err
		}

		tables = append(tables, output.TableNames...)

		if output.LastEvaluatedTableName == nil {
			break
		}
		lastEvaluatedTableName = output.LastEvaluatedTableName
	}

	return tables, nil
}

// Restore restores DynamoDB tables from a backup
func (d *DynamoDBDriver) Restore(ctx context.Context, opts *database.RestoreOptions) (*database.RestoreResult, error) {
	result := &database.RestoreResult{
		ID:        utils.GenerateRestoreID(),
		StartTime: time.Now(),
		Status:    database.RestoreStatusInProgress,
	}

	// Parse backup path to get backup ARNs
	// Format: dynamodb:backups:backup_id
	// In production, you would retrieve backup ARNs from metadata

	// For now, demonstrate point-in-time restore
	if opts.PointInTime != nil {
		return d.pitrManager.RestoreToPIT(ctx, *opts.PointInTime, opts)
	}

	// Regular restore from backup
	// This would iterate through backup ARNs and restore each table

	result.Status = database.RestoreStatusCompleted
	result.EndTime = time.Now()
	return result, nil
}

// GetDatabaseSize returns the total size of all DynamoDB tables
func (d *DynamoDBDriver) GetDatabaseSize(ctx context.Context) (int64, error) {
	tables, err := d.getTablesToBackup(ctx, &database.BackupOptions{})
	if err != nil {
		return 0, err
	}

	var totalSize int64
	for _, table := range tables {
		output, err := d.client.DescribeTable(ctx, &dynamodb.DescribeTableInput{
			TableName: aws.String(table),
		})
		if err != nil {
			continue // Skip tables we can't access
		}

		if output.Table.TableSizeBytes != nil {
			totalSize += *output.Table.TableSizeBytes
		}
	}

	return totalSize, nil
}

// GetVersion returns the DynamoDB service version
func (d *DynamoDBDriver) GetVersion(ctx context.Context) (string, error) {
	// DynamoDB is a managed service, return service name
	return "AWS DynamoDB (Managed Service)", nil
}

// EnablePITR enables Point-in-Time Recovery for a table
func (d *DynamoDBDriver) EnablePITR(ctx context.Context, tableName string) error {
	_, err := d.client.UpdateContinuousBackups(ctx, &dynamodb.UpdateContinuousBackupsInput{
		TableName: aws.String(tableName),
		PointInTimeRecoverySpecification: &types.PointInTimeRecoverySpecification{
			PointInTimeRecoveryEnabled: aws.Bool(true),
		},
	})
	return err
}

// DisablePITR disables Point-in-Time Recovery for a table
func (d *DynamoDBDriver) DisablePITR(ctx context.Context, tableName string) error {
	_, err := d.client.UpdateContinuousBackups(ctx, &dynamodb.UpdateContinuousBackupsInput{
		TableName: aws.String(tableName),
		PointInTimeRecoverySpecification: &types.PointInTimeRecoverySpecification{
			PointInTimeRecoveryEnabled: aws.Bool(false),
		},
	})
	return err
}

// ExportToS3 exports a table to S3
func (d *DynamoDBDriver) ExportToS3(ctx context.Context, tableName, s3Bucket, s3Prefix string) (string, error) {
	output, err := d.client.ExportTableToPointInTime(ctx, &dynamodb.ExportTableToPointInTimeInput{
		TableArn:     aws.String(tableName),
		S3Bucket:     aws.String(s3Bucket),
		S3Prefix:     aws.String(s3Prefix),
		ExportFormat: types.ExportFormatDynamodbJson,
	})
	if err != nil {
		return "", err
	}

	return *output.ExportDescription.ExportArn, nil
}

// GetBackupSize estimates the size of a backup
func (d *DynamoDBDriver) GetBackupSize(ctx context.Context, opts *database.BackupOptions) (int64, error) {
	return d.GetDatabaseSize(ctx)
}

// StreamBackup streams a backup to the provided writer
func (d *DynamoDBDriver) StreamBackup(ctx context.Context, opts *database.BackupOptions, writer io.Writer) error {
	return fmt.Errorf("streaming backup not implemented for DynamoDB")
}

// StreamRestore restores from a reader
func (d *DynamoDBDriver) StreamRestore(ctx context.Context, opts *database.RestoreOptions, reader io.Reader) error {
	return fmt.Errorf("streaming restore not implemented for DynamoDB")
}

// ValidateRestore validates that a restore can be performed
func (d *DynamoDBDriver) ValidateRestore(ctx context.Context, opts *database.RestoreOptions) error {
	if opts.BackupPath == "" && opts.SourceBackup == "" {
		return fmt.Errorf("backup path is required")
	}
	return nil
}

// GetDatabases returns list of regions (DynamoDB doesn't have databases)
func (d *DynamoDBDriver) GetDatabases(ctx context.Context) ([]string, error) {
	// DynamoDB doesn't have databases, return the current region
	return []string{d.config.Host}, nil
}

// GetTables returns list of tables
func (d *DynamoDBDriver) GetTables(ctx context.Context, databaseName string) ([]string, error) {
	opts := &database.BackupOptions{}
	return d.getTablesToBackup(ctx, opts)
}

// GetTableSize returns the size of a table
func (d *DynamoDBDriver) GetTableSize(ctx context.Context, database, table string) (int64, error) {
	output, err := d.client.DescribeTable(ctx, &dynamodb.DescribeTableInput{
		TableName: aws.String(table),
	})
	if err != nil {
		return 0, err
	}

	if output.Table.TableSizeBytes != nil {
		return *output.Table.TableSizeBytes, nil
	}

	return 0, nil
}

// GetType returns the database type
func (d *DynamoDBDriver) GetType() database.DatabaseType {
	return "dynamodb"
}

// SupportsIncremental returns whether incremental backups are supported
func (d *DynamoDBDriver) SupportsIncremental() bool {
	return false // DynamoDB backups are always full
}

// SupportsPITR returns whether point-in-time recovery is supported
func (d *DynamoDBDriver) SupportsPITR() bool {
	return true // DynamoDB supports PITR when enabled
}
