// Package dynamodb provides DynamoDB database driver implementation
package dynamodb

import (
	"context"
	"errors"
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

// dynamoDBAPI is the subset of the AWS DynamoDB client used by the driver.
// Declaring it as an interface lets unit tests substitute a mock client.
// The concrete *dynamodb.Client satisfies this interface.
type dynamoDBAPI interface {
	ListTables(ctx context.Context, params *dynamodb.ListTablesInput, optFns ...func(*dynamodb.Options)) (*dynamodb.ListTablesOutput, error)
	CreateBackup(ctx context.Context, params *dynamodb.CreateBackupInput, optFns ...func(*dynamodb.Options)) (*dynamodb.CreateBackupOutput, error)
	DescribeBackup(ctx context.Context, params *dynamodb.DescribeBackupInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DescribeBackupOutput, error)
	DescribeTable(ctx context.Context, params *dynamodb.DescribeTableInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DescribeTableOutput, error)
	RestoreTableFromBackup(ctx context.Context, params *dynamodb.RestoreTableFromBackupInput, optFns ...func(*dynamodb.Options)) (*dynamodb.RestoreTableFromBackupOutput, error)
	RestoreTableToPointInTime(ctx context.Context, params *dynamodb.RestoreTableToPointInTimeInput, optFns ...func(*dynamodb.Options)) (*dynamodb.RestoreTableToPointInTimeOutput, error)
	UpdateContinuousBackups(ctx context.Context, params *dynamodb.UpdateContinuousBackupsInput, optFns ...func(*dynamodb.Options)) (*dynamodb.UpdateContinuousBackupsOutput, error)
	DescribeContinuousBackups(ctx context.Context, params *dynamodb.DescribeContinuousBackupsInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DescribeContinuousBackupsOutput, error)
	ExportTableToPointInTime(ctx context.Context, params *dynamodb.ExportTableToPointInTimeInput, optFns ...func(*dynamodb.Options)) (*dynamodb.ExportTableToPointInTimeOutput, error)
}

// DynamoDBDriver implements the database.Driver interface for DynamoDB.
type DynamoDBDriver struct {
	client      dynamoDBAPI
	config      *database.ConnectionConfig
	pitrManager *PITRManager
	awsConfig   aws.Config

	// tableActivePollInterval controls how often the driver polls a table's
	// status while waiting for a restore to complete. Defaults to 5s when zero;
	// tests override it to keep runs fast.
	tableActivePollInterval time.Duration
}

func init() {
	database.RegisterDriver("dynamodb", func() database.Driver {
		return NewDynamoDBDriver()
	})
}

// NewDynamoDBDriver creates a new DynamoDB driver instance.
func NewDynamoDBDriver() *DynamoDBDriver {
	return &DynamoDBDriver{}
}

// Connect establishes a connection to DynamoDB.
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

// Disconnect closes the database connection.
func (d *DynamoDBDriver) Disconnect() error {
	// DynamoDB client doesn't need explicit disconnect
	return nil
}

// Ping tests the database connection.
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

// Backup creates a backup of the DynamoDB tables.
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

// createTableBackup creates a backup for a single table.
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

// waitForBackup waits for a backup to complete.
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

// getTablesToBackup returns the list of tables to backup.
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

// Restore restores DynamoDB tables from a backup.
//
// When opts.PointInTime is set, restore is delegated to the PITR manager.
// Otherwise each on-demand backup ARN is restored with RestoreTableFromBackup.
// DynamoDB always restores a backup into a NEW table, so the target table must
// not already exist; if it does, an honest error is returned. The method waits
// for every restored table to reach ACTIVE before reporting success and never
// reports success for work that was not actually performed.
func (d *DynamoDBDriver) Restore(ctx context.Context, opts *database.RestoreOptions) (*database.RestoreResult, error) {
	result := &database.RestoreResult{
		ID:        utils.GenerateRestoreID(),
		StartTime: time.Now(),
		Status:    database.RestoreStatusInProgress,
		Metadata:  map[string]interface{}{},
	}

	// Point-in-time recovery takes precedence when a target time is provided.
	if opts.PointInTime != nil {
		return d.pitrManager.RestoreToPIT(ctx, *opts.PointInTime, opts)
	}

	// Resolve the on-demand backup ARN(s) to restore from.
	backupARNs := resolveBackupARNs(opts)
	if len(backupARNs) == 0 {
		err := fmt.Errorf("no backup ARN provided: set RestoreOptions.SourceBackup or Metadata[\"backup_arns\"] to the on-demand backup ARN(s) to restore")
		result.Status = database.RestoreStatusFailed
		result.Error = err
		result.EndTime = time.Now()
		return result, pkgErrors.ErrDatabaseRestore(err)
	}

	for i, arn := range backupARNs {
		targetTable, err := d.resolveTargetTableName(ctx, opts, arn, i, len(backupARNs))
		if err != nil {
			result.Status = database.RestoreStatusFailed
			result.Error = err
			result.EndTime = time.Now()
			return result, pkgErrors.ErrDatabaseRestore(err)
		}

		if err := d.restoreTableFromBackup(ctx, arn, targetTable); err != nil {
			result.Status = database.RestoreStatusFailed
			result.Error = fmt.Errorf("failed to restore backup %q into table %q: %w", arn, targetTable, err)
			result.EndTime = time.Now()
			return result, pkgErrors.ErrDatabaseRestore(err)
		}

		result.RestoredTables = append(result.RestoredTables, targetTable)
	}

	result.Status = database.RestoreStatusCompleted
	result.EndTime = time.Now()
	result.Metadata["backup_arns"] = backupARNs
	result.Metadata["restored_tables"] = result.RestoredTables
	return result, nil
}

// resolveBackupARNs determines which on-demand backup ARNs to restore. An
// explicit SourceBackup takes precedence; otherwise the ARNs are read from
// Metadata["backup_arns"] (as written by Backup), tolerating both []string and
// the []interface{} form produced by JSON round-tripping.
func resolveBackupARNs(opts *database.RestoreOptions) []string {
	if opts.SourceBackup != "" {
		return []string{opts.SourceBackup}
	}

	raw, ok := opts.Metadata["backup_arns"]
	if !ok {
		return nil
	}

	switch v := raw.(type) {
	case []string:
		return v
	case []interface{}:
		arns := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok && s != "" {
				arns = append(arns, s)
			}
		}
		return arns
	default:
		return nil
	}
}

