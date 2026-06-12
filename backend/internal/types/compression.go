// Package types provides common types used across the application
package types

// CompressionType represents the type of compression
type CompressionType string

// Compression type constants
const (
	CompressionNone CompressionType = "none"
	CompressionGzip CompressionType = "gzip"
	CompressionZstd CompressionType = "zstd"
	CompressionLZ4  CompressionType = "lz4"
)
