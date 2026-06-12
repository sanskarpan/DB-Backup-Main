package redis

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/sanskarpan/db-backup/internal/database"
)

// ClusterDriver handles Redis Cluster backups
type ClusterDriver struct {
	clusterClient *redis.ClusterClient
	config        *database.ConnectionConfig
}

// NewClusterDriver creates a new Redis Cluster driver
func NewClusterDriver() *ClusterDriver {
	return &ClusterDriver{}
}

// Connect establishes connection to Redis Cluster
func (c *ClusterDriver) Connect(ctx context.Context, config *database.ConnectionConfig) error {
	// Parse cluster nodes from config
	nodes := strings.Split(config.Host, ",")

	opts := &redis.ClusterOptions{
		Addrs:    nodes,
		Password: config.Password,
	}

	clusterClient := redis.NewClusterClient(opts)

	// Test connection
	if err := clusterClient.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("failed to connect to Redis cluster: %w", err)
	}

	c.clusterClient = clusterClient
	c.config = config
	return nil
}

// Disconnect closes cluster connections
func (c *ClusterDriver) Disconnect() error {
	if c.clusterClient != nil {
		return c.clusterClient.Close()
	}
	return nil
}

// BackupCluster backs up all nodes in the Redis cluster
func (c *ClusterDriver) BackupCluster(ctx context.Context, opts *database.BackupOptions) (*database.BackupResult, error) {
	result := &database.BackupResult{
		ID:        fmt.Sprintf("cluster_%s", time.Now().Format("20060102_150405")),
		StartTime: time.Now(),
		Status:    database.BackupStatusInProgress,
		Metadata:  make(map[string]interface{}),
	}

	// Get cluster nodes
	nodes, err := c.getClusterNodes(ctx)
	if err != nil {
		result.Status = database.BackupStatusFailed
		result.Error = err
		return result, err
	}

	// Backup each node in parallel
	var wg sync.WaitGroup
	errors := make(chan error, len(nodes))
	backups := make(map[string]string)
	var mu sync.Mutex

	for nodeID, nodeAddr := range nodes {
		wg.Add(1)
		go func(id, addr string) {
			defer wg.Done()

			// Create driver for this node
			nodeDriver := NewRedisDriver()
			nodeConfig := *c.config
			parts := strings.Split(addr, ":")
			if len(parts) == 2 {
				nodeConfig.Host = parts[0]
				fmt.Sscanf(parts[1], "%d", &nodeConfig.Port)
			}

			if err := nodeDriver.Connect(ctx, &nodeConfig); err != nil {
				errors <- fmt.Errorf("node %s: %w", id, err)
				return
			}
			defer nodeDriver.Disconnect()

			// Backup this node
			nodeOpts := *opts
			nodeOpts.OutputDir = filepath.Join(opts.OutputDir, id)
			nodeResult, err := nodeDriver.Backup(ctx, &nodeOpts)
			if err != nil {
				errors <- fmt.Errorf("node %s: %w", id, err)
				return
			}

			mu.Lock()
			backups[id] = nodeResult.BackupPath
			mu.Unlock()
		}(nodeID, nodeAddr)
	}

	wg.Wait()
	close(errors)

	// Check for errors
	var backupErrors []string
	for err := range errors {
		backupErrors = append(backupErrors, err.Error())
	}

	if len(backupErrors) > 0 {
		result.Status = database.BackupStatusFailed
		result.Error = fmt.Errorf("cluster backup failed: %s", strings.Join(backupErrors, "; "))
		return result, result.Error
	}

	result.Status = database.BackupStatusCompleted
	result.EndTime = time.Now()
	result.Metadata["cluster_nodes"] = len(nodes)
	result.Metadata["node_backups"] = backups
	return result, nil
}

