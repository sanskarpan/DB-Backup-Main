package cassandra

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gocql/gocql"

	"github.com/sanskarpan/db-backup/internal/database"
)

// MultiDCDriver handles multi-datacenter Cassandra backups.
type MultiDCDriver struct {
	sessions map[string]*gocql.Session
	config   *database.ConnectionConfig
}

// NewMultiDCDriver creates a new multi-datacenter driver.
func NewMultiDCDriver() *MultiDCDriver {
	return &MultiDCDriver{
		sessions: make(map[string]*gocql.Session),
	}
}

// Connect establishes connections to all datacenters.
func (m *MultiDCDriver) Connect(ctx context.Context, config *database.ConnectionConfig, datacenters map[string][]string) error {
	for dc, hosts := range datacenters {
		cluster := gocql.NewCluster(hosts...)
		cluster.Port = config.Port
		cluster.Keyspace = config.Database
		cluster.Consistency = gocql.LocalQuorum // Use local quorum for DC-aware operations

		if config.Username != "" {
			cluster.Authenticator = gocql.PasswordAuthenticator{
				Username: config.Username,
				Password: config.Password,
			}
		}

		session, err := cluster.CreateSession()
		if err != nil {
			return fmt.Errorf("failed to connect to datacenter %s: %w", dc, err)
		}

		m.sessions[dc] = session
	}

	m.config = config
	return nil
}

// Disconnect closes all datacenter connections.
func (m *MultiDCDriver) Disconnect() error {
	for dc, session := range m.sessions {
		if session != nil {
			session.Close()
			delete(m.sessions, dc)
		}
	}
	return nil
}

// BackupAllDatacenters backs up all datacenters in parallel.
func (m *MultiDCDriver) BackupAllDatacenters(ctx context.Context, opts *database.BackupOptions) (*database.BackupResult, error) {
	result := &database.BackupResult{
		ID:        fmt.Sprintf("multidc_%s", time.Now().Format("20060102_150405")),
		StartTime: time.Now(),
		Status:    database.BackupStatusInProgress,
		Metadata:  make(map[string]interface{}),
	}

	var wg sync.WaitGroup
	errors := make(chan error, len(m.sessions))
	backups := make(map[string]string)
	var mu sync.Mutex

	for dc := range m.sessions {
		wg.Add(1)
		go func(datacenter string) {
			defer wg.Done()

			driver := NewCassandraDriver()
			driver.session = m.sessions[datacenter]
			driver.config = m.config

			dcOpts := *opts
			dcOpts.OutputDir = filepath.Join(opts.OutputDir, datacenter)

			dcResult, err := driver.Backup(ctx, &dcOpts)
			if err != nil {
				errors <- fmt.Errorf("datacenter %s: %w", datacenter, err)
				return
			}

			mu.Lock()
			backups[datacenter] = dcResult.BackupPath
			mu.Unlock()
		}(dc)
	}

	wg.Wait()
	close(errors)

	var backupErrors []string
	for err := range errors {
		backupErrors = append(backupErrors, err.Error())
	}

	if len(backupErrors) > 0 {
		result.Status = database.BackupStatusFailed
		result.Error = fmt.Errorf("multi-DC backup failed: %s", strings.Join(backupErrors, "; "))
		return result, result.Error
	}

	result.Status = database.BackupStatusCompleted
	result.EndTime = time.Now()
	result.Metadata["datacenters"] = len(m.sessions)
	result.Metadata["dc_backups"] = backups
	return result, nil
}

// EnableIncrementalBackups enables incremental backups on all nodes.
func (m *MultiDCDriver) EnableIncrementalBackups(ctx context.Context) error {
	// Use nodetool to enable incremental backups
	cmd := exec.CommandContext(ctx, "nodetool", "enablebackup")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to enable incremental backups: %w, output: %s", err, string(output))
	}
	return nil
}

// DisableIncrementalBackups disables incremental backups.
func (m *MultiDCDriver) DisableIncrementalBackups(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "nodetool", "disablebackup")
	return cmd.Run()
}
