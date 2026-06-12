package compression

import (
	"io"

	"github.com/pierrec/lz4/v4"
	"github.com/sanskarpan/db-backup/internal/types"
	pkgErrors "github.com/sanskarpan/db-backup/pkg/errors"
)

// LZ4Compressor implements LZ4 compression
type LZ4Compressor struct{}

// NewLZ4Compressor creates a new LZ4 compressor
func NewLZ4Compressor() *LZ4Compressor {
	return &LZ4Compressor{}
}

// Compress compresses data using LZ4
func (c *LZ4Compressor) Compress(input io.Reader, output io.Writer, level int) error {
	writer := lz4.NewWriter(output)
	defer writer.Close()

	if _, err := io.Copy(writer, input); err != nil {
		return pkgErrors.ErrCompressionFailed(err)
	}

	if err := writer.Close(); err != nil {
		return pkgErrors.ErrCompressionFailed(err)
	}

	return nil
}

// Decompress decompresses LZ4 data
func (c *LZ4Compressor) Decompress(input io.Reader, output io.Writer) error {
	reader := lz4.NewReader(input)

	if _, err := io.Copy(output, reader); err != nil {
		return pkgErrors.ErrDecompressionFailed(err)
	}

	return nil
}

// GetType returns the compression type
func (c *LZ4Compressor) GetType() types.CompressionType {
	return types.CompressionLZ4
}

// GetExtension returns the file extension
func (c *LZ4Compressor) GetExtension() string {
	return ".lz4"
}
