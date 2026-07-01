package dynamodb

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/sanskarpan/db-backup/internal/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockDynamoAPI is a configurable mock implementation of dynamoDBAPI used by the
// non-PITR restore unit tests. Only the methods exercised by restore have
// behavior hooks; the remaining methods satisfy the interface as no-ops.
type mockDynamoAPI struct {
	describeTableFunc          func(ctx context.Context, params *dynamodb.DescribeTableInput) (*dynamodb.DescribeTableOutput, error)
	describeBackupFunc         func(ctx context.Context, params *dynamodb.DescribeBackupInput) (*dynamodb.DescribeBackupOutput, error)
	restoreTableFromBackupFunc func(ctx context.Context, params *dynamodb.RestoreTableFromBackupInput) (*dynamodb.RestoreTableFromBackupOutput, error)

	restoreCalls int
}

func (m *mockDynamoAPI) DescribeTable(ctx context.Context, params *dynamodb.DescribeTableInput, _ ...func(*dynamodb.Options)) (*dynamodb.DescribeTableOutput, error) {
	if m.describeTableFunc != nil {
		return m.describeTableFunc(ctx, params)
	}
	return nil, &types.ResourceNotFoundException{Message: aws.String("not found")}
}

func (m *mockDynamoAPI) DescribeBackup(ctx context.Context, params *dynamodb.DescribeBackupInput, _ ...func(*dynamodb.Options)) (*dynamodb.DescribeBackupOutput, error) {
	if m.describeBackupFunc != nil {
		return m.describeBackupFunc(ctx, params)
	}
	return nil, errors.New("DescribeBackup not configured")
}

func (m *mockDynamoAPI) RestoreTableFromBackup(ctx context.Context, params *dynamodb.RestoreTableFromBackupInput, _ ...func(*dynamodb.Options)) (*dynamodb.RestoreTableFromBackupOutput, error) {
	m.restoreCalls++
	if m.restoreTableFromBackupFunc != nil {
		return m.restoreTableFromBackupFunc(ctx, params)
	}
	return &dynamodb.RestoreTableFromBackupOutput{}, nil
}

// Unused-by-restore interface methods.
func (m *mockDynamoAPI) ListTables(context.Context, *dynamodb.ListTablesInput, ...func(*dynamodb.Options)) (*dynamodb.ListTablesOutput, error) {
	return &dynamodb.ListTablesOutput{}, nil
}

func (m *mockDynamoAPI) CreateBackup(context.Context, *dynamodb.CreateBackupInput, ...func(*dynamodb.Options)) (*dynamodb.CreateBackupOutput, error) {
	return &dynamodb.CreateBackupOutput{}, nil
}

func (m *mockDynamoAPI) RestoreTableToPointInTime(context.Context, *dynamodb.RestoreTableToPointInTimeInput, ...func(*dynamodb.Options)) (*dynamodb.RestoreTableToPointInTimeOutput, error) {
	return &dynamodb.RestoreTableToPointInTimeOutput{}, nil
}

func (m *mockDynamoAPI) UpdateContinuousBackups(context.Context, *dynamodb.UpdateContinuousBackupsInput, ...func(*dynamodb.Options)) (*dynamodb.UpdateContinuousBackupsOutput, error) {
	return &dynamodb.UpdateContinuousBackupsOutput{}, nil
}

func (m *mockDynamoAPI) DescribeContinuousBackups(context.Context, *dynamodb.DescribeContinuousBackupsInput, ...func(*dynamodb.Options)) (*dynamodb.DescribeContinuousBackupsOutput, error) {
	return &dynamodb.DescribeContinuousBackupsOutput{}, nil
}

func (m *mockDynamoAPI) ExportTableToPointInTime(context.Context, *dynamodb.ExportTableToPointInTimeInput, ...func(*dynamodb.Options)) (*dynamodb.ExportTableToPointInTimeOutput, error) {
	return &dynamodb.ExportTableToPointInTimeOutput{}, nil
}