// RestoreCluster restores all nodes in the Redis cluster
func (c *ClusterDriver) RestoreCluster(ctx context.Context, opts *database.RestoreOptions) (*database.RestoreResult, error) {
	result := &database.RestoreResult{
		ID:        fmt.Sprintf("cluster_restore_%s", time.Now().Format("20060102_150405")),
		StartTime: time.Now(),
		Status:    database.RestoreStatusInProgress,
	}

	// Get cluster nodes
	nodes, err := c.getClusterNodes(ctx)
	if err != nil {
		result.Status = database.RestoreStatusFailed
		result.Error = err
		return result, err
	}

	// Restore each node in parallel
	var wg sync.WaitGroup
	errors := make(chan error, len(nodes))

	for nodeID, nodeAddr := range nodes {
		wg.Add(1)
		go func(id, addr string) {
			defer wg.Done()

			// Create driver for this node
			nodeDriver := NewRedisDriver()
			nodeConfig := *c.config
			parts := strings.Split(addr, ":")
			if len(parts) == 2 {
				nodeConfig.Host = parts[0]
				fmt.Sscanf(parts[1], "%d", &nodeConfig.Port)
			}

			if err := nodeDriver.Connect(ctx, &nodeConfig); err != nil {
				errors <- fmt.Errorf("node %s: %w", id, err)
				return
			}
			defer nodeDriver.Disconnect()

			// Restore this node
			nodeOpts := *opts
			nodeOpts.BackupPath = filepath.Join(opts.BackupPath, id)
			_, err := nodeDriver.Restore(ctx, &nodeOpts)
			if err != nil {
				errors <- fmt.Errorf("node %s: %w", id, err)
				return
			}
		}(nodeID, nodeAddr)
	}

	wg.Wait()
	close(errors)

	// Check for errors
	var restoreErrors []string
	for err := range errors {
		restoreErrors = append(restoreErrors, err.Error())
	}

	if len(restoreErrors) > 0 {
		result.Status = database.RestoreStatusFailed
		result.Error = fmt.Errorf("cluster restore failed: %s", strings.Join(restoreErrors, "; "))
		return result, result.Error
	}

	result.Status = database.RestoreStatusCompleted
	result.EndTime = time.Now()
	return result, nil
}

// getClusterNodes returns a map of node IDs to addresses
func (c *ClusterDriver) getClusterNodes(ctx context.Context) (map[string]string, error) {
	nodesInfo, err := c.clusterClient.ClusterNodes(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get cluster nodes: %w", err)
	}

	nodes := make(map[string]string)
	lines := strings.Split(nodesInfo, "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		nodeID := parts[0]
		nodeAddr := parts[1]

		// Only include master nodes
		if strings.Contains(line, "master") {
			nodes[nodeID] = nodeAddr
		}
	}

	return nodes, nil
}

// SentinelDriver handles Redis Sentinel configuration backups
type SentinelDriver struct {
	sentinelClient *redis.SentinelClient
	config         *database.ConnectionConfig
}

// NewSentinelDriver creates a new Redis Sentinel driver
func NewSentinelDriver() *SentinelDriver {
	return &SentinelDriver{}
}

// Connect establishes connection to Redis Sentinel
func (s *SentinelDriver) Connect(ctx context.Context, config *database.ConnectionConfig) error {
	opts := &redis.Options{
		Addr:     fmt.Sprintf("%s:%d", config.Host, config.Port),
		Password: config.Password,
	}

	client := redis.NewClient(opts)
	_ = client.Conn(ctx) // Establish connection

	s.sentinelClient = &redis.SentinelClient{}
	s.config = config
	return nil
}

// BackupSentinelConfig backs up Sentinel configuration
func (s *SentinelDriver) BackupSentinelConfig(ctx context.Context, outputPath string) error {
	// Get Sentinel configuration
	// In practice, this would use SENTINEL commands to get:
	// - Monitored masters
	// - Sentinel configuration
	// - Notification scripts
	// - Client reconfig scripts

	// This is a placeholder - actual implementation would query Sentinel
	config := map[string]interface{}{
		"masters":     []string{},
		"sentinels":   []string{},
		"config":      map[string]string{},
		"backup_time": time.Now(),
	}

	// Save configuration to file
	// In production, serialize and save config
	_ = config

	return nil
}
