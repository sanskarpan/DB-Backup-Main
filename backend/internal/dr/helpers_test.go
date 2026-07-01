package dr

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// testBackupSQL is a small logical SQL dump used as the "backup" restored into
// a SQLite test target during DR unit tests.
const testBackupSQL = `
-- users table
CREATE TABLE users (id INTEGER PRIMARY KEY, email TEXT NOT NULL);
INSERT INTO users (id, email) VALUES (1, 'a@example.com'), (2, 'b@example.com'), (3, 'c@example.com');

CREATE TABLE orders (id INTEGER PRIMARY KEY, user_id INTEGER, total REAL);
INSERT INTO orders (id, user_id, total) VALUES (1, 1, 9.99), (2, 2, 19.99);
`

// newSQLiteTarget builds a TargetConfig backed by a throwaway on-disk SQLite
// database plus a backup SQL file, wired with realistic validation
// expectations.
func newSQLiteTarget(t *testing.T) *TargetConfig {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "target.db")
	sqlPath := filepath.Join(dir, "backup.sql")
	require.NoError(t, os.WriteFile(sqlPath, []byte(testBackupSQL), 0o600))

	return &TargetConfig{
		Driver:         "sqlite3",
		DSN:            dbPath,
		BackupPath:     sqlPath,
		BackupTime:     time.Now().Add(-30 * time.Second),
		ExpectedTables: []string{"users", "orders"},
		MinRowCounts:   map[string]int64{"users": 3, "orders": 2},
		SampleTables:   []string{"users", "orders"},
	}
}

// testConfigForTarget returns a DR TestConfig wired to the given target with
// all validations enabled and generous thresholds.
func testConfigForTarget(target *TargetConfig) *TestConfig {
	cfg := DefaultTestConfig()
	cfg.Target = target
	return cfg
}

// provisionRestored provisions a test environment and applies the backup so
// callers can validate a genuinely restored database.
func provisionRestored(t *testing.T, target *TargetConfig) (*EnvironmentProvisioner, *TestEnvironment) {
	t.Helper()
	prov := NewEnvironmentProvisioner()
	env, err := prov.ProvisionEnvironment(context.Background(), &ProvisionConfig{
		DatabaseName: "testdb",
		Target:       target,
	})
	require.NoError(t, err)

	script, err := os.ReadFile(target.BackupPath)
	require.NoError(t, err)
	_, err = applySQLScript(context.Background(), env.db, string(script))
	require.NoError(t, err)

	t.Cleanup(func() { _ = prov.CleanupEnvironment(context.Background(), env.ID) })
	return prov, env
}