// notFoundErr is a reusable ResourceNotFoundException for the existence check.
func notFoundErr() error {
	return &types.ResourceNotFoundException{Message: aws.String("not found")}
}

// describeTableActive returns DescribeTable output reporting an ACTIVE table.
func describeTableActive(name string) *dynamodb.DescribeTableOutput {
	return &dynamodb.DescribeTableOutput{
		Table: &types.TableDescription{
			TableName:   aws.String(name),
			TableStatus: types.TableStatusActive,
		},
	}
}

// newActiveAfterCreatingMock returns a mock whose DescribeTable first reports
// the target absent (existence pre-check), then CREATING, then ACTIVE, so a
// restore into a pinned target must poll until the table becomes ACTIVE.
func newActiveAfterCreatingMock() *mockDynamoAPI {
	calls := 0
	return &mockDynamoAPI{
		describeTableFunc: func(_ context.Context, _ *dynamodb.DescribeTableInput) (*dynamodb.DescribeTableOutput, error) {
			calls++
			switch calls {
			case 1:
				// existence pre-check: target does not exist yet.
				return nil, notFoundErr()
			case 2:
				// first poll: still creating.
				return &dynamodb.DescribeTableOutput{
					Table: &types.TableDescription{TableStatus: types.TableStatusCreating},
				}, nil
			default:
				return describeTableActive("orders_restored"), nil
			}
		},
	}
}

// newDerivedTargetMock returns a mock for restoring into a name derived from
// the backup's source table: the derived target is absent then ACTIVE, and
// DescribeBackup reports the source table name.
func newDerivedTargetMock() *mockDynamoAPI {
	calls := 0
	return &mockDynamoAPI{
		describeTableFunc: func(_ context.Context, _ *dynamodb.DescribeTableInput) (*dynamodb.DescribeTableOutput, error) {
			calls++
			if calls == 1 {
				// existence pre-check on the derived name: not present.
				return nil, notFoundErr()
			}
			return describeTableActive("orders_restored"), nil
		},
		describeBackupFunc: func(_ context.Context, _ *dynamodb.DescribeBackupInput) (*dynamodb.DescribeBackupOutput, error) {
			return &dynamodb.DescribeBackupOutput{
				BackupDescription: &types.BackupDescription{
					SourceTableDetails: &types.SourceTableDetails{
						TableName: aws.String("orders"),
					},
				},
			}, nil
		},
	}
}

