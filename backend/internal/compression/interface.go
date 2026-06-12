// Package compression provides compression and decompression functionality
package compression

import (
	"io"

	"github.com/sanskarpan/db-backup/internal/types"
)

// Compressor interface for compression operations
type Compressor interface {
	// Compress compresses the input reader to the output writer
	Compress(input io.Reader, output io.Writer, level int) error

	// Decompress decompresses the input reader to the output writer
	Decompress(input io.Reader, output io.Writer) error

	// GetType returns the compression type
	GetType() types.CompressionType

	// GetExtension returns the file extension for this compression type
	GetExtension() string
}

// NewCompressor creates a new compressor based on the compression type
func NewCompressor(compressionType types.CompressionType) (Compressor, error) {
	switch compressionType {
	case types.CompressionGzip:
		return NewGzipCompressor(), nil
	case types.CompressionZstd:
		return NewZstdCompressor(), nil
	case types.CompressionLZ4:
		return NewLZ4Compressor(), nil
	case types.CompressionNone:
		return NewNoneCompressor(), nil
	default:
		return NewNoneCompressor(), nil
	}
}
