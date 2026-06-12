package timescaledb

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// CompressionManager handles TimescaleDB compression policies
type CompressionManager struct {
	driver *TimescaleDBDriver
}

// NewCompressionManager creates a new compression manager
func NewCompressionManager(driver *TimescaleDBDriver) *CompressionManager {
	return &CompressionManager{
		driver: driver,
	}
}

// CompressionPolicy represents a TimescaleDB compression policy
type CompressionPolicy struct {
	Hypertable        string `json:"hypertable"`
	Schema            string `json:"schema"`
	SegmentBy         string `json:"segment_by"`
	OrderBy           string `json:"order_by"`
	CompressAfter     string `json:"compress_after"`
	CompressionEnabled bool   `json:"compression_enabled"`
}

// BackupCompressionPolicies backs up all compression policies
func (m *CompressionManager) BackupCompressionPolicies(ctx context.Context, outputDir string) error {
	query := `
		SELECT
			ht.schema_name || '.' || ht.table_name as hypertable,
			ht.schema_name,
			d.compress_segmentby,
			d.compress_orderby,
			cp.compress_after,
			ht.compression_state != 0 as compression_enabled
		FROM _timescaledb_catalog.hypertable ht
		LEFT JOIN LATERAL (
			SELECT compress_segmentby, compress_orderby
			FROM _timescaledb_catalog.dimension_info(ht.id)
			LIMIT 1
		) d ON true
		LEFT JOIN LATERAL (
			SELECT config::json->>'compress_after' as compress_after
			FROM _timescaledb_config.bgw_policy
			WHERE hypertable_id = ht.id
			AND proc_name = 'policy_compression'
			LIMIT 1
		) cp ON true
		WHERE ht.compression_state != 0
	`

	rows, err := m.driver.pool.Query(ctx, query)
	if err != nil {
		// If query fails (e.g., no compression policies), just return nil
		return nil
	}
	defer rows.Close()

	var policies []CompressionPolicy
	for rows.Next() {
		var policy CompressionPolicy
		var segmentBy, orderBy, compressAfter *string

		err := rows.Scan(
			&policy.Hypertable,
			&policy.Schema,
			&segmentBy,
			&orderBy,
			&compressAfter,
			&policy.CompressionEnabled,
		)
		if err != nil {
			continue
		}

		if segmentBy != nil {
			policy.SegmentBy = *segmentBy
		}
		if orderBy != nil {
			policy.OrderBy = *orderBy
		}
		if compressAfter != nil {
			policy.CompressAfter = *compressAfter
		}

		policies = append(policies, policy)
	}

	if len(policies) == 0 {
		return nil
	}

	// Save to file
	policyFile := filepath.Join(outputDir, "compression_policies.json")
	file, err := os.Create(policyFile)
	if err != nil {
		return err
	}
	defer file.Close()

	return json.NewEncoder(file).Encode(policies)
}

// RestoreCompressionPolicies restores compression policies from backup
func (m *CompressionManager) RestoreCompressionPolicies(ctx context.Context, backupDir string) error {
	policyFile := filepath.Join(backupDir, "compression_policies.json")

	// Check if file exists
	if _, err := os.Stat(policyFile); os.IsNotExist(err) {
		return nil // No policies to restore
	}

	file, err := os.Open(policyFile)
	if err != nil {
		return err
	}
	defer file.Close()

	var policies []CompressionPolicy
	if err := json.NewDecoder(file).Decode(&policies); err != nil {
		return err
	}

	// Restore each compression policy
	for _, policy := range policies {
		// Enable compression on hypertable
		if policy.CompressionEnabled {
			alterQuery := fmt.Sprintf(
				"ALTER TABLE %s SET (timescaledb.compress",
				policy.Hypertable,
			)

			if policy.SegmentBy != "" {
				alterQuery += fmt.Sprintf(", timescaledb.compress_segmentby = '%s'", policy.SegmentBy)
			}
			if policy.OrderBy != "" {
				alterQuery += fmt.Sprintf(", timescaledb.compress_orderby = '%s'", policy.OrderBy)
			}

			alterQuery += ")"

			if _, err := m.driver.pool.Exec(ctx, alterQuery); err != nil {
				// Continue even if one fails
				continue
			}

			// Add compression policy if compress_after is specified
			if policy.CompressAfter != "" {
				policyQuery := fmt.Sprintf(
					"SELECT add_compression_policy('%s', INTERVAL '%s')",
					policy.Hypertable,
					policy.CompressAfter,
				)

				if _, err := m.driver.pool.Exec(ctx, policyQuery); err != nil {
					// Continue even if one fails
					continue
				}
			}
		}
	}

	return nil
}

// GetCompressedChunks returns information about compressed chunks
func (m *CompressionManager) GetCompressedChunks(ctx context.Context, hypertable string) (int, error) {
	query := `
		SELECT count(*)
		FROM timescaledb_information.chunks
		WHERE hypertable_name = $1
		AND is_compressed = true
	`

	var count int
	err := m.driver.pool.QueryRow(ctx, query, hypertable).Scan(&count)
	return count, err
}

// CompressChunk manually compresses a specific chunk
func (m *CompressionManager) CompressChunk(ctx context.Context, chunkName string) error {
	query := fmt.Sprintf("SELECT compress_chunk('%s')", chunkName)
	_, err := m.driver.pool.Exec(ctx, query)
	return err
}

// DecompressChunk manually decompresses a specific chunk
func (m *CompressionManager) DecompressChunk(ctx context.Context, chunkName string) error {
	query := fmt.Sprintf("SELECT decompress_chunk('%s')", chunkName)
	_, err := m.driver.pool.Exec(ctx, query)
	return err
}