// resolveTargetTableName determines the name of the new table to restore into.
// For a single backup the caller may pin the name via opts.Database or a single
// opts.Tables entry; for multiple backups opts.Tables is matched positionally.
// When no name is supplied it is derived from the backup's source table name
// with a unique restore suffix.
func (d *DynamoDBDriver) resolveTargetTableName(ctx context.Context, opts *database.RestoreOptions, backupARN string, idx, total int) (string, error) {
	if total == 1 {
		if opts.Database != "" {
			return opts.Database, nil
		}
		if len(opts.Tables) == 1 {
			return opts.Tables[0], nil
		}
	} else if idx < len(opts.Tables) && opts.Tables[idx] != "" {
		return opts.Tables[idx], nil
	}

	output, err := d.client.DescribeBackup(ctx, &dynamodb.DescribeBackupInput{
		BackupArn: aws.String(backupARN),
	})
	if err != nil {
		return "", fmt.Errorf("describe backup %q: %w", backupARN, err)
	}

	var sourceTable string
	if output.BackupDescription != nil && output.BackupDescription.SourceTableDetails != nil &&
		output.BackupDescription.SourceTableDetails.TableName != nil {
		sourceTable = *output.BackupDescription.SourceTableDetails.TableName
	}
	if sourceTable == "" {
		return "", fmt.Errorf("unable to determine a target table name for backup %q: source table name missing from backup metadata", backupARN)
	}

	return fmt.Sprintf("%s_restored_%s", sourceTable, time.Now().Format("20060102_150405")), nil
}

// restoreTableFromBackup restores a single on-demand backup into a new table and
// waits for that table to become ACTIVE. RestoreTableFromBackup always creates a
// new table, so an existing target is reported as an honest error rather than
// silently overwritten.
func (d *DynamoDBDriver) restoreTableFromBackup(ctx context.Context, backupARN, targetTable string) error {
	// Ensure the target table does not already exist.
	_, err := d.client.DescribeTable(ctx, &dynamodb.DescribeTableInput{
		TableName: aws.String(targetTable),
	})
	if err == nil {
		return fmt.Errorf("target table %q already exists: RestoreTableFromBackup creates a new table and cannot overwrite an existing one", targetTable)
	}
	var notFound *types.ResourceNotFoundException
	if !errors.As(err, &notFound) {
		return fmt.Errorf("checking target table %q: %w", targetTable, err)
	}

	if _, err := d.client.RestoreTableFromBackup(ctx, &dynamodb.RestoreTableFromBackupInput{
		BackupArn:       aws.String(backupARN),
		TargetTableName: aws.String(targetTable),
	}); err != nil {
		return err
	}

	return d.waitForTableActive(ctx, targetTable)
}

