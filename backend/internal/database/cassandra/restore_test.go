package cassandra

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sanskarpan/db-backup/internal/database"
)

func TestResolveDataDir(t *testing.T) {
	tests := []struct {
		name     string
		driver   *CassandraDriver
		expected string
	}{
		{
			name:     "cassandra default when unset",
			driver:   &CassandraDriver{config: &database.ConnectionConfig{}},
			expected: cassandraDataDir,
		},
		{
			name:     "scylla default when unset",
			driver:   &CassandraDriver{config: &database.ConnectionConfig{}, isScyllaDB: true},
			expected: scyllaDataDir,
		},
		{
			name: "option overrides default",
			driver: &CassandraDriver{config: &database.ConnectionConfig{
				Options: map[string]string{dataDirOption: "/data/cassandra"},
			}},
			expected: "/data/cassandra",
		},
		{
			name: "option overrides scylla default",
			driver: &CassandraDriver{config: &database.ConnectionConfig{
				Options: map[string]string{dataDirOption: "/mnt/scylla"},
			}, isScyllaDB: true},
			expected: "/mnt/scylla",
		},
		{
			name:     "nil config falls back to cassandra default",
			driver:   &CassandraDriver{},
			expected: cassandraDataDir,
		},
		{
			name: "empty option falls back to default",
			driver: &CassandraDriver{config: &database.ConnectionConfig{
				Options: map[string]string{dataDirOption: ""},
			}},
			expected: cassandraDataDir,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.driver.resolveDataDir())
		})
	}
}

func TestStripTableUUID(t *testing.T) {
	assert.Equal(t, "users", stripTableUUID("users-a1b2c3d4e5f6789012345678901234ab"))
	assert.Equal(t, "orders_v2", stripTableUUID("orders_v2-00000000000000000000000000000000"))
	// No UUID suffix: unchanged.
	assert.Equal(t, "users", stripTableUUID("users"))
	// Suffix that is not 32 hex chars: unchanged.
	assert.Equal(t, "users-notauuid", stripTableUUID("users-notauuid"))
}

func TestPlanRestoreEntry(t *testing.T) {
	tests := []struct {
		name     string
		rel      string
		wantDest string
		wantKT   keyspaceTable
		wantOK   bool
	}{
		{
			name:     "snapshot file is flattened into table dir",
			rel:      "myks/users-a1b2c3d4e5f6789012345678901234ab/snapshots/backup_1/md-1-big-Data.db",
			wantDest: filepath.Join("myks", "users-a1b2c3d4e5f6789012345678901234ab", "md-1-big-Data.db"),
			wantKT:   keyspaceTable{keyspace: "myks", table: "users"},
			wantOK:   true,
		},
		{
			name:     "backups file is flattened into table dir",
			rel:      "myks/orders-a1b2c3d4e5f6789012345678901234ab/backups/md-2-big-Data.db",
			wantDest: filepath.Join("myks", "orders-a1b2c3d4e5f6789012345678901234ab", "md-2-big-Data.db"),
			wantKT:   keyspaceTable{keyspace: "myks", table: "orders"},
			wantOK:   true,
		},
		{
			name:     "live sstable path kept as-is",
			rel:      "myks/events-a1b2c3d4e5f6789012345678901234ab/md-3-big-Data.db",
			wantDest: filepath.Join("myks", "events-a1b2c3d4e5f6789012345678901234ab", "md-3-big-Data.db"),
			wantKT:   keyspaceTable{keyspace: "myks", table: "events"},
			wantOK:   true,
		},
		{
			name:   "top-level manifest is not table data",
			rel:    "manifest.json",
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dest, kt, ok := planRestoreEntry(tt.rel)
			assert.Equal(t, tt.wantOK, ok)
			if tt.wantOK {
				assert.Equal(t, tt.wantDest, dest)
				assert.Equal(t, tt.wantKT, kt)
			}
		})
	}
}

func TestNodetoolRefreshArgs(t *testing.T) {
	args := nodetoolRefreshArgs(keyspaceTable{keyspace: "myks", table: "users"})
	assert.Equal(t, []string{"refresh", "myks", "users"}, args)
}

func TestStageRestoreFiles(t *testing.T) {
	backupPath := t.TempDir()
	dataDir := t.TempDir()

	// Build a snapshot-style backup layout.
	files := []string{
		"myks/users-a1b2c3d4e5f6789012345678901234ab/snapshots/backup_1/md-1-big-Data.db",
		"myks/users-a1b2c3d4e5f6789012345678901234ab/snapshots/backup_1/md-1-big-Index.db",
		"myks/orders-b1b2c3d4e5f6789012345678901234ab/snapshots/backup_1/md-1-big-Data.db",
	}
	for _, f := range files {
		full := filepath.Join(backupPath, filepath.FromSlash(f))
		require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
		require.NoError(t, os.WriteFile(full, []byte("sstable"), 0o600))
	}

	driver := &CassandraDriver{}
	tables, err := driver.stageRestoreFiles(backupPath, dataDir)
	require.NoError(t, err)

	// Two distinct tables restored.
	got := make([]string, 0, len(tables))
	for _, kt := range tables {
		got = append(got, kt.keyspace+"."+kt.table)
	}
	sort.Strings(got)
	assert.Equal(t, []string{"myks.orders", "myks.users"}, got)

	// Files should be flattened into the live table directory (no snapshots/ segment).
	staged := filepath.Join(dataDir, "myks", "users-a1b2c3d4e5f6789012345678901234ab", "md-1-big-Data.db")
	_, statErr := os.Stat(staged)
	assert.NoError(t, statErr, "sstable should be staged directly under the table directory")
}

func TestRestore_RequiresBackupPath(t *testing.T) {
	driver := &CassandraDriver{config: &database.ConnectionConfig{}}
	result, err := driver.Restore(t.Context(), &database.RestoreOptions{})
	require.Error(t, err)
	require.NotNil(t, result)
	assert.Equal(t, database.RestoreStatusFailed, result.Status)
}

func TestStreamOperations_Unsupported(t *testing.T) {
	driver := NewCassandraDriver()
	ctx := t.Context()

	assert.Error(t, driver.StreamBackup(ctx, &database.BackupOptions{}, nil))
	assert.Error(t, driver.StreamRestore(ctx, &database.RestoreOptions{}, nil))
}

func TestClusterManager_BackupMultiDC_Unsupported(t *testing.T) {
	cm := NewClusterManager(nil)
	result, err := cm.BackupMultiDC(t.Context(), &database.BackupOptions{})
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "not supported")
}

func TestSSHConnection_Unsupported(t *testing.T) {
	driver := NewCassandraDriver()
	err := driver.testSSHConnection(t.Context())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not supported")
}