func TestDynamoDBDriver_Restore_NonPITR(t *testing.T) {
	const backupARN = "arn:aws:dynamodb:us-east-1:123456789012:table/orders/backup/01234567890123-abcd1234"

	tests := []struct {
		opts           *database.RestoreOptions
		mock           *mockDynamoAPI
		name           string
		wantStatus     database.RestoreStatus
		wantErrSubstr  string
		wantRestored   []string
		wantErr        bool
		wantRestoreRun bool
	}{
		{
			name:          "no backup arn provided is an honest failure",
			opts:          &database.RestoreOptions{},
			mock:          &mockDynamoAPI{},
			wantStatus:    database.RestoreStatusFailed,
			wantErr:       true,
			wantErrSubstr: "no backup ARN",
		},
		{
			name: "target table already exists returns honest error",
			opts: &database.RestoreOptions{SourceBackup: backupARN, Database: "orders_restored"},
			mock: &mockDynamoAPI{
				describeTableFunc: func(_ context.Context, _ *dynamodb.DescribeTableInput) (*dynamodb.DescribeTableOutput, error) {
					return describeTableActive("orders_restored"), nil
				},
			},
			wantStatus:    database.RestoreStatusFailed,
			wantErr:       true,
			wantErrSubstr: "already exists",
		},
		{
			name:           "successful restore into pinned target waits for ACTIVE",
			opts:           &database.RestoreOptions{SourceBackup: backupARN, Database: "orders_restored"},
			mock:           newActiveAfterCreatingMock(),
			wantStatus:     database.RestoreStatusCompleted,
			wantRestored:   []string{"orders_restored"},
			wantRestoreRun: true,
		},
		{
			name: "SDK error from RestoreTableFromBackup is surfaced",
			opts: &database.RestoreOptions{SourceBackup: backupARN, Database: "orders_restored"},
			mock: &mockDynamoAPI{
				describeTableFunc: func(_ context.Context, _ *dynamodb.DescribeTableInput) (*dynamodb.DescribeTableOutput, error) {
					return nil, notFoundErr()
				},
				restoreTableFromBackupFunc: func(_ context.Context, _ *dynamodb.RestoreTableFromBackupInput) (*dynamodb.RestoreTableFromBackupOutput, error) {
					return nil, errors.New("BackupNotFoundException: backup is gone")
				},
			},
			wantStatus:     database.RestoreStatusFailed,
			wantErr:        true,
			wantErrSubstr:  "backup is gone",
			wantRestoreRun: true,
		},
		{
			name:           "target derived from backup source table when unspecified",
			opts:           &database.RestoreOptions{SourceBackup: backupARN},
			mock:           newDerivedTargetMock(),
			wantStatus:     database.RestoreStatusCompleted,
			wantRestoreRun: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			driver := &DynamoDBDriver{
				client:                  tt.mock,
				tableActivePollInterval: time.Millisecond,
			}

			result, err := driver.Restore(context.Background(), tt.opts)
			require.NotNil(t, result)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.wantErrSubstr != "" {
					assert.Contains(t, err.Error(), tt.wantErrSubstr)
					assert.Contains(t, result.Error.Error(), tt.wantErrSubstr)
				}
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.wantStatus, result.Status)
			assert.False(t, result.EndTime.IsZero(), "EndTime should be set")

			if tt.wantRestored != nil {
				assert.Equal(t, tt.wantRestored, result.RestoredTables)
			}
			if tt.wantRestoreRun {
				assert.Positive(t, tt.mock.restoreCalls, "RestoreTableFromBackup should have been invoked")
			} else if tt.name == "no backup arn provided is an honest failure" || tt.name == "target table already exists returns honest error" {
				assert.Zero(t, tt.mock.restoreCalls, "RestoreTableFromBackup must not run")
			}
		})
	}
}

func TestResolveBackupARNs(t *testing.T) {
	tests := []struct {
		name string
		opts *database.RestoreOptions
		want []string
	}{
		{
			name: "source backup takes precedence",
			opts: &database.RestoreOptions{
				SourceBackup: "arn:source",
				Metadata:     map[string]interface{}{"backup_arns": []string{"arn:meta"}},
			},
			want: []string{"arn:source"},
		},
		{
			name: "string slice from metadata",
			opts: &database.RestoreOptions{
				Metadata: map[string]interface{}{"backup_arns": []string{"arn:a", "arn:b"}},
			},
			want: []string{"arn:a", "arn:b"},
		},
		{
			name: "interface slice from JSON round-trip",
			opts: &database.RestoreOptions{
				Metadata: map[string]interface{}{"backup_arns": []interface{}{"arn:a", "", "arn:b"}},
			},
			want: []string{"arn:a", "arn:b"},
		},
		{
			name: "no arns available",
			opts: &database.RestoreOptions{Metadata: map[string]interface{}{}},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, resolveBackupARNs(tt.opts))
		})
	}
}

func TestWaitForTableActive_TerminalStatusFails(t *testing.T) {
	mock := &mockDynamoAPI{
		describeTableFunc: func(_ context.Context, _ *dynamodb.DescribeTableInput) (*dynamodb.DescribeTableOutput, error) {
			return &dynamodb.DescribeTableOutput{
				Table: &types.TableDescription{TableStatus: types.TableStatusArchived},
			}, nil
		},
	}
	driver := &DynamoDBDriver{client: mock, tableActivePollInterval: time.Millisecond}

	err := driver.waitForTableActive(context.Background(), "orders_restored")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected status")
}
