package timescaledb

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// ChunkManager handles TimescaleDB chunk operations.
type ChunkManager struct {
	driver *TimescaleDBDriver
}

// NewChunkManager creates a new chunk manager.
func NewChunkManager(driver *TimescaleDBDriver) *ChunkManager {
	return &ChunkManager{
		driver: driver,
	}
}

// ChunkInfo represents information about a TimescaleDB chunk.
type ChunkInfo struct {
	ChunkSchema      string
	ChunkName        string
	TableSchema      string
	TableName        string
	RangeStart       time.Time
	RangeEnd         time.Time
	IsCompressed     bool
	CompressedSize   int64
	UncompressedSize int64
}

// GetChunks returns all chunks for a hypertable.
func (m *ChunkManager) GetChunks(ctx context.Context, hypertable string) ([]ChunkInfo, error) {
	query := `
		SELECT
			chunk_schema,
			chunk_name,
			hypertable_schema,
			hypertable_name,
			range_start,
			range_end,
			is_compressed,
			COALESCE(compressed_total_bytes, 0) as compressed_size,
			COALESCE(uncompressed_total_bytes, 0) as uncompressed_size
		FROM timescaledb_information.chunks
		WHERE hypertable_name = $1
		ORDER BY range_start
	`

	rows, err := m.driver.pool.Query(ctx, query, hypertable)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chunks []ChunkInfo
	for rows.Next() {
		var chunk ChunkInfo
		err := rows.Scan(
			&chunk.ChunkSchema,
			&chunk.ChunkName,
			&chunk.TableSchema,
			&chunk.TableName,
			&chunk.RangeStart,
			&chunk.RangeEnd,
			&chunk.IsCompressed,
			&chunk.CompressedSize,
			&chunk.UncompressedSize,
		)
		if err != nil {
			continue
		}
		chunks = append(chunks, chunk)
	}

	return chunks, rows.Err()
}

// GetChunksByTimeRange returns chunks within a specific time range.
func (m *ChunkManager) GetChunksByTimeRange(ctx context.Context, hypertable string, start, end time.Time) ([]ChunkInfo, error) {
	query := `
		SELECT
			chunk_schema,
			chunk_name,
			hypertable_schema,
			hypertable_name,
			range_start,
			range_end,
			is_compressed,
			COALESCE(compressed_total_bytes, 0) as compressed_size,
			COALESCE(uncompressed_total_bytes, 0) as uncompressed_size
		FROM timescaledb_information.chunks
		WHERE hypertable_name = $1
		AND range_start >= $2
		AND range_end <= $3
		ORDER BY range_start
	`

	rows, err := m.driver.pool.Query(ctx, query, hypertable, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chunks []ChunkInfo
	for rows.Next() {
		var chunk ChunkInfo
		err := rows.Scan(
			&chunk.ChunkSchema,
			&chunk.ChunkName,
			&chunk.TableSchema,
			&chunk.TableName,
			&chunk.RangeStart,
			&chunk.RangeEnd,
			&chunk.IsCompressed,
			&chunk.CompressedSize,
			&chunk.UncompressedSize,
		)
		if err != nil {
			continue
		}
		chunks = append(chunks, chunk)
	}

	return chunks, rows.Err()
}

// DropChunk drops a specific chunk.
func (m *ChunkManager) DropChunk(ctx context.Context, chunkName string) error {
	// Use pgx.Identifier to properly quote the chunk name and prevent SQL injection.
	query := fmt.Sprintf("SELECT drop_chunks(%s)", pgx.Identifier{chunkName}.Sanitize())
	_, err := m.driver.pool.Exec(ctx, query)
	return err
}

// DropOldChunks drops chunks older than a specific time.
func (m *ChunkManager) DropOldChunks(ctx context.Context, hypertable string, olderThan time.Time) (int, error) {
	// Use pgx.Identifier to properly quote the hypertable name; pass timestamp as a parameter.
	query := fmt.Sprintf(
		"SELECT drop_chunks(%s, older_than => $1::timestamptz)",
		pgx.Identifier{hypertable}.Sanitize(),
	)

	var droppedCount int
	err := m.driver.pool.QueryRow(ctx, query, olderThan).Scan(&droppedCount)
	return droppedCount, err
}

// ReorderChunk reorders a chunk to improve compression.
func (m *ChunkManager) ReorderChunk(ctx context.Context, chunkName, indexName string) error {
	// Use pgx.Identifier to properly quote chunk and index names.
	query := fmt.Sprintf(
		"SELECT reorder_chunk(%s, index => %s)",
		pgx.Identifier{chunkName}.Sanitize(),
		pgx.Identifier{indexName}.Sanitize(),
	)
	_, err := m.driver.pool.Exec(ctx, query)
	return err
}

// GetChunkStatistics returns statistics for all chunks.
func (m *ChunkManager) GetChunkStatistics(ctx context.Context) (map[string]interface{}, error) {
	query := `
		SELECT
			COUNT(*) as total_chunks,
			SUM(CASE WHEN is_compressed THEN 1 ELSE 0 END) as compressed_chunks,
			SUM(CASE WHEN NOT is_compressed THEN 1 ELSE 0 END) as uncompressed_chunks,
			SUM(compressed_total_bytes) as total_compressed_size,
			SUM(uncompressed_total_bytes) as total_uncompressed_size
		FROM timescaledb_information.chunks
	`

	var totalChunks, compressedChunks, uncompressedChunks int
	var compressedSize, uncompressedSize *int64

	err := m.driver.pool.QueryRow(ctx, query).Scan(
		&totalChunks,
		&compressedChunks,
		&uncompressedChunks,
		&compressedSize,
		&uncompressedSize,
	)
	if err != nil {
		return nil, err
	}

	stats := map[string]interface{}{
		"total_chunks":        totalChunks,
		"compressed_chunks":   compressedChunks,
		"uncompressed_chunks": uncompressedChunks,
	}

	if compressedSize != nil {
		stats["total_compressed_size"] = *compressedSize
	}
	if uncompressedSize != nil {
		stats["total_uncompressed_size"] = *uncompressedSize
	}

	// Calculate compression ratio
	if compressedSize != nil && uncompressedSize != nil && *uncompressedSize > 0 {
		ratio := float64(*compressedSize) / float64(*uncompressedSize)
		stats["compression_ratio"] = ratio
	}

	return stats, nil
}

// BackupChunk backs up a specific chunk.
func (m *ChunkManager) BackupChunk(ctx context.Context, chunkSchema, chunkName, outputPath string) error {
	// This would use pg_dump to backup just the specific chunk table
	fullChunkName := fmt.Sprintf("%s.%s", chunkSchema, chunkName)

	query := fmt.Sprintf("COPY %s TO '%s' WITH (FORMAT binary)", fullChunkName, outputPath)
	_, err := m.driver.pool.Exec(ctx, query)
	return err
}

// RestoreChunk restores a specific chunk.
func (m *ChunkManager) RestoreChunk(ctx context.Context, chunkSchema, chunkName, inputPath string) error {
	fullChunkName := fmt.Sprintf("%s.%s", chunkSchema, chunkName)

	query := fmt.Sprintf("COPY %s FROM '%s' WITH (FORMAT binary)", fullChunkName, inputPath)
	_, err := m.driver.pool.Exec(ctx, query)
	return err
}
