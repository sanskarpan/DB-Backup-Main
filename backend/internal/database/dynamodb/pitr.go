package dynamodb

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/sanskarpan/db-backup/internal/database"
)

// PITRManager handles Point-in-Time Recovery for DynamoDB
type PITRManager struct {
	driver *DynamoDBDriver
}

// NewPITRManager creates a new PITR manager
func NewPITRManager(driver *DynamoDBDriver) *PITRManager {
	return &PITRManager{
		driver: driver,
	}
}

// RestoreToPIT restores a table to a specific point in time
func (p *PITRManager) RestoreToPIT(ctx context.Context, targetTime time.Time, opts *database.RestoreOptions) (*database.RestoreResult, error) {
	result := &database.RestoreResult{
		ID:        fmt.Sprintf("pitr_%s", time.Now().Format("20060102_150405")),
		StartTime: time.Now(),
		Status:    database.RestoreStatusInProgress,
	}

	// Get source table name from options
	sourceTable := opts.SourceDatabase
	if sourceTable == "" {
		result.Status = database.RestoreStatusFailed
		result.Error = fmt.Errorf("source table name is required")
		return result, result.Error
	}

	// Create target table name
	targetTable := fmt.Sprintf("%s_restored_%s", sourceTable, time.Now().Format("20060102_150405"))

	// Restore table to point in time
	_, err := p.driver.client.RestoreTableToPointInTime(ctx, &dynamodb.RestoreTableToPointInTimeInput{
		SourceTableName:       aws.String(sourceTable),
		TargetTableName:       aws.String(targetTable),
		RestoreDateTime:       aws.Time(targetTime),
		UseLatestRestorableTime: aws.Bool(false),
	})
	if err != nil {
		result.Status = database.RestoreStatusFailed
		result.Error = err
		result.EndTime = time.Now()
		return result, err
	}

	// Wait for table to be active
	waiter := dynamodb.NewTableExistsWaiter(p.driver.client)
	if err := waiter.Wait(ctx, &dynamodb.DescribeTableInput{
		TableName: aws.String(targetTable),
	}, 10*time.Minute); err != nil {
		result.Status = database.RestoreStatusFailed
		result.Error = err
		result.EndTime = time.Now()
		return result, err
	}

	result.Status = database.RestoreStatusCompleted
	result.EndTime = time.Now()
	result.Metadata = map[string]interface{}{
		"source_table": sourceTable,
		"target_table": targetTable,
		"restore_time": targetTime,
	}

	return result, nil
}

// GetRecoveryRange returns the available recovery time range for a table
func (p *PITRManager) GetRecoveryRange(ctx context.Context, tableName string) (start, end time.Time, err error) {
	output, err := p.driver.client.DescribeContinuousBackups(ctx, &dynamodb.DescribeContinuousBackupsInput{
		TableName: aws.String(tableName),
	})
	if err != nil {
		return time.Time{}, time.Time{}, err
	}

	pitr := output.ContinuousBackupsDescription.PointInTimeRecoveryDescription
	if pitr == nil || pitr.PointInTimeRecoveryStatus != types.PointInTimeRecoveryStatusEnabled {
		return time.Time{}, time.Time{}, fmt.Errorf("PITR is not enabled for table %s", tableName)
	}

	if pitr.EarliestRestorableDateTime != nil {
		start = *pitr.EarliestRestorableDateTime
	}

	if pitr.LatestRestorableDateTime != nil {
		end = *pitr.LatestRestorableDateTime
	}

	return start, end, nil
}

// IsPITREnabled checks if PITR is enabled for a table
func (p *PITRManager) IsPITREnabled(ctx context.Context, tableName string) (bool, error) {
	output, err := p.driver.client.DescribeContinuousBackups(ctx, &dynamodb.DescribeContinuousBackupsInput{
		TableName: aws.String(tableName),
	})
	if err != nil {
		return false, err
	}

	pitr := output.ContinuousBackupsDescription.PointInTimeRecoveryDescription
	if pitr == nil {
		return false, nil
	}

	return pitr.PointInTimeRecoveryStatus == types.PointInTimeRecoveryStatusEnabled, nil
}