// waitForTableActive polls the target table until it reports ACTIVE, surfacing
// the real CREATING -> ACTIVE lifecycle. Terminal non-active states are returned
// as errors and real SDK errors are surfaced unchanged.
func (d *DynamoDBDriver) waitForTableActive(ctx context.Context, tableName string) error {
	interval := d.tableActivePollInterval
	if interval <= 0 {
		interval = 5 * time.Second
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	timeout := time.After(30 * time.Minute)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeout:
			return fmt.Errorf("timed out waiting for table %q to become ACTIVE", tableName)
		case <-ticker.C:
			output, err := d.client.DescribeTable(ctx, &dynamodb.DescribeTableInput{
				TableName: aws.String(tableName),
			})
			if err != nil {
				// Immediately after RestoreTableFromBackup the new table may not
				// be visible yet; treat that as still being created.
				var notFound *types.ResourceNotFoundException
				if errors.As(err, &notFound) {
					continue
				}
				return err
			}

			if output.Table == nil {
				continue
			}

			switch output.Table.TableStatus {
			case types.TableStatusActive:
				return nil
			case types.TableStatusCreating, types.TableStatusUpdating:
				// Still provisioning; keep polling.
			default:
				return fmt.Errorf("table %q entered unexpected status %q during restore", tableName, output.Table.TableStatus)
			}
		}
	}
}

// GetDatabaseSize returns the total size of all DynamoDB tables.
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

// GetVersion returns the DynamoDB service version.
func (d *DynamoDBDriver) GetVersion(ctx context.Context) (string, error) {
	// DynamoDB is a managed service, return service name
	return "AWS DynamoDB (Managed Service)", nil
}

// EnablePITR enables Point-in-Time Recovery for a table.
func (d *DynamoDBDriver) EnablePITR(ctx context.Context, tableName string) error {
	_, err := d.client.UpdateContinuousBackups(ctx, &dynamodb.UpdateContinuousBackupsInput{
		TableName: aws.String(tableName),
		PointInTimeRecoverySpecification: &types.PointInTimeRecoverySpecification{
			PointInTimeRecoveryEnabled: aws.Bool(true),
		},
	})
	return err
}

// DisablePITR disables Point-in-Time Recovery for a table.
func (d *DynamoDBDriver) DisablePITR(ctx context.Context, tableName string) error {
	_, err := d.client.UpdateContinuousBackups(ctx, &dynamodb.UpdateContinuousBackupsInput{
		TableName: aws.String(tableName),
		PointInTimeRecoverySpecification: &types.PointInTimeRecoverySpecification{
			PointInTimeRecoveryEnabled: aws.Bool(false),
		},
	})
	return err
}

// ExportToS3 exports a table to S3.
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

// GetBackupSize estimates the size of a backup.
func (d *DynamoDBDriver) GetBackupSize(ctx context.Context, opts *database.BackupOptions) (int64, error) {
	return d.GetDatabaseSize(ctx)
}

// StreamBackup streams a backup to the provided writer.
func (d *DynamoDBDriver) StreamBackup(ctx context.Context, opts *database.BackupOptions, writer io.Writer) error {
	return fmt.Errorf("streaming backup not implemented for DynamoDB")
}

// StreamRestore restores from a reader.
func (d *DynamoDBDriver) StreamRestore(ctx context.Context, opts *database.RestoreOptions, reader io.Reader) error {
	return fmt.Errorf("streaming restore not implemented for DynamoDB")
}

// ValidateRestore validates that a restore can be performed.
func (d *DynamoDBDriver) ValidateRestore(ctx context.Context, opts *database.RestoreOptions) error {
	if opts.BackupPath == "" && opts.SourceBackup == "" {
		return fmt.Errorf("backup path is required")
	}
	return nil
}

// GetDatabases returns list of regions (DynamoDB doesn't have databases).
func (d *DynamoDBDriver) GetDatabases(ctx context.Context) ([]string, error) {
	// DynamoDB doesn't have databases, return the current region
	return []string{d.config.Host}, nil
}

// GetTables returns list of tables.
func (d *DynamoDBDriver) GetTables(ctx context.Context, databaseName string) ([]string, error) {
	opts := &database.BackupOptions{}
	return d.getTablesToBackup(ctx, opts)
}

// GetTableSize returns the size of a table.
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

// GetType returns the database type.
func (d *DynamoDBDriver) GetType() database.DatabaseType {
	return "dynamodb"
}

// SupportsIncremental returns whether incremental backups are supported.
func (d *DynamoDBDriver) SupportsIncremental() bool {
	return false // DynamoDB backups are always full
}

// SupportsPITR returns whether point-in-time recovery is supported.
func (d *DynamoDBDriver) SupportsPITR() bool {
	return true // DynamoDB supports PITR when enabled
}
